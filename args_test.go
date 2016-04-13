package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func assertFlagError(t *testing.T, cmdline string) {
	argv := strings.Split(cmdline, " ")
	_, err := docopt.Parse(usage, argv, false, "test", false, false)
	assert.Error(t, err, fmt.Sprintf("docopt.Parse should error for: %q", cmdline))
}

func assertFlagSuccess(t *testing.T, cmdline string) map[string]interface{} {
	argv := strings.Split(cmdline, " ")
	args, err := docopt.Parse(usage, argv, false, "test", false, false)
	assert.NoError(t, err, fmt.Sprintf("docopt.Parse should not error for: %q", cmdline))
	return args
}

func assertParseSuccess(t *testing.T, cmdline string) options {
	argv := strings.Split(cmdline, " ")
	opts, err := parseFlags(argv, false)
	assert.NoError(t, err, fmt.Sprintf("docopt.Parse should not error for: %q", cmdline))
	return opts
}

// Just some random tests for a few command line argument combinations. Not exhaustive or thorough by any means.
func TestArgs(t *testing.T) {
	assertFlagError(t, "init")
	assertFlagError(t, "init -f")
	assertFlagError(t, "backup")
	assertFlagError(t, "restore")
	assertFlagError(t, "restore foo/")

	args := assertFlagSuccess(t, "init --pass ABC")
	assert.EqualValues(t, "~/.backupinc.cfg", args["--cfg"], "default config path")
	assert.EqualValues(t, true, args["init"], "flag set")
	assert.EqualValues(t, false, args["backup"], "other flags not set")
	assert.EqualValues(t, false, args["restore"], "other flags not set")
	assert.EqualValues(t, "ABC", args["--pass"], "password set")

	args = assertFlagSuccess(t, "backup foo/")
	assert.EqualValues(t, "~/.backupinc.cfg", args["--cfg"], "default config path")
	assert.EqualValues(t, true, args["backup"], "flag set")
	assert.EqualValues(t, false, args["init"], "other flags not set")
	assert.EqualValues(t, false, args["restore"], "other flags not set")
	assert.EqualValues(t, []string{"foo/"}, args["<path>"], "paths set")

	args = assertFlagSuccess(t, "restore --dest ABC foo/")
	assert.EqualValues(t, "~/.backupinc.cfg", args["--cfg"], "default config path")
	assert.EqualValues(t, true, args["restore"], "flag set")
	assert.EqualValues(t, false, args["init"], "other flags not set")
	assert.EqualValues(t, false, args["backup"], "other flags not set")
	assert.EqualValues(t, "ABC", args["--dest"], "destination set")
	assert.EqualValues(t, []string{"foo/"}, args["<path>"], "paths set")
}

// A few more random tests for command line invocations and the resulting options that get set.
func TestOpts(t *testing.T) {
	opts := assertParseSuccess(t, "init --cfg /tmp/foo.cfg --pass ABC -f")
	assert.EqualValues(t, true, opts.storeInit)
	assert.EqualValues(t, "/tmp/foo.cfg", opts.configPath)
	assert.EqualValues(t, "ABC", opts.storeSecret)
	assert.EqualValues(t, true, opts.forceInit)

	opts = assertParseSuccess(t, "init --pass foobar --s3-bucket myspecialbucket --s3-region us-west-2")
	assert.EqualValues(t, true, opts.storeInit)
	assert.EqualValues(t, "foobar", opts.storeSecret)
	assert.EqualValues(t, "myspecialbucket", opts.s3Bucket)
	assert.EqualValues(t, "us-west-2", opts.s3Region)
	assert.EqualValues(t, false, opts.forceInit)

	opts = assertParseSuccess(t, "backup ~/pics ~/movies :~/movies/Hellboy.mkv")
	assert.EqualValues(t, []string{
		filepath.Join(os.Getenv("HOME"), "pics"),
		filepath.Join(os.Getenv("HOME"), "movies")}, opts.includePaths)
	assert.EqualValues(t, []string{
		filepath.Join(os.Getenv("HOME"), "movies/Hellboy.mkv")}, opts.excludePaths)

	opts = assertParseSuccess(t, "restore --dest /tmp/restore ~/code ~/pics")
	assert.EqualValues(t, []string{
		filepath.Join(os.Getenv("HOME"), "code"),
		filepath.Join(os.Getenv("HOME"), "pics")}, opts.includePaths)
	assert.EqualValues(t, "/tmp/restore", opts.restoreRoot)

	opts = assertParseSuccess(t, "backup --s3-key ABC --s3-secret DEF --storage fs --fs-root /tmp/fs ~")
	assert.EqualValues(t, "ABC", opts.awsAccessKey)
	assert.EqualValues(t, "DEF", opts.awsSecretKey)
	assert.EqualValues(t, "fs", opts.storageType)
	assert.EqualValues(t, "/tmp/fs", opts.fsRootFolder)
	assert.EqualValues(t, []string{os.Getenv("HOME")}, opts.includePaths)

	opts = assertParseSuccess(t, "scan ~")
	assert.EqualValues(t, true, opts.scanOnly)
	assert.EqualValues(t, []string{os.Getenv("HOME")}, opts.includePaths)
}
