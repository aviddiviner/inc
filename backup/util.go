package backup

import (
	"fmt"
	"time"
)

// How many hexadecimal places we need for a number of size n.
func hexLog(n int64) int64 {
	if n < 16 {
		return 1
	}
	return 1 + hexLog(n>>4)
}

func manifestKey(t time.Time) string {
	return fmt.Sprintf("%016x", t.UnixNano())
}

func keyFactory(size int) func() string {
	keyFormat := fmt.Sprintf("%%0%dx", hexLog(int64(size)))
	k := 0
	return func() (key string) {
		key = fmt.Sprintf(keyFormat, k) // unique storage key
		k += 1
		return
	}
}
