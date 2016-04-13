package store

import (
	"bytes"
	"errors"
	"github.com/aviddiviner/inc/store/crypto"
	"github.com/aviddiviner/inc/store/storage"
	"github.com/aviddiviner/inc/store/zip"
	"github.com/aviddiviner/inc/util"
	"io"
	"io/ioutil"
	"log"
)

// Error when attempting to unlock a store which hasn't been initialized.
var ErrStoreNotInitialized = errors.New("remote store not initialized")

// Error when attempting to read/write from the store before it's been opened, unlocked or wiped.
var ErrStoreNotConnected = errors.New("store not ready for reading/writing")

// Error when attempting to read/write to a forbidden key name (e.g. "metadata" which is used internally).
var ErrForbiddenKey = errors.New("read/write to key name is forbidden")

// ByteRange is an offset pair (zero-indexed, inclusive) used when requesting partial contents from the storage layer.
// See http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35 for more details.
type ByteRange [2]int

// StorageLayer is the interface used by the store for underlying storage.
type StorageLayer interface {
	// Exists returns true if the container (S3 bucket, folder, etc.) exists and is usable.
	Exists() (bool, error)
	// Create ensures the container exists and is usable.
	Create() error

	// Size returns the encrypted content length of an object identified by key.
	Size(key string) (int, error)
	// GetReader returns the contents of an object identified by key.
	GetReader(key string) (io.Reader, error)
	// PutReader reads in from r and stores the result to an object.
	PutReader(key string, r io.Reader) (int, error)
	// IsNotExist returns true if the error indicates an object does not exist.
	IsNotExist(err error) bool

	// GetReaderParts(key string, ranges []ByteRange) (io.Reader, error)
	// Delete(key string) error
}

// S3Config has the configuration options for creating a new S3 connection.
type S3Config struct {
	S3Region     string `json:"s3Region"`
	S3Bucket     string `json:"s3Bucket"`
	AWSAccessKey string `json:"awsAccessKey"`
	AWSSecretKey string `json:"awsSecretKey"`
}

// Keys is a struct for handling the encryption and authentication keys to store data.
type Keys struct {
	EncKey  []byte `json:"encKey"`  // base64 encoded at rest
	AuthKey []byte `json:"authKey"` // base64 encoded at rest
}

// -----------------------------------------------------------------------------

// Store handles compressing, encrypting and uploading blobs to some storage medium.
type Store struct {
	layer StorageLayer
	id    string
	meta  *storeMetadata
	enc   crypto.Crypter
}

// NewStore returns a store using some storage layer.
func NewStore(layer StorageLayer, id string) *Store {
	return &Store{layer: layer, id: id}
}

// NewStoreS3 returns a store using S3 as its storage layer.
func NewStoreS3(cfg S3Config) (s *Store, err error) {
	s3, err := storage.NewS3Connection(cfg.S3Region, cfg.S3Bucket, cfg.AWSAccessKey, cfg.AWSSecretKey)
	if err != nil {
		return
	}
	id := "s3/" + cfg.AWSAccessKey + "/" + cfg.S3Region + "/" + cfg.S3Bucket
	return NewStore(s3, id), nil
}

// NewStoreFS returns a store using the filesystem as its storage layer.
func NewStoreFS(root string) (s *Store, err error) {
	fs := storage.NewFileStorage(root)
	return NewStore(fs, "fs"+root), nil
}

// ID returns the unique identifier of the store.
func (s *Store) ID() string {
	return s.id
}

// IsEmpty returns true if the store is clean (has no metadata) and is thus safe to wipe.
func (s *Store) IsClean() bool {
	if _, err := s.getStoreMetadata(); s.layer.IsNotExist(err) {
		return true
	}
	return false
}

// Wipe initializes the store by (re)populating it with metadata/salt. All existing data will be lost!
// This will be followed by a call to Open so the store will be ready for use.
func (s *Store) Wipe(secret []byte) (keys Keys, err error) {
	salt, err := crypto.Salt()
	if err != nil {
		return
	}
	ok, err := s.layer.Exists()
	if err != nil {
		return
	}
	if !ok {
		err = s.layer.Create()
		if err != nil {
			return
		}
	}
	err = s.putStoreMetadata(newMetadata(salt))
	if err != nil {
		return
	}
	return s.Unlock(secret)
}

// Unlock reads the store metadata and returns encryption keys derived from our secret+salt.
// This will be followed by a call to Open so the store will be ready for use.
func (s *Store) Unlock(secret []byte) (keys Keys, err error) {
	md, err := s.getStoreMetadata()
	if err != nil {
		if s.layer.IsNotExist(err) {
			err = ErrStoreNotInitialized
		}
		return
	}
	keys.EncKey, keys.AuthKey = crypto.DeriveKeys(secret, md.Salt)
	err = s.Open(keys)
	return
}

