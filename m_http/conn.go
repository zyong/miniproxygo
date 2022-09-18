package m_http

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"sync"
	"time"
)

var requestLine string

type httpHandler struct {
	localConn *net.Conn
	header    *header
}

func NewConn(rwc *net.Conn) *httpHandler {
	return &httpHandler{
		localConn: rwc,
		header:    NewHeader(),
	}
}

// Serve tunnel the client connection to remote host
// if conn doesn't close，we should loop read
func (c *httpHandler) Serve() {
	defer (*c.localConn).Close()
	// 读取消息体
	err := c.readMessage()
	if err != nil {
		//logger.Errorf("read message err %v\n", err)
		return
	}

	// rebuild m_http request header
	var rawReqHeader bytes.Buffer
	remote, _ := c.header.Remote()
	//logger.Infof("request %s \n", remote)

	// GET http://www.baidu.com/
	rawReqHeader.WriteString(requestLine + "\r\n")
	for k, vs := range c.header.header {
		for _, v := range vs {
			rawReqHeader.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
		}
	}
	rawReqHeader.WriteString("\r\n")
	//logger.Debugf("raw req header %v\n", rawReqHeader.String())

	// 解析tunnel
	//logger.Info("connecting to " + remote)
	remoteConn, err := c.connect(remote)
	if err != nil {
		//logger.Errorf("connect error %v\n", err)
		return
	}
	defer remoteConn.Close()

	if c.header.isHttps() {
		// if https, should sent 200 to client
		_, err = (*c.localConn).Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		if err != nil {
			//logger.Error(err)
			return
		}
	} else {
		// if not https, should sent the request header to remote
		_, err := rawReqHeader.WriteTo(remoteConn)
		if err != nil {
			//logger.Error(err)
			return
		}
	}

	// build bidirectional-streams
	//logger.Info("begin tunnel", (*c.localConn).RemoteAddr(), "<->", remote)
	c.Relay(&remoteConn, c.localConn)
	//logger.Info("stop tunnel", (*c.localConn).RemoteAddr(), "<->", remote)
}

/**
 * 读取消息可能分几次读取，一次获取到的消息可能不完整
 */
func (c *httpHandler) readMessage() error {
	reader := bufio.NewReader(*c.localConn)
	// 读取完成header
	// readHeader(reader)
	tpReader := textproto.NewReader(reader)
	var err error
	requestLine, err = tpReader.ReadLine()
	if err != nil {
		return err
	}
	err = c.header.ResolveHeader(requestLine)
	if err != nil {
		return err
	}
	header, _ := tpReader.ReadMIMEHeader()
	c.header.SetHeader(header)

	// 由于ReadLineBytes返回不带end-of-line
	// 这里直接添加header解析,避免反复读取

	return nil
}

func (c *httpHandler) Relay(remoteConn *net.Conn, localConn *net.Conn) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		// reader := bufio.NewReader(remoteConn)
		// writer := bufio.NewWriter(localConn)
		// remoteConn.SetReadDeadline(time.Now().Add(10 * time.Second))
		// localConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		for {
			//logger.Debug("loop read remote write local\n")
			bs, err := c.read(remoteConn)
			if err != nil || err == io.EOF {
				goto noloop
			}
			//logger.Debugf("loop read read %s byte\n", string(bs))

			if len(bs) > 0 {
				_, err = (*localConn).Write(bs)
				//logger.Debugf("write local conn %v\n", string(bs))
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
			//logger.Debug("loop write remote read local\n")
			bs, err := c.read(localConn)
			if err != nil || err == io.EOF {
				goto noloop
			}
			//logger.Debugf("loop read local %s byte\n", string(bs))

			if len(bs) > 0 {
				_, err = (*remoteConn).Write(bs)
				//logger.Debugf("write remote conn %v\n", string(bs))
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

// tunnel m_http message between client and m_server
func (c *httpHandler) connect(remote string) (remoteConn net.Conn, err error) {
	remoteConn, err = net.DialTimeout("tcp", remote, 5*time.Second)
	if err != nil {
		//logger.Error(err)
		return
	}
	return
}

type BadRequestError struct {
	what string
}

func (b *BadRequestError) Error() string {
	return b.what
}

func (c *httpHandler) read(conn *net.Conn) (buf []byte, err error) {
	bs := make([]byte, 50)
	nr, err := (*conn).Read(bs)
	if err != nil {
		return
	}
	buf = append(buf, bs[:nr]...)
	return
}

func readHeader(reader *bufio.Reader) {
	var buf bytes.Buffer
	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			//logger.Errorf("reader read byte err %v\n", err)
			break
		}
		buf.WriteByte(b)
	}
	//logger.Infof("%v\n", buf.String())
	format(buf)
}

// 一行8个
func format(b bytes.Buffer) {
	n := b.Len()
	lines := n / 8
	bs := b.Bytes()
	for i := 0; i < lines; i++ {
		for j := 0; j < 8; j++ {
			if i*8+j >= b.Len() {
				goto Loop
			}
			fmt.Printf("%d ", bs[i*8+j])
		}
		fmt.Println()
	}
Loop:
	return
}
