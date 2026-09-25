package sitetpl

type fieldKind int

const (
	kindText fieldKind = iota
	kindLong
	kindHTML
	kindURL
	kindImage
	kindVideo
	kindIcon
	kindBool
	kindEnum
	kindInt
	kindList
)

type field struct {
	kind   fieldKind
	values []string
	min    int
	max    int
	item   map[string]field
}

var (
	fText  = field{kind: kindText}
	fLong  = field{kind: kindLong}
	fHTML  = field{kind: kindHTML}
	fURL   = field{kind: kindURL}
	fImage = field{kind: kindImage}
	fVideo = field{kind: kindVideo}
	fIcon  = field{kind: kindIcon}
	fBool  = field{kind: kindBool}
)

func fEnum(values ...string) field {
	return field{kind: kindEnum, values: values}
}

func fInt(min, max int) field {
	return field{kind: kindInt, min: min, max: max}
}

func fList(max int, item map[string]field) field {
	return field{kind: kindList, max: max, item: item}
}

var builtinSections = map[string]bool{
	"landing.hero":      true,
	"landing.games":     true,
	"landing.steps":     true,
	"landing.hardware":  true,
	"landing.panel":     true,
	"landing.locations": true,
	"landing.pricing":   true,
	"landing.faq":       true,
	"landing.cta":       true,
}

var blockSchemas = map[string]map[string]field{
	"heading": {
		"eyebrow": fText,
		"title":   fText,
		"text":    fLong,
		"align":   fEnum("left", "center"),
		"size":    fEnum("md", "lg", "xl"),
	},
	"text": {
		"html":  fHTML,
		"width": fEnum("narrow", "wide"),
		"align": fEnum("left", "center"),
	},
	"image": {
		"src":     fImage,
		"alt":     fText,
		"caption": fText,
		"url":     fURL,
		"new_tab": fBool,
		"width":   fEnum("narrow", "wide", "full"),
		"rounded": fBool,
	},
	"media": {
		"src":     fImage,
		"alt":     fText,
		"eyebrow": fText,
		"title":   fText,
		"text":    fLong,
		"button":  fText,
		"url":     fURL,
		"side":    fEnum("left", "right"),
	},
	"features": {
		"title":   fText,
		"text":    fLong,
		"columns": fInt(2, 4),
		"items": fList(24, map[string]field{
			"icon":  fIcon,
			"title": fText,
			"text":  fLong,
			"url":   fURL,
		}),
	},
	"cta": {
		"title":   fText,
		"text":    fLong,
		"button":  fText,
		"url":     fURL,
		"button2": fText,
		"url2":    fURL,
		"tone":    fEnum("accent", "muted"),
	},
	"faq": {
		"title": fText,
		"text":  fLong,
		"items": fList(40, map[string]field{
			"q": fText,
			"a": fLong,
		}),
	},
	"stats": {
		"title": fText,
		"items": fList(8, map[string]field{
			"value": fText,
			"label": fText,
		}),
	},
	"notice": {
		"tone":        fEnum("info", "success", "warning", "danger"),
		"title":       fText,
		"text":        fLong,
		"button":      fText,
		"url":         fURL,
		"dismissible": fBool,
	},
	"video": {
		"url":     fVideo,
		"title":   fText,
		"caption": fText,
	},
	"html": {
		"html":    fHTML,
		"scripts": fBool,
	},
	"spacer": {
		"size": fEnum("sm", "md", "lg", "xl"),
		"line": fBool,
	},
}
