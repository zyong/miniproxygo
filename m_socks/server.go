package m_socks

import (
	"errors"
	"fmt"
	"github.com/baidu/go-lib/log"
	"github.com/zyong/miniproxygo/m_debug"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"time"
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
	if username != "" && password != "" {
		m = MethodUsernamePassword
	}

	cs := cache.New(cache.NoExpiration, cache.NoExpiration)
	cs1 := cache.New(cache.NoExpiration, cache.NoExpiration)
	cs2 := cache.New(cache.NoExpiration, cache.NoExpiration)

	s := &Server{
		Method:	m,
		UserName:  username,
		Password: password,
		SupportedCommand: []byte{CmdConnect, CmdUDP},
		TCPAddr: taddr,
		UDPAddr: uaddr,
		ServerAddr: saddr,
		UDPExchanges: cs,
		TCPTimeout: tcpTimeout,
		UDPTimeout: udpTimeout,
		AssociatedUDP: cs1,
		UDPSrc: cs2,
		RunnerGroup: runnergroup.New(),
	}
	return s, nil
}

var (
	// ErrVersion is version errror
	ErrVersion = errors.New("Invalid Version")
	// ErrUserPassVersion is username/ password auth error
	ErrUserPassVersion = errors.New("Invalid Version of Username Password Auth")
	// ErrBadRequest is bad request error
	ErrBadRequest = errors.New("Bad Request")
)

// NewNegotiationRequestFrom read negotiation request packet from client
func NewNegotiationRequestFrom(r io.Reader) (*NegotiationRequest, error) {
	// memory strict
	bb := make([]byte, 2)
	if _, err := io.ReadFull(r, bb); err != nil {
		return nil, err
	}
	if bb[0] != Ver {
		return nil, ErrVersion
	}
	if bb[1] == 0 {
		return nil, ErrBadRequest
	}

	ms := make([]byte, int(bb[1]))
	if _, err := io.ReadFull(r, ms); err != nil {
		return nil, err
	}
	if m_debug.DebugIsOpen {
		log.Logger.Info("Got NegotiationRequest: %#V %#v %#v\n", bb[0], bb[1], ms)
	}

	return &NegotiationRequest{
		Ver: bb[0],
		NMethods: bb[1],
		Methods: ms,
	},nil

}

// NewNegotiationReply return negotiation reply packet can be writed into client
func NewNegotiationReply(method byte) *NegotiationReply {
	return &NegotiationReply{
		Ver:    Ver,
		Method: method,
	}
}

// WriteTo write negotiation reply packet into client
func (r *NegotiationReply) WriteTo(w io.Writer) (int64, error) {
	var n int
	i, err := w.Write([]byte{r.Ver, r.Method})
	n = n + i
	if err != nil {
		return int64(n), err
	}
	if m_debug.DebugIsOpen {
		log.Logger.Info("Sent NegotiationReply: %#v %#v\n", r.Ver, r.Method)
	}
	return int64(n), nil
}


// NewUserPassNegotiationRequestFrom read user password negotiation request packet from client
func NewUserPassNegotiationRequestFrom(r io.Reader) (*UserPassNegotiationRequest, error) {
	bb := make([]byte, 2)
	if _, err := io.ReadFull(r, bb); err != nil {
		return nil, err
	}
	if bb[0] != UserPassVer {
		return nil, ErrUserPassVersion
	}
	if bb[1] == 0 {
		return nil, ErrBadRequest
	}
	ub := make([]byte, int(bb[1])+1)
	if _, err := io.ReadFull(r, ub); err != nil {
		return nil, err
	}
	if ub[int(bb[1])] == 0 {
		return nil, ErrBadRequest
	}
	p := make([]byte, int(ub[int(bb[1])]))
	if _, err := io.ReadFull(r, p); err != nil {
		return nil, err
	}
	if m_debug.DebugIsOpen {
		log.Logger.Info("Got UserPassNegotiationRequest: %#v %#v %#v %#v %#v\n", bb[0], bb[1], ub[:int(bb[1])], ub[int(bb[1])], p)
	}
	return &UserPassNegotiationRequest{
		Ver:    bb[0],
		Ulen:   bb[1],
		Uname:  ub[:int(bb[1])],
		Plen:   ub[int(bb[1])],
		Passwd: p,
	}, nil
}


