package miniproxy

import (
	"net"
	"runtime"

	reuseport "github.com/kavu/go_reuseport"
	"github.com/op/go-logging"
	"github.com/panjf2000/ants/v2"
)

var debug bool = false
var logger *logging.Logger = logging.MustGetLogger("Server")

type Server struct {
	listener net.Listener
	addr     string
}

// NewServer create a proxy server
func NewServer(Addr string) *Server {
	return &Server{addr: Addr}
}

// Start a proxy server
func (s *Server) Start() {
	defer ants.Release()
	// 调整线程数为CPU数量
	runtime.GOMAXPROCS(runtime.NumCPU())

	var err error
	s.listener, err = reuseport.Listen("tcp", s.addr)

	if err != nil {
		logger.Fatal(err)
	}

	logger.Infof("proxy listen in %s, waiting for connection...\n", s.addr)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			logger.Error(err)
			continue
		}
		ants.Submit(func() { s.handlerConn(&conn) })
	}
}

// newConn create a conn to serve client request
func (s *Server) handlerConn(rwc *net.Conn) {
	conn := NewConn(s, rwc)
	conn.serve()
}
