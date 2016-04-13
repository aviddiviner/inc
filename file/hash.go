package file

import (
	"crypto/sha1"
	"github.com/aviddiviner/inc/util"
	"io"
	"log"
	"time"
)

func checksumFile(fs FileSystem, path string) (out [sha1.Size]byte, length int) {
	f, err := fs.OpenRead(path)
	if err != nil {
		log.Fatal("check: file read error. ", err, path)
	}
	defer f.Close()
	sum := sha1.New()
	n, err := io.Copy(sum, f)
	if err != nil {
		log.Fatal("check: file read error. ", err, path)
	}
	length = int(n)
	copy(out[:], sum.Sum(nil))
	return
}

func checksumSymlink(fs FileSystem, path string) ([sha1.Size]byte, int) {
	link, err := fs.Readlink(path)
	if err != nil {
		log.Fatal("check: link read error. ", err, path)
	}
	return sha1.Sum([]byte(link)), len(link)
}

// ChecksumFiles scans file contents on the DefaultFileSystem.
func ChecksumFiles(groups ...[]File) {
	ChecksumFilesFS(DefaultFileSystem, groups...)
}

// ChecksumFilesFS scans the contents of a list of files, calculating their SHA1
// checksums and populating the File details.
func ChecksumFilesFS(fs FileSystem, groups ...[]File) {
	start := time.Now()

	totalFiles := 0
	totalBytes := util.ByteCount(0)
	for _, files := range groups {
		for _, f := range files {
			if (f.IsRegular() || f.IsSymlink()) && !f.HasChecksum() {
				totalFiles += 1
				totalBytes += util.ByteCount(f.Size)
			}
		}
	}

	if totalFiles == 0 {
		return // no files to hash
	}

	log.Printf("check: calculating hashes for %d files (%s).\n", totalFiles,
		totalBytes)

	doneFiles := 0
	doneBytes := util.ByteCount(0)

	timer := util.NewTimer(1800, func() {
		progress := doneBytes / totalBytes * 100
		log.Printf("check: busy. %d files (%s, %.1f%%) hashed.\n", doneFiles,
			doneBytes, progress)
	})

	for i, files := range groups {
		for j, f := range files {
			var hash [sha1.Size]byte
			var length int
			if f.IsRegular() {
				hash, length = checksumFile(fs, f.Path())
			} else if f.IsSymlink() {
				hash, length = checksumSymlink(fs, f.Path())
			} else {
				continue
			}
			//log.Printf("check: read %q\n", f.Path())
			groups[i][j].SHA1 = hash
			doneFiles += 1
			doneBytes += util.ByteCount(length)
		}
	}

	timer.Stop()

	elapsed := time.Since(start)
	log.Printf("check: done. %d files (%s) hashed. took %s.\n", doneFiles,
		doneBytes, elapsed)
}
