package gamesettings

// ARK и Conan Exiles.
//
// Обе игры на Unreal и настраиваются одинаково по устройству: основное лежит в
// ini-файлах, а имя сессии, карта и число мест передаются строкой запуска вида
// `Карта?Ключ=Значение`. Файлы появляются только после первого запуска — до
// этого настройки задаются строкой запуска, поэтому самое важное вынесено
// именно туда.

const (
	arkGUS      = "ShooterGame/Saved/Config/LinuxServer/GameUserSettings.ini"
	arkGameINI  = "ShooterGame/Saved/Config/LinuxServer/Game.ini"
	conanServer = "ConanSandbox/Saved/Config/LinuxServer/ServerSettings.ini"
	conanEngine = "ConanSandbox/Saved/Config/LinuxServer/Engine.ini"
	conanGame   = "ConanSandbox/Saved/Config/LinuxServer/Game.ini"
)

// arkFields — общая часть для обеих версий ARK. Отличаются они только списком карт.
func arkFields(maps []Option) []Field {
	return []Field{
		{Key: "session_name", File: "startup", Prop: "SessionName",
			Label: "Название сервера", Section: SectionGeneral, Kind: KindString, MaxLen: 128},
		{Key: "map", File: "startup", Prop: MapKey,
			Label: "Карта", Section: SectionWorld, Kind: KindEnum,
			Default: maps[0].Value, Options: maps,
			Hint: "Смена карты открывает другой мир; прежние сохранения остаются на месте"},
		{Key: "max_players", File: "startup", Prop: "MaxPlayers", Slots: true,
			Label: "Мест на сервере", Section: SectionGeneral,
			Kind: KindInt, Min: AtLeast(1), Max: AtMost(200)},
		{Key: "server_password", File: "startup", Prop: "ServerPassword",
			Label: "Пароль для входа", Section: SectionGeneral,
			Kind: KindString, MaxLen: 128, Secret: true, Clearable: true},

		{Key: "admin_password", File: "main", Prop: "ServerSettings/ServerAdminPassword",
			Label: "Пароль администратора", Section: SectionAdmin,
			Kind: KindString, MaxLen: 128, Secret: true},
		{Key: "rcon_enabled", File: "main", Prop: "ServerSettings/RCONEnabled",
			Label: "Удалённое управление", Section: SectionAdmin,
			Kind: KindBool, True: "True", False: "False"},
		{Key: "rcon_port", File: "main", Prop: "ServerSettings/RCONPort",
			Label: "Порт RCON", Section: SectionAdmin,
			Kind: KindInt, Min: AtLeast(1), Max: AtMost(65535), ReadOnly: true},

		{Key: "pve", File: "main", Prop: "ServerSettings/ServerPVE",
			Label: "Режим без боя между игроками", Section: SectionGameplay,
			Kind: KindBool, True: "True", False: "False"},
		{Key: "hardcore", File: "main", Prop: "ServerSettings/ServerHardcore",
			Label: "Хардкор", Section: SectionGameplay,
			Kind: KindBool, True: "True", False: "False",
			Hint: "Смерть сбрасывает персонажа на первый уровень"},
		{Key: "third_person", File: "main", Prop: "ServerSettings/AllowThirdPersonPlayer",
			Label: "Вид от третьего лица", Section: SectionGameplay,
			Kind: KindBool, True: "True", False: "False", Default: "true"},
		{Key: "show_map_location", File: "main", Prop: "ServerSettings/ShowMapPlayerLocation",
			Label: "Показывать себя на карте", Section: SectionGameplay,
			Kind: KindBool, True: "True", False: "False", Default: "true"},
		{Key: "crosshair", File: "main", Prop: "ServerSettings/ServerCrosshair",
			Label: "Прицел", Section: SectionGameplay,
			Kind: KindBool, True: "True", False: "False", Default: "true"},
		{Key: "global_voice", File: "main", Prop: "ServerSettings/GlobalVoiceChat",
			Label: "Общий голосовой чат", Section: SectionPlayers,
			Kind: KindBool, True: "True", False: "False"},

		{Key: "difficulty_offset", File: "main", Prop: "ServerSettings/DifficultyOffset",
			Label: "Сложность", Section: SectionWorld,
			Kind: KindFloat, Min: AtLeast(0), Max: AtMost(1), Default: "0.2",
			Hint: "Задаёт максимальный уровень существ в мире"},
		{Key: "day_cycle_speed", File: "main", Prop: "ServerSettings/DayCycleSpeedScale",
			Label: "Скорость смены суток", Section: SectionWorld,
			Kind: KindFloat, Min: AtLeast(0.1), Max: AtMost(10), Default: "1.0"},

		{Key: "xp_multiplier", File: "main", Prop: "ServerSettings/XPMultiplier",
			Label: "Множитель опыта", Section: SectionRates,
			Kind: KindFloat, Min: AtLeast(0.1), Max: AtMost(100), Default: "1.0"},
		{Key: "taming_speed", File: "main", Prop: "ServerSettings/TamingSpeedMultiplier",
			Label: "Скорость приручения", Section: SectionRates,
			Kind: KindFloat, Min: AtLeast(0.1), Max: AtMost(100), Default: "1.0"},
		{Key: "harvest_amount", File: "main", Prop: "ServerSettings/HarvestAmountMultiplier",
			Label: "Множитель добычи", Section: SectionRates,
			Kind: KindFloat, Min: AtLeast(0.1), Max: AtMost(100), Default: "1.0"},

		{Key: "max_tamed_dinos", File: "main", Prop: "ServerSettings/MaxTamedDinos",
			Label: "Предел прирученных существ", Section: SectionPerformance,
			Kind: KindFloat, Min: AtLeast(100), Max: AtMost(20000), Default: "4000",
			Hint: "Главный расход памяти на больших серверах"},
		{Key: "autosave_minutes", File: "main", Prop: "ServerSettings/AutoSavePeriodMinutes",
			Label: "Интервал автосохранения, мин", Section: SectionSaves,
			Kind: KindFloat, Min: AtLeast(1), Max: AtMost(240), Default: "15"},
	}
}

