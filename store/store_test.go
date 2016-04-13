package store

import (
	"bytes"
	"github.com/aviddiviner/inc/store/storage"
	"github.com/stretchr/testify/assert"
	"testing"
	"testing/iotest"
)

var (
	testMetadata   = `{"version":1,"storeFormat":1,"salt":"5+ZOMGkPADM="}`
	testBadBase64  = `{"version":1,"storeFormat":1,"salt":"5+ZOMGkPADM"}`
	testBadSyntax  = `{"version":1,"storeFormat":1,"salt":"5+ZOMGkPADM=}`
	testBadVersion = `{"version":7,"storeFormat":1,"salt":"5+ZOMGkPADM="}`
	testSecret     = []byte("mysupersecretpassword")
	testData       = []byte("A quick brown fox jumps over the lazy dog.")
	testCryptoKeys = Keys{
		EncKey:  []uint8{0xd9, 0xe1, 0x8, 0xdf, 0xe2, 0xb6, 0xd8, 0xef, 0x70, 0x3d, 0x1b, 0xd, 0x37, 0xa, 0x8b, 0x3e, 0xa2, 0x4, 0xd2, 0x52, 0xae, 0x6b, 0xca, 0x6e, 0x68, 0x13, 0x97, 0x91, 0x2f, 0x6d, 0x53, 0x1e},
		AuthKey: []uint8{0xc2, 0xc1, 0xb0, 0x9f, 0xf3, 0x5, 0x3d, 0x78, 0x1e, 0xdd, 0xd1, 0x90, 0xfc, 0x93, 0xce, 0x86, 0xca, 0x7a, 0xfc, 0x40, 0xfd, 0xb5, 0x94, 0xac, 0x46, 0xc3, 0x1c, 0x2c, 0xf3, 0x99, 0x2d, 0xf, 0xf3, 0x28, 0x4, 0x30, 0x9e, 0xad, 0xab, 0xee, 0xf6, 0xcf, 0x1e, 0xab, 0x43, 0x6d, 0x2, 0x86, 0xb, 0xcb, 0x8c, 0xac, 0xb0, 0xe5, 0x79, 0xbd, 0x18, 0x2d, 0x4c, 0x2d, 0x90, 0xcc, 0x4f, 0x36},
	}
)

func useStoreRW(t *testing.T, store *Store) {
	_, err := store.Put("test", testData)
	assert.NoError(t, err)

	got, err := store.Get("test")
	assert.NoError(t, err)
	assert.Equal(t, testData, got)
}

func TestWipeAndUseStore(t *testing.T) {
	store := NewStore(storage.NewMockStorage(), "test")
	keys, err := store.Wipe(testSecret)
	assert.NotEmpty(t, keys)
	assert.NoError(t, err)

	useStoreRW(t, store)
}

func TestUnlockAndUseStore(t *testing.T) {
	layer := storage.NewMockStorage()
	store := NewStore(layer, "test")

	// Unlock without metadata
	keys, err := store.Unlock(testSecret)
	assert.Equal(t, Keys{}, keys)
	assert.Error(t, err)

	// Add some metadata, then unlock again
	layer.PutString(c_METADATA_KEY, testMetadata)
	keys, err = store.Unlock(testSecret)
	assert.NotEqual(t, Keys{}, keys)
	assert.NoError(t, err)

	useStoreRW(t, store)
}

func TestOpenAndUseStore(t *testing.T) {
	store := NewStore(storage.NewMockStorage(), "test")
	err := store.Open(testCryptoKeys)
	assert.NoError(t, err)

	useStoreRW(t, store)
}

func TestUnlockWithBadMetadata(t *testing.T) {
	layer := storage.NewMockStorage()
	store := NewStore(layer, "test")

	layer.PutString(c_METADATA_KEY, testBadBase64)
	_, err := store.Unlock(testSecret)
	assert.EqualError(t, err, ErrMalformedMetadata.Error())

	layer.PutString(c_METADATA_KEY, testBadSyntax)
	_, err = store.Unlock(testSecret)
	assert.EqualError(t, err, ErrMalformedMetadata.Error())

	layer.PutString(c_METADATA_KEY, testBadVersion)
	_, err = store.Unlock(testSecret)
	assert.EqualError(t, err, ErrBadVersion.Error())
}

func TestUseBeforeOpen(t *testing.T) {
	store := NewStore(storage.NewMockStorage(), "test")

	_, err := store.Put("test", testData)
	assert.EqualError(t, err, ErrStoreNotConnected.Error())

	_, err = store.Get("test")
	assert.EqualError(t, err, ErrStoreNotConnected.Error())

	_, err = store.Pack("test")
	assert.EqualError(t, err, ErrStoreNotConnected.Error())
}

func TestGetMissingObject(t *testing.T) {
	store := NewStore(storage.NewMockStorage(), "test")
	store.Open(testCryptoKeys)

	_, err := store.Get("test")
	assert.Error(t, err)
	assert.True(t, store.IsNotExist(err))
}

func TestPacker(t *testing.T) {
	store := NewStore(storage.NewMockStorage(), "test")
	store.Open(testCryptoKeys)

	packer, err := store.Pack("test")
	assert.NoError(t, err)

	_, err = packer.PutReader(iotest.OneByteReader(bytes.NewReader(testData)))
	assert.NoError(t, err)

	err = packer.Close()
	assert.NoError(t, err)

	err = packer.Close()
	assert.NoError(t, err)

	got, err := store.Get("test")
	assert.NoError(t, err)
	assert.Equal(t, testData, got)
}

func TestPackerErrors(t *testing.T) {
	store := NewStore(storage.NewMockStorage(), "test")
	store.Open(testCryptoKeys)

	packer, _ := store.Pack("test")

	_, err := packer.PutReader(iotest.TimeoutReader(bytes.NewReader(testData)))
	assert.EqualError(t, err, "timeout")

	_, err = packer.PutReader(bytes.NewReader(testData))
	assert.EqualError(t, err, "io: read/write on closed pipe")

	assert.EqualError(t, packer.Close(), "timeout")
	assert.EqualError(t, packer.Close(), "timeout")

	_, err = store.Get("test")
	assert.Error(t, err)
	assert.True(t, store.IsNotExist(err))
}
