package archive

import (
	"archive/tar"
	"errors"
	"github.com/aviddiviner/inc/file"
	"github.com/aviddiviner/inc/util"
	"io"
	"log"
	"os"
	"path/filepath"
)

var fs file.FileSystem = file.DefaultFileSystem

func RestoreDir(root string, entry file.File) error {
	if !entry.IsDir() {
		return errors.New("can only restore dirs from file header data")
	}
	path := filepath.Join(root, entry.Name)
	log.Printf("restore: %s (%s)\n", path, entry.Mode)

	// Create the directory.
	if err := fs.Mkdir(path, entry.Mode); err != nil {
		return err
	}
	// Set the owner uid/gid.
	if err := fs.Lchown(path, entry.UID, entry.GID); err != nil {
		return err
	}
	// Set the access/modification times.
	if err := fs.Chtimes(path, entry.ModTime, entry.ModTime); err != nil {
		return err
	}
	return nil
}

func UnpackReader(root string, tarball io.Reader, only map[string]file.File) error {
	var subdir string
	// Iterate through the files in the archive.
	tr := tar.NewReader(tarball)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // end of tarball
		}
		if err != nil {
			return err
		}

		if only != nil {
			// Does the tarball hdr.Name only have the filename and not the full path? (Old style tarballs.)
			// If so, we can be sure that all the files in this tarball are from the same root folder.
			if subdir == "" && filepath.Base(hdr.Name) == hdr.Name {
				log.Println("unpack: old-style tarball detected.")
				for _, f := range only {
					if subdir == "" {
						subdir = f.Root
					} else if subdir != f.Root { // Sanity check.
						return errors.New("tarball shouldn't contain files from different roots")
					}
				}
			}
			if subdir != "" {
				hdr.Name = filepath.Join(subdir, hdr.Name)
			}

			// If we couldn't find our path in the selected list, skip over this file.
			if _, ok := only[hdr.Name]; !ok {
				log.Printf("unpack: skipping file, not selected %s\n", hdr.Name)
				continue
			}
		}

		mode := hdr.FileInfo().Mode()
		path := filepath.Join(root, hdr.Name)

		// Ensure the folder exists.
		if err := file.MakeDir(filepath.Dir(path)); err != nil {
			return err
		}

		if _, err := fs.Lstat(path); !fs.IsNotExist(err) {
			log.Printf("unpack: skipping, already exists %s\n", path)
			continue
		}

		if mode.IsDir() {
			log.Printf("unpack: %s (%s)\n", path, mode)
			if err := fs.Mkdir(path, mode); err != nil {
				return err
			}
		} else if mode&os.ModeSymlink != 0 {
			log.Printf("unpack: %s (%s) (%s)\n", path, mode, util.ByteCount(len(hdr.Linkname)))
			if err := fs.Symlink(hdr.Linkname, path); err != nil {
				return err
			}
		} else {
			fh, err := fs.OpenWrite(path, mode)
			if err != nil {
				return err
			}
			n, err := io.Copy(fh, tr)
			if err != nil {
				return err
			}
			log.Printf("unpack: %s (%s) (%s)\n", path, mode, util.ByteCount(n))
			fh.Close()
		}

		// Set the owner uid/gid.
		if err := fs.Lchown(path, hdr.Uid, hdr.Gid); err != nil {
			// TODO: Handle this better.
		}
		// Set the access/modification times.
		if err := fs.Chtimes(path, hdr.AccessTime, hdr.ModTime); err != nil {
			// TODO: Handle this better.
		}
	}

	return nil
}

// -----------------------------------------------------------------------------

const c_FLUSH_SIZE = 65535

func PackReader(files ...file.File) io.ReadCloser {
	r, w := io.Pipe()
	tw := tar.NewWriter(w)

	go func() {
		for _, file := range files {
			// log.Printf("pack: %s (%s)\n", file.Path(), util.ByteCount(file.Size)) // TODO: Debug logging

			var link string
			var err error

			// Read link if symlink.
			if file.IsSymlink() {
				if link, err = fs.Readlink(file.Path()); err != nil {
					w.CloseWithError(err)
					return
				}
			}

			// Write the file header to the tarball.
			hdr, err := tar.FileInfoHeader(file.FileInfo(), link)
			if err != nil {
				w.CloseWithError(err)
				return
			}
			if err := tw.WriteHeader(hdr); err != nil {
				w.CloseWithError(err)
				return
			}

			// Move on to the next file header if symlink or dir.
			if file.IsSymlink() || file.IsDir() {
				continue
			}

			// Read the file contents and write them to the tarball. Flush at regular
			// intervals during the copy.
			fh, err := fs.OpenRead(file.Path())
			if err != nil {
				w.CloseWithError(err)
				return
			}
			for {
				if _, err = io.CopyN(tw, fh, c_FLUSH_SIZE); err != nil {
					if err == io.EOF {
						break
					}
					fh.Close()
					w.CloseWithError(err)
					return
				}
				tw.Flush()
			}
			fh.Close()
		}

		// Finished writing all files. Close tarball and write pipe.
		if err := tw.Close(); err != nil {
			w.CloseWithError(err)
			return
		}
		w.Close()
	}()

	return r
}
