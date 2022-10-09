package m_shadow

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"net"
)

import (
	"github.com/zyong/miniproxygo/m_internal"
)

// payloadSizeMask is the maximum size of payload in bytes.
const payloadSizeMask = 0x3FFF // 16*1024 - 1

type writer struct {
	io.Writer
	cipher.AEAD
	nonce []byte
	buf   []byte
}

// NewWriter wraps an io.Writer with AEAD encryption.
func NewWriter(w io.Writer, aead cipher.AEAD) io.Writer { return newWriter(w, aead) }

func newWriter(w io.Writer, aead cipher.AEAD) *writer {
	return &writer{
		Writer: w,
		AEAD:   aead,
		buf:    make([]byte, 2+aead.Overhead()+payloadSizeMask+aead.Overhead()),
		nonce:  make([]byte, aead.NonceSize()),
	}
}

// Write encrypts b and writes to the embedded io.Writer.
func (w *writer) Write(b []byte) (int, error) {
	n, err := w.ReadFrom(bytes.NewBuffer(b))
	return int(n), err
}

// ReadFrom reads from the given io.Reader until EOF or error, encrypts and
// writes to the embedded io.Writer. Returns number of bytes read from r and
// any error encountered.
func (w *writer) ReadFrom(r io.Reader) (n int64, err error) {
	for {
		buf := w.buf
		// 空出来的两个字节是留给写数据长度的
		// Overhead 给出plaintext和ciphertext的最大长度差
		// payloadSizeMask 是负载字节数
		payloadBuf := buf[2+w.Overhead() : 2+w.Overhead()+payloadSizeMask]
		// nr实际读取到的字节数
		nr, er := r.Read(payloadBuf)

		if nr > 0 {
			n += int64(nr)
			// 计算实际有数据的bytes数组
			buf = buf[:2+w.Overhead()+nr+w.Overhead()]
			payloadBuf = payloadBuf[:nr]
			// 填入空出的两个字节，写入实际数据长度
			buf[0], buf[1] = byte(nr>>8), byte(nr) // big-endian payload size
			// 产生加密数据，使用buf变量存储加密数据
			// nonce是NonceSize的随机字节数组
			w.Seal(buf[:0], w.nonce, buf[:2], nil)
			increment(w.nonce)

			w.Seal(payloadBuf[:0], w.nonce, payloadBuf, nil)
			increment(w.nonce)

			_, ew := w.Writer.Write(buf)
			if ew != nil {
				err = ew
				break
			}
		}

		if er != nil {
			if er != io.EOF { // ignore EOF as per io.ReaderFrom contract
				err = er
			}
			break
		}
	}

	return n, err
}

type reader struct {
	io.Reader
	cipher.AEAD
	nonce    []byte
	buf      []byte
	leftover []byte
}

// NewReader wraps an io.Reader with AEAD decryption.
func NewReader(r io.Reader, aead cipher.AEAD) io.Reader { return newReader(r, aead) }

func newReader(r io.Reader, aead cipher.AEAD) *reader {
	return &reader{
		Reader: r,
		AEAD:   aead,
		buf:    make([]byte, payloadSizeMask+aead.Overhead()),
		nonce:  make([]byte, aead.NonceSize()),
	}
}

// read and decrypt a record into the internal buffer. Return decrypted payload length and any error encountered.
func (r *reader) read() (int, error) {
	// decrypt payload size
	buf := r.buf[:2+r.Overhead()]
	_, err := io.ReadFull(r.Reader, buf)
	if err != nil {
		return 0, err
	}

	// 解密buf， 将结果写入buf
	_, err = r.Open(buf[:0], r.nonce, buf, nil)
	// 初始nonce为全0byte数组
	// increment是每次对第一个字节加1，如果第一个字节不等于1了就返回
	// 如果第一个字节再次等于0，就对第二个字节加1
	// littleEndian编码 左边最小
	increment(r.nonce)
	if err != nil {
		return 0, err
	}

	size := (int(buf[0])<<8 + int(buf[1])) & payloadSizeMask

	// decrypt payload
	buf = r.buf[:size+r.Overhead()]
	_, err = io.ReadFull(r.Reader, buf)
	if err != nil {
		return 0, err
	}

	_, err = r.Open(buf[:0], r.nonce, buf, nil)
	increment(r.nonce)
	if err != nil {
		return 0, err
	}

	return size, nil
}

