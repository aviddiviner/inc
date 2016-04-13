package main

import (
	"errors"
	"fmt"
	"github.com/aviddiviner/inc/backup"
	"github.com/aviddiviner/inc/file"
	"github.com/aviddiviner/inc/store"
	"github.com/docopt/docopt-go"
	"log"
	"os"
	"runtime"
	"strings"
)

var usage = `Incremental remote backup utility.

Usage:
  inc init    [--cfg FILE] --pass SECRET [--storage TYPE] [--s3-key KEY]
              [--s3-secret KEY] [--s3-region NAME] [--s3-bucket NAME]
              [--fs-root PATH] [-f]
  inc backup  [--cfg FILE] [--pass SECRET] [--storage TYPE] [--s3-key KEY]
              [--s3-secret KEY] [--s3-region NAME] [--s3-bucket NAME]
              [--fs-root PATH] <path>...
  inc restore [--cfg FILE] [--pass SECRET] [--storage TYPE] [--s3-key KEY]
              [--s3-secret KEY] [--s3-region NAME] [--s3-bucket NAME]
              [--fs-root PATH] --dest DIR <path>...
  inc scan <path>...
  inc -h | --help
  inc --version

Commands:
  init              Initialize the store for first use. Will create the S3 bucket or folder as required.
  backup            Back up files to the store.
  restore           Restore files from the store.
  scan              Scan files and generate a manifest.json file. Don't perform any backup/restore.

Options:
  --cfg FILE        Config file to read (if it exists) or write to. [default: ~/.inc.cfg]
  -f --force        Force initialization. (WARNING: This will overwrite existing data in the store.)
  --pass SECRET     Encryption password. Used on first initialization, or when unlocking the store.
  --storage TYPE    Storage medium to use (s3, fs). [default: s3]
  --s3-key KEY      AWS access key. (defaults to $AWS_ACCESS_KEY, or reads $HOME/.aws/credentials)
  --s3-secret KEY   AWS secret key. (defaults to $AWS_SECRET_KEY, or reads $HOME/.aws/credentials)
  --s3-region NAME  AWS region where S3 bucket should be located. (e.g. us-west-2)
  --s3-bucket NAME  S3 bucket name. Note: bucket names are globally unique.
  --fs-root PATH    Root path to store files when using filesystem (fs) as storage.
  --dest DIR        Destination path to restore files to.
  -h --help         Show this screen.
  --version         Show version.

Backup examples:
  inc init --pass foobar --s3-bucket myspecialbucket --s3-region us-west-2
  inc backup ~/code ~/pics ~/movies

Any path with a leading colon (:) will be excluded from the backup. For example:
  inc backup ~/pics ~/movies :~/movies/Hellboy.mkv

Restore examples:
  inc restore --dest /tmp/restore ~/code ~/pics`

var buildTag = fmt.Sprintf("%s [%s] %s/%s", BUILD_DATE, BUILD_COMMIT, runtime.GOOS, runtime.GOARCH)

func parseFlags(argv []string, exit ...bool) (opt options, err error) {
	args, err := docopt.Parse(
		fmt.Sprintf("%s\n\n(build: %s)", usage, buildTag),
		argv,     // command line args
		true,     // help enabled
		buildTag, // --version tag
		false,    // require options first
		exit...,  // os.Exit on usage
	)
	if err != nil {
		return
	}

	if val, ok := args["--s3-key"].(string); ok {
		opt.awsAccessKey = val
	}
	if val, ok := args["--s3-secret"].(string); ok {
		opt.awsSecretKey = val
	}
	if val, ok := args["--force"].(bool); ok {
		opt.forceInit = val
	}
	if val, ok := args["--fs-root"].(string); ok {
		opt.fsRootFolder = val
	}
	if val, ok := args["--dest"].(string); ok {
		opt.restoreRoot = val
	}
	if val, ok := args["--s3-bucket"].(string); ok {
		opt.s3Bucket = val
	}
	if val, ok := args["--s3-region"].(string); ok {
		opt.s3Region = val
	}
	if val, ok := args["scan"].(bool); ok {
		opt.scanOnly = val
	}
	if val, ok := args["--storage"].(string); ok {
		opt.storageType = val
	}
	if val, ok := args["init"].(bool); ok {
		opt.storeInit = val
	}
	if val, ok := args["--pass"].(string); ok {
		opt.storeSecret = val
	}

	for _, p := range args["<path>"].([]string) {
		if strings.HasPrefix(p, ":") {
			opt.excludePaths = append(opt.excludePaths, file.CleanPath(p[1:]))
		} else {
			opt.includePaths = append(opt.includePaths, file.CleanPath(p))
		}
	}

	opt.configPath = file.CleanPath(args["--cfg"].(string))
	return
}

