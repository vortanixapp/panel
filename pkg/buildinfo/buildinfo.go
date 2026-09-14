package buildinfo

import (
	"os"
	"strings"
)

var Version = "dev"

func Current() string {
	if v := Normalize(Version); v != "" && v != "dev" {
		return v
	}
	if v := Normalize(os.Getenv("VORTANIX_VERSION")); v != "" {
		return v
	}
	return "dev"
}

func Normalize(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}
