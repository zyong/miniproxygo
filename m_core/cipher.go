package m_core

import (
	"crypto/md5"
	"errors"
	"net"
	"sort"
	"strings"
)

import (
	"github.com/zyong/miniproxygo/m_shadow"
)

type Cipher interface {
	StreamConnCipher
	PacketConnCipher
}

type StreamConnCipher interface {
	StreamConn(net.Conn) net.Conn
}

type PacketConnCipher interface {
	PacketConn(net.PacketConn) net.PacketConn
}

// ErrCipherNotSupported occurs when a cipher is not supported (likely because of security concerns).
var ErrCipherNotSupported = errors.New("cipher not supported")

const (
	aeadAes128Gcm        = "AEAD_AES_128_GCM"
	aeadAes256Gcm        = "AEAD_AES_256_GCM"
	aeadChacha20Poly1305 = "AEAD_CHACHA20_POLY1305"
)

// List of AEAD ciphers: key size in bytes and constructor
var aeadList = map[string]struct {
	KeySize int
	New     func([]byte) (m_shadow.Cipher, error)
}{
	aeadAes128Gcm:        {16, m_shadow.AESGCM},
	aeadAes256Gcm:        {32, m_shadow.AESGCM},
	aeadChacha20Poly1305: {32, m_shadow.Chacha20Poly1305},
}

// ListCipher returns a list of available cipher names sorted alphabetically.
func ListCipher() []string {
	var l []string
	for k := range aeadList {
		l = append(l, k)
	}
	sort.Strings(l)
	return l
}

// PickCipher returns a Cipher of the given name. Derive key from password if given key is empty.
func PickCipher(name string, key []byte, password string) (Cipher, error) {
	name = strings.ToUpper(name)

	switch name {
	case "DUMMY":
		return &dummy{}, nil
	case "CHACHA20-IETF-POLY1305":
		name = aeadChacha20Poly1305
	case "AES-128-GCM":
		name = aeadAes128Gcm
	case "AES-256-GCM":
		name = aeadAes256Gcm
	}

	if choice, ok := aeadList[name]; ok {
		//密码为空的情况下生成一个新的key
		if len(key) == 0 {
			key = kdf(password, choice.KeySize)
		}
		if len(key) != choice.KeySize {
			return nil, m_shadow.KeySizeError(choice.KeySize)
		}
		aead, err := choice.New(key)
		return &aeadCipher{aead}, err
	}

	return nil, ErrCipherNotSupported
}

type aeadCipher struct{ m_shadow.Cipher }

// 通过connection参数包装一个带加解密功能的conn
func (aead *aeadCipher) StreamConn(c net.Conn) net.Conn {
	return m_shadow.NewConn(c, aead)
}

func (aead *aeadCipher) PacketConn(c net.PacketConn) net.PacketConn {
	return m_shadow.NewPacketConn(c, aead)
}

// dummy cipher does not encrypt
type dummy struct{}

func (dummy) StreamConn(c net.Conn) net.Conn             { return c }
func (dummy) PacketConn(c net.PacketConn) net.PacketConn { return c }

// key-derivation function from original Shadowsocks
func kdf(password string, keyLen int) []byte {
	var b, prev []byte
	h := md5.New()
	//计算多轮，如果b的长度大于keyLen的时候返回keyLen长度的数据
	// keylen的最大长度为32位，只要设计的hash实现能产出32位及以上长度的就可以
	for len(b) < keyLen {
		h.Write(prev)
		h.Write([]byte(password))
		b = h.Sum(b)
		prev = b[len(b)-h.Size():]
		h.Reset()
	}
	return b[:keyLen]
}
