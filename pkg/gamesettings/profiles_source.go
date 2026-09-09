package gamesettings

import "regexp"

var mapNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func sourceCommonFields(fileID string) []Field {
	return []Field{
		{
			Key: "hostname", File: fileID, Prop: "hostname",
			Label: "Название сервера", Section: SectionGeneral,
			Kind: KindString, MaxLen: 128,
			Hint:        "Видно в списке серверов",
			AppliesLive: false,
		},
		{
			Key: "rcon_password", File: fileID, Prop: "rcon_password",
			Label: "Пароль RCON", Section: SectionAdmin,
			Kind: KindString, MaxLen: 128, Secret: true, Clearable: true,
			Hint: "Пустой пароль полностью отключает удалённое управление",
		},
		{
			Key: "sv_password", File: fileID, Prop: "sv_password",
			Label: "Пароль для входа", Section: SectionGeneral,
			Kind: KindString, MaxLen: 128, Secret: true, Clearable: true,
			Hint: "Пусто — сервер открыт для всех",
		},
		{
			Key: "sv_lan", File: fileID, Prop: "sv_lan",
			Label: "Только локальная сеть", Section: SectionNetwork,
			Kind: KindBool, True: "1", False: "0", Default: "0",
		},
	}
}

func sourceInternetFields(fileID string) []Field {
	return []Field{
		{
			Key: "sv_region", File: fileID, Prop: "sv_region",
			Label: "Регион", Section: SectionNetwork, Kind: KindEnum, Default: "3",
			Options: []Option{
				{Value: "0", Label: "США — восток"},
				{Value: "1", Label: "США — запад"},
				{Value: "2", Label: "Южная Америка"},
				{Value: "3", Label: "Европа"},
				{Value: "4", Label: "Азия"},
				{Value: "5", Label: "Австралия"},
				{Value: "6", Label: "Ближний Восток"},
				{Value: "7", Label: "Африка"},
				{Value: "255", Label: "Весь мир"},
			},
		},
		{
			Key: "sv_contact", File: fileID, Prop: "sv_contact",
			Label: "Контакт администратора", Section: SectionGeneral,
			Kind: KindString, MaxLen: 128, Clearable: true,
		},
		{
			Key: "sv_tags", File: fileID, Prop: "sv_tags",
			Label: "Метки сервера", Section: SectionNetwork,
			Kind: KindString, MaxLen: 128, Clearable: true,
			Hint: "Через запятую, по ним сервер ищут в списке",
		},
		{
			Key: "sv_downloadurl", File: fileID, Prop: "sv_downloadurl",
			Label: "Адрес быстрой загрузки", Section: SectionNetwork,
			Kind: KindString, MaxLen: 255, Clearable: true,
			Hint: "HTTP-зеркало с картами и материалами",
		},
	}
}

func init() {
	register(Profile{
		Key: "cs16",
		Files: []ConfigFile{
			{ID: "main", Path: "cstrike/server.cfg", Format: FormatSourceCfg, Title: "cstrike/server.cfg"},
			{ID: "root", Path: "server.cfg", Format: FormatSourceCfg, Title: "server.cfg", RawOnly: true},
			{ID: "startup", Path: StartupPath, Format: FormatArgs, Title: "Параметры запуска"},
		},
		Fields: append(append(sourceCommonFields("main"), Field{
			Key: "map", File: "startup", Prop: "map",
			Label: "Стартовая карта", Section: SectionWorld,
			Kind: KindString, MaxLen: 64, Pattern: mapNamePattern, Default: "de_dust2",
			Hint: "Имя файла карты без расширения",
		}), []Field{
			{
				Key: "mp_timelimit", File: "main", Prop: "mp_timelimit",
				Label: "Длительность карты, мин", Section: SectionGameplay,
				Kind: KindInt, Max: AtMost(10000), AppliesLive: true,
			},
			{
				Key: "mp_friendlyfire", File: "main", Prop: "mp_friendlyfire",
				Label: "Огонь по своим", Section: SectionGameplay,
				Kind: KindBool, True: "1", False: "0", AppliesLive: true,
			},
			{
				Key: "mp_autoteambalance", File: "main", Prop: "mp_autoteambalance",
				Label: "Автобаланс команд", Section: SectionGameplay,
				Kind: KindBool, True: "1", False: "0", AppliesLive: true,
			},
			{
				Key: "mp_autokick", File: "main", Prop: "mp_autokick",
				Label: "Кик за бездействие", Section: SectionPlayers,
				Kind: KindBool, True: "1", False: "0", AppliesLive: true,
			},
			{
				Key: "mp_limitteams", File: "main", Prop: "mp_limitteams",
				Label: "Разница в размере команд", Section: SectionGameplay,
				Kind: KindInt, Max: AtMost(1000), AppliesLive: true,
			},
		}...),
	})

	register(Profile{
		Key: "cs2",
		Files: []ConfigFile{
			{ID: "main", Path: "game/csgo/cfg/server.cfg", Format: FormatSourceCfg, Title: "server.cfg"},
			{ID: "startup", Path: StartupPath, Format: FormatArgs, Title: "Параметры запуска"},
		},
		Fields: append(sourceCommonFields("main"), Field{
			Key: "sv_setsteamaccount", File: "main", Prop: "sv_setsteamaccount",
			Label: "Токен игрового сервера Steam", Section: SectionAdmin,
			Kind: KindString, MaxLen: 128, Secret: true, Clearable: true,
			Hint: "Без токена сервер не появится в публичном списке",
		}),
	})

	register(Profile{
		Key: "css",
		Files: []ConfigFile{
			{ID: "main", Path: "cstrike/cfg/server.cfg", Format: FormatSourceCfg, Title: "server.cfg"},
			{ID: "startup", Path: StartupPath, Format: FormatArgs, Title: "Параметры запуска"},
		},
		Fields: append(sourceCommonFields("main"), sourceInternetFields("main")...),
	})

	register(Profile{
		Key: "gmod",
		Files: []ConfigFile{
			{ID: "main", Path: "garrysmod/cfg/server.cfg", Format: FormatSourceCfg, Title: "server.cfg"},
			{ID: "startup", Path: StartupPath, Format: FormatArgs, Title: "Параметры запуска"},
		},
		Fields: append(sourceCommonFields("main"), sourceInternetFields("main")...),
	})

	register(Profile{
		Key: "tf2",
		Files: []ConfigFile{
			{ID: "main", Path: "tf/cfg/server.cfg", Format: FormatSourceCfg, Title: "server.cfg"},
			{ID: "startup", Path: StartupPath, Format: FormatArgs, Title: "Параметры запуска"},
		},
		Fields: append(append(sourceCommonFields("main"), sourceInternetFields("main")...), Field{
			Key: "tf_bot_quota", File: "main", Prop: "tf_bot_quota",
			Label: "Число ботов", Section: SectionGameplay,
			Kind: KindInt, Min: AtLeast(0), Max: AtMost(1000), AppliesLive: true,
		}),
	})
}
