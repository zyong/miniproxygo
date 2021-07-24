package miniproxy

import (
	"net"

	"github.com/op/go-logging"
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
	var err error
	s.listener, err = net.Listen("tcp", s.addr)

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
		s.handlerConn(conn)
	}
}

// newConn create a conn to serve client request
func (s *Server) handlerConn(rwc net.Conn) {
	conn := NewConn(s, rwc)
	go conn.serve()
}
