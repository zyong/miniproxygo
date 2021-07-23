package miniproxy

import (
	"bytes"
	"net/textproto"
	"strings"
)

type header struct {
	header      textproto.MIMEHeader
	method      string
	host        string
	port        string
	uri         string
	isCompleted bool
}

var (
	lineBuf []byte = make([]byte, 1024)
)

func NewHeader() *header {
	return &header{
		isCompleted: false,
	}
}

/**
*
解析header信息，建立连接
HTTP 请求头如下
GET http://www.baidu.com/ HTTP/1.1
Host: www.baidu.com
User-Agent: curl/7.69.1
Accept: *

Proxy-Connection:Keep-Alive

HTTPS请求头如下
CONNECT www.baidu.com:443 HTTP/1.1
Host: www.baidu.com:443
User-Agent: curl/7.69.1
Proxy-Connection: Keep-Alive
*
*/
func (h *header) ResolveHeader(line string) error {
	s1 := strings.Index(line, " ")
	s2 := strings.Index(line[s1+1:], " ")

	if h.method == "" {
		if s1 != -1 {
			h.method = line[:s1]
			switch strings.ToLower(h.method) {
			case "get":
			case "post":
			case "header":
			case "connect":
			case "patch":
			case "options":
				if s2 != -1 {
					h.uri = line[s1+1 : s1+s2+1]
				} else {
					h.uri = line[s1+1:]
				}
			default:
				logger.Errorf("error line %s\n", line)
				return &BadRequestError{
					what: "bad request",
				}
			}

		} else {
			logger.Errorf("error line %s\n", line)
		}
	}

	return nil
}

func (h *header) SetHeader(header textproto.MIMEHeader) error {
	h.header = header
	host := header.Get("Host")

	s3 := strings.Index(host, ":")
	if s3 != -1 {
		h.host = host[:s3]
	} else {
		h.host = host
	}
	if strings.EqualFold(h.method, "connect") {
		h.port = "443"
	} else {
		h.port = "80"
	}
	if h.method != "" && h.host != "" && h.port != "" {
		h.isCompleted = true
	}
	return nil
}

func (h *header) Remote() (remote string, err error) {
	return h.host + ":" + h.port, nil
}

func (h *header) isHttps() bool {
	// https request
	return strings.EqualFold(h.method, "CONNECT")
}

func (h *header) readLine(in []byte) (line string, err error) {
	reader := bytes.NewReader(in)
	for reader.Len() > 0 {
		b, err := reader.ReadByte()
		lineBuf = append(lineBuf, b)
		if len(lineBuf) >= 2 && bytes.Equal(lineBuf[len(lineBuf)-2:], []byte("\r\n")) {
			line := string(lineBuf[:])
			lineBuf = lineBuf[:cap(lineBuf)]
			return line, err
		}
	}
	return
}
