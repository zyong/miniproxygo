package miniproxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/textproto"
)

var requestLine string

type conn struct {
	localConn net.Conn
	server    *Server
	header    *header
}

func NewConn(s *Server, rwc net.Conn) *conn {
	return &conn{
		server:    s,
		localConn: rwc,
		header:    NewHeader(),
	}
}

// serve tunnel the client connection to remote host
// if conn doesn't close，we should loop read
func (c *conn) serve() {
	// 读取消息体
	c.readMessage()

	// rebuild http request header
	var rawReqHeader bytes.Buffer
	remote, _ := c.header.Remote()
	// GET http://www.baidu.com/
	rawReqHeader.WriteString(requestLine + "\r\n")
	for k, vs := range c.header.header {
		for _, v := range vs {
			rawReqHeader.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
		}
	}
	rawReqHeader.WriteString("\r\n")

	// 解析tunnel
	logger.Info("connecting to " + remote)
	remoteConn, err := c.connect(remote)
	if err != nil {
		logger.Errorf("connect error %v\n", err)
		return
	}

	if c.header.isHttps() {
		// if https, should sent 200 to client
		_, err = c.localConn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		if err != nil {
			logger.Error(err)
			return
		}
	} else {
		// if not https, should sent the request header to remote
		writer := bufio.NewWriter(remoteConn)
		writer.Write(rawReqHeader.Bytes())
		err := writer.Flush()
		if err != nil {
			logger.Error(err)
			return
		}
	}

	// build bidirectional-streams
	logger.Info("begin tunnel", c.localConn.RemoteAddr(), "<->", remote)
	c.Relay(remoteConn, c.localConn)
	logger.Info("stop tunnel", c.localConn.RemoteAddr(), "<->", remote)
}

/**
 * 读取消息可能分几次读取，一次获取到的消息可能不完整
 */
func (c *conn) readMessage() error {
	reader := bufio.NewReader(c.localConn)
	tpReader := textproto.NewReader(reader)
	requestLine, err := tpReader.ReadLine()
	if err != nil {
		return err
	}
	c.header.ResolveHeader(requestLine)
	header, _ := tpReader.ReadMIMEHeader()
	c.header.SetHeader(header)

	// 由于ReadLineBytes返回不带end-of-line
	// 这里直接添加header解析,避免反复读取

	return nil
}

func (c *conn) Relay(remoteConn net.Conn, localConn net.Conn) {
	go func() {
		for {
			reader := bufio.NewReader(remoteConn)
			writer := bufio.NewWriter(localConn)
			if reader.Size() > 0 {
				_, err := io.Copy(writer, reader)
				if err != nil {
					logger.Errorf("err %v\n", err)
				}
			}
		}
	}()

	go func() {
		for {
			reader := bufio.NewReader(localConn)
			writer := bufio.NewWriter(remoteConn)
			if reader.Size() > 0 {
				_, err := io.Copy(writer, reader)
				if err != nil {
					logger.Errorf("err %v\n", err)
				}
			}
		}
	}()

	// reader := bufio.NewReader(remoteConn)
	// var buf []byte = make([]byte, 4096)
	// for {
	// 	_, err := reader.Read(buf)
	// 	if err != nil {
	// 		if err == io.EOF {
	// 			break
	// 		}
	// 	}
	// 	localConn.Write(buf)
	// 	logger.Info(buf)
	// }
}

// tunnel http message between client and server
func (c *conn) connect(remote string) (remoteConn net.Conn, err error) {
	remoteConn, err = net.Dial("tcp", remote)
	if err != nil {
		logger.Error(err)
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
