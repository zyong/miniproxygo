package m_server

import (
	"bufio"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

import (
	"github.com/baidu/go-lib/log"
	"github.com/zyong/miniproxygo/m_socks"
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

// relay copies between left and right bidirectionally
func (srv *Server) relay(left, right net.Conn) error {
	var err, err1 error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err1 = io.Copy(right, left)
		if srv.ReadTimeout > 0 {
			_ = right.SetReadDeadline(time.Now().Add(srv.ReadTimeout)) // unblock read on right
		}
	}()
	_, err = io.Copy(left, right)
	if srv.ReadTimeout > 0 {
		_ = left.SetReadDeadline(time.Now().Add(srv.ReadTimeout)) // unblock read on left
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

type corkedConn struct {
	net.Conn
	bufw   *bufio.Writer
	corked bool
	delay  time.Duration
	err    error
	lock   sync.Mutex
	once   sync.Once
}

func timedCork(c net.Conn, d time.Duration, bufSize int) net.Conn {
	return &corkedConn{
		Conn:   c,
		bufw:   bufio.NewWriterSize(c, bufSize),
		corked: true,
		delay:  d,
	}
}

func (w *corkedConn) Write(p []byte) (int, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	if w.err != nil {
		return 0, w.err
	}
	if w.corked {
		w.once.Do(func() {
			time.AfterFunc(w.delay, func() {
				w.lock.Lock()
				defer w.lock.Unlock()
				w.corked = false
				w.err = w.bufw.Flush()
			})
		})
		return w.bufw.Write(p)
	}
	return w.Conn.Write(p)
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
func (srv *Server) ServeLocal(shadow func(net.Conn) net.Conn, getAddr func(net.Conn) (m_socks.Addr, error)) error {
	var tempDelay time.Duration // how long to sleep on accept failure

	l, err := net.Listen("tcp", srv.Addr)

	if err != nil {
		log.Logger.Warn("socks: failed to listen to %s: %v", srv.Config.Base.Port, err)
		return err
	}

	var start time.Time

	for {
		// accept new connection
		c, e := l.Accept()
		if e != nil {
			if isTimeout(e) {
				continue
			}

			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				tempDelay = delayCalc(tempDelay)

				_ = log.Logger.Error("socks: Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}

			// if in GraceShutdown state, exit accept loop after timeout
			if srv.CheckGracefulShutdown() {
				shutdownTimeout := srv.Config.Base.GracefulShutdownTimeout
				time.Sleep(time.Duration(shutdownTimeout) * time.Second)
			}

			return e
		}

		atomic.AddInt64(&srv.stats.ReqNum, 1)

		// start go-routine for new connection
		go func() {
			defer func() {
				_ = c.Close()
				atomic.AddInt64(&srv.stats.ReqNum, -1)
			}()

			tgt, err := getAddr(c)

			log.Logger.Info("socks: get target address: %s", string(tgt))
			if err != nil {
				_ = log.Logger.Warn("socks: failed to get target address from %v: %v", c.RemoteAddr(), err)

				_, err = io.Copy(ioutil.Discard, c)
				if err != nil {
					_ = log.Logger.Warn("socks: failed to discard error: %v", err)
				}
				return
			}

			start = time.Now()
			// todo add concurrent pool
			rc, err := net.Dial("tcp", srv.Config.Client.RemoteServer)
			if err != nil {
				_ = log.Logger.Warn("socks: failed to connect to RemoteServer: %v", err)
				return
			}
			log.Logger.Info("socks: proxy %s <-> %s, connect elapsed time:%fs, total req num %d",
				c.RemoteAddr(), rc.RemoteAddr(), time.Since(start).Seconds(), srv.stats.ReqNum)

			defer rc.Close()

			rc = timedCork(rc, 10*time.Millisecond, 1280)

			// create data structure for new connection
			// create stream connect instance
			rc = shadow(rc)

			// 一次性打包发送，约定好格式，这样既减少请求次数，也实现了验证用户的目的
			// 用户信息验证可以是缓存验证，也可以是数据库验证
			// 使用16个字节记录用户名、密码，用户名8个字节，密码8个字节

			initBuf := make([]byte, 40+len(tgt))
			n := copy(initBuf[:8], []byte(srv.Config.Client.Username))
			if n < 8 {
				for i := n; i < 8; i++ {
					initBuf[i] = 0x20
				}
			}
			_ = copy(initBuf[8:40], []byte(srv.Config.Client.Password))
			_ = copy(initBuf[40:], tgt)

			if _, err = rc.Write(initBuf); err != nil {
				log.Logger.Warn("socks: failed to send target address: %v", err)
				return
			}
			log.Logger.Info("socks: write user %s succ", srv.Config.Client.Username)

			log.Logger.Info("socks: proxy %s <-> %s", c.RemoteAddr(), tgt)
			if err = srv.relay(rc, c); err != nil {
				_ = log.Logger.Warn("socks: relay error from %v:%v", c.RemoteAddr(), err)
			}

		}()
	}
}

// Listen on addr for incoming connections.
func (srv *Server) ServeServer(shadow func(net.Conn) net.Conn) error {
	l, err := net.Listen("tcp", srv.Addr)

	if err != nil {
		_ = log.Logger.Warn("socks: failed to listen to %s: %v", srv.Config.Base.Port, err)
		return err
	}

	var start time.Time

	for {
		c, err := l.Accept()
		if err != nil {
			_ = log.Logger.Warn("socks: failed to accept: %v", err)
			continue
		}

		atomic.AddInt64(&srv.stats.ReqNum, 1)
		go func() {
			defer c.Close()

			c = timedCork(c, 10*time.Millisecond, 1280)

			sc := shadow(c)

			start = time.Now()
			// todo add user certification
			user, pass, err := m_socks.ReadUserPass(sc)

			if !srv.Accounts.Valid(user, pass) {
				log.Logger.Info("socks: server account valid failed user: %s", user)
				return
			}

			log.Logger.Info("socks: server account valid success user: %s", user)

			tgt, err := m_socks.ReadAddr(sc)
			log.Logger.Info("socks: server read addr elapsed time :%fs", time.Since(start).Seconds())

			if err != nil {
				_ = log.Logger.Warn("socks: failed to get target address from %v: %v", c.RemoteAddr(), err)
				// drain c to avoid leaking server behavioral features
				// see https://www.ndss-symposium.org/ndss-paper/detecting-probe-resistant-proxies/
				_, err = io.Copy(ioutil.Discard, c)
				if err != nil {
					_ = log.Logger.Warn("socks: discard error: %v", err)
				}
				return
			}

			start = time.Now()
			// # todo add dns resolve module
			rc, err := net.Dial("tcp", tgt.String())
			if err != nil {
				_ = log.Logger.Warn("socks: failed to connect to target: %v", err)
				return
			}
			atomic.AddInt64(&srv.stats.ReqNum, 1)

			log.Logger.Info("socks: proxy %s <-> %s, connect elapsed time:%fs, total req num %d",
				c.RemoteAddr(), rc.RemoteAddr(), time.Since(start).Seconds(), srv.stats.ReqNum)

			defer rc.Close()

			if err = srv.relay(sc, rc); err != nil {
				_ = log.Logger.Warn("socks: relay error: %v", err)
			}
		}()
	}
}
