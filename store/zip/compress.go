package zip

import (
	"compress/gzip"
	"github.com/aviddiviner/inc/util"
	"io"
)

const c_COMPRESS_LEVEL = gzip.DefaultCompression

// The amount of uncompressed bytes to read before flushing the gzip buffer.
// Note: this affects compression ratios, so the more we read in, the better.
const c_FLUSH_SIZE = 65535

// CompressReader reads from a stream and compresses using gzip at the default
// compression ratio.  Will flush to output at regular intervals of ~64KB.
func CompressReader(in io.Reader) (out io.Reader, err error) {
	r, w := io.Pipe()
	gz, err := gzip.NewWriterLevel(w, c_COMPRESS_LEVEL)
	if err != nil {
		return
	}
	go func() {
		// Compress input steam, flushing at regular intervals during the copy.
		for {
			if _, err = io.CopyN(gz, in, c_FLUSH_SIZE); err != nil {
				if err == io.EOF {
					break
				}
				w.CloseWithError(err)
				return
			}
			gz.Flush()
		}
		// Finished compressing. Close gzip and write pipe.
		if err := gz.Close(); err != nil {
			w.CloseWithError(err)
			return
		}
		w.Close()
	}()
	return r, nil
}

// DecompressReader read from a stream and decompresses using gzip.
func DecompressReader(in io.Reader) (out io.Reader, err error) {
	rc, err := gzip.NewReader(in)
	if err != nil {
		return
	}
	return &util.AutoCloseReader{rc}, nil
}
