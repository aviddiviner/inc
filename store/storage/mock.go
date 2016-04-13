package storage

import (
	"bytes"
	"errors"
	"io"
)

var MockErrNoSuchKey = errors.New("The specified key does not exist.")

type MockRequestFault func(key string) error
type MockReaderWrapper func(r io.Reader) io.Reader

// MockStorage is used for testing; data is stored in memory in a map.
type MockStorage struct {
	dataStore map[string][]byte
	requestFn []MockRequestFault
	readerFn  []MockReaderWrapper
}

func NewMockStorage() *MockStorage {
	return &MockStorage{dataStore: make(map[string][]byte)}
}

func (s *MockStorage) Exists() (bool, error) { return true, nil }
func (s *MockStorage) Create() error         { return nil }

func (s *MockStorage) Size(key string) (int, error) {
	if data, ok := s.dataStore[key]; ok {
		return len(data), nil
	}
	return 0, MockErrNoSuchKey
}

func (s *MockStorage) GetReader(key string) (r io.Reader, err error) {
	for _, fn := range s.requestFn {
		err = fn(key)
		if err != nil {
			return
		}
	}
	if data, ok := s.dataStore[key]; ok {
		r = bytes.NewReader(data)
		for _, fn := range s.readerFn {
			r = fn(r)
		}
		return
	}
	err = MockErrNoSuchKey
	return
}

func (s *MockStorage) PutReader(key string, r io.Reader) (int, error) {
	var buf bytes.Buffer
	if n, err := io.Copy(&buf, r); err != nil {
		return int(n), err
	}
	written := buf.Len()
	s.dataStore[key] = buf.Bytes()
	return written, nil
}

func (s *MockStorage) IsNotExist(err error) bool {
	if err == MockErrNoSuchKey {
		return true
	}
	return false
}

// Put allows easy writing to a key for tests.
func (s *MockStorage) PutString(key, value string) {
	s.dataStore[key] = []byte(value)
}

// InjectRequestFault adds a callback which gets called when a key is requested by GetReader.
// This func should return an error or nil.
func (s *MockStorage) InjectRequestFault(fn MockRequestFault) {
	s.requestFn = append(s.requestFn, fn)
}

// InjectReaderWrapper adds a callback which gets called when an io.Reader is returned by GetReader.
// This func should wrap that Reader or return it unchanged.
func (s *MockStorage) InjectReaderWrapper(fn MockReaderWrapper) {
	s.readerFn = append(s.readerFn, fn)
}

// ClearRequestFaults clears any request callbacks that have been added by InjectRequestFault.
func (s *MockStorage) ClearRequestFaults() {
	s.requestFn = nil
}

// ClearReaderWrappers clears any reader callbacks that have been added by InjectReaderWrapper.
func (s *MockStorage) ClearReaderWrappers() {
	s.readerFn = nil
}
