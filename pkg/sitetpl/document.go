package sitetpl

import (
	"bytes"
	"encoding/json"
)

type L map[string]string

type Item struct {
	ID       string   `json:"id"`
	Ref      string   `json:"ref,omitempty"`
	Kind     string   `json:"kind,omitempty"`
	Label    L        `json:"label,omitempty"`
	Icon     string   `json:"icon,omitempty"`
	URL      string   `json:"url,omitempty"`
	NewTab   bool     `json:"new_tab,omitempty"`
	Badge    L        `json:"badge,omitempty"`
	Audience string   `json:"audience,omitempty"`
	Roles    []string `json:"roles,omitempty"`
	Hidden   bool     `json:"hidden,omitempty"`
	Items    []Item   `json:"items,omitempty"`
}

type Menu struct {
	Items []Item `json:"items"`
}

type Block struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Hidden bool           `json:"hidden,omitempty"`
	Props  map[string]any `json:"props,omitempty"`
}

type Page struct {
	Blocks *[]Block `json:"blocks,omitempty"`
	Top    []Block  `json:"top,omitempty"`
	Bottom []Block  `json:"bottom,omitempty"`
	Hidden []string `json:"hidden,omitempty"`
}

type CustomPage struct {
	ID          string  `json:"id"`
	Slug        string  `json:"slug"`
	Title       L       `json:"title,omitempty"`
	Description L       `json:"description,omitempty"`
	Layout      string  `json:"layout,omitempty"`
	Audience    string  `json:"audience,omitempty"`
	Hidden      bool    `json:"hidden,omitempty"`
	Blocks      []Block `json:"blocks"`
}

type Document struct {
	Menus       map[string]Menu              `json:"menus,omitempty"`
	Pages       map[string]Page              `json:"pages,omitempty"`
	CustomPages []CustomPage                 `json:"custom_pages,omitempty"`
	Texts       map[string]map[string]string `json:"texts,omitempty"`
}

const AdminMenu = "admin_sidebar"

func Parse(raw []byte) (Document, error) {
	var doc Document
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return doc, nil
	}
	err := json.Unmarshal(raw, &doc)
	return doc, err
}

func (d Document) WithoutTexts() Document {
	d.Texts = nil
	return d
}

func (d Document) Public() Document {
	d.Texts = nil
	if _, ok := d.Menus[AdminMenu]; ok {
		menus := make(map[string]Menu, len(d.Menus))
		for key, menu := range d.Menus {
			if key != AdminMenu {
				menus[key] = menu
			}
		}
		d.Menus = menus
	}
	return d
}

func (d *Document) EachBlock(fn func(b *Block)) {
	for key, page := range d.Pages {
		if page.Blocks != nil {
			blocks := *page.Blocks
			for i := range blocks {
				fn(&blocks[i])
			}
		}
		for i := range page.Top {
			fn(&page.Top[i])
		}
		for i := range page.Bottom {
			fn(&page.Bottom[i])
		}
		d.Pages[key] = page
	}
	for i := range d.CustomPages {
		for j := range d.CustomPages[i].Blocks {
			fn(&d.CustomPages[i].Blocks[j])
		}
	}
}

func (d Document) TextCount() int {
	n := 0
	for _, keys := range d.Texts {
		n += len(keys)
	}
	return n
}

func Same(a, b Document) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return bytes.Equal(left, right)
}
