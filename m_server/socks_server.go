package m_server

import (
	"errors"
	"io"
	"io/ioutil"
	"net"
	"os"
	"sync"
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
func (srv *Server) ServeLocal(l net.Listener, shadow func(net.Conn) net.Conn, getAddr func(net.Conn) (m_socks.Addr, error)) error {
	var tempDelay time.Duration // how long to sleep on accept failure

	l, err := net.Listen("tcp", srv.Addr)

	if err != nil {
		log.Logger.Warn("socks: failed to listen to %s: %v", srv.Config.Server.Port, err)
		return err
	}

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

			tgt, err := getAddr(c)

			if err != nil {
				log.Logger.Warn("socks: failed to get target address from %v: %v", c.RemoteAddr(), err)

				_, err = io.Copy(ioutil.Discard, c)
				if err != nil {
					log.Logger.Warn("socks: failed to discard error: %v", err)
				}
				return
			}

			rc, err := net.Dial("tcp", srv.Config.Server.RemoteServer)
			if err != nil {
				log.Logger.Warn("socks: failed to connect to RemoteServer: %v", err)
				return
			}

			defer rc.Close()

			// create data structure for new connection
			rc = shadow(rc)

			if _, err = rc.Write(tgt); err != nil {
				log.Logger.Warn("socks: failed to send target address: %v", err)
				return
			}

			log.Logger.Info("socks: proxy %s <-> %s", c.RemoteAddr(), tgt)
			if err = srv.relay(rc, c); err != nil {
				log.Logger.Warn("socks: relay error from %v:%v", c.RemoteAddr(), err)
			}

		}()
	}
}


// Listen on addr for incoming connections.
func (srv *Server) ServeServer(l net.Listener, shadow func(net.Conn) net.Conn) error {
	l, err := net.Listen("tcp", srv.Addr)

	if err != nil {
		log.Logger.Warn("socks: failed to listen to %s: %v", srv.Config.Server.Port, err)
		return err
	}

	for {
		c, err := l.Accept()
		if err != nil {
			log.Logger.Warn("socks: failed to accept: %v", err)
			continue
		}

		go func() {
			defer c.Close()

			sc := shadow(c)

			tgt, err := m_socks.ReadAddr(sc)
			if err != nil {
				log.Logger.Warn("socks: failed to get target address from %v: %v", c.RemoteAddr(), err)
				// drain c to avoid leaking server behavioral features
				// see https://www.ndss-symposium.org/ndss-paper/detecting-probe-resistant-proxies/
				_, err = io.Copy(ioutil.Discard, c)
				if err != nil {
					log.Logger.Warn("socks: discard error: %v", err)
				}
				return
			}

			rc, err := net.Dial("tcp", tgt.String())
			log.Logger.Info("socks: proxy %s <-> %s", c.RemoteAddr(), tgt)

			if err != nil {
				log.Logger.Warn("socks: failed to connect to target: %v", err)
				return
			}
			defer rc.Close()

			if err = srv.relay(sc, rc); err != nil {
				log.Logger.Warn("socks: relay error: %v", err)
			}
		}()
	}
}