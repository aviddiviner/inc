package fs

import (
	"os"
	"path/filepath"
	"time"
)

// NewSubdirFS returns a virtual filesystem where the real root is located on
// disk in some subdirectory. It transparently maps files to their real paths.
func NewSubdirFS(root string) (fs FileSystem, err error) {
	root, err = filepath.Abs(root)
	if err != nil {
		return
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return
	}
	return &subdirFs{root, OS}, nil
}

type subdirFs struct {
	root string
	osFs FileSystem
}

func (fs *subdirFs) realPath(path string) string {
	return filepath.Join(fs.root, path)
}
func (fs *subdirFs) AbsPath(path string) (abs string, err error) {
	abs, err = filepath.Abs(path)
	if err != nil {
		return
	}
	// Translate to the real path before we follow symlinks.
	abs = fs.realPath(abs)
	defer func() {
		abs, _ = filepath.Rel(fs.root, abs)
	}()
	abs, err = filepath.EvalSymlinks(abs)
	return
}
func (fs *subdirFs) Lstat(name string) (os.FileInfo, error) {
	return fs.osFs.Lstat(fs.realPath(name))
}
func (fs *subdirFs) Mkdir(name string, perm os.FileMode) error {
	return fs.osFs.Mkdir(fs.realPath(name), perm)
}
func (fs *subdirFs) MkdirAll(path string, perm os.FileMode) error {
	return fs.osFs.MkdirAll(fs.realPath(path), perm)
}
func (fs *subdirFs) RemoveAll(path string) error {
	return fs.osFs.RemoveAll(fs.realPath(path))
}
func (fs *subdirFs) OpenRead(name string) (FileHandle, error) {
	return fs.osFs.OpenRead(fs.realPath(name))
}
func (fs *subdirFs) OpenWrite(name string, perm os.FileMode) (FileHandle, error) {
	return fs.osFs.OpenWrite(fs.realPath(name), perm)
}
func (fs *subdirFs) Readlink(name string) (string, error) {
	return fs.osFs.Readlink(fs.realPath(name))
}
func (fs *subdirFs) Symlink(oldname, newname string) error {
	return fs.osFs.Symlink(fs.realPath(oldname), fs.realPath(newname))
}
func (fs *subdirFs) Lchown(name string, uid, gid int) error {
	return fs.osFs.Lchown(fs.realPath(name), uid, gid)
}
func (fs *subdirFs) Chtimes(name string, atime, mtime time.Time) error {
	return fs.osFs.Chtimes(fs.realPath(name), atime, mtime)
}
func (fs *subdirFs) IsNotExist(err error) bool {
	return fs.osFs.IsNotExist(err)
}
func (fs *subdirFs) SysStat(fi os.FileInfo) (*FileStat_t, error) {
	return fs.osFs.SysStat(fi)
}
func (fs *subdirFs) ReadFile(filename string) ([]byte, error) {
	return fs.osFs.ReadFile(fs.realPath(filename))
}
func (fs *subdirFs) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return fs.osFs.WriteFile(fs.realPath(filename), data, perm)
}
