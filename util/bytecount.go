package util

import "fmt"

type ByteCount float64

// Return a byte count, cleaned up for pretty output.
func (t ByteCount) String() string {
	if t/1e9 > 1 {
		return fmt.Sprintf("%.2f GB", t/1e9)
	} else if t/1e6 > 1 {
		return fmt.Sprintf("%.3g MB", t/1e6)
	} else if t/1e3 > 1 {
		return fmt.Sprintf("%.3g KB", t/1e3)
	} else {
		return fmt.Sprintf("%.f B", t)
	}
}
