package main

import (
	"errors"
	"github.com/aviddiviner/inc/backup"
	"github.com/aviddiviner/inc/file"
	"github.com/aviddiviner/inc/store"
	"github.com/aviddiviner/inc/store/storage"
	"github.com/aviddiviner/inc/util/test"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"testing"
	"time"
)

var testConfigPath = "test_fixtures/config.v2.json"
var testPassword = "mysupersecretpassword"

var testRequestFault = errors.New("general test fault")

// Create a callback that will fail every n-th request.
func newRequestFaultEveryN(n int) storage.MockRequestFault {
	reqCount := 0
	return func(key string) (err error) {
		switch {
		case key == "metadata":
			// Always succeed for metadata.
		case strings.HasPrefix(key, "manifest/"):
			// Always succeed for manifests.
		default:
			reqCount += 1
			if reqCount%n == 0 {
				err = testRequestFault
				log.Printf("store.mock: failing with %s\n", err)
			}
			log.Println("store.mock: succeeding")
		}
		return
	}
}

// -----------------------------------------------------------------------------

func setupMockStore(t *testing.T, opt options) (vault *store.Store, layer *storage.MockStorage, origCfg, storeCfg LocalConfigStore) {
	cfg, err := LoadConfigFile(testConfigPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, cfg)

	origCfg = cfg.Store
	storeCfg = cfg.Store

	layer = storage.NewMockStorage()
	layer.PutString("metadata", `{"version":1,"storeFormat":1,"salt":"5+ZOMGkPADM="}`)

	vault = store.NewStore(layer, "test")
	assert.NoError(t, err)

	if opt.storeInit {
		if opt.storeSecret != "" {
			storeCfg.Keys, err = vault.Wipe([]byte(opt.storeSecret))
			assert.NoError(t, err)
		} else {
			t.Fail()
		}
	} else {
		if opt.storeSecret != "" {
			storeCfg.Keys, err = vault.Unlock([]byte(opt.storeSecret))
			assert.NoError(t, err)
		}
	}

	err = vault.Open(storeCfg.Keys)
	assert.NoError(t, err)

	return
}

// -----------------------------------------------------------------------------

var tempTestDir string

func createTempDir() {
	deleteTempDir()
	if err := file.DefaultFileSystem.MkdirAll(tempTestDir, 0755); err != nil {
		panic(err)
	}
}
func deleteTempDir() {
	if err := file.DefaultFileSystem.RemoveAll(tempTestDir); err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	result := m.Run()
	test.CleanupTempFiles()
	os.Exit(result)
}

func TestLoadingOlderConfigVersions(t *testing.T) {
	cfg, err := LoadConfigFile("test_fixtures/config.v1.json")
	assert.NoError(t, err)
	assert.NotEmpty(t, cfg)
	assert.Equal(t, "us-west-2", cfg.Store.S3Region)
}

func TestLoadingAndSavingConfig(t *testing.T) {
	cfg, err := LoadConfigFile(testConfigPath)
	assert.NoError(t, err)
	assert.Equal(t, "test_fixtures/", cfg.Paths.Include[0])
	err = cfg.WriteToFile(testConfigPath + "~")
	assert.NoError(t, err)
}

func TestStoreSetup(t *testing.T) {
	_, _, o, cfg := setupMockStore(t, options{})
	assert.Equal(t, o, cfg)

	_, _, o, cfg = setupMockStore(t, options{storeSecret: testPassword})
	assert.Equal(t, o, cfg)

	_, _, o, cfg = setupMockStore(t, options{storeInit: true, storeSecret: testPassword})
	assert.NotEqual(t, o, cfg)
}

// Get a list of files, sorted by path, ready for comparing.
func lsFiles(path string) []file.File {
	ls := file.NewScanner().IncludePath(path).ScanRelativeTo(path)
	sort.Sort(file.ByPath(ls))
	for i := range ls {
		// Folder mod times will probably differ, so just zero them out.
		ls[i].ModTime = time.Time{}
	}
	return ls
}

func TestBackupAndRestore(t *testing.T) {
	test.RandSeed(42)
	tempTestDir := test.CreateTempDir(t)

	var cfg LocalConfig
	opts := options{
		includePaths: []string{"test_fixtures/sample_files/"},
	}
	vault, _, _, _ := setupMockStore(t, opts)

	backupPath, err := file.DefaultFileSystem.AbsPath("test_fixtures/sample_files/")
	assert.NoError(t, err)
	restorePath := path.Join(tempTestDir, backupPath)
	restorePaths := []string{backupPath}

	assert.NoError(t, backup.ScanAndBackup(vault, scanFiles(cfg.Paths, opts))) // Backup (all files).
	assert.NoError(t, backup.RestoreToPath(vault, tempTestDir, restorePaths))  // Restore (all files).

	lsBackup := lsFiles(backupPath)
	lsRestore := lsFiles(restorePath)

	assert.Equal(t, lsBackup, lsRestore, "restored files are the same")
}

