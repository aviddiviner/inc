package zip

import (
	"bytes"
	"compress/gzip"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
	"testing/iotest"
)

var sample = randWords(10000) // enough to trigger a flush

func TestCompressReader(t *testing.T) {
	zipped := compress(t, sample)

	r, err := CompressReader(bytes.NewReader(sample))
	assert.NoError(t, err, "create reader without errors")
	streamed, err := ioutil.ReadAll(iotest.OneByteReader(r))
	assert.NoError(t, err, "compress without errors")

	assert.InEpsilon(t, len(zipped), len(streamed), 0.1, "within 10% of the size")

	unzipped := decompress(t, streamed)
	assert.Equal(t, sample, unzipped, "decompresses back to the original")
}

func TestCompressErrors(t *testing.T) {
	r, _ := CompressReader(iotest.TimeoutReader(bytes.NewReader(sample)))
	_, err := ioutil.ReadAll(r)
	assert.Error(t, err, "errors propagate through the reader")
}

func TestDecompressReader(t *testing.T) {
	zipped := compress(t, sample)

	r, err := DecompressReader(bytes.NewReader(zipped))
	assert.NoError(t, err, "create reader without errors")
	streamed, err := ioutil.ReadAll(iotest.OneByteReader(r))
	assert.NoError(t, err, "decompress without errors")

	assert.Equal(t, sample, streamed, "decompresses back to the original")
}

// -----------------------------------------------------------------------------
// Reference functions to test against.

func compress(t *testing.T, data []byte) []byte {
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, c_COMPRESS_LEVEL)
	assert.NoError(t, err)
	_, err = w.Write(data)
	assert.NoError(t, err)
	assert.NoError(t, w.Close())
	return buf.Bytes()
}

func decompress(t *testing.T, data []byte) []byte {
	r, err := gzip.NewReader(bytes.NewReader(data))
	assert.NoError(t, err)
	out, err := ioutil.ReadAll(r)
	assert.NoError(t, err)
	assert.NoError(t, r.Close())
	return out
}
