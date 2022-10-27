package m_shadow

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"errors"
	"io"
	"strconv"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

// ErrRepeatedSalt means detected a reused salt
var ErrRepeatedSalt = errors.New("repeated salt detected")

type Cipher interface {
	KeySize() int
	SaltSize() int
	// generate encryption funciton
	Encrypter(salt []byte) (cipher.AEAD, error)
	// generate decryption function
	Decrypter(salt []byte) (cipher.AEAD, error)
}

type KeySizeError int

func (e KeySizeError) Error() string {
	return "key size error: need " + strconv.Itoa(int(e)) + " bytes"
}

func hkdfSHA1(secret, salt, info, outkey []byte) {
	r := hkdf.New(sha1.New, secret, salt, info)
	if _, err := io.ReadFull(r, outkey); err != nil {
		panic(err) // should never happen
	}
}

type metaCipher struct {
	psk      []byte
	makeAEAD func(key []byte) (cipher.AEAD, error)
}

func (a *metaCipher) KeySize() int { return len(a.psk) }
func (a *metaCipher) SaltSize() int {
	if ks := a.KeySize(); ks > 16 {
		return ks
	}
	return 16
}

// 生成一个加解密实例
func (a *metaCipher) Encrypter(salt []byte) (cipher.AEAD, error) {
	// 以psk的长度来生成hkdf key
	subkey := make([]byte, a.KeySize())
	// 通过psk 来源于命令行参数 或 password生成
	// salt 来源于rand
	// subkey 通过hkdf
	// 通过kdf 算出更好的key,并写入到subkey
	hkdfSHA1(a.psk, salt, []byte("ss-subkey"), subkey)
	// 构建AEAD需要的实例
	// 通过subkey生产aead实例
	return a.makeAEAD(subkey)
}

// 生成一个加解密实例
func (a *metaCipher) Decrypter(salt []byte) (cipher.AEAD, error) {
	subkey := make([]byte, a.KeySize())
	hkdfSHA1(a.psk, salt, []byte("ss-subkey"), subkey)
	return a.makeAEAD(subkey)
}

func aesGCM(key []byte) (cipher.AEAD, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(blk)
}

// AESGCM creates a new Cipher with a pre-shared key. len(psk) must be
// one of 16, 24, or 32 to select AES-128/196/256-GCM.
func AESGCM(psk []byte) (Cipher, error) {
	switch l := len(psk); l {
	case 16, 24, 32: // AES 128/196/256
	default:
		return nil, aes.KeySizeError(l)
	}
	return &metaCipher{psk: psk, makeAEAD: aesGCM}, nil
}

// Chacha20Poly1305 creates a new Cipher with a pre-shared key. len(psk)
// must be 32.
func Chacha20Poly1305(psk []byte) (Cipher, error) {
	if len(psk) != chacha20poly1305.KeySize {
		return nil, KeySizeError(chacha20poly1305.KeySize)
	}
	return &metaCipher{psk: psk, makeAEAD: chacha20poly1305.New}, nil
}
