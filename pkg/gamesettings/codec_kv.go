package gamesettings

func init() {
	// SA-MP и CRMP: server.cfg вида `ключ значение`, без кавычек.
	// Комментарии в SA-MP пишут и решёткой, и точкой с запятой, и двойной косой.
	registerCodec(FormatKVSpace, lineCodec{syn: lineSyntax{
		separator:      " ",
		comments:       []string{"#", ";", "//"},
		inlineComments: []string{"#", ";", "//"},
		quote:          quoteNever,
	}})

	// GoldSrc и Source: `hostname "Мой сервер"`. Значения принято брать в
	// кавычки, комментарий — двойная косая черта.
	//
	// quoteWhenNeeded, а не quoteAlways: числовые cvar'ы вроде `mp_timelimit 30`
	// в кавычки не оборачивают, и переписывать весь конфиг под один стиль при
	// правке одного поля — значит выдать за свои изменения весь файл.
	registerCodec(FormatSourceCfg, lineCodec{syn: lineSyntax{
		separator:      " ",
		comments:       []string{"//", "#"},
		inlineComments: []string{"//"},
		quote:          quoteWhenNeeded,
	}})

	// server.properties (Minecraft Java и Bedrock) и rust.env.
	//
	// Ключи здесь физические, ровно как в файле: `max-players`, `rcon.password`,
	// `query.port`. Прежний аппликатор переводил их из `max_players` сам, а
	// парсер возвращал как есть — из-за чего прочитанное значение нельзя было
	// записать обратно тем же именем. Перевод переехал в Field.Prop.
	registerCodec(FormatProperties, lineCodec{syn: lineSyntax{
		separator:      "=",
		equals:         true,
		comments:       []string{"#", "!"},
		inlineComments: nil, // в properties символ # внутри значения законен
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
