package backup

import (
	"github.com/aviddiviner/inc/file"
	"github.com/aviddiviner/inc/file/archive"
	"github.com/aviddiviner/inc/store"
	"github.com/aviddiviner/inc/util"
	"log"
	"sort"
	"time"
)

const c_BUNDLE_LIMIT_SIZE = 1000 << 6 // don't bundle files >64KB
const c_BUNDLE_MAX_SIZE = 1000 << 10  // bundles are max. 1MB

// Create the initial bundle.
func makeBundle(files []file.File) (bundles [][]file.File) {
	if len(files) > 0 {
		bundles = append(bundles, files)
	}
	return
}

// Sort each bundle by file size.
func sortBundlesBySize(bundles [][]file.File) [][]file.File {
	for _, bundle := range bundles {
		sort.Sort(file.BySize(bundle))
	}
	return bundles
}

// Sort each bundle by path.
func sortBundlesByPath(bundles [][]file.File) [][]file.File {
	for _, bundle := range bundles {
		sort.Sort(file.ByPath(bundle))
	}
	return bundles
}

// Split each bundle on c_BUNDLE_LIMIT_SIZE. Splits into 3 parts; small files
// (<= cutoff size), large files (> cutoff size) and directories.
func splitBundlesBySizeLimit(in [][]file.File) (out [][]file.File) {
	for _, bundle := range in {
		var small, big, dirs []file.File

		for _, f := range bundle {
			switch {
			case f.IsDir():
				dirs = append(dirs, f)
			case f.Size > c_BUNDLE_LIMIT_SIZE:
				big = append(big, f)
			default:
				small = append(small, f)
			}
		}

		out = append(out, small, big, dirs)
	}
	return
}

// Bundle files with the same root together (note: you should sort before doing this).
func bundleByPath(in [][]file.File) (out [][]file.File) {
	var currRoot string
	var currBundle []file.File

	nextBundle := func() {
		if len(currBundle) > 0 {
			out = append(out, currBundle)
			currBundle = nil
		}
	}

	for _, bundle := range in {
		for _, f := range bundle {
			if f.Root != currRoot {
				nextBundle()
				currRoot = f.Root
			}
			currBundle = append(currBundle, f)
		}
		nextBundle()
	}

	return
}

// Bundle small files together.
func bundleSmallFiles(in [][]file.File) (out [][]file.File) {
	var currBytes int64
	var currBundle, dirBundle []file.File

	nextBundle := func() {
		if len(currBundle) > 0 {
			out = append(out, currBundle)
			currBundle = nil
			currBytes = 0
		}
	}

	for _, bundle := range in {
		for _, f := range bundle {
			if f.IsDir() {
				dirBundle = append(dirBundle, f)
				continue
			}
			if f.Size > c_BUNDLE_LIMIT_SIZE {
				nextBundle()
			}
			if currBytes > c_BUNDLE_MAX_SIZE {
				nextBundle()
			}
			currBundle = append(currBundle, f)
			currBytes += f.Size
		}
		nextBundle()
	}

	if len(dirBundle) > 0 {
		out = append(out, dirBundle)
	}

	return
}

// Traditional bundling approach; group files into similar paths, and then bundle by size within each path.
func bundleByPathAndSize(files []file.File) [][]file.File {
	return bundleSmallFiles(sortBundlesBySize(bundleByPath(sortBundlesByPath(makeBundle(files)))))
}

// New bundling approach; first split by size cutoff, then sort by path and bundle up small files.
func bundleSmallFilesAcrossPaths(files []file.File) [][]file.File {
	return bundleSmallFiles(sortBundlesByPath(splitBundlesBySizeLimit(makeBundle(files))))
}

// -----------------------------------------------------------------------------

func (m *Manifest) Update(files []file.File) time.Time {
	file.ChecksumFiles(files) // pre-populate hashes

	now := time.Now()
	m.LastSet = manifestKey(now)
	m.Updated = now.Truncate(time.Second)
	bundles := bundleSmallFilesAcrossPaths(files)
	nextKey := keyFactory(len(bundles))

	for _, bundle := range bundles {
		key := nextKey()
		for _, f := range bundle {
			var parts []ManifestEntryPart
			if !f.IsDir() {
				parts = []ManifestEntryPart{{Key: key}}
			}
			newEntry := &ManifestEntry{f, m.LastSet, parts}
			if m.Has(f) {
				*m.pathMap[f.Path()] = *newEntry // replace the entry
			} else {
				m.pathMap[f.Path()] = newEntry // add a new entry
				m.Entries = append(m.Entries, newEntry)
			}
		}
	}

	return now
}

func (m *Manifest) LatestEntries() map[string][]*ManifestEntry {
	entries := make(map[string][]*ManifestEntry)
	for _, e := range m.Entries {
		if e.Set == m.LastSet {
			for _, p := range e.Parts {
				obj := m.LastSet + "/" + p.Key
				entries[obj] = append(entries[obj], e)
			}
		}
	}
	return entries
}

// -----------------------------------------------------------------------------

// Semaphores for limiting how many uploads can run concurrently.
var uploadSem chan bool

// Chosen arbitrarily. TODO: Tweak this to find the best number.
const c_CONCURRENT_UPLOADS = 20

func init() {
	uploadSem = make(chan bool, c_CONCURRENT_UPLOADS)
	for i := 0; i < cap(uploadSem); i++ {
		uploadSem <- true
	}
}

// Backup changed files only.
func backupLatest(store *store.Store, m Manifest) (err error) {
	latest := m.LatestEntries()
	totalPuts := len(latest)
	if totalPuts > 0 {
		var donePuts int
		var doneBytes util.ByteCount
		for key, entries := range latest {
			var files []file.File
			for _, e := range entries {
				if e.IsDir() {
					continue // don't put an object in the store, use manifest
				}
				files = append(files, e.File)
			}
			<-uploadSem
			// This func is blocked on our semaphore above.
			go func(key string, files []file.File) {

				dropFromManifest := func() {
					totalPuts -= 1
					log.Printf("backup: failed to put %s, removing files from manifest.\n", key)
					for _, f := range files {
						m.Remove(f)
						log.Printf("backup: removed %q\n", f.Path())
					}
					uploadSem <- true
				}

				tarball := archive.PackReader(files...)
				packer, err := store.Pack("blob/" + key)
				if err != nil {
					dropFromManifest()
					return
				}

				n, err := packer.PutReader(tarball)
				if err != nil {
					dropFromManifest()
					return
				}

				err = packer.Close()
				if err != nil {
					dropFromManifest()
					return
				}

				donePuts += 1
				doneBytes += util.ByteCount(n)
				log.Printf("backup: [%s] stored %d files (%s, %d/%d)\n", key, len(files), util.ByteCount(n), donePuts, totalPuts)
				uploadSem <- true
			}(key, files)
		}
		// Synchronize on all uploads being finished.
		for i := 0; i < cap(uploadSem); i++ {
			<-uploadSem
		}
		for i := 0; i < cap(uploadSem); i++ {
			uploadSem <- true
		}
		log.Printf("backup: finished saving data. put %d objects (%s)\n", donePuts, doneBytes)
		return saveManifest(store, m)
	}
	log.Println("no new entries to store.")
	return nil
}
