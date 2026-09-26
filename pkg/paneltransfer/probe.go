package paneltransfer

type Probe struct {
	Project       string   `json:"project"`
	Mode          string   `json:"mode"`
	Version       string   `json:"version"`
	SourceAddress string   `json:"source_address,omitempty"`
	DBBytes       int64    `json:"db_bytes"`
	UploadsBytes  int64    `json:"uploads_bytes"`
	FreeBytes     int64    `json:"free_bytes"`
	EnvKeys       []string `json:"env_keys"`
}
