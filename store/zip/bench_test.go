package zip

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
	"testing"
)

type compressor interface {
	io.WriteCloser
	Reset(io.Writer)
}

type compressorFactory func(io.Writer) compressor

func flateWriter(buf io.Writer) compressor {
	w, _ := flate.NewWriter(buf, flate.DefaultCompression)
	return w
}
func flate1Writer(buf io.Writer) compressor {
	w, _ := flate.NewWriter(buf, flate.BestSpeed)
	return w
}
func flate9Writer(buf io.Writer) compressor {
	w, _ := flate.NewWriter(buf, flate.BestCompression)
	return w
}
func gzipWriter(buf io.Writer) compressor {
	return gzip.NewWriter(buf)
}
func zlibWriter(buf io.Writer) compressor {
	return zlib.NewWriter(buf)
}

// -----------------------------------------------------------------------------

var smallSample = randWords(500)
var largeSample = randWords(10000)

func compressAndPrintSizes(sample []byte) {
	fmt.Printf("sample:%d\n", len(sample))
	var buf bytes.Buffer

	compressWith := func(label string, f compressorFactory) {
		buf.Reset()
		wc := f(&buf)
		wc.Write(sample)
		wc.Close()
		fmt.Printf("... %s.size:%d\n", label, buf.Len())
	}

	compressWith("flate", flateWriter)
	compressWith("flate1", flate1Writer)
	compressWith("flate9", flate9Writer)
	compressWith("gzip", gzipWriter)
	compressWith("zlib", zlibWriter)
}

func TestXCompressionSizes(t *testing.T) {
	compressAndPrintSizes(smallSample)
	compressAndPrintSizes(largeSample)
}

// -----------------------------------------------------------------------------

func benchmarkWriter(b *testing.B, sample []byte, f compressorFactory) {
	b.ReportAllocs()
	var buf bytes.Buffer
	wc := f(&buf)
	for n := 0; n < b.N; n++ {
		wc.Write(sample)
		wc.Reset(&buf)
		buf.Reset()
	}
	b.SetBytes(int64(len(sample)))
}

func BenchmarkFlateSmall(b *testing.B)  { benchmarkWriter(b, smallSample, flateWriter) }
func BenchmarkFlateLarge(b *testing.B)  { benchmarkWriter(b, largeSample, flateWriter) }
func BenchmarkFlate1Small(b *testing.B) { benchmarkWriter(b, smallSample, flate1Writer) }
func BenchmarkFlate1Large(b *testing.B) { benchmarkWriter(b, largeSample, flate1Writer) }
func BenchmarkFlate9Small(b *testing.B) { benchmarkWriter(b, smallSample, flate9Writer) }
func BenchmarkFlate9Large(b *testing.B) { benchmarkWriter(b, largeSample, flate9Writer) }
func BenchmarkGzipSmall(b *testing.B)   { benchmarkWriter(b, smallSample, gzipWriter) }
func BenchmarkGzipLarge(b *testing.B)   { benchmarkWriter(b, largeSample, gzipWriter) }
func BenchmarkZlibSmall(b *testing.B)   { benchmarkWriter(b, smallSample, zlibWriter) }
func BenchmarkZlibLarge(b *testing.B)   { benchmarkWriter(b, largeSample, zlibWriter) }

// === RUN   TestXCompressionSizes
// sample:5276
// ... flate.size:2843
// ... flate1.size:2922
// ... flate9.size:2843
// ... gzip.size:2861
// ... zlib.size:2849
// sample:105829
// ... flate.size:51714
// ... flate1.size:56614
// ... flate9.size:51704
// ... gzip.size:51732
// ... zlib.size:51720

// BenchmarkFlateSmall-8 	   10000	    101935 ns/op	  52.03 MB/s	     145 B/op	       0 allocs/op
// BenchmarkFlateLarge-8 	     100	  11425684 ns/op	   9.25 MB/s	   20050 B/op	      68 allocs/op
// BenchmarkFlate1Small-8	   10000	    100112 ns/op	  52.98 MB/s	     145 B/op	       0 allocs/op
// BenchmarkFlate1Large-8	     500	   3055749 ns/op	  34.60 MB/s	    7837 B/op	      66 allocs/op
// BenchmarkFlate9Small-8	   20000	    100978 ns/op	  52.53 MB/s	      72 B/op	       0 allocs/op
// BenchmarkFlate9Large-8	     100	  11368950 ns/op	   9.30 MB/s	   20050 B/op	      68 allocs/op
// BenchmarkGzipSmall-8  	   10000	    108537 ns/op	  48.87 MB/s	     145 B/op	       0 allocs/op
// BenchmarkGzipLarge-8  	     100	  11337999 ns/op	   9.33 MB/s	   20052 B/op	      68 allocs/op
// BenchmarkZlibSmall-8  	   10000	    106671 ns/op	  49.72 MB/s	     145 B/op	       0 allocs/op
// BenchmarkZlibLarge-8  	     100	  11247187 ns/op	   9.40 MB/s	   20051 B/op	      68 allocs/op