func TestRestoreSelectedPaths(t *testing.T) {
	test.RandSeed(324)
	tempTestDir := test.CreateTempDir(t)

	var cfg LocalConfig
	opts := options{
		includePaths: []string{"test_fixtures/sample_files/"},
	}
	vault, _, _, _ := setupMockStore(t, opts)

	backupPath, err := file.DefaultFileSystem.AbsPath("test_fixtures/sample_files/")
	assert.NoError(t, err)
	restorePath := path.Join(tempTestDir, backupPath)
	restorePaths := []string{path.Join(backupPath, "1-lorem")}

	assert.NoError(t, backup.ScanAndBackup(vault, scanFiles(cfg.Paths, opts))) // Backup (all files).
	assert.NoError(t, backup.RestoreToPath(vault, tempTestDir, restorePaths))  // Restore (single files).

	lsBackup := lsFiles(backupPath)
	lsRestore := lsFiles(restorePath)

	assert.NotEqual(t, lsBackup, lsRestore, "restored files are different")
	assert.Equal(t, 1, len(lsRestore), "only 1 file restored")
}

func TestAnotherBackupOverBrokenNetwork(t *testing.T) {
	test.RandSeed(43)
	tempTestDir := test.CreateTempDir(t)

	var cfg LocalConfig
	var opts options
	vault, layer, _, _ := setupMockStore(t, opts)

	// Start with all files except one.
	opts = options{
		includePaths: []string{"test_fixtures/sample_files/"},
		excludePaths: []string{"test_fixtures/sample_files/1-lorem", "test_fixtures/does_not_exist.txt"},
	}

	backupPath, err := file.DefaultFileSystem.AbsPath("test_fixtures/sample_files/")
	assert.NoError(t, err)
	restorePaths := []string{path.Join(backupPath)}

	layer.InjectRequestFault(newRequestFaultEveryN(2)) // Break the network.

	assert.NoError(t, backup.ScanAndBackup(vault, scanFiles(cfg.Paths, opts)))
	assert.NoError(t, backup.RestoreToPath(vault, tempTestDir, restorePaths))

	// Now backup all files.
	opts = options{
		includePaths: []string{"test_fixtures/sample_files/"},
	}

	assert.NoError(t, backup.ScanAndBackup(vault, scanFiles(cfg.Paths, opts)))
	assert.Error(t, backup.RestoreToPath(vault, tempTestDir, restorePaths)) // Injected fault.
	assert.NoError(t, backup.RestoreToPath(vault, tempTestDir, restorePaths))
	assert.NoError(t, backup.RestoreToPath(vault, tempTestDir, restorePaths)) // No files changed.

	layer.ClearRequestFaults() // Fix the network.

	nextTestDir := test.CreateTempDir(t)
	assert.NoError(t, backup.RestoreToPath(vault, nextTestDir, restorePaths))
}

// -----------------------------------------------------------------------------

func TestLoadingV1ManifestFile(t *testing.T) {
	data, err := ioutil.ReadFile("test_fixtures/manifest.v1.json")
	assert.NoError(t, err)

	m, err := backup.ReadManifestData(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, m)

	assert.Equal(t, 7, len(m.Entries))
	assert.Equal(t, "056842ac4", m.LastSet)
	assert.True(t, m.Updated.IsZero())

	json, err := m.JSON()
	assert.NoError(t, err)
	ioutil.WriteFile("test_fixtures/manifest.v1.json~", json, 0644)
}

func TestLoadingV2ManifestFile(t *testing.T) {
	data, err := ioutil.ReadFile("test_fixtures/manifest.v2.json")
	assert.NoError(t, err)

	m, err := backup.ReadManifestData(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, m)

	assert.Equal(t, 9, len(m.Entries))
	assert.NotNil(t, *m.Entries[0])
	assert.Equal(t, "1424c5b57fef6895", m.LastSet)
	assert.True(t, m.Updated.IsZero())

	json, err := m.JSON()
	assert.NoError(t, err)
	ioutil.WriteFile("test_fixtures/manifest.v2.json~", json, 0644)
}

func TestLoadingV3ManifestFile(t *testing.T) {
	data, err := ioutil.ReadFile("test_fixtures/manifest.v3.json")
	assert.NoError(t, err)

	m, err := backup.ReadManifestData(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, m)

	assert.Equal(t, 13, len(m.Entries))
	assert.NotNil(t, *m.Entries[0])
	assert.Equal(t, "1426f9f4131b13f8", m.LastSet)
	assert.False(t, m.Updated.IsZero())

	json, err := m.JSON()
	assert.NoError(t, err)
	ioutil.WriteFile("test_fixtures/manifest.v3.json~", json, 0644)
}
