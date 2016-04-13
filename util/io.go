package util

import "io"

// Wrapper around io.ReadCloser making an io.Reader that automatically closes.
type AutoCloseReader struct {
	RC io.ReadCloser
}

func (a *AutoCloseReader) Read(b []byte) (int, error) {
	n, err := a.RC.Read(b)
	if err == io.EOF {
		a.RC.Close()
	}
	return n, err
}

// -----------------------------------------------------------------------------

// An io.Reader that outputs its progress every N milliseconds.
type ProgressReader struct {
	Reader   io.Reader // underlying reader
	Size     int       // total size of data
	timer    IntervalTimer
	progress int
}

type ProgressFunc func(progress, total int)

func NewProgressReader(r io.Reader, size int, millis int, f ProgressFunc) *ProgressReader {
	p := &ProgressReader{
		Reader:   r,
		Size:     size,
		progress: 0,
	}
	p.timer = NewTimer(millis, func() {
		println("progress", p.progress, p.Size)
	})
	return p
}

// Read from the underlying source, increment our progress and stop the timer
// if we're done.
func (r *ProgressReader) Read(b []byte) (int, error) {
	n, err := r.Reader.Read(b)
	r.progress += n
	if err != nil {
		r.timer.Stop()
	}
	return n, err
}

// -----------------------------------------------------------------------------

func CopyAndClose(dst io.WriteCloser, src io.Reader) (written int64, err error) {
	written, err = io.Copy(dst, src)
	if err == nil {
		err = dst.Close()
	}
	return
}
