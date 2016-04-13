package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/aviddiviner/inc/util"
	"io/ioutil"
)

type storeMetadata struct {
	Version     int    `json:"version"`
	StoreFormat int    `json:"storeFormat"`
	Salt        []byte `json:"salt"` // base64 encoded

	UserData map[string]interface{} `json:"userData"`
}

// Error when retrieving store metadata that's unreadable.
var ErrMalformedMetadata = errors.New("malformed metadata")

// Error when retrieving store metadata that has an unknown version.
var ErrBadVersion = errors.New("bad version")

// Error when attempting to read a custom metadata field which hasn't been set.
var ErrMissingMetadata = errors.New("user metadata not set")

// -----------------------------------------------------------------------------

const c_METADATA_KEY = "metadata"

func newMetadata(salt []byte) storeMetadata {
	return storeMetadata{Version: 1, StoreFormat: 1, Salt: salt}
}

func (s *Store) getStoreMetadata() (md storeMetadata, err error) {
	if s.meta != nil {
		md = *s.meta
		return
	}
	reader, err := s.layer.GetReader(c_METADATA_KEY)
	if err != nil {
		return
	}
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return
	}
	if ver, ok := util.ParseVersionJSON(data); ok {
		switch ver {
		case 1:
			if f := json.Unmarshal(data, &md); f == nil {
				s.meta = &md
				return
			}
		default:
			err = ErrBadVersion
			return
		}
	}
	err = ErrMalformedMetadata
	return
}

func (s *Store) putStoreMetadata(md storeMetadata) (err error) {
	data, err := json.Marshal(md)
	if err != nil {
		return
	}
	_, err = s.layer.PutReader(c_METADATA_KEY, bytes.NewReader(data))
	if err != nil {
		return
	}
	s.meta = &md
	return
}

func (s *Store) getUserMetadata(key string) (interface{}, error) {
	md, err := s.getStoreMetadata()
	if err != nil {
		return nil, err
	}
	if data, ok := md.UserData[key]; ok {
		return data, nil
	}
	return nil, ErrMissingMetadata
}

func (s *Store) putUserMetadata(key string, data interface{}) error {
	md, err := s.getStoreMetadata()
	if err != nil {
		return err
	}
	if md.UserData == nil {
		md.UserData = make(map[string]interface{})
	}
	md.UserData[key] = data
	return s.putStoreMetadata(md)
}