// Read reads from the embedded io.Reader, decrypts and writes to b.
func (r *reader) Read(b []byte) (int, error) {
	// copy decrypted bytes (if any) from previous record first
	if len(r.leftover) > 0 {
		n := copy(b, r.leftover)
		r.leftover = r.leftover[n:]
		return n, nil
	}

	n, err := r.read()
	// 实际拷贝数量和读取到的数量不一致
	m := copy(b, r.buf[:n])
	if m < n { // insufficient len(b), keep leftover for next read
		r.leftover = r.buf[m:n]
	}
	return m, err
}

// WriteTo reads from the embedded io.Reader, decrypts and writes to w until
// there's no more data to write or when an error occurs. Return number of
// bytes written to w and any error encountered.
func (r *reader) WriteTo(w io.Writer) (n int64, err error) {
	// write decrypted bytes left over from previous record
	for len(r.leftover) > 0 {
		nw, ew := w.Write(r.leftover)
		r.leftover = r.leftover[nw:]
		n += int64(nw)
		if ew != nil {
			return n, ew
		}
	}

	for {
		nr, er := r.read()
		if nr > 0 {
			nw, ew := w.Write(r.buf[:nr])
			n += int64(nw)

			if ew != nil {
				err = ew
				break
			}
		}

		if er != nil {
			if er != io.EOF { // ignore EOF as per io.Copy contract (using src.WriteTo shortcut)
				err = er
			}
			break
		}
	}

	return n, err
}

// increment little-endian encoded unsigned integer b. Wrap around on overflow.
// 相当于对4个字节的无符号整数累加1
func increment(b []byte) {
	for i := range b {
		b[i]++
		if b[i] != 0 {
			return
		}
	}
}

type streamConn struct {
	net.Conn
	Cipher
	r *reader
	w *writer
}

func (c *streamConn) initReader() error {
	// make byte array by salt size
	salt := make([]byte, c.SaltSize())
	// ReadFull exactly read byte array in salt size
	if _, err := io.ReadFull(c.Conn, salt); err != nil {
		return err
	}
	aead, err := c.Decrypter(salt)
	if err != nil {
		return err
	}

	// 通过bloomfilter 检查
	if m_internal.CheckSalt(salt) {
		return ErrRepeatedSalt
	}

	// 通过将对称加解密算法和套接字绑定
	c.r = newReader(c.Conn, aead)
	return nil
}

func (c *streamConn) Read(b []byte) (int, error) {
	if c.r == nil {
		if err := c.initReader(); err != nil {
			return 0, err
		}
	}
	return c.r.Read(b)
}

func (c *streamConn) WriteTo(w io.Writer) (int64, error) {
	if c.r == nil {
		if err := c.initReader(); err != nil {
			return 0, err
		}
	}
	return c.r.WriteTo(w)
}

func (c *streamConn) initWriter() error {
	salt := make([]byte, c.SaltSize())
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}
	aead, err := c.Encrypter(salt)
	if err != nil {
		return err
	}
	// write key to dst
	// salt is key
	_, err = c.Conn.Write(salt)
	if err != nil {
		return err
	}
	m_internal.AddSalt(salt)
	c.w = newWriter(c.Conn, aead)
	return nil
}

func (c *streamConn) Write(b []byte) (int, error) {
	if c.w == nil {
		if err := c.initWriter(); err != nil {
			return 0, err
		}
	}
	return c.w.Write(b)
}

func (c *streamConn) ReadFrom(r io.Reader) (int64, error) {
	if c.w == nil {
		if err := c.initWriter(); err != nil {
			return 0, err
		}
	}
	return c.w.ReadFrom(r)
}

// NewConn wraps a stream-oriented net.Conn with cipher.
func NewConn(c net.Conn, ciph Cipher) net.Conn {
	return &streamConn{Conn: c, Cipher: ciph}
}
