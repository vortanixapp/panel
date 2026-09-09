package storage

import "strings"

type EggCDN struct {
	Prefix string
}

func NewEggCDN(prefix string) *EggCDN {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil
	}
	return &EggCDN{Prefix: strings.TrimRight(prefix, "/")}
}

func (c *EggCDN) URL(path *string) any {
	if c == nil || path == nil || *path == "" {
		if path != nil {
			return *path
		}
		return nil
	}
	p := strings.TrimSpace(*path)
	if p == "" {
		return nil
	}
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}
	return c.Prefix + "/" + strings.TrimLeft(p, "/")
}
