package backup

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
	"time"
)

func TestHexFormat(t *testing.T) {
	assert.Equal(t, "%016x", fmt.Sprintf("%%0%dx", hexLog(1e9*60*60*24*365*45))) // 45y
	assert.Equal(t, "%05x", fmt.Sprintf("%%0%dx", hexLog(65536)))
	assert.Equal(t, "%04x", fmt.Sprintf("%%0%dx", hexLog(65535)))
	assert.Equal(t, "%04x", fmt.Sprintf("%%0%dx", hexLog(4096)))
	assert.Equal(t, "%03x", fmt.Sprintf("%%0%dx", hexLog(4095)))
	assert.Equal(t, "%03x", fmt.Sprintf("%%0%dx", hexLog(256)))
	assert.Equal(t, "%02x", fmt.Sprintf("%%0%dx", hexLog(255)))
	assert.Equal(t, "%02x", fmt.Sprintf("%%0%dx", hexLog(16)))
	assert.Equal(t, "%01x", fmt.Sprintf("%%0%dx", hexLog(15)))
	assert.Equal(t, "%01x", fmt.Sprintf("%%0%dx", hexLog(1)))
	assert.Equal(t, "%01x", fmt.Sprintf("%%0%dx", hexLog(0)))
}

// -----------------------------------------------------------------------------

const testStep = 1e9 * 60 * 60 * 24 * 365 // nanosecs in a year
const testN = 100                         // 100 years

// Playing with ideas of how to key each file upload to S3...
// Check that hex encoded values of time.Now().Unix() are lexically ascending.
func TestSortedHexIsLexicallyAscending(t *testing.T) {
	start := time.Now().UnixNano()
	var strs []string
	for i := start; i < start+(testN*testStep); i = i + testStep {
		strs = append(strs, fmt.Sprintf("%016x", i)) // 141d a263 1e4c 3e82
	}
	assert.True(t, sort.StringsAreSorted(strs))
}