// NewUserPassNegotiationReply return negotiation username password reply packet can be writed into client
func NewUserPassNegotiationReply(status byte) *UserPassNegotiationReply {
	return &UserPassNegotiationReply{
		Ver:    UserPassVer,
		Status: status,
	}
}


// WriteTo write negotiation username password reply packet into client
func (r *UserPassNegotiationReply) WriteTo(w io.Writer) (int64, error) {
	var n int
	i, err := w.Write([]byte{r.Ver, r.Status})
	n = n + i
	if err != nil {
		return int64(n), err
	}
	if m_debug.DebugIsOpen {
		log.Logger.Info("Sent UserPassNegotiationReply: %#v %#v \n", r.Ver, r.Status)
	}
	return int64(n), nil
}


// NewRequestFrom read requst packet from client
func NewRequestFrom(r io.Reader) (*Request, error) {
	bb := make([]byte, 4)
	if _, err := io.ReadFull(r, bb); err != nil {
		return nil, err
	}
	if bb[0] != Ver {
		return nil, ErrVersion
	}
	var addr []byte
	if bb[3] == ATYPIPv4 {
		addr = make([]byte, 4)
		if _, err := io.ReadFull(r, addr); err != nil {
			return nil, err
		}
	} else if bb[3] == ATYPIPv6 {
		addr = make([]byte, 16)
		if _, err := io.ReadFull(r, addr); err != nil {
			return nil, err
		}
	} else if bb[3] == ATYPDomain {
		dal := make([]byte, 1)
		if _, err := io.ReadFull(r, dal); err != nil {
			return nil, err
		}
		if dal[0] == 0 {
			return nil, ErrBadRequest
		}
		addr = make([]byte, int(dal[0]))
		if _, err := io.ReadFull(r, addr); err != nil {
			return nil, err
		}
		addr = append(dal, addr...)
	} else {
		return nil, ErrBadRequest
	}
	port := make([]byte, 2)
	if _, err := io.ReadFull(r, port); err != nil {
		return nil, err
	}
	if m_debug.DebugIsOpen {
		log.Logger.Info("Got Request: %#v %#v %#v %#v %#v %#v\n", bb[0], bb[1], bb[2], bb[3], addr, port)
	}
	return &Request{
		Ver:     bb[0],
		Cmd:     bb[1],
		Rsv:     bb[2],
		Atyp:    bb[3],
		DstAddr: addr,
		DstPort: port,
	}, nil
}

// NewReply return reply packet can be writed into client, bndaddr should not have domain length
func NewReply(rep byte, atyp byte, bndaddr []byte, bndport []byte) *Reply {
	if atyp == ATYPDomain {
		bndaddr = append([]byte{byte(len(bndaddr))}, bndaddr...)
	}
	return &Reply{
		Ver:     Ver,
		Rep:     rep,
		Rsv:     0x00,
		Atyp:    atyp,
		BndAddr: bndaddr,
		BndPort: bndport,
	}
}


// WriteTo write reply packet into client
func (r *Reply) WriteTo(w io.Writer) (int64, error) {
	var n int
	i, err := w.Write([]byte{r.Ver, r.Rep, r.Rsv, r.Atyp})
	n = n + i
	if err != nil {
		return int64(n), err
	}
	i, err = w.Write(r.BndAddr)
	n = n + i
	if err != nil {
		return int64(n), err
	}
	i, err = w.Write(r.BndPort)
	n = n + i
	if err != nil {
		return int64(n), err
	}
	if m_debug.DebugIsOpen {
		log.Logger.Info("Sent Reply: %#v %#v %#v %#v %#v %#v\n", r.Ver, r.Rep, r.Rsv, r.Atyp, r.BndAddr, r.BndPort)
	}
	return int64(n), nil
}


// Negotiate handle negotiate packet.
// Error or OK both replied
func (s *Server) Negotiate(rw io.ReadWriter) error {
	rq, err := NewNegotiationRequestFrom(rw)
	if err != nil {
		return err
	}
	var got bool
	var m byte
	for _, m = range rq.Methods {
		if m == s.Method {
			got = true
		}
	}

	rp := NewNegotiationReply(s.Method)
	if !got {
		rp.Method = MethodUnsupportAll
	}
	if _, err := rp.WriteTo(rw); err != nil {
		return err
	}

	if s.Method == MethodUsernamePassword {
		urq, err := NewUserPassNegotiationRequestFrom(rw)
		if err != nil {
			return err
		}
		if string(urq.Uname) != s.UserName || string(urq.Passwd) != s.Password {
			urp := NewUserPassNegotiationReply(UserPassStatusFailure)
			if _, err := urp.WriteTo(rw); err != nil {
				return err
			}
			return ErrUserPassAuth
		}
		urp := NewUserPassNegotiationReply(UserPassStatusSuccess)
		if _, err := urp.WriteTo(rw); err != nil {
			return err
		}
	}
	return nil
}

