package handler

import (
	"log"
	"miniproxygo"

	"github.com/panjf2000/gnet"
)

type ServerHandler struct {
	*gnet.EventServer
}

var (
	res         string
	errMsg      = "Internal Server Error"
	errMsgBytes = []byte(errMsg)
)

func (sh *ServerHandler) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	log.Printf("HTTP server is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumEventLoop)
	return
}

func (sh *ServerHandler) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	if c.Context() != nil {
		// bad thing happened
		out = errMsgBytes
		action = gnet.Close
		return
	}
	// handle the request
	out = frame
	res = "Hello World!\r\n"
	out = appendHandle(out, res)

	return
}

// appendHandle handles the incoming request and appends the response to
// the provided bytes, which is then returned to the caller.
func appendHandle(b []byte, res string) []byte {
	return miniproxygo.AppendResp(b, "200 OK", "", res)
}
