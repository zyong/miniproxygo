package m_server

import (
	"fmt"
	"github.com/baidu/go-lib/log"
	"net"
	"os"
	"sync"
	"time"
)

import (
	"github.com/zyong/miniproxygo/m_config"
)

type Server struct {
	Addr		string
	ReadTimeout             time.Duration // maximum duration before timing out read of the request
	WriteTimeout            time.Duration // maximum duration before timing out write of the response
	TlsHandshakeTimeout     time.Duration // maximum duration before timing out handshake
	GracefulShutdownTimeout time.Duration // maximum duration before timing out graceful shutdown

	// CloseNotifyCh allow detecting when the server in graceful shutdown state
	CloseNotifyCh chan bool

	listener        net.Listener

	connWaitGroup	sync.WaitGroup // waits for server conns to finish

	Config		m_config.Conf
	ConfRoot	string

	Version string // version of bfe server

}

// NewServer create a proxy m_server
func NewServer(cfg m_config.Conf, confRoot string, version string) *Server {
	s := new(Server)

	s.Config = cfg
	s.ConfRoot = confRoot
	s.InitConfig()

	s.CloseNotifyCh = make(chan bool)

	s.Version = version

	return s
}

// Start a proxy m_server
func Start(cfg m_config.Conf, version string, confRoot string) error {
	var err error

	s := NewServer(cfg, confRoot, version)

	// initial http
	//if err = s.InitHttp(); err != nil {
	//	log.Logger.Error("Start: InitHttp():%s", err.Error())
	//	return err
	//}

	// initial Socks
	if err = s.InitSocks(); err != nil {
		log.Logger.Error("Start: InitSocks():%s", err.Error())
		return err
	}

	serveChan := make(chan error)
	go func() {
		err := s.ServeSocks()
		serveChan <- err
	}()

	err = <-serveChan
	return err
}


// InitConfig set some parameter based on config.
func (srv *Server) InitConfig() {
	// set service port, according to config
	srv.Addr = fmt.Sprintf(":%d", srv.Config.Server.Port)

	// set TlsHandshakeTimeout
	if srv.Config.Server.TlsHandshakeTimeout != 0 {
		srv.TlsHandshakeTimeout = time.Duration(srv.Config.Server.TlsHandshakeTimeout) * time.Second
	}

	// set ReadTimeout
	if srv.Config.Server.ClientReadTimeout != 0 {
		srv.ReadTimeout = time.Duration(srv.Config.Server.ClientReadTimeout) * time.Second
	}

	// set GracefulShutdownTimeout
	srv.GracefulShutdownTimeout = time.Duration(srv.Config.Server.GracefulShutdownTimeout) * time.Second
}


func (srv *Server) InitHttp() (err error) {
	// initialize http next proto handlers

	return nil
}

func (srv *Server) InitSocks() (err error) {
	// initialize socks proto handlers
	// initialize socks
	return nil
}

func (srv *Server) InitListeners(config m_config.Conf) (err error) {
	srv.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", config.Server.Port))
	if err != nil {
		return err
	}
	// wrap underlying listener according to balancer type
	log.Logger.Info("InitListeners(): begin to listen [:%d]", config.Server.Port)
	return err
}

// newConn create a conn to serve client request
func (s *Server) ServeSocks() (err error) {
	return s.Serve(s.listener, s.listener, "tcp")
}


// ShutdownHandler is signal handler for QUIT
func (srv *Server) ShutdownHandler(sig os.Signal) {
	shutdownTimeout := srv.Config.Server.GracefulShutdownTimeout
	log.Logger.Info("get signal %s, graceful shutdown in %ds", sig, shutdownTimeout)

	// notify that server is in graceful shutdown state
	close(srv.CloseNotifyCh)

	// close server listeners
	srv.closeListeners()

	// waits server conns to finish
	connFinCh := make(chan bool)
	go func() {
		srv.connWaitGroup.Wait()
		connFinCh <- true
	}()

	shutdownTimer := time.After(time.Duration(shutdownTimeout) * time.Second)

Loop:
	for {
		select {
		// waits server conns to finish
		case <-connFinCh:
			log.Logger.Info("graceful shutdown success.")
			break Loop

		// wait for shutdown timeout
		case <-shutdownTimer:
			log.Logger.Info("graceful shutdown timeout.")
			break Loop
		}
	}

	// shutdown server
	log.Logger.Close()
	os.Exit(0)
}

func (srv *Server) closeListeners() {
	if err := srv.listener.Close(); err != nil {
		log.Logger.Error("closeListeners(): %s, %s", err, srv.listener.Addr())
	}
}

// CheckGracefulShutdown check wether the server is in graceful shutdown state.
func (srv *Server) CheckGracefulShutdown() bool {
	select {
	case <-srv.CloseNotifyCh:
		return true
	default:
		return false
	}
}
