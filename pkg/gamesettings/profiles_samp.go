package gamesettings

func sampFields() []Field {
	return []Field{
		{
			Key: "hostname", Prop: "hostname",
			Label: "Название сервера", Section: SectionGeneral,
			Kind: KindString, MaxLen: 128,
		},
		{
			Key: "maxplayers", Prop: "maxplayers", Slots: true,
			Label: "Мест на сервере", Section: SectionGeneral,
			Kind: KindInt, Min: AtLeast(1), Max: AtMost(1000),
		},
		{
			Key: "rcon_password", Prop: "rcon_password",
			Label: "Пароль RCON", Section: SectionAdmin,
			Kind: KindString, MaxLen: 128, Secret: true,
		},
		{
			Key: "password", Prop: "password",
			Label: "Пароль для входа", Section: SectionGeneral,
			Kind: KindString, MaxLen: 128, Secret: true, Clearable: true,
			Hint: "Пусто — сервер открыт для всех",
		},
		{
			Key: "weburl", Prop: "weburl",
			Label: "Сайт проекта", Section: SectionGeneral,
			Kind: KindString, MaxLen: 255, Clearable: true,
		},
		{
			Key: "announce", Prop: "announce",
			Label: "Объявлять в общем списке", Section: SectionNetwork,
			Kind: KindBool, True: "1", False: "0", Default: "1",
		},
		{
			Key: "query", Prop: "query",
			Label: "Отвечать на запросы состояния", Section: SectionNetwork,
			Kind: KindBool, True: "1", False: "0", Default: "1",
			Hint: "Выключение скрывает сервер из мониторингов",
		},
		{
			Key: "lanmode", Prop: "lanmode",
			Label: "Только локальная сеть", Section: SectionNetwork,
			Kind: KindBool, True: "1", False: "0", Default: "0",
		},
		{
			Key: "anticheat", Prop: "anticheat",
			Label: "Встроенная защита от читов", Section: SectionPlayers,
			Kind: KindBool, True: "1", False: "0",
		},
		{
			Key: "lagcompmode", Prop: "lagcompmode",
			Label: "Компенсация задержки", Section: SectionPerformance,
			Kind: KindBool, True: "1", False: "0",
		},
		{
			Key: "onfoot_rate", Prop: "onfoot_rate",
			Label: "Частота синхронизации пешком", Section: SectionPerformance,
			Kind: KindInt, Min: AtLeast(1), Max: AtMost(1000), Default: "40",
			Hint: "Меньше значение — чаще обновления и выше нагрузка",
		},
		{
			Key: "incar_rate", Prop: "incar_rate",
			Label: "Частота синхронизации в транспорте", Section: SectionPerformance,
			Kind: KindInt, Min: AtLeast(1), Max: AtMost(1000), Default: "40",
		},
		{
			Key: "weapon_rate", Prop: "weapon_rate",
			Label: "Частота синхронизации стрельбы", Section: SectionPerformance,
			Kind: KindInt, Min: AtLeast(1), Max: AtMost(1000), Default: "40",
		},
		{
			Key: "stream_rate", Prop: "stream_rate",
			Label: "Частота стриминга объектов", Section: SectionPerformance,
			Kind: KindInt, Min: AtLeast(1), Max: AtMost(1000000),
		},
		{
			Key: "maxnpc", Prop: "maxnpc",
			Label: "Предел NPC", Section: SectionGameplay,
			Kind: KindInt, Min: AtLeast(0), Max: AtMost(10000), Default: "0",
		},
		{
			Key: "worldtime", Prop: "worldtime",
			Label: "Время суток в мире", Section: SectionWorld,
			Kind: KindInt, Min: AtLeast(0), Max: AtMost(23),
		},
		{
			Key: "logqueries", Prop: "logqueries",
			Label: "Писать запросы в журнал", Section: SectionAdvanced,
			Kind: KindBool, True: "1", False: "0",
		},
		{
			Key: "logbans", Prop: "logbans",
			Label: "Писать баны в журнал", Section: SectionAdvanced,
			Kind: KindBool, True: "1", False: "0",
		},
		{
			Key: "logtimeformat", Prop: "logtimeformat",
			Label: "Формат времени в журнале", Section: SectionAdvanced,
			Kind: KindString, MaxLen: 128, Clearable: true,
		},
	}
}

func init() {
	register(Profile{
		Key: "samp",
		Files: []ConfigFile{
			{ID: "main", Path: "server.cfg", Format: FormatKVSpace, Title: "server.cfg"},
			{ID: "startup", Path: StartupPath, Format: FormatArgs, Title: "Параметры запуска"},
		},
		Fields: sampFields(),
	})

	register(Profile{
		Key: "crmp",
		Files: []ConfigFile{
			{ID: "main", Path: "server.cfg", Format: FormatKVSpace, Title: "server.cfg"},
			{ID: "startup", Path: StartupPath, Format: FormatArgs, Title: "Параметры запуска"},
		},
		Fields: sampFields(),
	})

	register(Profile{
		Key: "mta",
		Files: []ConfigFile{
			{ID: "main", Path: "mods/deathmatch/mtaserver.conf", Format: FormatXMLTags, Title: "mtaserver.conf"},
			{ID: "acl", Path: "mods/deathmatch/acl.xml", Format: FormatXMLTags, Title: "acl.xml", RawOnly: true},
			{ID: "startup", Path: StartupPath, Format: FormatArgs, Title: "Параметры запуска"},
		},
		Fields: []Field{
			{
				Key: "servername", File: "main", Prop: "servername",
				Label: "Название сервера", Section: SectionGeneral,
				Kind: KindString, MaxLen: 128,
			},
			{
				Key: "maxplayers", File: "main", Prop: "maxplayers", Slots: true,
				Label: "Мест на сервере", Section: SectionGeneral,
				Kind: KindInt, Min: AtLeast(1), Max: AtMost(1000),
			},
			{
				Key: "password", File: "main", Prop: "password",
				Label: "Пароль для входа", Section: SectionGeneral,
				Kind: KindString, MaxLen: 128, Secret: true, Clearable: true,
			},
			{
				Key: "httpport", File: "main", Prop: "httpport",
				Label: "Порт встроенного HTTP", Section: SectionNetwork,
				Kind: KindInt, Min: AtLeast(1), Max: AtMost(65535), ReadOnly: true,
				Hint: "Назначается панелью вместе с игровым портом",
			},
			{
				Key: "fpslimit", File: "main", Prop: "fpslimit",
				Label: "Предел кадров у клиентов", Section: SectionPerformance,
				Kind: KindInt, Min: AtLeast(25), Max: AtMost(100), Default: "36",
			},
			{
				Key: "ase", File: "main", Prop: "ase",
				Label: "Объявлять в общем списке", Section: SectionNetwork,
				Kind: KindBool, True: "1", False: "0", Default: "1",
			},
			{
				Key: "donotbroadcastlan", File: "main", Prop: "donotbroadcastlan",
				Label: "Не вещать в локальную сеть", Section: SectionNetwork,
				Kind: KindBool, True: "1", False: "0",
			},
		},
	})
}
