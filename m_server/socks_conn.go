package m_server

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"
)


// This should be >= 512 bytes for DetectContentType,
// but otherwise it's somewhat arbitrary.
const bufferBeforeChunkingSize = 512

// Actions to do with current connection.
const (
	// Reuse the connection (default aciton)
	keepAlive = iota
	// Close the connection after send response to client.
	closeAfterReply
	// Close the connection directly, do not send any data,
	// it usually means some attacks may happened.
	closeDirectly
)

var errTooLarge = errors.New("http: request too large")

// A switchReader can have its Reader changed at runtime.
// It's not safe for concurrent Reads and switches.
type switchReader struct {
	io.Reader
}

// A liveSwitchReader is a switchReader that's safe for concurrent
// reads and switches, if its mutex is held.
type liveSwitchReader struct {
	sync.Mutex
	r io.Reader
}

func (sr *liveSwitchReader) Read(p []byte) (n int, err error) {
	sr.Lock()
	r := sr.r
	sr.Unlock()
	return r.Read(p)
}


// A conn represents the server side of an HTTP/HTTPS connection.
type conn struct {
	// immutable:
	remoteAddr string             // network address of remote side
	server     *Server         // the Server on which the connection arrived
	rwc        net.Conn           // i/o connection

	// for http/https:
	sr    liveSwitchReader      // where the LimitReader reads from; usually the rwc
	lr    *io.LimitedReader     // io.LimitReader(sr)
	reqSN uint32                //number of requests arrived on this connection

	mu           sync.Mutex // guards the following
	clientGone   bool       // if client has disconnected mid-request
	closeNotifyc chan bool  // made lazily
}

func (c *conn) closeNotify() <-chan bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closeNotifyc == nil {
		c.closeNotifyc = make(chan bool, 1)

		pr, pw := io.Pipe()

		readSource := c.sr.r
		c.sr.Lock()
		c.sr.r = pr
		c.sr.Unlock()
		go func() {
			_, err := io.Copy(pw, readSource)
			if err == nil {
				err = io.EOF
			}
			pw.CloseWithError(err)
			c.noteClientGone()
		}()
	}
	return c.closeNotifyc
}

func (c *conn) noteClientGone() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closeNotifyc != nil && !c.clientGone {
		c.closeNotifyc <- true
	}
	c.clientGone = true
}

// noLimit is an effective infinite upper bound for io.LimitedReader
const noLimit int64 = (1 << 63) - 1

// Create new connection from rwc.
func newConn(rwc net.Conn, srv *Server) (c *conn, err error) {
	c = new(conn)
	c.remoteAddr = rwc.RemoteAddr().String()
	c.server = srv
	c.rwc = rwc
	c.sr = liveSwitchReader{r: c.rwc}
	c.lr = io.LimitReader(&c.sr, noLimit).(*io.LimitedReader)
	c.reqSN = 0

	return c, nil
}


// rstAvoidanceDelay is the amount of time we sleep after closing the
// write side of a TCP connection before closing the entire socket.
// By sleeping, we increase the chances that the client sees our FIN
// and processes its final data before they process the subsequent RST
// from closing a connection with known unread data.
// This RST seems to occur mostly on BSD systems. (And Windows?)
// This timeout is somewhat arbitrary (~latency around the planet).
const rstAvoidanceDelay = 500 * time.Millisecond
