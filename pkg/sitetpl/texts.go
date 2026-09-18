package sitetpl

type Change struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Changes map[string]map[string]Change

func (c Changes) Count() int {
	n := 0
	for _, keys := range c {
		n += len(keys)
	}
	return n
}

func (c Changes) Set(locale, key string, change Change) {
	if c[locale] == nil {
		c[locale] = map[string]Change{}
	}
	c[locale][key] = change
}

func Reverts(newer []Changes) map[string]map[string]string {
	out := map[string]map[string]string{}
	for _, changes := range newer {
		for locale, keys := range changes {
			for key, change := range keys {
				if out[locale] == nil {
					out[locale] = map[string]string{}
				}
				if _, seen := out[locale][key]; !seen {
					out[locale][key] = change.From
				}
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
