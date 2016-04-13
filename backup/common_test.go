package backup

import (
	"github.com/aviddiviner/inc/file"
	"github.com/aviddiviner/inc/util/test"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	result := m.Run()
	test.CleanupTempFiles()
	os.Exit(result)
}

func createTestFile(t *testing.T) file.File {
	return file.ScanFile(test.CreateTempFile(t))
}
func createLargeTestFile(t *testing.T) file.File {
	return file.ScanFile(test.CreateLargeTempFile(t))
}
