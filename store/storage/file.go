package storage

import (
	"github.com/aviddiviner/inc/util"
	"io"
	"os"
	"path/filepath"
)

// FileStorage stores data locally on disk in some directory.
type FileStorage struct {
	root string
}

func makeDir(path string) (err error) {
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModeDir|0755)
	}
	return
}

func NewFileStorage(root string) *FileStorage {
	return &FileStorage{root}
}

func (fs *FileStorage) Exists() (bool, error) {
	fi, err := os.Stat(fs.root)
	if err == nil && fi.Mode()&(os.ModeDir|0700) != 0 {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (fs *FileStorage) Create() error {
	if err := os.MkdirAll(fs.root, os.ModeDir|0755); err != nil {
		return err
	}
	ok, err := fs.Exists()
	if err == nil && !ok {
		return os.ErrPermission // permission denied
	}
	return err
}

func (fs *FileStorage) Size(key string) (int, error) {
	fi, err := os.Stat(key)
	if err != nil {
		return 0, err
	}
	return int(fi.Size()), nil
}

func (fs *FileStorage) GetReader(key string) (io.Reader, error) {
	path := filepath.Join(fs.root, key)
	rc, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &util.AutoCloseReader{rc}, nil
}

func (fs *FileStorage) PutReader(key string, r io.Reader) (length int, err error) {
	path := filepath.Join(fs.root, key)
	err = makeDir(filepath.Dir(path))
	if err != nil {
		return
	}
	fh, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer fh.Close()
	n, err := io.Copy(fh, r)
	if err != nil {
		return
	}
	length = int(n)
	return
}

func (fs *FileStorage) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}
