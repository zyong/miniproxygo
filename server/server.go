package server

import (
	"net"
	"runtime"
	"strconv"

	reuseport "github.com/kavu/go_reuseport"
	"github.com/op/go-logging"
	"github.com/panjf2000/ants/v2"
	"github.com/zyong/miniproxygo/http"
	"github.com/zyong/miniproxygo/socks"
)

var logger *logging.Logger = logging.MustGetLogger("Server")

type Server struct {
	listener        net.Listener
	isReuseport     bool
	addr            string
	isGoroutinepool bool
}

// NewServer create a proxy server
func NewServer() *Server {
	return &Server{}
}

func (s *Server) Bind(addr string) {
	s.addr = addr
}

func (s *Server) WithNumCPU() {
	// 调整线程数为CPU数量
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func (s *Server) WithReusePort() {
	// 是否重用端口
	s.isReuseport = true
}

func (s *Server) WithGoroutinePool() {
	s.isGoroutinepool = true
}

// Start a proxy server
func (s *Server) Start() {
	if s.isGoroutinepool {
		defer ants.Release()
	}

	var err error
	if s.isReuseport {
		s.listener, err = reuseport.Listen("tcp", s.addr)
	} else {
		s.listener, err = net.Listen("tcp", s.addr)
	}

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
		if s.isGoroutinepool {
			ants.Submit(func() { s.serv(&conn) })
		} else {
			go s.serv(&conn)
		}
	}
}

// newConn create a conn to serve client request
func (s *Server) serv(conn *net.Conn) {
	hostport := (*conn).LocalAddr().String()
	_, sport, _ := net.SplitHostPort(hostport)

	port, _ := strconv.Atoi(sport)
	if port == 8080 {
		request := socks.NewRequest(conn)
		request.Serv()
	} else if port == 10000 {
		handler := http.NewConn(conn)
		handler.Serve()
	}
}
