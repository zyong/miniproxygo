package handler

import "github.com/panjf2000/gnet"

type httpCodec struct {
	req request
}

func (hc *httpCodec) Encode(c gnet.Conn, buf []byte) (out []byte, err error) {
	if c.Context() == nil {
		return buf, nil
	}
	return appendResp(out, "500 Error", "", errMsg+"\n"), nil
}

func (hc *httpCodec) Decode(c gnet.Conn) (out []byte, err error) {
	// Read  all data from ring buffer
	buf := c.Read()
	// 清空buf
	c.ResetBuffer()

	// process the pipeline
	var leftover []byte
pipeline:
	leftover, err = parseReq(buf, &hc.req)
	// bad thing happened
	if err != nil {
		c.SetContext(err)
		return nil, err
	} else if len(leftover) == len(buf) {
		// request not ready, yet
		return
	}
	out = appendHandle(out, res)
	buf = leftover
	goto pipeline
}
