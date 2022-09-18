package m_socks

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/op/go-logging"
)

// 进度
const (
	STATUS_VERIFY  uint = 0
	STATUS_CONNECT uint = 1
)

// 版本
const (
	V5  uint = 0
	V4  uint = 1
	V4A uint = 2
)

// 命令类型
const (
	CONNECT      uint = 0
	BIND         uint = 1
	UDPASSOCIATE uint = 2
)

// 地址类型
const (
	IPV4     uint = 0
	IPV6     uint = 1
	HOSTNAME uint = 2
)

var logger *logging.Logger = logging.MustGetLogger("httpHandler")

type request struct {
	conn    *net.Conn
	version uint
	command uint
	host    string
	port    string
}

func NewRequest(conn *net.Conn) *request {
	return &request{
		conn: conn,
	}
}

// 提供服务给socks
func (r *request) Serv() {
	defer func() {
		err := recover()
		if err != nil {
			fmt.Println("HERE")
			fmt.Println(err)
			fmt.Println(0)
		}
	}()

	defer (*r.conn).Close()

	if r.Resolve(STATUS_VERIFY) {
		r.ResponseVerify()
	}

	if r.Resolve(STATUS_CONNECT) {
		r.ResponseConnect()
		remoteConn := r.connect()
		defer remoteConn.Close()
		r.Relay(r.conn, &remoteConn)
	}

}

// 提供socks数据解析
// 1、第一次解析连接信息
// 2、第二次解析请求
func (r *request) Resolve(status uint) bool {
	var b [1024]byte
	n, err := (*r.conn).Read(b[:])
	if err != nil {
		logger.Panic(err)
	}

	// read length less 4 byte is bad,
	// m_socks init 4byte
	if n < 3 {
		logger.Panicf("m_socks request read byte length small: %d", n)
	}

	if status == STATUS_VERIFY {
		//只处理Socks5协议
		if b[0] == 0x05 {
			r.version = 5
		} else {
			logger.Panic("m_socks request other version protocol")
		}
	}

	if status == STATUS_CONNECT {
		// 获取命令
		switch b[1] {
		case 0x01:
			r.command = CONNECT
		case 0x02:
			r.command = BIND
		case 0x03:
			r.command = UDPASSOCIATE
		}

		// 地址类型
		switch b[3] {
		case 0x01:
			r.host = net.IPv4(b[4], b[5], b[6], b[7]).String()
		case 0x03:
			r.host = string(b[5 : n-2])
		case 0x04:
			r.host = net.IP{b[4], b[5], b[6], b[7], b[8], b[9],
				b[10], b[11], b[12], b[13], b[14], b[15], b[16],
				b[17], b[18], b[19]}.String()
		}
		r.port = strconv.Itoa(int(b[n-2])<<8 | int(b[n-1]))

	}
	return true
}

// 根据需求做不同的应答
func (r *request) ResponseVerify() {
	if r.version == 5 {
		(*r.conn).Write([]byte{0x05, 0x00})
	}
}

// 提供连接结果返回数据
func (r *request) ResponseConnect() {
	(*r.conn).Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00}) //响应客户端连接成功

}

// 提供中继操作
func (r *request) Relay(localConn *net.Conn, remoteConn *net.Conn) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for {
			logger.Debug("loop read remote write local\n")
			bs, err := r.read(remoteConn)
			if err != nil || err == io.EOF {
				goto noloop
			}
			logger.Debugf("loop read read %s byte\n", string(bs))

			if len(bs) > 0 {
				_, err = (*localConn).Write(bs)
				logger.Debugf("write local conn %v\n", string(bs))
			}
			if err != nil || err == io.EOF {
				goto noloop
			}
		}
	noloop:
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		// (*localConn).SetReadDeadline(time.Now().Add(10 * time.Second))
		// (*remoteConn).SetWriteDeadline(time.Now().Add(10 * time.Second))
		for {
			logger.Debug("loop write remote read local\n")
			bs, err := r.read(localConn)
			if err != nil || err == io.EOF {
				goto noloop
			}
			logger.Debugf("loop read local %s byte\n", string(bs))

			if len(bs) > 0 {
				_, err = (*remoteConn).Write(bs)
				logger.Debugf("write remote conn %v\n", string(bs))
			}

			if err != nil || err == io.EOF {
				goto noloop
			}
		}
	noloop:
		wg.Done()
	}()
	wg.Wait()
}

// 提供请求结果返回数据
func (r *request) connect() net.Conn {
	remoteconn, err := net.Dial("tcp", net.JoinHostPort(r.host, r.port))
	if err != nil {
		logger.Panicf("connect remote failed:%v", err)
	}
	return remoteconn
}

func (c *request) read(conn *net.Conn) (buf []byte, err error) {
	bs := make([]byte, 50)
	nr, err := (*conn).Read(bs)
	if err != nil {
		return
	}
	buf = append(buf, bs[:nr]...)
	return
}
