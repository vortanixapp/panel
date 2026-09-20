package gameconsole

func init() {
	register(Profile{
		Key:   "rust",
		Title: "Rust",
		Rules: []Rule{
			{Kind: KindChat, Pattern: `^\[CHAT\] (?<player>[^:\[]+?)(?:\[\d+\])?\s*:\s*(?<text>.*)$`},
			{Kind: KindJoin, Pattern: `(?<player>.+?)\[\d+/\d+\] has entered the game`},
			{Kind: KindLeave, Pattern: `(?<player>.+?)\[\d+/\d+\] disconnecting:`},
			{Kind: KindReady, Pattern: `Server startup complete`},
			{Kind: KindStop, Pattern: `(?i)^(?:saving and )?shutting down`},
			{Kind: KindError, Pattern: `(?i)\b(error|exception|failed)\b`},
			{Kind: KindWarn, Pattern: `(?i)\bwarning\b`},
		},
		Commands: []Command{
			{ID: "players", Label: "Кто на сервере", Template: "players"},
			{ID: "say", Label: "Сообщение в чат", Template: "say {text}",
				Args: []Arg{{Name: "text", Label: "Текст", Kind: "text"}}},
			{ID: "save", Label: "Сохранить мир", Template: "server.save"},
			{ID: "kick", Label: "Выгнать игрока", Template: "kick {player} {reason}",
				Args: []Arg{
					{Name: "player", Label: "Игрок", Kind: "player"},
					{Name: "reason", Label: "Причина", Kind: "text", Optional: true},
				}},
			{ID: "ban", Label: "Забанить", Template: "ban {player}", Danger: true,
				Args: []Arg{{Name: "player", Label: "Игрок", Kind: "player"}}},
			{ID: "quit", Label: "Остановить сервер", Template: "quit", Danger: true,
				Hint: "Перед выключением мир сохраняется"},
		},
	})

	register(Profile{
		Key:       "arkse",
		SharedBy:  []string{"arksa"},
		Title:     "ARK",
		Note:      "ARK не принимает команды через консоль — управляйте сервером по RCON или из игры",
		Timestamp: `^(?<time>\d{4}\.\d{2}\.\d{2}_\d{2}\.\d{2}\.\d{2}):\s*`,
		Rules: []Rule{
			{Kind: KindJoin, Pattern: `(?<player>[^:]+) joined this ARK!`},
			{Kind: KindLeave, Pattern: `(?<player>[^:]+) left this ARK!`},
			{Kind: KindReady, Pattern: `Full Startup: [\d.]+ seconds`},
			{Kind: KindError, Pattern: `(?i)\b(error|fatal|assertion failed)\b`},
			{Kind: KindWarn, Pattern: `(?i)\bwarning\b`},
		},
	})

	register(Profile{
		Key:       "valheim",
		Title:     "Valheim",
		Timestamp: `^(?<time>\d{2}/\d{2}/\d{4} \d{2}:\d{2}:\d{2}):\s*`,
		Note:      "Valheim не принимает команды через консоль — используйте внутриигровую консоль администратора",
		Rules: []Rule{
			{Kind: KindJoin, Pattern: `Got character ZDOID from (?<player>.+?) : `},
			{Kind: KindReady, Pattern: `Game server connected`},
			{Kind: KindStop, Pattern: `Shuting down|Shutting down`},
			{Kind: KindError, Pattern: `(?i)\b(error|exception|failed)\b`},
			{Kind: KindWarn, Pattern: `(?i)\bwarning\b`},
		},
	})

	register(Profile{
		Key:       "7d2d",
		Title:     "7 Days to Die",
		Timestamp: `^(?<time>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\s+[\d.]+\s*`,
		Rules: []Rule{
			{Kind: KindChat, Pattern: `Chat \(from [^)]*\): '(?<player>[^']+)': (?<text>.*)$`},
			{Kind: KindJoin, Pattern: `GMSG: Player '(?<player>[^']+)' joined the game`},
			{Kind: KindLeave, Pattern: `GMSG: Player '(?<player>[^']+)' left the game`},
			{Kind: KindReady, Pattern: `StartGame done`},
			{Kind: KindStop, Pattern: `Exiting the game|Shutdown initiated`},
			{Kind: KindError, Pattern: `(?i)\bERR\b|\b(?:error|exception)\b`},
			{Kind: KindWarn, Pattern: `(?i)\bWRN\b|\bwarning\b`},
		},
		Commands: []Command{
			{ID: "lp", Label: "Кто на сервере", Template: "lp"},
			{ID: "say", Label: "Сообщение в чат", Template: "say \"{text}\"",
				Args: []Arg{{Name: "text", Label: "Текст", Kind: "text"}}},
			{ID: "saveworld", Label: "Сохранить мир", Template: "saveworld"},
			{ID: "kick", Label: "Выгнать игрока", Template: "kick \"{player}\" \"{reason}\"",
				Args: []Arg{
					{Name: "player", Label: "Игрок", Kind: "player"},
					{Name: "reason", Label: "Причина", Kind: "text", Optional: true},
				}},
			{ID: "shutdown", Label: "Остановить сервер", Template: "shutdown", Danger: true},
		},
	})

	register(Profile{
		Key:   "pzomboid",
		Title: "Project Zomboid",
		Rules: []Rule{
			{Kind: KindReady, Pattern: `Server Steam ID|SERVER STARTED`},
			{Kind: KindError, Pattern: `(?i)\b(error|exception|severe)\b`},
			{Kind: KindWarn, Pattern: `(?i)\bwarn(?:ing)?\b`},
		},
		Commands: []Command{
			{ID: "players", Label: "Кто на сервере", Template: "players"},
			{ID: "servermsg", Label: "Сообщение в чат", Template: "servermsg \"{text}\"",
				Args: []Arg{{Name: "text", Label: "Текст", Kind: "text"}}},
			{ID: "save", Label: "Сохранить мир", Template: "save"},
			{ID: "kick", Label: "Выгнать игрока", Template: "kickuser \"{player}\"",
				Args: []Arg{{Name: "player", Label: "Игрок", Kind: "player"}}},
			{ID: "quit", Label: "Остановить сервер", Template: "quit", Danger: true},
		},
	})

	register(Profile{
		Key:       "factorio",
		Title:     "Factorio",
		Timestamp: `^(?<time>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\s*`,
		Rules: []Rule{
			{Kind: KindChat, Pattern: `\[CHAT\] (?<player>[^:]+): (?<text>.*)$`},
			{Kind: KindJoin, Pattern: `\[JOIN\] (?<player>.+?) joined the game`},
			{Kind: KindLeave, Pattern: `\[LEAVE\] (?<player>.+?) left the game`},
			{Kind: KindReady, Pattern: `Hosting game at|changing state from\(CreatingGame\) to\(InGame\)`},
			{Kind: KindStop, Pattern: `Goodbye|changing state from\(InGame\) to\(DisconnectingScheduled\)`},
			{Kind: KindError, Pattern: `(?i)\b(error|failed)\b`},
			{Kind: KindWarn, Pattern: `(?i)\bwarning\b`},
		},
		Commands: []Command{
			{ID: "players", Label: "Кто на сервере", Template: "/players online"},
			{ID: "save", Label: "Сохранить мир", Template: "/server-save"},
			{ID: "say", Label: "Сообщение в чат", Template: "{text}",
				Args: []Arg{{Name: "text", Label: "Текст", Kind: "text"}},
				Hint: "Текст без слеша уходит в игровой чат"},
		},
	})

	register(Profile{
		Key:   "palworld",
		Title: "Palworld",
		Note:  "Palworld не принимает команды через консоль — включите RCON в настройках сервера",
		Rules: []Rule{
			{Kind: KindReady, Pattern: `Running Palworld dedicated server|Setting breakpad minidump`},
			{Kind: KindError, Pattern: `(?i)\b(error|fatal|assertion failed)\b`},
			{Kind: KindWarn, Pattern: `(?i)\bwarning\b`},
		},
	})
}
