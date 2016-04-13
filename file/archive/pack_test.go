package archive

import (
	"archive/tar"
	"bytes"
	"github.com/aviddiviner/inc/file"
	"github.com/aviddiviner/inc/util/test"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"sort"
	"testing"
)

func pack(files ...file.File) (tarball []byte, err error) {
	return ioutil.ReadAll(PackReader(files...))
}

func unpack(root string, tarball []byte) error {
	return UnpackReader(root, bytes.NewReader(tarball), nil)
}

// -----------------------------------------------------------------------------

func TestPackUnpack(t *testing.T) {
	testFiles := []file.File{createTestFile(t), createTestFile(t), createTestFile(t)}
	sort.Sort(file.ByPath(testFiles))

	tarball, err := pack(testFiles...)
	assert.NoError(t, err, "no errors creating tarball")
	assert.NotEmpty(t, tarball, "tarball isn't empty")

	tempDir := test.CreateTempDir(t)
	err = unpack(tempDir, tarball)
	assert.NoError(t, err, "no errors restoring tarball")

	var found []file.File
	for _, f := range file.NewScanner().IncludePath(tempDir).ScanRelativeTo(tempDir) {
		if !f.IsDir() {
			f.Root = test.TempDir
			found = append(found, f)
		}
	}
	sort.Sort(file.ByPath(found))

	assert.EqualValues(t, testFiles, found, "files are the same")
}

func TestPackReader(t *testing.T) {
	testFiles := []file.File{createTestFile(t), createTestFile(t), createLargeTestFile(t)}

	tarball, _ := pack(testFiles...)
	stream := PackReader(testFiles...)

	var buf bytes.Buffer
	n, err := io.Copy(&buf, stream)
	assert.NoError(t, err, "no errors when reading")
	err = stream.Close()
	assert.NoError(t, err, "no errors closing the stream")

	assert.EqualValues(t, len(tarball), n, "same number of bytes")
	assert.EqualValues(t, tarball, buf.Bytes(), "archives are the same")
}

// -----------------------------------------------------------------------------

func TestFlushMidFileWorks(t *testing.T) {
	testFiles := []file.File{createTestFile(t), createTestFile(t), createTestFile(t)}
	tarball, _ := pack(testFiles...)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for _, f := range testFiles {
		hdr, _ := tar.FileInfoHeader(f.FileInfo(), "")
		assert.NoError(t, tw.WriteHeader(hdr), "write header without errors")
		fh, err := fs.OpenRead(f.Path())
		assert.NoError(t, err, "open file without errors")

		flushes := 0
		for {
			// Write contents 2 bytes at a time and Flush() after each write.
			if _, err := io.CopyN(tw, fh, 2); err != nil {
				if err == io.EOF {
					break
				}
				t.FailNow()
			}
			tw.Flush()
			flushes += 1
		}

		assert.True(t, flushes > 1, "we flushed more than once")
	}

	tw.Close()
	assert.EqualValues(t, len(tarball), buf.Len(), "same number of bytes")
	assert.EqualValues(t, tarball, buf.Bytes(), "archives are the same")
}
