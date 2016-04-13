package zip

import (
	"io/ioutil"
	"math/rand"
	"strings"
	"time"
)

func init() {
	rand.Seed(int64(time.Now().Nanosecond()))
}

func shuffle(words []string) {
	for i := len(words) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		words[i], words[j] = words[j], words[i]
	}
}

func randWords(n int) []byte {
	raw, _ := ioutil.ReadFile("/usr/share/dict/words")
	words := strings.Split(string(raw), "\n")
	shuffle(words)
	if n > len(words)-1 {
		n = len(words) - 1
	}
	return []byte(strings.Join(words[:n], "\n"))
}
