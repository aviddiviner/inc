package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"errors"
)

// Pad a plaintext to the next whole block size. We use PKCS7-style padding.
func pad(slice []byte) []byte {
	oldLen := len(slice)
	pad := aes.BlockSize - (oldLen % aes.BlockSize)
	newLen := oldLen + pad

	// Reallocate only once, if needed.
	if newLen > cap(slice) {
		newSlice := make([]byte, newLen)
		copy(newSlice, slice)
		slice = newSlice
	}

	slice = slice[:newLen]
	for i := oldLen; i < newLen; i++ {
		slice[i] = byte(pad)
	}
	return slice
}

// Strip a padded plaintext.
func unpad(slice []byte) ([]byte, error) {
	pad := int(slice[len(slice)-1])
	if pad > aes.BlockSize || pad == 0 {
		return nil, errors.New("invalid padding length")
	}

	// Validate the padding bytes.
	for _, p := range slice[len(slice)-pad:] {
		if p != byte(pad) {
			return nil, errors.New("invalid padding bytes")
		}
	}

	return slice[:len(slice)-pad], nil
}

// -----------------------------------------------------------------------------

// Our encryption envelope consists of (iv||ciphertext||hmac).
type envelope []byte

// Create a new envelope, generating a random IV.
func newEnvelope(size int) (envelope, error) {
	slice := make([]byte, aes.BlockSize+size+c_HMAC_SIZE)
	// The IV must be unique, but need not be secure. It precedes the ciphertext.
	iv := slice[:aes.BlockSize]
	_, err := rand.Read(iv)
	return slice, err
}

// Read an encryption envelope and validate its length.
func readEnvelope(blob []byte) (envelope, error) {
	msg := envelope(blob)
	// The IV and ciphertext will both be at least aes.BlockSize.
	if msg.msgLen() < aes.BlockSize*2 {
		return nil, errors.New("ciphertext too short")
	}
	// CBC mode always works in whole blocks.
	if msg.msgLen()%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}
	return msg, nil
}

// The length of the message part (iv||ciphertext).
func (e envelope) msgLen() int {
	return len(e) - c_HMAC_SIZE
}

// Get the first (iv) part of the envelope.
func (e envelope) iv() []byte {
	return e[:aes.BlockSize]
}

// Get the middle (ciphertext) part of the envelope.
func (e envelope) ciphertext() []byte {
	return e[aes.BlockSize:e.msgLen()]
}

// Get the last (hmac) part of the envelope.
func (e envelope) hmac() []byte {
	return e[e.msgLen():]
}

// Sign the contents of the envelope and append the signature.
func (e envelope) sign(authKey []byte) []byte {
	mac := hmac.New(sha1.New, authKey)
	mac.Write(e[:e.msgLen()])
	copy(e.hmac(), mac.Sum(nil))
	return e.hmac()
}

// Authenticate the contents of the envelope against its signature.
func (e envelope) auth(authKey []byte) bool {
	mac := hmac.New(sha1.New, authKey)
	mac.Write(e[:e.msgLen()])
	return hmac.Equal(e.hmac(), mac.Sum(nil))
}

// -----------------------------------------------------------------------------

// Encrypt some plaintext using AES-256 in CBC (block chaining) mode.
// The ciphertext is then signed by HMAC-SHA1 signature.
func encrypt(plaintext []byte, bc cipher.Block, authKey []byte) ([]byte, error) {
	plaintext = pad(plaintext)
	blob, err := newEnvelope(len(plaintext))
	if err != nil {
		return nil, err
	}

	mode := cipher.NewCBCEncrypter(bc, blob.iv())
	mode.CryptBlocks(blob.ciphertext(), plaintext)

	blob.sign(authKey)
	return blob, nil
}

// Decrypt some ciphertext using AES-256 in CBC (block chaining) mode.
// Authenticated by its HMAC-SHA1 signature before decrypting.
func decrypt(ciphertext []byte, bc cipher.Block, authKey []byte) ([]byte, error) {
	blob, err := readEnvelope(ciphertext)
	ciphertext = blob.ciphertext()

	if !blob.auth(authKey) {
		return nil, errors.New("ciphertext not authentic")
	}

	mode := cipher.NewCBCDecrypter(bc, blob.iv())
	mode.CryptBlocks(ciphertext, ciphertext) // decrypt in place

	plaintext, err := unpad(ciphertext)
	return plaintext, err
}
