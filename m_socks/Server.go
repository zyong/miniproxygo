package m_socks

import (
	"errors"
	"net"
)

import (
	cache "github.com/patrickmn/go-cache"
	"github.com/txthinking/runnergroup"
)

var (
	ErrUnsupportCmd = errors.New("Unsupport Command")
	ErrUserPassAuth = errors.New("Invalid Username or Password for Auth")
)

type Server struct {
	UserName string
	Password	string
	Method 		byte
	SupportedCommand []byte
	TCPAddr *net.TCPAddr
	UDPAddr *net.UDPAddr
	ServerAddr *net.UDPAddr
	TCPListen *net.TCPListener
	UDPConn *net.UDPConn
	UDPExchanges *cache.Cache
	TCPTimeout	int
	UDPTimeout	int
	Handle 		Handler
	AssociatedUDP	*cache.Cache
	UDPSrc		*cache.Cache
	RunnerGroup	*runnergroup.RunnerGroup
	LimitUDP bool
}

type UDPExchange struct {
	ClientAddr *net.UDPAddr
	RemoteConn *net.UDPConn
}

func NewClassicServer(addr, ip, username, password string, tcpTimeout, udpTimeout int) (*Server, error) {
	_, p, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	taddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}

	uaddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	saddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(ip, p))
	if err != nil {
		return nil, err
	}

	m := MethodNone
	if username
}