// GetRequest get request packet from client, and check command according to SupportedCommands
// Error replied.
func (s *Server) GetRequest(rw io.ReadWriter) (*Request, error) {
	r, err := NewRequestFrom(rw)
	if err != nil {
		return nil, err
	}
	var supported bool
	for _, c := range s.SupportedCommand {
		if r.Cmd == c {
			supported = true
			break
		}
	}

	if !supported {
		var p *Reply
		if r.Atyp == ATYPIPv4 || r.Atyp == ATYPDomain {
			p = NewReply(RepCommandNotSupported, ATYPIPv4, []byte{0x00, 0x00, 0x00, 0x00}, []byte{0x00, 0x00})
		} else {
			p = NewReply(RepCommandNotSupported, ATYPIPv6, []byte(net.IPv6zero), []byte{0x00, 0x00})
		}
		if _, err := p.WriteTo(rw); err != nil {
			return nil, err
		}
		return nil, ErrUnsupportCmd
	}
	return r, nil
}


func NewDatagramFromBytes(bb []byte) (*Datagram, error) {
	n := len(bb)
	minl := 4
	if n < minl {
		return nil, ErrBadRequest
	}
	var addr []byte
	if bb[3] == ATYPIPv4 {
		minl += 4
		if n < minl {
			return nil, ErrBadRequest
		}
		addr = bb[minl-4 : minl]
	} else if bb[3] == ATYPIPv6 {
		minl += 16
		if n < minl {
			return nil, ErrBadRequest
		}
		addr = bb[minl-16 : minl]
	} else if bb[3] == ATYPDomain {
		minl += 1
		if n < minl {
			return nil, ErrBadRequest
		}
		l := bb[4]
		if l == 0 {
			return nil, ErrBadRequest
		}
		minl += int(l)
		if n < minl {
			return nil, ErrBadRequest
		}
		addr = bb[minl-int(l) : minl]
		addr = append([]byte{l}, addr...)
	} else {
		return nil, ErrBadRequest
	}
	minl += 2
	if n <= minl {
		return nil, ErrBadRequest
	}
	port := bb[minl-2 : minl]
	data := bb[minl:]
	d := &Datagram{
		Rsv:     bb[0:2],
		Frag:    bb[2],
		Atyp:    bb[3],
		DstAddr: addr,
		DstPort: port,
		Data:    data,
	}
	if m_debug.DebugIsOpen {
		log.Logger.Debug("Got Datagram. data: %#v %#v %#v %#v %#v %#v datagram address: %#v\n", d.Rsv, d.Frag, d.Atyp, d.DstAddr, d.DstPort, d.Data, d.Address())
	}
	return d, nil
}

// NewDatagram return datagram packet can be writed into client, dstaddr should not have domain length
func NewDatagram(atyp byte, dstaddr []byte, dstport []byte, data []byte) *Datagram {
	if atyp == ATYPDomain {
		dstaddr = append([]byte{byte(len(dstaddr))}, dstaddr...)
	}
	return &Datagram{
		Rsv:     []byte{0x00, 0x00},
		Frag:    0x00,
		Atyp:    atyp,
		DstAddr: dstaddr,
		DstPort: dstport,
		Data:    data,
	}
}

// Bytes return []byte
func (d *Datagram) Bytes() []byte {
	b := make([]byte, 0)
	b = append(b, d.Rsv...)
	b = append(b, d.Frag)
	b = append(b, d.Atyp)
	b = append(b, d.DstAddr...)
	b = append(b, d.DstPort...)
	b = append(b, d.Data...)
	return b
}


// Handler handle tcp, udp request
type Handler interface {
	// Request has not been replied yet
	TCPHandle(*Server, *net.TCPConn, *Request) error
	UDPHandle(*Server, *net.UDPAddr, *Datagram) error
}

// DefaultHandle implements Handler interface
type DefaultHandle struct {
}

