package m_server

import (
	"net"
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

// ServeHttp accept incoming http connections
func (srv *Server) ServeHttp(ln net.Listener) error {
	return srv.Serve(ln, ln, "SOCKS")
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
func (srv *Server) Serve(l net.Listener, raw net.Listener, proto string) error {
	var tempDelay time.Duration // how long to sleep on accept failure

	for {
		// accept new connection
		rw, e := l.Accept()
		if e != nil {
			if isTimeout(e) {
				continue
			}

			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				tempDelay = delayCalc(tempDelay)

				log.Logger.Error("http: Accept error: %v; retrying in %v", e, tempDelay)
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
		go func(rwc net.Conn, srv *Server) {
			// create data structure for new connection
			c, err := newConn(rw, srv)
			if err != nil {
				// current, here is unreachable
				return
			}

			// process new connection
			c.serve()
		}(rw, srv)
	}
}
