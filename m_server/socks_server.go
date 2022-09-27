package m_server

import (
	"errors"
	"github.com/shadowsocks/go-shadowsocks2/socks"
	"github.com/zyong/miniproxygo/m_socks"
	"io"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"time"
)

import (
	"github.com/baidu/go-lib/log"
)

func delayCalc(delay time.Duration) time.Duration {
	if delay == 0 {
		delay = 5 * time.Millisecond
	} else {
		delay *= 2
	}
	if max := 1 * time.Second; delay > max {
		delay = max
	}
	return delay
}

func isTimeout(err error) bool {
	e, ok := err.(net.Error)
	return ok && e.Timeout()
}

func (srv *Server) ServeClient(l net.Listener, server string) error {
	for {
		c, err := l.Accept()
		if err != nil {
			log.Logger.Warn("failed to accept: %s", err)
			continue
		}

		go func() {
			defer c.Close()
			tgt, err := m_socks.HandShake(c)
			if err != nil {

				// UDP: keep the connection until disconnect then free the UDP socket
				if err == socks.InfoUDPAssociate {
					buf := make([]byte, 1)
					// block here
					for {
						_, err := c.Read(buf)
						if err, ok := err.(net.Error); ok && err.Timeout() {
							continue
						}
						log.Logger.Info("UDP Associate End.")
						return
					}
				}

				log.Logger.Warn("failed to get target address: %v", err)
				return
			}

			rc, err := net.Dial("tcp", server)
			if err != nil {
				log.Logger.Warn("failed to connect to server %v: %v", server, err)
				return
			}
			defer rc.Close()
			// 配置的意思是合并写入前几个包
			// 实际逻辑是写缓存，缓存大小为bufSize，然后10s后flush数据
			//if config.TCPCork {
			//	rc = timedCork(rc, 10*time.Millisecond, 1280)
			//}
			// 使用加密算法包装rc来读取和写入数据
			// shadow代表具体传入的生成加密连接实例的函数
			// 如果读到数据就使用算法解密
			// 如果写入数据就先用算法加密
			rc = srv.Cipher.StreamConn(rc)

			// 第一次要写入远端的目标地址
			if _, err = rc.Write(tgt); err != nil {
				log.Logger.Warn("failed to send target address: %v", err)
				return
			}

			log.Logger.Info("proxy %s <-> %s <-> %s", c.RemoteAddr(), server, tgt)

			// 地址写入成功后就开始中继数据
			if err = srv.relay(rc, c); err != nil {
				log.Logger.Warn("relay error: %v", err)
			}
		}()
	}
}


// relay copies between left and right bidirectionally
func (srv *Server) relay(left, right net.Conn) error {
	var err, err1 error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err1 = io.Copy(right, left)
		if (srv.ReadTimeout > 0) {
			right.SetReadDeadline(time.Now().Add(srv.ReadTimeout)) // unblock read on right
		}
	}()
	_, err = io.Copy(left, right)
	if (srv.WriteTimeout > 0) {
		left.SetReadDeadline(time.Now().Add(srv.WriteTimeout)) // unblock read on left
	}
	wg.Wait()
	if err1 != nil && !errors.Is(err1, os.ErrDeadlineExceeded) { // requires Go 1.15+
		return err1
	}
	if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
		return err
	}
	return nil
}

// Serve accepts incoming connections on the Listener l, creating a
// new service goroutine for each.  The service goroutines read requests and
// then call srv.Handler to reply to them.
//
// Params
//     - l  : net listener
//     - raw: underlying tcp listener (different from `l` in HTTPS)
//
// Return
//     - err: error
func (srv *Server) ServeServer(l net.Listener, raw net.Listener, proto string) error {
	var tempDelay time.Duration // how long to sleep on accept failure

	for {
		// accept new connection
		c, e := l.Accept()
		if e != nil {
			if isTimeout(e) {
				continue
			}

			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				tempDelay = delayCalc(tempDelay)

				log.Logger.Error("socks: Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}

			// if in GraceShutdown state, exit accept loop after timeout
			if srv.CheckGracefulShutdown() {
				shutdownTimeout := srv.Config.Server.GracefulShutdownTimeout
				time.Sleep(time.Duration(shutdownTimeout) * time.Second)
			}

			return e
		}

		// start go-routine for new connection
		go func() {
			defer c.Close()
			// create data structure for new connection
			sc := srv.Cipher.StreamConn(c)

			tgt, err := m_socks.ReadAddr(sc)

			if err != nil {
				log.Logger.Warn("socks: failed to get target address from %v: %v", c.RemoteAddr(), err)

				_, err = io.Copy(ioutil.Discard, c)
				if err != nil {
					log.Logger.Warn("socks: failed to discard error: %v", err)
				}
				return
			}

			rc, err := net.Dial("tcp", tgt.String())
			if err != nil {
				log.Logger.Warn("socks: failed to connect to target: %v", err)
				return
			}

			defer rc.Close()

			log.Logger.Info("socks: proxy %s <-> %s", c.RemoteAddr(), tgt)
			if err = srv.relay(sc, rc); err != nil {
				log.Logger.Warn("socks: relay error from %v:%v", c.RemoteAddr(), err)
			}

		}()
	}
}
