package test

import (
	"github.com/aviddiviner/inc/file"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var fs file.FileSystem = file.DefaultFileSystem

// The absolute path (after following symlinks) of os.TempDir()
var TempDir = func() string {
	abs, err := fs.AbsPath(os.TempDir())
	if err != nil {
		panic(err)
	}
	return abs
}()

func randTestFilePath() string {
	return filepath.Join(TempDir, "go-test-"+RandString(9))
}

var tempFilePaths []string

func CreateTempDir(t *testing.T) (path string) {
	path = randTestFilePath()
	tempFilePaths = append(tempFilePaths, path)
	assert.NoError(t, fs.MkdirAll(path, 0755))
	return
}

// Write a 50 byte file to the TempDir and return the path.
func CreateTempFile(t *testing.T) (path string) {
	path = randTestFilePath()
	tempFilePaths = append(tempFilePaths, path)
	AppendToFile(t, path, RandString(50))
	return
}

// Write a 500KB file to the TempDir and return the path.
func CreateLargeTempFile(t *testing.T) (path string) {
	path = randTestFilePath()
	tempFilePaths = append(tempFilePaths, path)
	AppendToFile(t, path, RandString(500000))
	return
}

func CleanupTempFiles() {
	println("Cleaning up test files...")
	for _, path := range tempFilePaths {
		println("--", path)
		if err := fs.RemoveAll(path); err != nil {
			println(err)
		}
	}
}

func AppendToFile(t *testing.T, filepath, str string) {
	fd, err := fs.OpenWrite(filepath, 0644)
	assert.NoError(t, err)
	defer fd.Close()
	_, err = fd.Write([]byte(str))
	assert.NoError(t, err)
}

func TouchFileTime(t *testing.T, filepath string, ts time.Time) {
	assert.NoError(t, fs.Chtimes(filepath, ts, ts))
}
