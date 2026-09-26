package gamecatalog

type RuntimeSelector struct {
	Kind     string   `json:"kind"`
	File     string   `json:"file"`
	URLFile  string   `json:"url_file,omitempty"`
	Versions []string `json:"versions"`
}

var runtimeSelectors = map[string]RuntimeSelector{
	"java": {Kind: "java", File: ".vtx/java", Versions: []string{"8", "17", "21", "25"}},
	"php":  {Kind: "php", File: ".vtx/php", URLFile: ".vtx/php_url", Versions: []string{"8.1", "8.2", "8.3"}},
}

var gameRuntimeSelector = map[string]string{
	"pocketmine": "php",
}

func RuntimeSelectorOf(code string) (RuntimeSelector, bool) {
	g, ok := Resolve(code)
	if !ok {
		return RuntimeSelector{}, false
	}
	if kind, ok := gameRuntimeSelector[g.Key]; ok {
		sel, found := runtimeSelectors[kind]
		return sel, found
	}
	if l, ok := launches[g.Key]; ok && l.Runtime == RuntimeJava {
		sel, found := runtimeSelectors["java"]
		return sel, found
	}
	return RuntimeSelector{}, false
}

func RuntimeVersionAllowed(code, version string) bool {
	sel, ok := RuntimeSelectorOf(code)
	if !ok {
		return false
	}
	for _, v := range sel.Versions {
		if v == version {
			return true
		}
	}
	return false
}
