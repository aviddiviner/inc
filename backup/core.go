package backup

import (
	"github.com/aviddiviner/inc/file"
	"github.com/aviddiviner/inc/file/archive"
	"github.com/aviddiviner/inc/file/fs"
	"github.com/aviddiviner/inc/store"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Write the latest manifest to the store and to disk.
func saveManifest(bucket *store.Store, m Manifest) (err error) {
	localFile := "manifest.json~"
	data, err := m.JSON()
	if err != nil {
		return
	}
	err = file.WriteFile(localFile, data)
	if err != nil {
		return
	}
	log.Printf("core: wrote file %q with manifest data\n", localFile)
	_, err = bucket.Put("manifest/"+m.LastSet, data)
	if err != nil {
		return
	}
	err = bucket.PutMetadata("manifest/latest", m.LastSet)
	return
}

var defaultCachePath = filepath.Join(os.Getenv("HOME"), ".inc", "cache")

func cacheGetStoreObject(bucket *store.Store, key string) (data []byte, err error) {
	cacheFile := filepath.Join(defaultCachePath, strings.Replace(bucket.ID()+"/"+key, "/", "_", -1))
	if _, e := os.Stat(cacheFile); os.IsNotExist(e) {
		data, err = bucket.Get(key)
		if err == nil {
			file.DefaultFileSystem.MkdirAll(defaultCachePath, 0755)
			file.DefaultFileSystem.WriteFile(cacheFile, data, 0644)
		}
		return
	}
	log.Printf("core: found cached object: %q (%s)\n", key, cacheFile)
	return file.DefaultFileSystem.ReadFile(cacheFile)
}

// Get the latest manifest that was written to the store.
func getLatestManifest(bucket *store.Store) (data []byte, err error) {
	lastSet, err := bucket.GetMetadata("manifest/latest")
	switch err {
	case store.ErrMissingMetadata:
		// No metadata; older or empty store. Check "manifest/latest" object.
		return cacheGetStoreObject(bucket, "manifest/latest")
	case nil:
		// Found metadata; fetch the last manifest id.
		if val, ok := lastSet.(string); ok {
			return cacheGetStoreObject(bucket, "manifest/"+val)
		}
		err = store.ErrMalformedMetadata
	}
	return nil, err
}

// Write a manifest file from some path scan.
func WriteManifest(filename string, scanner *file.PathScanner) (err error) {
	m := NewManifest(scanner.Scan())
	data, err := m.JSON()
	if err != nil {
		return
	}
	err = file.WriteFile(filename, data)
	return
}

// Scan a path for changes (compared to latest manifest) and upload the diff.
func ScanAndBackup(bucket *store.Store, scanner *file.PathScanner) error {
	ls := scanner.Scan()
	if len(ls) > 0 {
		// Fetch last manifest.
		data, err := getLatestManifest(bucket)
		switch {
		case err == nil:
			// No error; read the manifest and update it with new files for backup.
			m, err := ReadManifestData(data)
			if err != nil {
				return err
			}
			changed := m.Compare(ls)
			if len(changed) > 0 {
				m.Update(changed)
				err = backupLatest(bucket, m)
				if err != nil {
					return err
				}
			}
		case bucket.IsNotExist(err):
			// Manifest not found; create a new one with files for backup.
			m := NewManifest(ls)
			err = backupLatest(bucket, m)
			if err != nil {
				return err
			}
		default:
			// Other error; bail out.
			return err
		}
	}
	return nil
}

// Restore changed files from the store to a particular folder.
// Will do an incremental restore and only write the files that are different.
func RestoreToPath(bucket *store.Store, root string, incl []string) error {
	// Fetch last manifest.
	data, err := getLatestManifest(bucket)
	if err != nil {
		return err
	}
	m, err := ReadManifestData(data)
	if err != nil {
		return err
	}

	// Ensure the root folder exists.
	if err := file.MakeDir(root); err != nil {
		return err
	}

	// TODO: Optimise.
	included := func(path string) bool {
		for _, p := range incl {
			if strings.HasPrefix(path, p) {
				return true
			}
		}
		return false
	}

	// Map of which blobs to fetch, containing the files for restore.
	targets := make(map[string][]file.File)

	// Scan and tag for restore.
	localFiles := file.NewScanner().IncludePath(root).ScanRelativeTo(root)
	// TODO: localFiles := file.NewScannerFS(subFs).IncludePath("/").Scan()
	subFs, err := fs.NewSubdirFS(root)
	if err != nil {
		panic(err)
	}
	file.ChecksumFilesFS(subFs, localFiles)

	local := NewManifest(localFiles)
	for _, e := range m.Entries {
		if !local.HasIdentical(e.File) && included(e.Path()) {
			subdir := path.Join(root, path.Dir(e.Path()))
			if e.IsDir() { // restore directly from the manifest data
				file.MakeDir(subdir)
				if err := archive.RestoreDir(subdir, e.File); err != nil {
					// TODO: Handle this better.
				}
			} else {
				key := "blob/" + e.Set + "/" + e.Parts[0].Key
				targets[key] = append(targets[key], e.File)
			}
		}
	}

	// Fetch blobs and restore selected files from each blob.
	for key, list := range targets {
		only := make(map[string]file.File)
		for _, f := range list {
			only[f.Path()] = f
		}
		tarball, err := bucket.GetReader(key)
		if err != nil {
			return err
		}
		if err := archive.UnpackReader(root, tarball, only); err != nil {
			return err
		}
	}

	return nil
}
