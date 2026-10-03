package settingsreg

import "strings"

const (
	GroupAccounting = "accounting"
)

var (
	AccountingRegions = def(Setting{
		Key: "accounting.regions", Group: GroupAccounting, Section: "regions",
		Kind: KindMulti, Default: "cis", Options: []string{"cis", "eu", "europe", "us"}, Public: true,
	})
)

func AccountingRegionList() []string {
	out := []string{}
	for _, part := range strings.Split(AccountingRegions.Str(), ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
