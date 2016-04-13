package fs

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type FileSystem interface {
	// AbsPath returns an absolute representation of a path, after the evaluation of
	// any symbolic links. If the path is not absolute it will first be joined with
	// the current working directory to turn it into an absolute path.
	AbsPath(path string) (string, error)

	// Lstat returns a FileInfo describing the named file. If the file is a symbolic
	// link, the returned FileInfo describes the symbolic link. Lstat makes no
	// attempt to follow the link. If there is an error, it will be of type *os.PathError.
	Lstat(name string) (os.FileInfo, error)

	// Mkdir creates a new directory with the specified name and permission bits. If
	// there is an error, it will be of type *os.PathError.
	Mkdir(name string, perm os.FileMode) error

	// MkdirAll creates a directory named path, along with any necessary parents,
	// and returns nil, or else returns an error. The permission bits perm are used
	// for all directories that MkdirAll creates. If path is already a directory,
	// MkdirAll does nothing and returns nil.
	MkdirAll(path string, perm os.FileMode) error

	// RemoveAll removes path and any children it contains. It removes everything it can
	// but returns the first error it encounters. If the path does not exist, RemoveAll
	// returns nil (no error).
	RemoveAll(path string) error

	// OpenRead opens the named file for reading. If successful, methods on the returned
	// file can be used for reading; the associated file descriptor has mode O_RDONLY.
	// If there is an error, it will be of type *os.PathError.
	OpenRead(name string) (FileHandle, error)

	// OpenWrite creates the named file, truncating it if it already exists. If
	// successful, methods on the returned file can be used for writing; the associated
	// file descriptor has mode O_WRONLY. If there is an error, it will be of type
	// *os.PathError.
	OpenWrite(name string, perm os.FileMode) (FileHandle, error)

	// Readlink returns the destination of the named symbolic link. If there is an
	// error, it will be of type *os.PathError.
	Readlink(name string) (string, error)

	// Symlink creates newname as a symbolic link to oldname. If there is an error, it
	// will be of type *os.LinkError.
	Symlink(oldname, newname string) error

	// Lchown changes the numeric uid and gid of the named file. If the file is a
	// symbolic link, it changes the uid and gid of the link itself. If there is an
	// error, it will be of type *PathError.
	Lchown(name string, uid, gid int) error

	// Chtimes changes the access and modification times of the named file, similar to
	// the Unix utime() or utimes() functions.
	// The underlying filesystem may truncate or round the values to a less precise
	// time unit. If there is an error, it will be of type *os.PathError.
	Chtimes(name string, atime, mtime time.Time) error

	// IsNotExist returns a boolean indicating whether the error is known to report
	// that a file or directory does not exist. It is satisfied by ErrNotExist as
	// well as some syscall errors.
	IsNotExist(err error) bool

	// SysStat returns some additional metadata from system-dependent fields of fi.
	SysStat(fi os.FileInfo) (*FileStat_t, error)

	// ReadFile reads the file named by filename and returns the contents. A successful
	// call returns err == nil, not err == EOF. Because ReadFile reads the whole file,
	// it does not treat an EOF from Read as an error to be reported.
	ReadFile(filename string) ([]byte, error)

	// WriteFile writes data to a file named by filename. If the file does not exist,
	// WriteFile creates it with permissions perm; otherwise WriteFile truncates it
	// before writing.
	WriteFile(filename string, data []byte, perm os.FileMode) error
}

type FileStat_t struct {
	Uid int // User id of the file owner.
	Gid int // Group id of the file owner.

	// Atime is the file access time. OS dependant. For Linux, the atime gets updated
	// when you open a file but also when a file is used for other operations like
	// grep, sort, cat, head, tail and so on.
	Atime time.Time

	// Ctime is the inode or file change time. OS dependant. For Linux, the ctime
	// gets updated when the file attributes are changed, like changing the owner,
	// changing the permission or moving the file to an other filesystem but will
	// also be updated when you modify a file.
	// Mostly ctime and mtime will be the same, unless only the file attributes are
	// updated. In that case only the ctime gets updated.
	Ctime time.Time
}

// Subset of the os.File interface.
type FileHandle interface {
	// Readdir reads the contents of the directory associated with file and returns
	// a slice of up to n FileInfo values, as would be returned by Lstat, in directory
	// order. Subsequent calls on the same file will yield further FileInfos.
	Readdir(n int) (fi []os.FileInfo, err error)
	io.ReadWriteCloser
}

// -----------------------------------------------------------------------------

// OS is an interface to the actual OS filesystem.
var OS FileSystem = new(osFs)

type osFs struct{}

func (*osFs) AbsPath(path string) (abs string, err error) {
	abs, err = filepath.Abs(path)
	if err != nil {
		return
	}
	return filepath.EvalSymlinks(abs)
}
func (*osFs) Lstat(name string) (os.FileInfo, error) {
	return os.Lstat(name)
}
func (*osFs) Mkdir(name string, perm os.FileMode) error {
	return os.Mkdir(name, perm)
}
func (*osFs) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, os.ModeDir|perm)
}
func (*osFs) RemoveAll(path string) error {
	return os.RemoveAll(path)
}
func (*osFs) OpenRead(name string) (FileHandle, error) {
	return os.OpenFile(name, os.O_RDONLY, 0)
}
func (*osFs) OpenWrite(name string, perm os.FileMode) (FileHandle, error) {
	return os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
}
func (*osFs) Readlink(name string) (string, error) {
	return os.Readlink(name)
}
func (*osFs) Symlink(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}
func (*osFs) Lchown(name string, uid, gid int) error {
	return os.Lchown(name, uid, gid)
}
func (*osFs) Chtimes(name string, atime, mtime time.Time) error {
	return os.Chtimes(name, atime, mtime)
}
func (*osFs) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}
func (*osFs) SysStat(fi os.FileInfo) (*FileStat_t, error) {
	return sysStat(fi)
}
func (*osFs) ReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}
func (*osFs) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(filename, data, perm)
}
