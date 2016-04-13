package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/aviddiviner/inc/file"
	"github.com/aviddiviner/inc/store"
	"github.com/aviddiviner/inc/util"
)

type options struct {
	storeInit    bool
	forceInit    bool
	storeSecret  string
	storageType  string
	awsAccessKey string
	awsSecretKey string
	s3Region     string
	s3Bucket     string
	fsRootFolder string
	configPath   string
	includePaths []string
	excludePaths []string
	restoreRoot  string

	scanOnly bool
}

var ErrMalformedConfig = errors.New("malformed config data")
var ErrBadVersion = errors.New("bad version")

type LocalConfig struct {
	Version int              `json:"version"`
	Store   LocalConfigStore `json:"store"`
	Paths   LocalConfigPaths `json:"paths"`
}

type LocalConfigStore struct {
	store.S3Config
	store.Keys
}

type LocalConfigPaths struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

func NewConfig() LocalConfig {
	return LocalConfig{Version: 2}
}

func LoadConfigFile(filePath string) (cfg LocalConfig, err error) {
	data, err := file.ReadFile(filePath)
	if err != nil {
		return
	}
	if ver, ok := util.ParseVersionJSON(data); ok {
		switch ver {
		case 1:
			cfg = NewConfig()
			err = json.Unmarshal(data, &cfg.Store)
		case 2:
			err = json.Unmarshal(data, &cfg)
		default:
			err = ErrBadVersion
		}
		return
	}
	err = ErrMalformedConfig
	return
}

func (cfg *LocalConfig) WriteToFile(path string) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return file.WriteFile(path, data)
}

func (cfg *LocalConfig) Equal(other LocalConfig) bool {
	if cfg == &other {
		return true
	}
	myJson, err := json.Marshal(cfg)
	if err != nil {
		return false
	}
	otherJson, err := json.Marshal(other)
	if err != nil {
		return false
	}
	return bytes.Equal(myJson, otherJson)
}
