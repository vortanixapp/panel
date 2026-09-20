package gameconsole

func init() {
	register(Profile{
		Key:       "cs2",
		SharedBy:  []string{"cs16", "css", "tf2", "gmod"},
		Title:     "Source",
		Timestamp: `^L (?<time>\d{2}/\d{2}/\d{4} - \d{2}:\d{2}:\d{2}):\s*`,
		Rules: []Rule{
			{Kind: KindChat, Pattern: `"(?<player>[^"<]+)<\d+><[^>]*><[^>]*>" say(?:_team)? "(?<text>[^"]*)"`},
			{Kind: KindJoin, Pattern: `"(?<player>[^"<]+)<\d+><[^>]*><[^>]*>" connected`},
			{Kind: KindLeave, Pattern: `"(?<player>[^"<]+)<\d+><[^>]*><[^>]*>" disconnected`},
			{Kind: KindJoin, Pattern: `^(?<player>[^"]+?) connected, address`},
			{Kind: KindLeave, Pattern: `^Dropped (?<player>.+?) from server`},
			{Kind: KindMap, Pattern: `(?:Started|Loading) map "(?<value>[^"]+)"`},
			{Kind: KindReady, Pattern: `Connection to Steam servers successful`},
			{Kind: KindStop, Pattern: `^Server shutting down|^Shutdown function`},
			{Kind: KindError, Pattern: `(?i)\b(error|failed|couldn't|missing map)\b`},
			{Kind: KindWarn, Pattern: `(?i)\bwarning\b`},
		},
		Commands: []Command{
			{ID: "status", Label: "Состояние сервера", Template: "status",
				Hint: "Карта, слоты и список игроков"},
			{ID: "say", Label: "Сообщение в чат", Template: "say {text}",
				Args: []Arg{{Name: "text", Label: "Текст", Kind: "text"}}},
			{ID: "changelevel", Label: "Сменить карту", Template: "changelevel {map}",
				Args: []Arg{{Name: "map", Label: "Карта", Placeholder: "de_dust2", Kind: "text"}}},
			{ID: "kick", Label: "Выгнать игрока", Template: "kick {player}",
				Args: []Arg{{Name: "player", Label: "Игрок", Kind: "player"}}},
			{ID: "restart", Label: "Перезапустить раунд", Template: "mp_restartgame 1"},
			{ID: "quit", Label: "Остановить сервер", Template: "quit", Danger: true},
		},
	})
}
