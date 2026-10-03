package regions

var namesEN = map[string]string{
	"AT": "Austria", "BE": "Belgium", "BG": "Bulgaria", "HR": "Croatia", "CY": "Cyprus",
	"CZ": "Czechia", "DK": "Denmark", "EE": "Estonia", "FI": "Finland", "FR": "France",
	"DE": "Germany", "GR": "Greece", "HU": "Hungary", "IE": "Ireland", "IT": "Italy",
	"LV": "Latvia", "LT": "Lithuania", "LU": "Luxembourg", "MT": "Malta", "NL": "Netherlands",
	"PL": "Poland", "PT": "Portugal", "RO": "Romania", "SK": "Slovakia", "SI": "Slovenia",
	"ES": "Spain", "SE": "Sweden",
	"GB": "United Kingdom", "CH": "Switzerland", "NO": "Norway", "IS": "Iceland",
	"LI": "Liechtenstein", "AL": "Albania", "AD": "Andorra", "BA": "Bosnia and Herzegovina",
	"ME": "Montenegro", "MK": "North Macedonia", "RS": "Serbia", "SM": "San Marino",
	"VA": "Vatican City", "MC": "Monaco", "XK": "Kosovo",
	"RU": "Russia", "BY": "Belarus", "KZ": "Kazakhstan", "UA": "Ukraine", "AM": "Armenia",
	"AZ": "Azerbaijan", "GE": "Georgia", "KG": "Kyrgyzstan", "MD": "Moldova", "TJ": "Tajikistan",
	"TM": "Turkmenistan", "UZ": "Uzbekistan",
	"US": "United States", "CA": "Canada", "AU": "Australia", "JP": "Japan", "CN": "China",
	"IN": "India", "BR": "Brazil", "TR": "Turkey", "IL": "Israel", "AE": "United Arab Emirates",
}

func NameEN(code string) string {
	code = NormalizeCountry(code)
	if name, ok := namesEN[code]; ok {
		return name
	}
	return code
}
