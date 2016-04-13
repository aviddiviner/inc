package store

import (
	"github.com/aviddiviner/inc/store/storage"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestCustomMetadataWorks(t *testing.T) {
	salt := []byte("salty")
	layer := storage.NewMockStorage()
	store := NewStore(layer, "test")

	assertMetadataEquals := func(expected storeMetadata) {
		got, _ := store.getStoreMetadata()
		assert.Equal(t, expected, got)
	}
	assertRawMetadataEquals := func(expected string) {
		r, _ := layer.GetReader(c_METADATA_KEY)
		actual, _ := ioutil.ReadAll(r)
		assert.Equal(t, expected, string(actual))
	}

	simple := storeMetadata{Version: 1, StoreFormat: 1, Salt: salt}
	store.putStoreMetadata(simple)

	assertMetadataEquals(simple)
	assertRawMetadataEquals(`{"version":1,"storeFormat":1,"salt":"c2FsdHk=","userData":null}`)

	userData := map[string]interface{}{
		"123": 123,
		"foo": "bar",
	}
	full := storeMetadata{Version: 1, StoreFormat: 1, Salt: salt, UserData: userData}
	store.putStoreMetadata(full)

	assertMetadataEquals(full)
	assertRawMetadataEquals(`{"version":1,"storeFormat":1,"salt":"c2FsdHk=","userData":{"123":123,"foo":"bar"}}`)
}

func TestCustomMetadataGetAndSet(t *testing.T) {
	layer := storage.NewMockStorage()
	store := NewStore(layer, "test")

	layer.PutString(c_METADATA_KEY, `{"version":1}`)
	_, err := store.GetMetadata("foo")
	assert.EqualError(t, err, ErrMissingMetadata.Error())

	store.PutMetadata("foo", "bar")
	foo, err := store.GetMetadata("foo")
	assert.NoError(t, err)
	assert.Equal(t, "bar", foo.(string))

	layer = storage.NewMockStorage()
	store = NewStore(layer, "test")

	layer.PutString(c_METADATA_KEY, `{"version":1,"userData":{"foo":"bar"}}`)
	foo, err = store.GetMetadata("foo")
	assert.NoError(t, err)
	assert.Equal(t, "bar", foo.(string))
}
