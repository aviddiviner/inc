package crypto

import (
	"crypto/hmac"
	"crypto/sha1"
	"github.com/stretchr/testify/assert"
	"testing"
)

var key = []byte("myspecialkey")
var message = []byte("some message which we'll hmac in parts")

func TestHmacInParts(t *testing.T) {
	mac1 := hmac.New(sha1.New, key)
	mac2 := hmac.New(sha1.New, key)

	mac1.Write(message)

	mac2.Write(message[:5])
	mac2.Write(message[5:10])
	mac2.Write(message[10:])

	assert.True(t, hmac.Equal(mac1.Sum(nil), mac2.Sum(nil)))
}

func TestHmacReset(t *testing.T) {
	mac1 := hmac.New(sha1.New, key)
	mac2 := hmac.New(sha1.New, key)

	mac1.Write(message)
	mac1.Reset()
	mac1.Write(message)

	mac2.Write(message)

	assert.True(t, hmac.Equal(mac1.Sum(nil), mac2.Sum(nil)))
}
