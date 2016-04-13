package backup

import (
	"encoding/json"
	"errors"
	"github.com/aviddiviner/inc/file"
	"github.com/aviddiviner/inc/store"
	"github.com/aviddiviner/inc/util"
	"time"
)

type Manifest struct {
	Version int              `json:"version"`
	LastSet string           `json:"lastSet"`
	Created time.Time        `json:"created"`
	Updated time.Time        `json:"updated"`
	Entries []*ManifestEntry `json:"entries"`

	//sync.RWMutex
	pathMap map[string]*ManifestEntry
}

type ManifestEntry struct {
	file.File
	Set   string
	Parts []ManifestEntryPart
}

type ManifestEntryPart struct {
	Key   string          `json:"key"`
	Range store.ByteRange `json:"range"`
}

// -----------------------------------------------------------------------------

func (m *Manifest) Has(f file.File) bool {
	//m.RLock()
	_, ok := m.pathMap[f.Path()]
	//m.RUnlock()
	return ok
}

func (m *Manifest) removeEntry(ptr *ManifestEntry) {
	oldPath := ptr.Path()
	// Replace the deleted entry arbitrarily with the last entry from the list.
	// We take over the deleted entry pointer and use that now for the last entry.
	lastIndex := len(m.Entries) - 1
	lastEntry := m.Entries[lastIndex] // pointer to the last entry
	*ptr = *lastEntry                 // replace the actual entry data
	m.pathMap[lastEntry.Path()] = ptr // use the new pointer to last entry
	m.Entries[lastIndex] = nil        // ensure we don't leak memory
	m.Entries = m.Entries[:lastIndex] // truncate the slice
	delete(m.pathMap, oldPath)        // delete the old pointer
}

func (m *Manifest) Remove(f file.File) bool {
	ptr, found := m.pathMap[f.Path()]
	if !found {
		return false
	}
	m.removeEntry(ptr)
	return true
}

// -----------------------------------------------------------------------------

func (m *Manifest) HasIdentical(their file.File) bool {
	our, ok := m.pathMap[their.Path()]
	if !ok {
		return false
	}
	if !our.IsDir() && our.Size != their.Size {
		return false
	}
	if our.Mode != their.Mode {
		return false
	}
	if !our.ModTime.Equal(their.ModTime) {
		if our.SHA1 == their.SHA1 {
			return true
		}
		return false
	}
	return true
}

func (before *Manifest) Compare(after []file.File) []file.File {
	var touched, changed []file.File

	for _, a := range after {
		if before.Has(a) {
			b := before.pathMap[a.Path()]
			if !a.IsDir() && a.Size != b.Size { // non-dir, size different
				changed = append(changed, a)
			} else if !a.ModTime.Equal(b.ModTime) { // timestamp touched
				touched = append(touched, a)
			}
		} else { // not found; must be new
			changed = append(changed, a)
		}
	}

	file.ChecksumFiles(touched, changed)

	for _, a := range touched {
		b := before.pathMap[a.Path()]
		if a.SHA1 != b.SHA1 { // can compare byte array contents directly
			changed = append(changed, a)
		}
	}

	return changed
}

// -----------------------------------------------------------------------------

// Used to avoid infinite recursion in UnmarshalJSON below.
type manifest Manifest

func (m *Manifest) buildPathMap() {
	m.pathMap = make(map[string]*ManifestEntry)
	for _, f := range m.Entries {
		m.pathMap[f.Path()] = f
	}
}

func (m *Manifest) MarshalJSON() ([]byte, error) {
	m.Version = 3
	return json.Marshal(*m)
}

func (m *Manifest) UnmarshalJSON(data []byte) (err error) {
	var t manifest
	err = json.Unmarshal(data, &t)
	if err != nil {
		return
	}
	*m = Manifest(t)
	m.buildPathMap()
	return
}

// -----------------------------------------------------------------------------

func (f *ManifestEntry) MarshalJSON() ([]byte, error) {
	jsonMap := map[string]interface{}{
		"root":  f.Root,
		"name":  f.Name,
		"mode":  f.Mode,
		"mtime": f.ModTime,
		"uid":   f.UID,
		"gid":   f.GID,
		"set":   f.Set,
	}

	if len(f.Parts) > 0 {
		jsonMap["parts"] = f.Parts
	}
	if !f.IsDir() {
		jsonMap["size"] = f.Size
	}
	if f.HasChecksum() {
		jsonMap["sha1"] = f.SHA1[:] // convert to slice = base64 encoded
	}

	return json.Marshal(jsonMap)
}

func (f *ManifestEntry) UnmarshalJSON(data []byte) (err error) {
	var keymap map[string]*json.RawMessage
	err = json.Unmarshal(data, &keymap)
	if err != nil {
		return
	}

	errors := map[string]error{
		"root":  json.Unmarshal(*keymap["root"], &f.Root),
		"name":  json.Unmarshal(*keymap["name"], &f.Name),
		"mode":  json.Unmarshal(*keymap["mode"], &f.Mode),
		"mtime": json.Unmarshal(*keymap["mtime"], &f.ModTime),
		"uid":   json.Unmarshal(*keymap["uid"], &f.UID),
		"gid":   json.Unmarshal(*keymap["gid"], &f.GID),
		"set":   json.Unmarshal(*keymap["set"], &f.Set),
	}
	if parts, ok := keymap["parts"]; ok {
		errors["parts"] = json.Unmarshal(*parts, &f.Parts)
	}
	if size, ok := keymap["size"]; ok {
		errors["size"] = json.Unmarshal(*size, &f.Size)
	}
	if sha1, ok := keymap["sha1"]; ok {
		var b []byte
		errors["sha1"] = json.Unmarshal(*sha1, &b)
		copy(f.SHA1[:], b)
	}
	for _, v := range errors {
		if v != nil {
			return v
		}
	}

	return
}

// -----------------------------------------------------------------------------

var emptyRange = [2]int{}

func (p *ManifestEntryPart) MarshalJSON() ([]byte, error) {
	jsonMap := map[string]interface{}{
		"key": p.Key,
	}

	if p.Range != emptyRange {
		jsonMap["range"] = p.Range
	}

	return json.Marshal(jsonMap)
}

// -----------------------------------------------------------------------------

func NewManifest(files []file.File) Manifest {
	m := Manifest{pathMap: make(map[string]*ManifestEntry)}
	now := m.Update(files)
	m.Created = now.Truncate(time.Second)
	return m
}

func (m *Manifest) JSON() ([]byte, error) {
	return json.Marshal(m)
}

var ErrMalformedConfig = errors.New("malformed config data")
var ErrBadVersion = errors.New("bad version")

func ReadManifestData(data []byte) (m Manifest, err error) {
	if ver, ok := util.ParseVersionJSON(data); ok {
		switch ver {
		case 1, 2:
			err = unmarshalV2Manifest(data, &m)
		case 3:
			err = json.Unmarshal(data, &m)
		default:
			err = ErrBadVersion
		}
		return
	}
	err = ErrMalformedConfig
	return
}
