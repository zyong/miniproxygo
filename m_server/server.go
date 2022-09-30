package m_server

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

import (
	"github.com/baidu/go-lib/log"
	"github.com/zyong/miniproxygo/m_config"
	"github.com/zyong/miniproxygo/m_core"
	"github.com/zyong/miniproxygo/m_socks"
)

type Stats struct {
	ReqNum int64
	CoNum  int64
	Users  int32
}

type client struct {
	cid      uint64
	username string
	conn     net.Conn
}

type Server struct {
	Addr                    string
	Cipher                  m_core.Cipher
	ReadTimeout             time.Duration // maximum duration before timing out read of the request
	WriteTimeout            time.Duration // maximum duration before timing out write of the response
	TlsHandshakeTimeout     time.Duration // maximum duration before timing out handshake
	GracefulShutdownTimeout time.Duration // maximum duration before timing out graceful shutdown

	// CloseNotifyCh allow detecting when the server in graceful shutdown state
	CloseNotifyCh chan bool

	listener net.Listener

	gcid    uint64
	clients map[uint64]*client

	connWaitGroup sync.WaitGroup // waits for server conns to finish

	Config   m_config.Conf
	ConfRoot string

	stats   Stats
	Version string // version of bfe server

}

// NewServer create a proxy m_server
func NewServer(cfg m_config.Conf, confRoot string, version string) *Server {
	s := new(Server)

	s.Config = cfg
	s.ConfRoot = confRoot
	s.InitConfig()

	s.CloseNotifyCh = make(chan bool)

	s.stats.ReqNum = 0
	s.stats.CoNum = 0
	s.Version = version

	return s
}

// Start a proxy client
func Start(cfg m_config.Conf, version string, confRoot string) error {
	var err error

	s := NewServer(cfg, confRoot, version)

	// 选择一个加密算法，可以不加密？和简单密码
	ciph, err := m_core.PickCipher(s.Config.Server.Cipher, []byte{}, "password")
	if err != nil {
		return err
	}
	s.Cipher = ciph

	serveChan := make(chan error)

	if s.Config.Server.Local {
		go func() {
			err := s.ServeSocksLocal()
			serveChan <- err
		}()
	} else {
		go func() {
			err := s.ServeSocksServer()
			serveChan <- err
		}()
	}

	err = <-serveChan
	return err
}

func (s *Server) ServeSocksLocal() (err error) {
	log.Logger.Info("Start: SOCKS proxy local %s <-> %s", s.Addr, s.Config.Server.RemoteServer)
	shadow := s.Cipher.StreamConn
	return s.ServeLocal(s.listener, shadow, func(c net.Conn) (m_socks.Addr, error) { return m_socks.HandShake(c) })
}

// newConn create a conn to serve client request
func (s *Server) ServeSocksServer() (err error) {
	log.Logger.Info("Start: SOCKS proxy server %s", s.Addr)
	shadow := s.Cipher.StreamConn
	return s.ServeServer(s.listener, shadow)
}

// InitConfig set some parameter based on config.
func (srv *Server) InitConfig() {
	// set service port, according to config
	srv.Addr = fmt.Sprintf(":%d", srv.Config.Server.Port)

	// set ReadTimeout
	if srv.Config.Server.ClientReadTimeout != 0 {
		srv.ReadTimeout = time.Duration(srv.Config.Server.ClientReadTimeout) * time.Second
	}

	// set GracefulShutdownTimeout
	srv.GracefulShutdownTimeout = time.Duration(srv.Config.Server.GracefulShutdownTimeout) * time.Second
}

func (srv *Server) InitSocks() (err error) {
	// initialize socks proto handlers
	// initialize socks
	return nil
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
