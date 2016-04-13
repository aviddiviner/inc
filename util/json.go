package util

import (
	"encoding/json"
)

// Parses any JSON map of (string) keys to whatever, and returns the value of
// the "version" key as an integer. Otherwise, returns (-1, false).
func ParseVersionJSON(raw []byte) (int, bool) {
	var f struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(raw, &f); err == nil {
		return f.Version, true
	}
	return -1, false
}
