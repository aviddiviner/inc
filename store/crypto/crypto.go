package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"golang.org/x/crypto/pbkdf2"
	"hash"
	"io"
	"io/ioutil"
)

const c_SALT_SIZE = 8
const c_HASH_ITER = 4096
const c_ENC_KEY_SIZE = 32              // 32-byte key for AES-256
const c_HMAC_KEY_SIZE = sha1.BlockSize // 64-byte key for HMAC-SHA1
const c_HMAC_SIZE = sha1.Size          // 20-byte checksum for SHA1

// Create a new cyptographically secure salt for hashing passwords.
func Salt() ([]byte, error) {
	buf := make([]byte, c_SALT_SIZE)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// PBKDF2 for generating encryption and auth keys, derived from secret/salt.
// Returns two keys; 32-byte encryption key (AES-256) and 64-byte auth key (SHA1).
func DeriveKeys(secret, salt []byte) (encKey, authKey []byte) {
	key := pbkdf2.Key(secret, salt, c_HASH_ITER, c_ENC_KEY_SIZE+c_HMAC_KEY_SIZE, sha1.New)
	return key[:c_ENC_KEY_SIZE], key[c_ENC_KEY_SIZE:]
}

// -----------------------------------------------------------------------------

// Pad a plaintext to the next whole block size. We use PKCS7-style padding.
func padBuffer(buf *bytes.Buffer) {
	pad := aes.BlockSize - (buf.Len() % aes.BlockSize)
	buf.Grow(pad)
	for i := 0; i < pad; i++ {
		buf.WriteByte(byte(pad))
	}
}

// Strip a padded plaintext.
func unpadBuffer(buf *bytes.Buffer) error {
	length := buf.Len()
	data := buf.Bytes()
	pad := int(data[length-1])
	// Validate the padding length and bytes.
	if pad > aes.BlockSize || pad <= 0 {
		return errors.New("invalid padding length")
	}
	for _, p := range data[length-pad:] {
		if p != byte(pad) {
			return errors.New("invalid padding bytes")
		}
	}
	// Discard the padding.
	buf.Truncate(length - pad)
	return nil
}

// -----------------------------------------------------------------------------

type Crypter interface {
	Encrypt(plaintext []byte) (ciphertext []byte, err error)
	Decrypt(ciphertext []byte) (plaintext []byte, err error)
	EncryptReader(plaintext io.Reader) (ciphertext io.Reader, err error)
	DecryptReader(ciphertext io.Reader) (plaintext io.Reader, err error)
}

type aesCrypter struct {
	bc      cipher.Block
	authKey []byte
}

// Encrypt and decrypt using AES-256 in CBC (block chaining) mode.
// The ciphertext is signed/authenticated by an HMAC-SHA1 signature.
// Our encryption envelope consists of (iv||ciphertext||hmac).
func NewCrypter(encKey, authKey []byte) (Crypter, error) {
	if len(encKey) != c_ENC_KEY_SIZE {
		return nil, errors.New("invalid encryption key length")
	}
	if len(authKey) != c_HMAC_KEY_SIZE {
		return nil, errors.New("invalid authentication key length")
	}
	bc, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	return &aesCrypter{bc, authKey}, nil
}

// Encrypt plaintext, then sign ciphertext with HMAC-SHA1 signature.
func (e *aesCrypter) Encrypt(plaintext []byte) ([]byte, error) {
	r, err := e.EncryptReader(bytes.NewReader(plaintext))
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(r)
}

// Authenticate ciphertext by its HMAC-SHA1 signature, then decrypt.
func (e *aesCrypter) Decrypt(ciphertext []byte) ([]byte, error) {
	r, err := e.DecryptReader(bytes.NewReader(ciphertext))
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(r)
}

// -----------------------------------------------------------------------------

type encryptReader struct {
	text io.Reader
	cbc  cipher.BlockMode
	mac  hash.Hash
	buf  bytes.Buffer
	fin  bool
}

// Implement io.Reader.
func (e *encryptReader) Read(b []byte) (n int, err error) {
	// Drain the buffer.
	n, err = e.buf.Read(b)
	if n == len(b) || e.fin {
		return
	}

	// Read the remaining required, rounded up to the nearest aes.BlockSize.
	target := ((len(b)-n)/aes.BlockSize + 1) * aes.BlockSize
	_, err = io.CopyN(&e.buf, e.text, int64(target))
	switch err {
	case io.EOF:
		// We've reached the end of our input. Pad plaintext, sign ciphertext.
		e.fin = true
	case nil:
		// Do nothing.
	default:
		// We had trouble reading from our input. Return with the error.
		return
	}

	// Pad the plaintext to ensure we're a multiple of aes.BlockSize.
	if e.fin {
		padBuffer(&e.buf)
	}

	// Encrypt the buffer contents and continue to HMAC the ciphertext.
	e.cbc.CryptBlocks(e.buf.Bytes(), e.buf.Bytes())
	_, err = e.mac.Write(e.buf.Bytes())
	if err != nil {
		return
	}

	// Append the final HMAC signature.
	if e.fin {
		e.buf.Write(e.mac.Sum(nil))
	}

	// Remaining read from the buffer.
	m, err := e.buf.Read(b[n:])
	return n + m, err
}

// -----------------------------------------------------------------------------

type decryptReader struct {
	text io.Reader
	cbc  cipher.BlockMode
	mac  hash.Hash
	buf  bytes.Buffer
	fin  bool

	// Read bytes not yet decrypted.
	enc bytes.Buffer
}

// Implement io.Reader.
func (e *decryptReader) Read(b []byte) (n int, err error) {
	// Drain the buffer.
	n, err = e.buf.Read(b)
	if n == len(b) || e.fin {
		return
	}

	// Read the remaining required, rounded up to the nearest aes.BlockSize, but
	// also ensure we always read ahead by at least c_HMAC_SIZE+1 so we know when
	// we hit the end of the ciphertext stream and need to unpad and authenticate.
	target := ((len(b)-n)/aes.BlockSize + 1) * aes.BlockSize
	_, err = io.CopyN(&e.enc, e.text, int64(target+c_HMAC_SIZE+1))
	switch err {
	case io.EOF:
		// We've reached the end of our input. Auth ciphertext, unpad plaintext.
		e.fin = true
	case nil:
		// Do nothing.
	default:
		// We had trouble reading from our input. Return with the error.
		return
	}

	// Found our message boundary. Leave the HMAC, decrypt the rest.
	if e.fin {
		target = e.enc.Len() - c_HMAC_SIZE

		// Check message length before CryptBlocks() panics below.
		if target%aes.BlockSize != 0 {
			err = errors.New("ciphertext is not a multiple of the block size")
			return
		}
	}

	// Put the required amount of ciphertext in the buffer. Leave the rest.
	copied, err := io.CopyN(&e.buf, &e.enc, int64(target))
	if int(copied) != target { // sanity check
		panic(err)
	}

	// Continue calculating the HMAC and decrypt what's in the buffer.
	_, err = e.mac.Write(e.buf.Bytes())
	if err != nil {
		return
	}
	e.cbc.CryptBlocks(e.buf.Bytes(), e.buf.Bytes())

	// Check that message is authentic, then unpad the decrypted buffer contents.
	if e.fin {
		if !hmac.Equal(e.enc.Bytes(), e.mac.Sum(nil)) {
			err = errors.New("ciphertext not authentic")
			return
		}
		err = unpadBuffer(&e.buf)
		if err != nil {
			return
		}
	}

	// Remaining read from the buffer.
	m, err := e.buf.Read(b[n:])
	return n + m, err
}

// -----------------------------------------------------------------------------

func (e *aesCrypter) EncryptReader(plaintext io.Reader) (ciphertext io.Reader, err error) {
	// The IV must be unique, but need not be secure. It precedes the ciphertext.
	var iv bytes.Buffer
	_, err = io.CopyN(&iv, rand.Reader, aes.BlockSize)
	if err != nil {
		return
	}

	// Start the signing HMAC and add the IV.
	mac := hmac.New(sha1.New, e.authKey)
	_, err = mac.Write(iv.Bytes())
	if err != nil {
		return
	}

	return &encryptReader{
		text: plaintext,
		cbc:  cipher.NewCBCEncrypter(e.bc, iv.Bytes()),
		mac:  mac,
		buf:  iv,
	}, nil
}

func (e *aesCrypter) DecryptReader(ciphertext io.Reader) (plaintext io.Reader, err error) {
	// Read the IV from the start of the stream.
	var iv bytes.Buffer
	n, err := io.CopyN(&iv, ciphertext, aes.BlockSize)
	if n < aes.BlockSize || err == io.EOF {
		return nil, errors.New("ciphertext too short")
	}
	if err != nil {
		return
	}

	// Start the signing HMAC and add the IV.
	mac := hmac.New(sha1.New, e.authKey)
	_, err = mac.Write(iv.Bytes())
	if err != nil {
		return
	}

	return &decryptReader{
		text: ciphertext,
		cbc:  cipher.NewCBCDecrypter(e.bc, iv.Bytes()),
		mac:  mac,
	}, nil
}
