package paneltransfer

import (
	"encoding/json"
	"errors"
	"io"
)

const Schema = 1

type Manifest struct {
	Schema          int      `json:"schema"`
	PanelVersion    string   `json:"panel_version"`
	DBSchemaVersion string   `json:"db_schema_version,omitempty"`
	CreatedAt       string   `json:"created_at"`
	SourceAddress   string   `json:"source_address,omitempty"`
	Mode            string   `json:"mode,omitempty"`
	Contents        []string `json:"contents"`
	DBBytes         int64    `json:"db_bytes"`
	UploadsBytes    int64    `json:"uploads_bytes"`
}

func (m *Manifest) Encode(w io.Writer) error {
	if m.Schema == 0 {
		m.Schema = Schema
	}
	return json.NewEncoder(w).Encode(m)
}

func DecodeManifest(r io.Reader) (*Manifest, error) {
	var m Manifest
	if err := json.NewDecoder(io.LimitReader(r, 1<<20)).Decode(&m); err != nil {
		return nil, errors.New("описание архива не читается")
	}
	if m.Schema == 0 || m.Schema > Schema {
		return nil, errors.New("архив создан более новой панелью")
	}
	return &m, nil
}
