package zip

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"testing"
)

var flushSample = randWords(100000)

func flushEvery(blockSize int) (compressedSize, flushes int) {
	var src, dst bytes.Buffer
	src.Write(flushSample)

	gz, _ := gzip.NewWriterLevel(&dst, gzip.DefaultCompression)
	for src.Len() > 0 {
		io.CopyN(gz, &src, int64(blockSize))
		gz.Flush()
		flushes += 1
	}
	gz.Close()

	compressedSize = dst.Len()
	return
}

func benchFlush(b *testing.B, blockSize int) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		flushEvery(blockSize)
	}
	b.SetBytes(int64(len(flushSample)))
}

// func BenchmarkFlush1(b *testing.B)    { benchFlush(b, 1) }
func BenchmarkFlush5(b *testing.B)       { benchFlush(b, 5) }
func BenchmarkFlush10(b *testing.B)      { benchFlush(b, 10) }
func BenchmarkFlush50(b *testing.B)      { benchFlush(b, 50) }
func BenchmarkFlush100(b *testing.B)     { benchFlush(b, 100) }
func BenchmarkFlush500(b *testing.B)     { benchFlush(b, 500) }
func BenchmarkFlush1000(b *testing.B)    { benchFlush(b, 1000) }
func BenchmarkFlush5000(b *testing.B)    { benchFlush(b, 5000) }
func BenchmarkFlush10000(b *testing.B)   { benchFlush(b, 10000) }
func BenchmarkFlush50000(b *testing.B)   { benchFlush(b, 50000) }
func BenchmarkFlush100000(b *testing.B)  { benchFlush(b, 100000) }
func BenchmarkFlush500000(b *testing.B)  { benchFlush(b, 500000) }
func BenchmarkFlush1000000(b *testing.B) { benchFlush(b, 1000000) }

func TestXFlushSizes(t *testing.T) {
	sample := randWords(100000)
	fmt.Printf("\nsample:%d\n", len(sample))

	reportFlushSize := func(blockSize int) {
		fmt.Printf("blockSize:%d ", blockSize)
		n, f := flushEvery(blockSize)
		fmt.Printf("... size:%d (%d)\n", n, f)
	}

	reportFlushSize(1)
	reportFlushSize(5)
	reportFlushSize(10)
	reportFlushSize(50)
	reportFlushSize(100)
	reportFlushSize(500)
	reportFlushSize(1000)
	reportFlushSize(5000)
	reportFlushSize(10000)
	reportFlushSize(50000)
	reportFlushSize(100000)
	reportFlushSize(500000)
	reportFlushSize(1000000)
}

// === RUN   TestXFlushSizes
// sample:1056979
// blockSize:1 ... size:7403377 (1057622)
// blockSize:5 ... size:2079348 (211525)
// blockSize:10 ... size:1364824 (105763)
// blockSize:50 ... size:789612 (21153)
// blockSize:100 ... size:717438 (10577)
// blockSize:500 ... size:579691 (2116)
// blockSize:1000 ... size:546255 (1058)
// blockSize:5000 ... size:514473 (212)
// blockSize:10000 ... size:509743 (106)
// blockSize:50000 ... size:506010 (22)
// blockSize:100000 ... size:505935 (11)
// blockSize:500000 ... size:505791 (3)
// blockSize:1000000 ... size:505727 (2)

// BenchmarkFlush5-8      	       1	1954527369 ns/op	   0.54 MB/s	7598651840 B/op	 3646390 allocs/op
// BenchmarkFlush10-8     	       1	1170523485 ns/op	   0.90 MB/s	3811302928 B/op	 2052867 allocs/op
// BenchmarkFlush50-8     	       3	 475743623 ns/op	   2.22 MB/s	773956618 B/op	  667106 allocs/op
// BenchmarkFlush100-8    	       3	 356509710 ns/op	   2.97 MB/s	391014304 B/op	  373865 allocs/op
// BenchmarkFlush500-8    	       5	 229411593 ns/op	   4.61 MB/s	82690854 B/op	   97494 allocs/op
// BenchmarkFlush1000-8   	       5	 203013465 ns/op	   5.21 MB/s	43794518 B/op	   53174 allocs/op
// BenchmarkFlush5000-8   	      10	 160545176 ns/op	   6.59 MB/s	11469524 B/op	   12307 allocs/op
// BenchmarkFlush10000-8  	      10	 153791464 ns/op	   6.88 MB/s	 7530459 B/op	    6553 allocs/op
// BenchmarkFlush50000-8  	      10	 146506643 ns/op	   7.22 MB/s	 4395020 B/op	    1558 allocs/op
// BenchmarkFlush100000-8 	      10	 146298058 ns/op	   7.23 MB/s	 4029796 B/op	    1482 allocs/op
// BenchmarkFlush500000-8 	      10	 155239146 ns/op	   6.81 MB/s	 3757995 B/op	    1340 allocs/op
// BenchmarkFlush1000000-8	      10	 156329253 ns/op	   6.77 MB/s	 3720504 B/op	    1276 allocs/op
