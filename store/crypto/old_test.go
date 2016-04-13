package crypto

import (
	"crypto/aes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEnvelope(t *testing.T) {
	plaintext := []byte("foobar")
	plaintext = pad(plaintext)

	empty, err := newEnvelope(len(plaintext))
	assert.NoError(t, err, "we can create an empty envelope")
	assert.NotEmpty(t, empty.iv(), "it has an IV")

	raw := []byte(empty)
	read, err := readEnvelope(raw)
	assert.NoError(t, err, "we can read an empty envelope")
	assert.EqualValues(t, empty.iv(), read.iv(), "it has the same IV")

	_, err = readEnvelope([]byte{})
	assert.Error(t, err, "envelope can't be zero size")
}

func TestOldStyleCrypto(t *testing.T) {
	salt, _ := Salt()
	pass := []byte("some password")
	encKey, authKey := DeriveKeys(pass, salt)
	bc, _ := aes.NewCipher(encKey)

	for _, plaintext := range samples {
		ciphertext, _ := encrypt(plaintext, bc, authKey)
		decrypted, _ := decrypt(ciphertext, bc, authKey)
		assert.Equal(t, plaintext, decrypted, "decrypted plaintext is the same")
	}
}

func TestNewAgainstOldCrypto(t *testing.T) {
	salt, _ := Salt()
	pass := []byte("some password")
	encKey, authKey := DeriveKeys(pass, salt)
	bc, _ := aes.NewCipher(encKey)
	enc, _ := NewCrypter(encKey, authKey)

	for _, plaintext := range samples {
		ciphertext, _ := encrypt(plaintext, bc, authKey)
		decrypted, _ := enc.Decrypt(ciphertext)
		assert.Equal(t, plaintext, decrypted, "decrypted plaintext is the same")
	}

	for _, plaintext := range samples {
		ciphertext, _ := enc.Encrypt(plaintext)
		decrypted, _ := decrypt(ciphertext, bc, authKey)
		assert.Equal(t, plaintext, decrypted, "decrypted plaintext is the same")
	}
}