// TCPHandle auto handle request. You may prefer to do yourself.
func (h *DefaultHandle) TCPHandle(s *Server, c *net.TCPConn, r *Request) error {
	if r.Cmd == CmdConnect {
		rc, err := r.Connect(c)
		if err != nil {
			return err
		}
		defer rc.Close()
		go func() {
			var bf [1024 * 2]byte
			for {
				if s.TCPTimeout != 0 {
					if err := rc.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
						return
					}
				}
				i, err := rc.Read(bf[:])
				if err != nil {
					return
				}
				if _, err := c.Write(bf[0:i]); err != nil {
					return
				}
			}
		}()
		var bf [1024 * 2]byte
		for {
			if s.TCPTimeout != 0 {
				if err := c.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
					return nil
				}
			}
			i, err := c.Read(bf[:])
			if err != nil {
				return nil
			}
			if _, err := rc.Write(bf[0:i]); err != nil {
				return nil
			}
		}
		return nil
	}
	if r.Cmd == CmdUDP {
		caddr, err := r.UDP(c, s.ServerAddr)
		if err != nil {
			return err
		}
		ch := make(chan byte)
		defer close(ch)
		s.AssociatedUDP.Set(caddr.String(), ch, -1)
		defer s.AssociatedUDP.Delete(caddr.String())
		io.Copy(ioutil.Discard, c)
		if m_debug.DebugIsOpen {
			log.Logger.Debug("A tcp connection that udp %#v associated closed\n", caddr.String())
		}
		return nil
	}
	return ErrUnsupportCmd
}

// UDPHandle auto handle packet. You may prefer to do yourself.
func (h *DefaultHandle) UDPHandle(s *Server, addr *net.UDPAddr, d *Datagram) error {
	src := addr.String()
	var ch chan byte
	if s.LimitUDP {
		any, ok := s.AssociatedUDP.Get(src)
		if !ok {
			return fmt.Errorf("This udp address %s is not associated with tcp", src)
		}
		ch = any.(chan byte)
	}
	send := func(ue *UDPExchange, data []byte) error {
		select {
		case <-ch:
			return fmt.Errorf("This udp address %s is not associated with tcp", src)
		default:
			_, err := ue.RemoteConn.Write(data)
			if err != nil {
				return err
			}
			if m_debug.DebugIsOpen {
				log.Logger.Debug("Sent UDP data to remote. client: %#v server: %#v remote: %#v data: %#v\n", ue.ClientAddr.String(), ue.RemoteConn.LocalAddr().String(), ue.RemoteConn.RemoteAddr().String(), data)
			}
		}
		return nil
	}

	dst := d.Address()
	var ue *UDPExchange
	iue, ok := s.UDPExchanges.Get(src + dst)
	if ok {
		ue = iue.(*UDPExchange)
		return send(ue, d.Data)
	}

	if m_debug.DebugIsOpen {
		log.Logger.Debug("Call udp: %#v\n", dst)
	}
	var laddr *net.UDPAddr
	any, ok := s.UDPSrc.Get(src + dst)
	if ok {
		laddr = any.(*net.UDPAddr)
	}
	raddr, err := net.ResolveUDPAddr("udp", dst)
	if err != nil {
		return err
	}
	rc, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		if !strings.Contains(err.Error(), "address already in use") {
			return err
		}
		rc, err = net.DialUDP("udp", nil, raddr)
		if err != nil {
			return err
		}
		laddr = nil
	}
	if laddr == nil {
		s.UDPSrc.Set(src+dst, rc.LocalAddr().(*net.UDPAddr), -1)
	}
	ue = &UDPExchange{
		ClientAddr: addr,
		RemoteConn: rc,
	}
	if m_debug.DebugIsOpen {
		log.Logger.Debug("Created remote UDP conn for client. client: %#v server: %#v remote: %#v\n", addr.String(), ue.RemoteConn.LocalAddr().String(), d.Address())
	}
	if err := send(ue, d.Data); err != nil {
		ue.RemoteConn.Close()
		return err
	}
	s.UDPExchanges.Set(src+dst, ue, -1)
	go func(ue *UDPExchange, dst string) {
		defer func() {
			ue.RemoteConn.Close()
			s.UDPExchanges.Delete(ue.ClientAddr.String() + dst)
		}()
		var b [65507]byte
		for {
			select {
			case <-ch:
				if m_debug.DebugIsOpen {
					log.Logger.Debug("The tcp that udp address %s associated closed\n", ue.ClientAddr.String())
				}
				return
			default:
				if s.UDPTimeout != 0 {
					if err := ue.RemoteConn.SetDeadline(time.Now().Add(time.Duration(s.UDPTimeout) * time.Second)); err != nil {
						log.Logger.Info(err)
						return
					}
				}
				n, err := ue.RemoteConn.Read(b[:])
				if err != nil {
					return
				}
				if m_debug.DebugIsOpen {
					log.Logger.Debug("Got UDP data from remote. client: %#v server: %#v remote: %#v data: %#v\n", ue.ClientAddr.String(), ue.RemoteConn.LocalAddr().String(), ue.RemoteConn.RemoteAddr().String(), b[0:n])
				}
				a, addr, port, err := ParseAddress(dst)
				if err != nil {
					log.Logger.Info(err)
					return
				}
				d1 := NewDatagram(a, addr, port, b[0:n])
				if _, err := s.UDPConn.WriteToUDP(d1.Bytes(), ue.ClientAddr); err != nil {
					return
				}
				if m_debug.DebugIsOpen {
					log.Logger.Debug("Sent Datagram. client: %#v server: %#v remote: %#v data: %#v %#v %#v %#v %#v %#v datagram address: %#v\n", ue.ClientAddr.String(), ue.RemoteConn.LocalAddr().String(), ue.RemoteConn.RemoteAddr().String(), d1.Rsv, d1.Frag, d1.Atyp, d1.DstAddr, d1.DstPort, d1.Data, d1.Address())
				}
			}
		}
	}(ue, dst)
	return nil
}



