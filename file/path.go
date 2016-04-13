package file

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

// CleanPath resolves the shortest path name equivalent, also replacing any
// occurrences of the ~/ prefix with the actual home directory path.
func CleanPath(path string) string {
	if path == "~" {
		path = os.Getenv("HOME")
	}
	if strings.HasPrefix(path, "~/") { // In case our shell doesn't do it for us.
		path = filepath.Join(os.Getenv("HOME"), path[2:])
	}
	return filepath.Clean(path)
}

// MakeDir creates a directory path, along with any necessary parents, using the
// default permissions of 0755 (drwxr-xr-x).
func MakeDir(path string) error {
	return MakeDirFS(DefaultFileSystem, path)
}

func MakeDirFS(fs FileSystem, path string) error {
	dir, err := fs.Lstat(path)
	if err == nil && dir.IsDir() {
		return nil
	}
	log.Printf("mkdir: %s\n", path)
	return fs.MkdirAll(path, 0755)
}

// WriteFile writes data to a file. If the file doesn't exist, it is created with
// default permissions of 0644 (-rw-r--r--), otherwise it is truncated.
func WriteFile(filename string, data []byte) error {
	return WriteFileFS(DefaultFileSystem, filename, data)
}

func WriteFileFS(fs FileSystem, filename string, data []byte) error {
	return fs.WriteFile(filename, data, 0644)
}

// ReadFile reads the contents of a file.
func ReadFile(filename string) ([]byte, error) {
	return ReadFileFS(DefaultFileSystem, filename)
}

func ReadFileFS(fs FileSystem, filename string) ([]byte, error) {
	return fs.ReadFile(filename)
}
