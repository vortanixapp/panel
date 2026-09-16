package gamecatalog

import (
	"path"
	"strings"
)

const RuntimeKeyPrefix = "runtime:"

type BuildTarget struct {
	Key     string
	Label   string
	Image   string
	Recipe  string
	Keys    []string
	Runtime bool
}

func RuntimeKey(kind string) string {
	return RuntimeKeyPrefix + kind
}

func RuntimeKeys() []string {
	builds := RuntimeBuilds()
	out := make([]string, 0, len(builds))
	for _, rt := range builds {
		out = append(out, RuntimeKey(rt.Kind))
	}
	return out
}

func RuntimeOf(code string) string {
	l, ok := LaunchOf(code)
	if !ok {
		return ""
	}
	return string(l.Runtime)
}

func NormalizeImageKey(key string) (string, bool) {
	key = strings.TrimSpace(key)
	if kind, ok := strings.CutPrefix(key, RuntimeKeyPrefix); ok {
		for _, rt := range RuntimeBuilds() {
			if rt.Kind == kind {
				return RuntimeKey(kind), true
			}
		}
		return "", false
	}
	g, ok := Resolve(Normalize(key))
	if !ok {
		return "", false
	}
	return g.Key, true
}

func GameBuildImage(code string) (image, recipe string) {
	image = ImageWithTag(code, DefaultTag(code))
	recipe = path.Base(Repository(code))
	if image == "" || recipe == "." || recipe == "/" {
		return "vortanix/" + code + ":latest", code
	}
	return image, recipe
}

func SharedImageKeys(code string) []string {
	image, _ := GameBuildImage(code)
	out := []string{}
	for _, g := range All() {
		if other, _ := GameBuildImage(g.Key); other == image {
			out = append(out, g.Key)
		}
	}
	return out
}

func BuildTargets(keys []string) []BuildTarget {
	runtimes := map[string]bool{}
	var games []BuildTarget
	images := map[string]bool{}
	for _, raw := range keys {
		key, ok := NormalizeImageKey(raw)
		if !ok {
			continue
		}
		if strings.HasPrefix(key, RuntimeKeyPrefix) {
			runtimes[key] = true
			continue
		}
		image, recipe := GameBuildImage(key)
		if images[image] {
			continue
		}
		images[image] = true
		shared := SharedImageKeys(key)
		games = append(games, BuildTarget{
			Key:    key,
			Label:  strings.Join(shared, ", "),
			Image:  image,
			Recipe: recipe,
			Keys:   shared,
		})
	}

	out := make([]BuildTarget, 0, len(runtimes)+len(games))
	for _, rt := range RuntimeBuilds() {
		key := RuntimeKey(rt.Kind)
		if !runtimes[key] {
			continue
		}
		out = append(out, BuildTarget{
			Key:     key,
			Label:   "runtime " + rt.Kind,
			Image:   rt.Image,
			Recipe:  "_runtime/" + rt.Kind,
			Keys:    []string{key},
			Runtime: true,
		})
	}
	return append(out, games...)
}
