package gameconsole

func init() {
	register(Profile{
		Key:       "mcjava",
		SharedBy:  []string{"mcpaper", "mcspigot", "mcforge", "mcfabric", "pocketmine"},
		Title:     "Minecraft",
		Timestamp: `^\[(?<time>\d{2}:\d{2}:\d{2})\]\s*`,
		Trim:      []string{`^\[[^\]]*\]:?\s*`},
		Rules: []Rule{
			{Kind: KindChat, Pattern: `\]:? <(?<player>[A-Za-z0-9_]{1,16})> (?<text>.*)$`},
			{Kind: KindJoin, Pattern: `\]:? (?<player>[A-Za-z0-9_]{1,16}) joined the game`},
			{Kind: KindLeave, Pattern: `\]:? (?<player>[A-Za-z0-9_]{1,16}) left the game`},
			{Kind: KindJoin, Pattern: `\]:? (?<player>[A-Za-z0-9_]{1,16})\[/[^\]]+\] logged in with entity id`},
			{Kind: KindLeave, Pattern: `\]:? (?<player>[A-Za-z0-9_]{1,16}) lost connection:`},
			{Kind: KindMap, Pattern: `Preparing level "(?<value>[^"]+)"`},
			{Kind: KindReady, Pattern: `Done \([0-9.]+s\)! For help, type`},
			{Kind: KindStop, Pattern: `Stopping (?:the )?server`},
			{Kind: KindError, Pattern: `\[[^\]]*(?:ERROR|SEVERE|FATAL)\]`},
			{Kind: KindWarn, Pattern: `\[[^\]]*WARN[^\]]*\]`},
		},
		Commands: []Command{
			{ID: "list", Label: "Кто на сервере", Template: "list", Hint: "Показывает игроков онлайн"},
			{ID: "say", Label: "Сообщение в чат", Template: "say {text}",
				Args: []Arg{{Name: "text", Label: "Текст", Placeholder: "Сервер перезапустится через 5 минут", Kind: "text"}}},
			{ID: "save", Label: "Сохранить мир", Template: "save-all", Hint: "Сбрасывает мир на диск"},
			{ID: "op", Label: "Выдать права", Template: "op {player}",
				Args: []Arg{{Name: "player", Label: "Игрок", Kind: "player"}}},
			{ID: "deop", Label: "Забрать права", Template: "deop {player}",
				Args: []Arg{{Name: "player", Label: "Игрок", Kind: "player"}}},
			{ID: "kick", Label: "Выгнать игрока", Template: "kick {player} {reason}",
				Args: []Arg{
					{Name: "player", Label: "Игрок", Kind: "player"},
					{Name: "reason", Label: "Причина", Kind: "text", Optional: true},
				}},
			{ID: "ban", Label: "Забанить", Template: "ban {player} {reason}", Danger: true,
				Args: []Arg{
					{Name: "player", Label: "Игрок", Kind: "player"},
					{Name: "reason", Label: "Причина", Kind: "text", Optional: true},
				}},
			{ID: "whitelist_add", Label: "В белый список", Template: "whitelist add {player}",
				Args: []Arg{{Name: "player", Label: "Игрок", Kind: "player"}}},
			{ID: "time_day", Label: "Сделать день", Template: "time set day"},
			{ID: "weather_clear", Label: "Разогнать тучи", Template: "weather clear"},
			{ID: "stop", Label: "Остановить сервер", Template: "stop", Danger: true,
				Hint: "Сервер корректно сохранится и выключится"},
		},
	})

	register(Profile{
		Key:       "mcbedrock",
		Title:     "Minecraft Bedrock",
		Timestamp: `^\[(?<time>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?::\d+)?)`,
		Trim:      []string{`^\[[^\]]*\]\s*`},
		Rules: []Rule{
			{Kind: KindJoin, Pattern: `Player connected:\s*(?<player>[^,]+)`},
			{Kind: KindLeave, Pattern: `Player disconnected:\s*(?<player>[^,]+)`},
			{Kind: KindReady, Pattern: `Server started\.`},
			{Kind: KindStop, Pattern: `Stopping server|Quit correctly`},
			{Kind: KindError, Pattern: `(?i)\bERROR\b`},
			{Kind: KindWarn, Pattern: `(?i)\bWARN(?:ING)?\b`},
		},
		Commands: []Command{
			{ID: "list", Label: "Кто на сервере", Template: "list"},
			{ID: "say", Label: "Сообщение в чат", Template: "say {text}",
				Args: []Arg{{Name: "text", Label: "Текст", Kind: "text"}}},
			{ID: "save", Label: "Сохранить мир", Template: "save hold"},
			{ID: "op", Label: "Выдать права", Template: "op {player}",
				Args: []Arg{{Name: "player", Label: "Игрок", Kind: "player"}}},
			{ID: "kick", Label: "Выгнать игрока", Template: "kick {player} {reason}",
				Args: []Arg{
					{Name: "player", Label: "Игрок", Kind: "player"},
					{Name: "reason", Label: "Причина", Kind: "text", Optional: true},
				}},
			{ID: "whitelist_add", Label: "В белый список", Template: "whitelist add {player}",
				Args: []Arg{{Name: "player", Label: "Игрок", Kind: "player"}}},
			{ID: "stop", Label: "Остановить сервер", Template: "stop", Danger: true},
		},
	})
}