// Load the config file (or default path), then set any specific command line overrides provided.
func loadConfig(opt options) (original, cfg LocalConfig) {
	cfg, err := LoadConfigFile(opt.configPath)
	if err != nil {
		log.Printf("unable to load config file: %q\n", opt.configPath)
		cfg = NewConfig()
	}
	original = cfg

	// Override any store specific settings.
	if opt.s3Region != "" {
		cfg.Store.S3Region = opt.s3Region
	}
	if opt.s3Bucket != "" {
		cfg.Store.S3Bucket = opt.s3Bucket
	}
	if opt.awsAccessKey != "" {
		cfg.Store.AWSAccessKey = opt.awsAccessKey
	}
	if opt.awsSecretKey != "" {
		cfg.Store.AWSSecretKey = opt.awsSecretKey
	}

	return
}

func setupStore(cfg *LocalConfigStore, opt options) (bucket *store.Store, err error) {
	switch opt.storageType {
	case "s3":
		bucket, err = store.NewStoreS3(cfg.S3Config)
		if err != nil {
			log.Println("failed to connect to the remote store")
			return
		}
	case "fs":
		bucket, err = store.NewStoreFS(opt.fsRootFolder)
		if err != nil {
			log.Println("failed to setup file storage")
			return
		}
	default:
		err = errors.New("invalid storage type")
		return
	}

	// If we tried to initialize the store, check that a password was provided. Otherwise,
	// if a password was given, derive new crypto keys, or else just try the existing keys.
	if opt.storeInit {
		if opt.storeSecret == "" {
			err = errors.New("You must provide a password to initialize the store.")
			return
		}
		if !bucket.IsClean() && !opt.forceInit {
			err = errors.New("Store is already initialized. Cannot wipe store without forcing.")
			return
		}
		log.Println("initializing the store for first use")
		cfg.Keys, err = bucket.Wipe([]byte(opt.storeSecret))
		return
	}
	if opt.storeSecret != "" {
		log.Println("attempting to access the store with the password provided")
		cfg.Keys, err = bucket.Unlock([]byte(opt.storeSecret))
		return
	}
	log.Println("using the crypto keys from config to read the store")
	err = bucket.Open(cfg.Keys)
	return
}

func exitIfError(err error) {
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func main() {
	opts, err := parseFlags(os.Args[1:])
	exitIfError(err)

	log.Printf("starting. build: %s\n", buildTag)
	//fmt.Printf("%#v\n", opts)

	if opts.scanOnly {
		backup.WriteManifest("scan.json", scanFiles(LocalConfigPaths{}, opts))
		return
	}

	original, cfg := loadConfig(opts)
	bucket, err := setupStore(&cfg.Store, opts)
	exitIfError(err)

	// Save the config if it changed.
	if !original.Equal(cfg) {
		log.Printf("saving updated config file: %q\n", opts.configPath)
		exitIfError(cfg.WriteToFile(opts.configPath))
	}

	if opts.restoreRoot != "" {
		exitIfError(backup.RestoreToPath(bucket, opts.restoreRoot, opts.includePaths))
	} else {
		exitIfError(backup.ScanAndBackup(bucket, scanFiles(cfg.Paths, opts)))
	}

	fmt.Println("<exited normally>")
}
