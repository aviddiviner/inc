package file

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type File struct {
	Root    string          // Parent path of the file.
	Name    string          // Base name of the file.
	Size    int64           // Length in bytes for regular files.
	Mode    os.FileMode     // File mode and permission bits.
	ModTime time.Time       // Last modification time.
	UID     int             // User identifier of owner.
	GID     int             // Group identifier of owner.
	SHA1    [sha1.Size]byte // Checksum of the file contents.
}

// Implements os.FileInfo.
type fileInfo struct{ f *File }

func (f *File) FileInfo() os.FileInfo  { return fileInfo{f} }
func (fi fileInfo) Name() string       { return fi.f.Path() }
func (fi fileInfo) Size() int64        { return fi.f.Size }
func (fi fileInfo) Mode() os.FileMode  { return fi.f.Mode }
func (fi fileInfo) ModTime() time.Time { return fi.f.ModTime }
func (fi fileInfo) IsDir() bool        { return fi.f.Mode.IsDir() }
func (fi fileInfo) Sys() interface{}   { return fi.f }

// Path returns the full file path (Root+Name).
func (f File) Path() string {
	return filepath.Join(f.Root, f.Name)
}

// IsDir checks if this is a directory.
func (f File) IsDir() bool {
	return f.Mode.IsDir()
}

// IsRegular checks if this is a regular file.
func (f File) IsRegular() bool {
	return f.Mode.IsRegular()
}

// IsSymlink checks if this is a symlink.
func (f File) IsSymlink() bool {
	return f.Mode&os.ModeSymlink != 0
}

// HasChecksum checks if the SHA1 for this file has been populated.
func (f File) HasChecksum() bool {
	return f.SHA1 != [sha1.Size]byte{}
}

// String representation for nicer logging / console output.
func (f File) String() string {
	if f.HasChecksum() {
		return fmt.Sprintf("File{Root:%q, Name:%q, Size:%d, Mode:%q, ModTime:%q, SHA1:%x}",
			f.Root, f.Name, f.Size, f.Mode, f.ModTime, f.SHA1)
	} else {
		return fmt.Sprintf("File{Root:%q, Name:%q, Size:%d, Mode:%q, ModTime:%q}",
			f.Root, f.Name, f.Size, f.Mode, f.ModTime)
	}
}
