package backup

import (
	// "fmt"
	"github.com/aviddiviner/inc/file"
	"github.com/aviddiviner/inc/util/test"
	"github.com/stretchr/testify/assert"
	"os"
	"sort"
	"testing"
	"time"
)

func mockFileBare() file.File {
	return file.File{
		Root:    "/",
		Name:    test.RandString(10),
		Size:    1234,
		Mode:    0644,
		ModTime: time.Now().Truncate(time.Second),
	}
}

func mockFile() file.File {
	f := mockFileBare()
	f.SHA1 = test.RandSHA1()
	return f
}

func mockSymlink() file.File {
	f := mockFile()
	f.Mode = os.ModeSymlink
	return f
}

func mockFileIn(root string) file.File {
	f := mockFile()
	f.Root = root
	return f
}

// -----------------------------------------------------------------------------

func reStatFiles(t *testing.T, files []file.File) []file.File {
	for i, f := range files {
		fi, err := file.DefaultFileSystem.Lstat(f.Path())
		assert.NoError(t, err)
		files[i].Size = fi.Size()
		files[i].ModTime = fi.ModTime()
	}
	return files
}

// -----------------------------------------------------------------------------

func TestFileBasics(t *testing.T) {
	f1 := mockFile()
	f2 := mockFileBare()

	assert.True(t, f1.HasChecksum())
	assert.False(t, f2.HasChecksum())

	symlink := mockSymlink()
	assert.True(t, symlink.IsSymlink())
}

func TestManifestMarshalling(t *testing.T) {
	files := []file.File{mockFile(), mockFile(), mockSymlink()}
	before := NewManifest(files)

	data, err := before.JSON()
	assert.NoError(t, err)

	after, err := ReadManifestData(data)
	assert.NoError(t, err)

	afterFiles := make([]file.File, len(after.Entries))
	for i, f := range after.Entries {
		afterFiles[i] = f.File
	}
	sort.Sort(file.ByPath(files))
	sort.Sort(file.ByPath(afterFiles))

	assert.Equal(t, files, afterFiles)
	assert.Equal(t, before.Entries, after.Entries)
}

// -----------------------------------------------------------------------------

func TestManifestAlterations(t *testing.T) {
	files := []file.File{mockFile(), mockFile(), mockFile()}
	sort.Sort(file.ByPath(files))
	m := NewManifest(files)

	checkEntriesAndPathMap := func() {
		// fmt.Println("----- checkEntriesAndPathMap -----")
		for i := 0; i < len(m.Entries); i++ {
			path := m.Entries[i].Path()
			// fmt.Printf("m.Entries[%d]: (*%p) %+v\n", i, m.Entries[i], *m.Entries[i])
			// fmt.Printf("m.pathMap[%q]: (*%p) %+v\n\n", path, m.pathMap[path], *m.pathMap[path])
			assert.True(t, m.Entries[i] == m.pathMap[path])  // pointers
			assert.Equal(t, *m.Entries[i], *m.pathMap[path]) // values
		}
	}

	assert.Equal(t, 3, len(m.Entries))
	assert.Equal(t, files[0], m.Entries[0].File)
	assert.Equal(t, files[1], m.Entries[1].File)
	assert.Equal(t, files[2], m.Entries[2].File)

	m.Remove(files[0])

	assert.Equal(t, 2, len(m.Entries))
	assert.Equal(t, 2, len(m.pathMap))
	checkEntriesAndPathMap()

	m.Update(files)

	assert.Equal(t, 3, len(m.Entries))
	assert.Equal(t, 3, len(m.pathMap))
	checkEntriesAndPathMap()

	// Check that we're still okay after expanding slices, etc.
	for i := 0; i < 5; i++ {
		bunchOfFiles := make([]file.File, 20)
		for j := 0; j < 20; j++ {
			bunchOfFiles[j] = mockFile()
		}
		m.Update(bunchOfFiles)
		checkEntriesAndPathMap()
	}
}

func TestManifestKeyIsAlwaysUnique(t *testing.T) {
	oldFiles := []file.File{mockFile(), mockFile(), mockFile()}
	newFiles := []file.File{mockFile(), mockFile()}

	m := NewManifest(oldFiles)
	oldKey := m.LastSet
	m.Update(newFiles)
	assert.NotEqual(t, oldKey, m.LastSet, "key should be different")
}

// -----------------------------------------------------------------------------

func TestBundlingFiles(t *testing.T) {
	fs := []file.File{
		mockFileIn("/"),
		mockFileIn("/bar"),
		mockFileIn("/"),
		mockFileIn("/foo"),
		mockFileIn("/"),
		mockFileIn("/bar"),
	}

	// Bundle by path and then by size within paths.
	bundled := bundleByPathAndSize(fs)
	assert.Len(t, bundled, 3, "correct number of bundles")
	assert.Len(t, bundled[0], 3, "bundle sized correctly /")
	assert.Len(t, bundled[1], 2, "bundle sized correctly /bar")
	assert.Len(t, bundled[2], 1, "bundle sized correctly /foo")

	// Bundle by size only.
	bundled = bundleSmallFiles(sortBundlesBySize(makeBundle(fs)))
	assert.Len(t, bundled, 1, "correct number of bundles")
	assert.Len(t, bundled[0], 6, "bundle sized correctly")

	fs[0].Size = 1e6 // Change a file to be 1MB

	// Bundle by path and then by size within paths.
	bundled = bundleByPathAndSize(fs)
	assert.Len(t, bundled, 4, "correct number of bundles")
	assert.Len(t, bundled[0], 2, "bundle sized correctly / (small files)")
	assert.Len(t, bundled[1], 1, "bundle sized correctly / (big file)")
	assert.Len(t, bundled[2], 2, "bundle sized correctly /bar")
	assert.Len(t, bundled[3], 1, "bundle sized correctly /foo")

	// Bundle by size only.
	bundled = bundleSmallFiles(sortBundlesBySize(makeBundle(fs)))
	assert.Len(t, bundled, 2, "correct number of bundles")
	assert.Len(t, bundled[0], 5, "bundle sized correctly (small files)")
	assert.Len(t, bundled[1], 1, "bundle sized correctly (big file)")
}

func TestComparingManifests(t *testing.T) {
	files := []file.File{createTestFile(t), createTestFile(t), createTestFile(t)}
	reScanFiles := func() []file.File {
		return file.NewScanner().
			IncludePath(files[0].Path()).
			IncludePath(files[1].Path()).
			IncludePath(files[2].Path()).
			Scan()
	}

	var diffs []file.File

	// Identical catalogs should compare the same.
	file.ChecksumFiles(files)
	manifest := NewManifest(files)
	diffs = manifest.Compare(reScanFiles())
	assert.Nil(t, diffs)

	// Touch a file, manifests should compare the same.
	oldTime := files[0].ModTime
	test.TouchFileTime(t, files[0].Path(), time.Now().Add(5*time.Second))
	files = reStatFiles(t, files)
	assert.NotEqual(t, oldTime, files[0].ModTime)
	diffs = manifest.Compare(reScanFiles())
	assert.Nil(t, diffs)

	// Append to a file, manifests should be different.
	test.AppendToFile(t, files[1].Path(), "\n")
	diffs = manifest.Compare(reScanFiles())
	assert.NotNil(t, diffs)
	assert.Len(t, diffs, 1)
}
