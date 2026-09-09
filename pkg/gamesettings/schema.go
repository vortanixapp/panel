package gamesettings

import "regexp"

type Format string

const (
	FormatKVSpace    Format = "kv_space"
	FormatSourceCfg  Format = "source_cfg"
	FormatProperties Format = "properties"
	FormatEnv        Format = "env"
	FormatINI        Format = "ini"
	FormatUEOption   Format = "ue_option"
	FormatJSON       Format = "json"
	FormatXMLProps   Format = "xml_props"
	FormatXMLTags    Format = "xml_tags"
	FormatYAMLFlat   Format = "yaml_flat"
	FormatArgs       Format = "args"
	FormatUEQuery    Format = "ue_query"
)

const StartupPath = "@startup"

type Kind string

const (
	KindString Kind = "string"
	KindText   Kind = "text"
	KindInt    Kind = "int"
	KindFloat  Kind = "float"
	KindBool   Kind = "bool"
	KindEnum   Kind = "enum"
)

type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type Field struct {
	Key string `json:"key"`

	File string `json:"file,omitempty"`

	Prop string `json:"-"`

	Label   string    `json:"label"`
	Hint    string    `json:"hint,omitempty"`
	Section SectionID `json:"section"`
	Kind    Kind      `json:"kind"`
	Options []Option  `json:"options,omitempty"`
	Default string    `json:"default,omitempty"`

	Min *float64 `json:"min,omitempty"`
	Max *float64 `json:"max,omitempty"`

	MaxLen  int            `json:"max_len,omitempty"`
	Pattern *regexp.Regexp `json:"-"`

	True  string `json:"-"`
	False string `json:"-"`

	AppliesLive bool `json:"applies_live,omitempty"`

	ReadOnly bool `json:"read_only,omitempty"`

	Secret bool `json:"secret,omitempty"`

	Slots bool `json:"slots,omitempty"`

	Clearable bool `json:"clearable,omitempty"`
}

type ConfigFile struct {
	ID string `json:"id"`

	Path string `json:"path"`

	Format Format `json:"format"`
	Title  string `json:"title"`

	Template string `json:"-"`

	MaxBytes int `json:"max_bytes,omitempty"`

	RawOnly bool `json:"raw_only,omitempty"`
}

type Profile struct {
	Key string `json:"key"`

	SharedBy []string `json:"shared_by,omitempty"`

	Files  []ConfigFile `json:"files"`
	Fields []Field      `json:"fields"`

	Note string `json:"note,omitempty"`
}

const DefaultMaxBytes = 256 << 10

func (f ConfigFile) MaxFileBytes() int {
	if f.MaxBytes > 0 {
		return f.MaxBytes
	}
	return DefaultMaxBytes
}

func (f ConfigFile) IsStartup() bool {
	return f.Path == StartupPath
}

func (p Profile) FileByID(id string) (ConfigFile, bool) {
	if id == "" {
		if len(p.Files) == 0 {
			return ConfigFile{}, false
		}
		return p.Files[0], true
	}
	for _, f := range p.Files {
		if f.ID == id {
			return f, true
		}
	}
	return ConfigFile{}, false
}

func (p Profile) FieldByKey(key string) (Field, bool) {
	for _, f := range p.Fields {
		if f.Key == key {
			return f, true
		}
	}
	return Field{}, false
}

func (p Profile) Games() []string {
	return append([]string{p.Key}, p.SharedBy...)
}

func (f Field) BoolLiteral(on bool) string {
	if on {
		if f.True != "" {
			return f.True
		}
		return "true"
	}
	if f.False != "" {
		return f.False
	}
	return "false"
}

func floatPtr(v float64) *float64 { return &v }

func Range(min, max float64) (*float64, *float64) {
	return floatPtr(min), floatPtr(max)
}

func AtLeast(min float64) *float64 { return floatPtr(min) }

func AtMost(max float64) *float64 { return floatPtr(max) }
