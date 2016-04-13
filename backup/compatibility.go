package backup

import (
	"encoding/json"
	"strings"
)

func unmarshalV2Manifest(data []byte, m *Manifest) error {
	var keymap map[string]*json.RawMessage
	if err := json.Unmarshal(data, &keymap); err != nil {
		return err
	}

	errors := map[string]error{
		"version": json.Unmarshal(*keymap["version"], &m.Version),
		"key":     json.Unmarshal(*keymap["key"], &m.LastSet),
		"created": json.Unmarshal(*keymap["created"], &m.Created),
		"entries": unmarshalV2ManifestEntries(*keymap["entries"], &m.Entries),
	}
	for _, v := range errors {
		if v != nil {
			return v
		}
	}

	m.buildPathMap()
	return nil
}

func unmarshalV2ManifestEntries(raw json.RawMessage, m *[]*ManifestEntry) error {
	var entries []*json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return err
	}

	*m = make([]*ManifestEntry, len(entries))
	for i := range entries {
		var entry ManifestEntry
		if err := unmarshalV2ManifestEntry(*entries[i], &entry); err != nil {
			return err
		}
		(*m)[i] = &entry
	}

	return nil
}

func unmarshalV2ManifestEntry(raw json.RawMessage, f *ManifestEntry) error {
	var keymap map[string]*json.RawMessage
	if err := json.Unmarshal(raw, &keymap); err != nil {
		return err
	}

	errors := map[string]error{
		"root":  json.Unmarshal(*keymap["root"], &f.Root),
		"name":  json.Unmarshal(*keymap["name"], &f.Name),
		"mode":  json.Unmarshal(*keymap["mode"], &f.Mode),
		"mtime": json.Unmarshal(*keymap["mtime"], &f.ModTime),
		"uid":   json.Unmarshal(*keymap["uid"], &f.UID),
		"gid":   json.Unmarshal(*keymap["gid"], &f.GID),
	}
	if size, ok := keymap["size"]; ok {
		errors["size"] = json.Unmarshal(*size, &f.Size)
	}
	for _, v := range errors {
		if v != nil {
			return v
		}
	}

	if key, ok := keymap["_"]; ok {
		// The possibility exited for a V2 manifest to have multiple keys, but in
		// practice this never happened, so just unmarshall the single string key.
		var obj string
		if err := json.Unmarshal(*key, &obj); err != nil {
			return err
		}
		setKey := strings.Split(obj, "/")
		f.Set = setKey[0]
		f.Parts = []ManifestEntryPart{ManifestEntryPart{Key: setKey[1]}}
	}

	if sha1, ok := keymap["sha1"]; ok {
		var b []byte
		if err := json.Unmarshal(*sha1, &b); err != nil {
			return err
		}
		copy(f.SHA1[:], b)
	}

	return nil
}