func arkFiles(gus, gameINI string) []ConfigFile {
	return []ConfigFile{
		{ID: "main", Path: gus, Format: FormatINI, Title: "GameUserSettings.ini"},
		{ID: "game", Path: gameINI, Format: FormatINI, Title: "Game.ini", RawOnly: true},
		{ID: "startup", Path: StartupPath, Format: FormatUEQuery, Title: "Параметры запуска"},
	}
}

func init() {
	register(Profile{
		Key:   "arkse",
		Files: arkFiles(arkGUS, arkGameINI),
		Note:  "Файлы настроек появляются после первого запуска сервера. До этого действуют только параметры запуска.",
		Fields: arkFields([]Option{
			{Value: "TheIsland", Label: "The Island"},
			{Value: "TheCenter", Label: "The Center"},
			{Value: "ScorchedEarth_P", Label: "Scorched Earth"},
			{Value: "Ragnarok", Label: "Ragnarok"},
			{Value: "Aberration_P", Label: "Aberration"},
			{Value: "Extinction", Label: "Extinction"},
			{Value: "Valguero_P", Label: "Valguero"},
			{Value: "CrystalIsles", Label: "Crystal Isles"},
			{Value: "Genesis", Label: "Genesis"},
			{Value: "Gen2", Label: "Genesis 2"},
			{Value: "LostIsland", Label: "Lost Island"},
			{Value: "Fjordur", Label: "Fjordur"},
		}),
	})

	register(Profile{
		Key:   "arksa",
		Files: arkFiles(arkGUS, arkGameINI),
		Note:  "Файлы настроек появляются после первого запуска сервера. До этого действуют только параметры запуска.",
		Fields: arkFields([]Option{
			{Value: "TheIsland_WP", Label: "The Island"},
			{Value: "TheCenter_WP", Label: "The Center"},
			{Value: "ScorchedEarth_WP", Label: "Scorched Earth"},
			{Value: "Aberration_WP", Label: "Aberration"},
			{Value: "Extinction_WP", Label: "Extinction"},
			{Value: "Ragnarok_WP", Label: "Ragnarok"},
		}),
	})

	// У Conan имя сервера и число мест лежат не там, где остальное: они в
	// Engine.ini, а игровые правила — в ServerSettings.ini.
	register(Profile{
		Key: "conan",
		Files: []ConfigFile{
			{ID: "main", Path: conanServer, Format: FormatINI, Title: "ServerSettings.ini"},
			{ID: "engine", Path: conanEngine, Format: FormatINI, Title: "Engine.ini"},
			{ID: "game", Path: conanGame, Format: FormatINI, Title: "Game.ini", RawOnly: true},
			{ID: "startup", Path: StartupPath, Format: FormatUEQuery, Title: "Параметры запуска"},
		},
		Note: "Файлы настроек появляются после первого запуска сервера.",
		Fields: []Field{
			{Key: "server_name", File: "engine", Prop: "OnlineSubsystem/ServerName",
				Label: "Название сервера", Section: SectionGeneral, Kind: KindString, MaxLen: 128},
			{Key: "max_players", File: "engine", Prop: "/Script/Engine.GameSession/MaxPlayers", Slots: true,
				Label: "Мест на сервере", Section: SectionGeneral,
				Kind: KindInt, Min: AtLeast(1), Max: AtMost(70)},
			{Key: "server_password", File: "engine", Prop: "OnlineSubsystem/ServerPassword",
				Label: "Пароль для входа", Section: SectionGeneral,
				Kind: KindString, MaxLen: 128, Secret: true, Clearable: true},
			{Key: "admin_password", File: "main", Prop: "ServerSettings/AdminPassword",
				Label: "Пароль администратора", Section: SectionAdmin,
				Kind: KindString, MaxLen: 128, Secret: true},

			{Key: "pvp", File: "main", Prop: "ServerSettings/PVPEnabled",
				Label: "Бой между игроками", Section: SectionGameplay,
				Kind: KindBool, True: "True", False: "False"},
			{Key: "max_nudity", File: "main", Prop: "ServerSettings/MaxNudity",
				Label: "Уровень откровенности", Section: SectionGameplay,
				Kind: KindEnum, Default: "0", Options: []Option{
					{Value: "0", Label: "Нет"},
					{Value: "1", Label: "Частичная"},
					{Value: "2", Label: "Полная"},
				}},
			{Key: "battleye", File: "main", Prop: "ServerSettings/IsBattlEyeEnabled",
				Label: "Защита BattlEye", Section: SectionPlayers,
				Kind: KindBool, True: "True", False: "False"},
			{Key: "server_region", File: "main", Prop: "ServerSettings/serverRegion",
				Label: "Регион", Section: SectionNetwork, Kind: KindEnum, Default: "1",
				Options: []Option{
					{Value: "0", Label: "Восточная Америка"},
					{Value: "1", Label: "Европа"},
					{Value: "2", Label: "Китай"},
					{Value: "3", Label: "Азия"},
				}},

			{Key: "xp_rate", File: "main", Prop: "ServerSettings/PlayerXPRateMultiplier",
				Label: "Множитель опыта", Section: SectionRates,
				Kind: KindFloat, Min: AtLeast(0.1), Max: AtMost(100), Default: "1.0"},
			{Key: "harvest_amount", File: "main", Prop: "ServerSettings/HarvestAmountMultiplier",
				Label: "Множитель добычи", Section: SectionRates,
				Kind: KindFloat, Min: AtLeast(0.1), Max: AtMost(100), Default: "1.0"},
			{Key: "item_conversion", File: "main", Prop: "ServerSettings/ItemConvertionMultiplier",
				Label: "Скорость крафта", Section: SectionRates,
				Kind: KindFloat, Min: AtLeast(0.1), Max: AtMost(100), Default: "1.0"},
			{Key: "stamina_cost", File: "main", Prop: "ServerSettings/StaminaCostMultiplier",
				Label: "Расход выносливости", Section: SectionRates,
				Kind: KindFloat, Min: AtLeast(0.1), Max: AtMost(10), Default: "1.0"},
			{Key: "day_cycle_speed", File: "main", Prop: "ServerSettings/DayCycleSpeedScale",
				Label: "Скорость смены суток", Section: SectionWorld,
				Kind: KindFloat, Min: AtLeast(0.1), Max: AtMost(10), Default: "1.0"},
			{Key: "npc_respawn", File: "main", Prop: "ServerSettings/NPCRespawnMultiplier",
				Label: "Скорость возрождения NPC", Section: SectionWorld,
				Kind: KindFloat, Min: AtLeast(0.1), Max: AtMost(10), Default: "1.0"},
		},
	})
}
