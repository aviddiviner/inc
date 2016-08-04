package crypto

import (
	"bytes"
	"crypto/aes"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"testing"
	"testing/iotest"
)

var samples = [][]byte{
	{},                            // empty (non-nil)
	[]byte(""),                    // empty (0 bytes)
	[]byte("f"),                   // tiny
	[]byte("foo"),                 // short
	[]byte("justshortof16.."),     // just short of 16 bytes
	[]byte("exampleplaintext"),    // exact (16 bytes == aes.BlockSize)
	[]byte("exampleplaintext!!1"), // longer
	{0x3b, 0x19, 0xec, 0x8a, 0x79, 0x37, 0xc4, 0xa4},
	[]byte(`
Lorem ipsum dolor sit amet, consectetur adipiscing elit. Cras porta volutpat leo eget dapibus. Duis scelerisque tellus
commodo magna ultrices sagittis. Duis eu imperdiet elit. Etiam convallis mauris lobortis pretium gravida. Phasellus ac
felis a leo bibendum egestas porttitor at quam. Proin laoreet aliquam nisl sit amet elementum. Duis elit quam, finibus
vitae semper eu, interdum ac ante. Duis magna urna, vulputate quis nisi vitae, tincidunt laoreet dui. Curabitur mattis
tellus sed mauris placerat, gravida porta eros lobortis. Nulla luctus lectus eget dolor congue lacinia. Aenean lacinia
neque diam, id vehicula arcu varius eget.`),
}

func TestPadding(t *testing.T) {
	for _, slice := range samples {
		padded := pad(slice)
		assert.True(t, len(padded)%aes.BlockSize == 0, "pads to whole block")
		unpadded, _ := unpad(padded)
		assert.Equal(t, slice, unpadded, "unpads back to the original")
	}
}

func TestCrypto(t *testing.T) {
	salt, _ := Salt()
	pass := []byte("some password")
	enc, _ := NewCrypter(DeriveKeys(pass, salt))

	for _, plaintext := range samples {
		ciphertext, _ := enc.Encrypt(plaintext)
		decrypted, _ := enc.Decrypt(ciphertext)
		assert.Equal(t, plaintext, decrypted, "decrypted plaintext is the same")
	}
}

func TestCryptoReaders(t *testing.T) {
	salt, _ := Salt()
	pass := []byte("some password")
	enc, _ := NewCrypter(DeriveKeys(pass, salt))

	// Normal io.Readers
	for _, plaintext := range samples {
		r := bytes.NewReader(plaintext)
		e, _ := enc.EncryptReader(r)
		d, _ := enc.DecryptReader(e)

		decrypted, _ := ioutil.ReadAll(d)
		assert.Equal(t, plaintext, decrypted, "decrypted plaintext is the same")
	}

	// Wrap readers in iotest.OneByteReader
	for _, plaintext := range samples {
		r := bytes.NewReader(plaintext)
		e, _ := enc.EncryptReader(iotest.OneByteReader(r))
		d, _ := enc.DecryptReader(iotest.OneByteReader(e))

		decrypted, _ := ioutil.ReadAll(iotest.OneByteReader(d))
		assert.Equal(t, plaintext, decrypted, "decrypted plaintext is the same")
	}

	// Wrap readers in iotest.DataErrReader (return io.EOF on last data)
	for _, plaintext := range samples {
		r := bytes.NewReader(plaintext)
		e, _ := enc.EncryptReader(iotest.DataErrReader(r))
		d, _ := enc.DecryptReader(iotest.DataErrReader(e))

		decrypted, _ := ioutil.ReadAll(d)
		assert.Equal(t, plaintext, decrypted, "decrypted plaintext is the same")
	}
}

func TestCryptoReaderErrors(t *testing.T) {
	salt, _ := Salt()
	pass := []byte("some password")
	enc, _ := NewCrypter(DeriveKeys(pass, salt))

	// Append a few extra bytes to the ciphertext
	for _, plaintext := range samples {
		r := bytes.NewReader(plaintext)
		extraBytes := bytes.NewReader([]byte("abc"))
		e, _ := enc.EncryptReader(r)
		d, _ := enc.DecryptReader(io.MultiReader(e, extraBytes))

		_, err := ioutil.ReadAll(d)
		assert.EqualError(t, err, "ciphertext is not a multiple of the block size")
	}

	// Append a whole extra block (16 bytes) to the ciphertext
	for _, plaintext := range samples {
		r := bytes.NewReader(plaintext)
		extraBytes := bytes.NewReader([]byte("exactly 16 bytes"))
		e, _ := enc.EncryptReader(r)
		d, _ := enc.DecryptReader(io.MultiReader(e, extraBytes))

		_, err := ioutil.ReadAll(d)
		assert.EqualError(t, err, "ciphertext not authentic")
	}

	// Truncate the ciphertext by a few bytes
	for _, plaintext := range samples {
		r := bytes.NewReader(plaintext)
		e, _ := enc.EncryptReader(r)
		d, err := enc.DecryptReader(io.LimitReader(e, 36))

		assert.NotNil(t, d, "we get a reader for decrypting")
		_, err = ioutil.ReadAll(d)
		assert.EqualError(t, err, "ciphertext not authentic")
	}

	// Truncate the ciphertext to less than a full block
	for _, plaintext := range samples {
		r := bytes.NewReader(plaintext)
		e, _ := enc.EncryptReader(r)
		d, err := enc.DecryptReader(io.LimitReader(e, 15))

		assert.Nil(t, d, "we don't get a reader for decrypting")
		assert.EqualError(t, err, "ciphertext too short")
	}

	// Timeout on the reader pipeline
	for _, plaintext := range samples {
		r := bytes.NewReader(plaintext)
		e, _ := enc.EncryptReader(iotest.TimeoutReader(r))
		d, _ := enc.DecryptReader(e)

		_, err := ioutil.ReadAll(d)
		if !bytes.Equal(plaintext, []byte{}) {
			assert.EqualError(t, err, "timeout")
		}
	}
}
