package file

import (
	"github.com/aviddiviner/inc/file/fs"
	"github.com/aviddiviner/inc/util"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Chosen arbitrarily. Higher = faster (to a point). Can probably be 1000 or
// more if we want. Limited by the OS; max allowed open file handles.
const c_CONCURRENT_FILES = 10

func foundFile(fs fs.FileSystem, pwd string, fi os.FileInfo) File {
	stat, _ := fs.SysStat(fi)
	f := File{
		Root:    pwd,
		Name:    fi.Name(),
		Size:    fi.Size(),
		Mode:    fi.Mode(),
		ModTime: fi.ModTime(),
		UID:     stat.Uid,
		GID:     stat.Gid,
	}
	return f
}

// ScanFile scans a single file path.
func ScanFile(path string) File {
	return ScanFileFS(DefaultFileSystem, path)
}

func ScanFileFS(fs fs.FileSystem, path string) File {
	abs, err := fs.AbsPath(path)
	if err != nil {
		log.Fatal("scan: error finding absolute path. ", err)
	}
	fi, err := fs.Lstat(path)
	if err != nil {
		log.Fatal("scan: file stat error. ", err)
	}
	return foundFile(fs, filepath.Dir(abs), fi)
}

// -----------------------------------------------------------------------------

// PathScanner scans multiple file paths in parallel.
type PathScanner struct {
	fs   fs.FileSystem
	incl []string
	excl map[string]bool
	wait sync.WaitGroup
	//chErr chan error
}

// NewScanner scans paths on the DefaultFileSystem.
func NewScanner() *PathScanner {
	return NewScannerFS(DefaultFileSystem)
}

func NewScannerFS(fs fs.FileSystem) *PathScanner {
	return &PathScanner{fs: fs, excl: make(map[string]bool)}
}

// Include a path to be scanned. Will always resolve paths to an absolute path,
// based on current working dir.
func (s *PathScanner) IncludePath(path string) *PathScanner {
	abs, err := s.fs.AbsPath(path)
	if err != nil {
		log.Fatal("scan: error finding absolute path. ", err)
	}
	log.Printf("scan: including path: %q\n", abs)
	s.incl = append(s.incl, abs)
	return s
}

// Exclude a path from scanning. This supersedes the included paths.
func (s *PathScanner) ExcludePath(path string) *PathScanner {
	abs, err := s.fs.AbsPath(path)
	if err != nil {
		log.Println("scan: error finding absolute path. ", err)
		return s
	}
	if !s.excl[abs] {
		log.Printf("scan: excluding path: %q\n", abs)
		s.excl[abs] = true
	}
	return s
}

// Walk the contents of a folder and send the results over the channels.
func (s *PathScanner) walkDir(pwd string, chDir, chAll chan File) {
	fd, err := s.fs.OpenRead(pwd)
	if err != nil {
		log.Fatal("scan: open error. ", err)
	}
	defer fd.Close()
loop:
	for {
		files, err := fd.Readdir(1) // TODO: tweak batch size.. 1 seems fastest?
		switch err {
		case io.EOF:
			break loop
		case nil:
			for _, fi := range files {
				s.tagFile(pwd, fi, chDir, chAll)
			}
		default:
			log.Fatal("scan: readdir error. ", err)
		}
	}
}

// Emit a new found file entry over the channels. If it's a directory, we queue
// it up for scanning its contents.
func (s *PathScanner) tagFile(pwd string, fi os.FileInfo, chDir, chAll chan File) {
	if f := foundFile(s.fs, pwd, fi); !s.excl[f.Path()] {
		if f.IsDir() {
			s.wait.Add(1)
			chDir <- f
		}
		chAll <- f
	}
}

// Recursively scan the included paths, sending found files and dirs on the
// returned channel.
func (s *PathScanner) scanRecursive() chan File {
	sem := make(chan bool, c_CONCURRENT_FILES)
	for i := 0; i < cap(sem); i++ {
		sem <- true
	}

	chDir := make(chan File)
	chAll := make(chan File)

	// Kick off a directory listing, limiting the number of open dirs at once.
	doWalkDir := func(pwd string) {
		<-sem
		s.walkDir(pwd, chDir, chAll)
		sem <- true
		s.wait.Done()
	}

	// Start walking all of the included paths.
	s.wait.Add(len(s.incl))
	for _, root := range s.incl {
		go func(path string) {
			<-sem

			// Get the file details (os.FileInfo).
			fi, err := s.fs.Lstat(path)
			if err != nil {
				log.Fatal("scan: file stat error. ", err)
			}
			s.tagFile(filepath.Dir(path), fi, chDir, chAll)

			sem <- true
			s.wait.Done()
		}(root)
	}

	// Keep grabbing new found dirs and kicking off new directory listings.
	go func() {
	loop:
		for {
			select {
			case d, ok := <-chDir:
				if ok {
					go doWalkDir(d.Path())
				} else { // channel closed
					break loop
				}
			}
		}
	}()

	// Wait for all the recursive directory listing to finish and when we're done
	// close the channels, signalling the goroutine above to exit.
	go func() {
		s.wait.Wait()
		close(chDir)
		close(chAll)
	}()

	return chAll
}

// -----------------------------------------------------------------------------

// Scan performs the scan. Comparable speed to a `find ... -mtime 1`, as it does
// a syscall.ReadDirent as well as syscall.Lstat (and Stat_t) for every file.
func (s *PathScanner) Scan() []File {
	start := time.Now()
	chAll := s.scanRecursive()

	foundDirs := 0
	foundFiles := 0
	foundBytes := util.ByteCount(0)

	// Get everything into a big slice.
	var entries []File
	timer := util.NewTimer(1800, func() {
		log.Printf("scan: busy. %d folders crawled. %d files (%s) found.\n",
			foundDirs, foundFiles, foundBytes)
	})
	for f := range chAll {
		if f.IsDir() {
			foundDirs += 1
		} else {
			if f.IsRegular() || f.IsSymlink() {
				foundFiles += 1
				foundBytes += util.ByteCount(f.Size)
			}
		}
		entries = append(entries, f)
	}
	timer.Stop()

	elapsed := time.Since(start)
	log.Printf("scan: done. %d folders crawled. %d files (%s) found. took %s.\n",
		foundDirs, foundFiles, foundBytes, elapsed)
	return entries
}

// ScanRelativeTo performs the scan, changing the root path of scanned files to
// be relative to some new root.
func (s *PathScanner) ScanRelativeTo(root string) []File {
	entries := s.Scan()
	updated := entries[:0] // same backing array
	basepath, err := s.fs.AbsPath(root)
	if err != nil {
		log.Fatal("scan: error finding absolute path. ", err)
	}

	log.Printf("scan: finding root paths relative to: %q.\n", basepath)
	for _, file := range entries {
		rel, err := filepath.Rel(basepath, file.Root)
		if err != nil {
			log.Fatal("scan: error finding relative path. ", err)
		}
		if rel != ".." { // ignore scanned files outside our new root. TODO: tests.
			file.Root = filepath.Join("/", rel)
			updated = append(updated, file)
		}
	}

	return updated
}
