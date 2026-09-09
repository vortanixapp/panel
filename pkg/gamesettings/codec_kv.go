package gamesettings

func init() {
	registerCodec(FormatKVSpace, lineCodec{syn: lineSyntax{
		separator:      " ",
		comments:       []string{"#", ";", "//"},
		inlineComments: []string{"#", ";", "//"},
		quote:          quoteNever,
	}})

	registerCodec(FormatSourceCfg, lineCodec{syn: lineSyntax{
		separator:      " ",
		comments:       []string{"//", "#"},
		inlineComments: []string{"//"},
		quote:          quoteWhenNeeded,
	}})

	registerCodec(FormatProperties, lineCodec{syn: lineSyntax{
		separator:      "=",
		equals:         true,
		comments:       []string{"#", "!"},
		inlineComments: nil,
		quote:          quoteNever,
	}})

	registerCodec(FormatEnv, lineCodec{syn: lineSyntax{
		separator:      "=",
		equals:         true,
		comments:       []string{"#"},
		inlineComments: nil,
		quote:          quoteWhenNeeded,
	}})
}
