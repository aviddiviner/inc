inc
===

**inc** is an incremental backup tool. It compresses, encrypts, and stores changed files to your preferred medium (currently, local volume or Amazon S3).

## Usage

	# Initialise store
	inc init --pass foobar --s3-bucket myspecialbucket --s3-region us-west-2

	# Backup files
	inc backup ~/code ~/pics ~/movies

	# Restore files
	inc restore --dest /tmp/restore ~/code ~/pics

## Usability

This project is currently a work in progress. Having said that, it is quite usable. I made this tool to handle some personal backups and I still use it for those. As such, having working, bug-free code is quite important to me.

The file / store formats are all versioned, so as I make changes I'll continue to bump versions and maintain backward compatibility. Rest assured, any breaking changes will be documented, and there will always be a tagged release you can use to restore files for older versions.

If you want to start using this today, check out the latest tagged release. That's what I use.

## Installation

Make sure you have recent versions of [**git**](https://git-scm.com/), [**rake**](https://ruby.github.io/rake/), [**dep**](https://golang.github.io/dep/) and [**go**](https://golang.org/). Clone the repo, then `rake` to run tests, build and install.

## Design

#### Package layout

I've tried to keep the code clean and easy to reason about by practicing radial encapsulation and decoupling things as far as possible. Here is the package/folder structure:

	├── backup            -- backup/restore logic
	├── file              -- OS and file functions
	│   ├── archive       -- file bundling (tarball)
	│   └── fs            -- filesystem layers (OS, test)
	├── store             -- remote store interface
	│   ├── crypto        -- encryption
	│   ├── storage       -- storage layers (S3, etc.)
	│   └── zip           -- compression
	└── util              -- shared useful functions

#### Testing plans

Testing is an important part of the project, and the goal is to have thorough tests for most plausible backup scenarios; files that change as we read from them, disks and remote storage layers that sporadically throw errors, etc.

Right now, the filesystem and storage layers are both interchangeable, so I'm working on more extensive testing, particularly on a mock filesystem that behaves strangely.

#### Encryption

If you want details, take a look at [the source](store/crypto/crypto.go) since it's all quite readable. The basic points are:

- AES-256 in CBC (block chaining) mode, with an
- HMAC-SHA1 signature for the payload, and
- PBKDF2 for your encryption and auth keys, derived from secret+salt.

#### Store format

Your store (S3 bucket, or wherever) should look a little something like this:

	├── blob
	│   └── 1444cc251df313a5
	│       ├── 00
	│       ├── 01
	│       ├── 02
	│       └── ...
	├── manifest
	│   └── 1444cc251df313a5
	└── metadata

The `metadata` object is an unencrypted JSON file with the version number, cryptographic salt and other metadata (pointer to latest manifest, and so on).

Everything else in the store is encrypted. The `blob` folder contains bundled, compressed file data objects. The `manifest` folder contains manifests of the files in each backup set and their size, SHA1 of their contents, etc.

#### File scanning

Scanning the disk and indexing files is fast; comparable to a `find . -mtime 1`. New or potentially changed files are then read from disk to calculate a SHA1 hash of their contents. This should also be fairly quick and use minimal RAM during this step. In general though, there is still lots of optimization to do.

## Contribution

Right now, since I'm still changing large chunks of the project, I would advise against getting stuck in with any meaningful changes. But feel free to submit any bug fixes, tweaks or suggestions you have. I may not use everything, but thanks for showing interest!

## See Also

- [restic](https://restic.github.io/)
- [tarsnap](http://www.tarsnap.com/)

## License

[MIT](LICENSE)