// RunTCPServer starts tcp server
func (s *Server) RunTCPServer() error {
	var err error
	s.TCPListen, err = net.ListenTCP("tcp", s.TCPAddr)
	if err != nil {
		return err
	}
	defer s.TCPListen.Close()
	for {
		c, err := s.TCPListen.AcceptTCP()
		if err != nil {
			return err
		}
		go func(c *net.TCPConn) {
			defer c.Close()
			if s.TCPTimeout != 0 {
				if err := c.SetDeadline(time.Now().Add(time.Duration(s.TCPTimeout) * time.Second)); err != nil {
					log.Logger.Info(err)
					return
				}
			}
			if err := s.Negotiate(c); err != nil {
				log.Logger.Info(err)
				return
			}
			r, err := s.GetRequest(c)
			if err != nil {
				log.Logger.Info(err)
				return
			}
			if err := s.Handle.TCPHandle(s, c, r); err != nil {
				log.Logger.Info(err)
			}
		}(c)
	}
	return nil
}

// RunUDPServer starts udp server
func (s *Server) RunUDPServer() error {
	var err error
	s.UDPConn, err = net.ListenUDP("udp", s.UDPAddr)
	if err != nil {
		return err
	}
	defer s.UDPConn.Close()
	for {
		b := make([]byte, 65507)
		n, addr, err := s.UDPConn.ReadFromUDP(b)
		if err != nil {
			return err
		}
		go func(addr *net.UDPAddr, b []byte) {
			d, err := NewDatagramFromBytes(b)
			if err != nil {
				log.Logger.Info(err)
				return
			}
			if d.Frag != 0x00 {
				log.Logger.Info("Ignore frag", d.Frag)
				return
			}
			if err := s.Handle.UDPHandle(s, addr, d); err != nil {
				log.Logger.Info(err)
				return
			}
		}(addr, b[0:n])
	}
	return nil
}

// Stop server
func (s *Server) Shutdown() error {
	return s.RunnerGroup.Done()
}

// Run server
func (s *Server) ListenAndServe(h Handler) error {
	if h == nil {
		s.Handle = &DefaultHandle{}
	} else {
		s.Handle = h
	}

	s.RunnerGroup.Add(&runnergroup.Runner{
		Start: func() error {
			return s.RunTCPServer()
		},
		Stop: func() error {
			if s.TCPListen != nil {
				return s.TCPListen.Close()
			}
			return nil
		},
	})
	s.RunnerGroup.Add(&runnergroup.Runner{
		Start: func() error {
			return s.RunUDPServer()
		},
		Stop: func() error {
			if s.UDPConn != nil {
				return s.UDPConn.Close()
			}
			return nil
		},
	})
	return s.RunnerGroup.Wait()
}