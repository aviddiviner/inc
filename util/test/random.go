package test

import (
	"crypto/sha1"
	"io/ioutil"
	"math/rand"
	"strings"
	"time"
)

func init() {
	RandSeed(time.Now().Nanosecond())
}

func RandSeed(seed int) {
	rand.Seed(int64(seed))
}

// -----------------------------------------------------------------------------

const letters = "abcdefghijklmnopqrstuvwxyz"

func RandBytes(n int) (b []byte) {
	b = make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return
}

func RandString(n int) string {
	return string(RandBytes(n))
}

// -----------------------------------------------------------------------------

func shuffle(words []string) {
	for i := len(words) - 1; i > 0; i-- {
		j := rand.Intn(i)
		words[i], words[j] = words[j], words[i]
	}
}

func RandWords(n int) []byte {
	raw, _ := ioutil.ReadFile("/usr/share/dict/words")
	words := strings.Split(string(raw), "\n")
	shuffle(words)
	if n > len(words)-1 {
		n = len(words) - 1
	}
	return []byte(strings.Join(words[:n], "\n"))
}

// -----------------------------------------------------------------------------

func RandSHA1() (out [sha1.Size]byte) {
	buf := RandBytes(sha1.Size)
	copy(out[:], buf)
	return
}