// Open sets the encryption keys to use, tests that we can read from the store.
func (s *Store) Open(keys Keys) (err error) {
	ok, err := s.layer.Exists()
	if err != nil {
		return
	}
	if !ok {
		return ErrStoreNotInitialized
	}
	enc, err := crypto.NewCrypter(keys.EncKey, keys.AuthKey)
	if err != nil {
		return
	}
	// TODO: Test we can decrypt something from the store. Maybe not a known plaintext, as that might compromise our crypto?
	// We could derive other keys from the secret for this? Maybe not an issue:
	// http://crypto.stackexchange.com/questions/3952/is-it-possible-to-obtain-aes-128-key-from-a-known-ciphertext-plaintext-pair
	// http://crypto.stackexchange.com/questions/1512/why-is-aes-resistant-to-known-plaintext-attacks
	s.enc = enc
	return
}

func (s *Store) isConnected() bool { return s.enc != nil }

func isForbiddenKey(key string) bool { return key == c_METADATA_KEY }

// -----------------------------------------------------------------------------

// PutMetadata writes some arbitrary (unencrypted) field to the store metadata. Ideally, this should be some small
// configuration data or similar.
func (s *Store) PutMetadata(key string, data interface{}) error {
	return s.putUserMetadata(key, data)
}

// GetMetadata returns the value of a custom metadata field.
func (s *Store) GetMetadata(key string) (interface{}, error) {
	return s.getUserMetadata(key)
}

// -----------------------------------------------------------------------------

// Put some blob as an object in the store. Returns the bytes written. Will overwrite existing keys.
func (s *Store) Put(key string, data []byte) (written int, err error) {
	return s.PutReader(key, bytes.NewReader(data))
}

// PutReader reads data into an object in the store. Returns the bytes written. Will overwrite existing keys.
func (s *Store) PutReader(key string, r io.Reader) (written int, err error) {
	if isForbiddenKey(key) {
		err = ErrForbiddenKey
		return
	}
	if !s.isConnected() {
		err = ErrStoreNotConnected
		return
	}
	data, err := zip.CompressReader(r)
	if err != nil {
		return
	}
	ciphertext, err := s.enc.EncryptReader(data)
	if err != nil {
		return
	}
	written, err = s.layer.PutReader(key, ciphertext)
	log.Printf("store: put.reader: %s (%s)\n", key, util.ByteCount(written))
	return
}

// Get returns the data stored by the given key.
func (s *Store) Get(key string) (data []byte, err error) {
	r, err := s.GetReader(key)
	if err != nil {
		return
	}
	return ioutil.ReadAll(r)
}

// GetReader returns a reader for the data stored by the given key.
func (s *Store) GetReader(key string) (data io.Reader, err error) {
	if isForbiddenKey(key) {
		err = ErrForbiddenKey
		return
	}
	if !s.isConnected() {
		err = ErrStoreNotConnected
		return
	}
	log.Printf("store: get.reader: %s\n", key)
	ciphertext, err := s.layer.GetReader(key)
	if err != nil {
		return
	}
	plaintext, err := s.enc.DecryptReader(ciphertext)
	if err != nil {
		return
	}
	return zip.DecompressReader(plaintext)
}

// IsNotExist returns a boolean indicating whether the error is because the object does not exist.
func (s *Store) IsNotExist(err error) bool {
	return s.layer.IsNotExist(err)
}

// -----------------------------------------------------------------------------

// Pack multiple blobs as a single object in the store. Will overwrite existing keys.
func (s *Store) Pack(key string) (*Packer, error) {
	if isForbiddenKey(key) {
		return nil, ErrForbiddenKey
	}
	if !s.isConnected() {
		return nil, ErrStoreNotConnected
	}
	r, w := io.Pipe()
	p := &Packer{s: s, key: key, r: r, w: w, err: make(chan error)}
	go func() {
		_, err := s.layer.PutReader(key, r)
		r.CloseWithError(err)
		p.err <- err
	}()
	return p, nil
}

// A Packer is used for storing multiple blobs as a single object.
type Packer struct {
	s   *Store
	key string
	r   *io.PipeReader
	w   *io.PipeWriter
	err chan error

	closed   bool
	closeErr error
	// sync.Mutex
}

// PutReader reads data into the object. Returns the bytes written.
func (p *Packer) PutReader(r io.Reader) (written int, err error) {
	data, err := zip.CompressReader(r)
	if err != nil {
		p.closeWriter(err)
		return
	}
	ciphertext, err := p.s.enc.EncryptReader(data)
	if err != nil {
		p.closeWriter(err)
		return
	}
	n, err := io.Copy(p.w, ciphertext)
	if err != nil {
		// We could retry reads/writes here, but we rather just close the write end
		// of the pipe with the error, so we know we've failed with this packer.
		p.closeWriter(err)
		log.Printf("store: put.packer: error: %s\n", err)
		return
	}
	written = int(n)
	// log.Printf("store: put.packer: %s (%s)\n", p.key, util.ByteCount(written)) // TODO: Debug logging
	return
}

// Close finishes writing, returning any last errors.
func (p *Packer) Close() error {
	p.closeWriter(nil)
	return p.closeErr
}

func (p *Packer) closeWriter(err error) {
	if !p.closed {
		p.closed = true
		p.w.CloseWithError(err) // close the write end of the pipe
		p.closeErr = <-p.err    // block on hearing back from the go func
	}
}
