package gamesettings

// Оставшиеся игры с Linux-сервером: Arma 3, Arma Reforger, Empyrion, Icarus,
// Mordhau, The Isle, Hytale, Space Engineers.
//
// Три из них получают профиль без типизированных полей — только правильные пути
// к файлам и объяснение почему. Выдумывать ключи, которых мы не проверяли на
// живом сервере, хуже, чем не показывать форму: поле, которое молча ничего не
// делает, выглядит работающим.

func init() {
	register(Profile{
		Key: "arma3",
		Files: []ConfigFile{
			{ID: "main", Path: "server.cfg", Format: FormatProperties, Title: "server.cfg"},
			{ID: "basic", Path: "basic.cfg", Format: FormatProperties, Title: "basic.cfg"},
			{ID: "startup", Path: StartupPath, Format: FormatArgs, Title: "Параметры запуска"},
		},
		Note: "Блоки class в server.cfg панель не трогает — правьте их во вкладке «Конфиг».",
		Fields: []Field{
			{Key: "hostname", File: "main", Prop: "hostname",
				Label: "Название сервера", Section: SectionGeneral, Kind: KindString, MaxLen: 128},
			{Key: "password", File: "main", Prop: "password",
				Label: "Пароль для входа", Section: SectionGeneral,
				Kind: KindString, MaxLen: 128, Secret: true, Clearable: true},
			{Key: "password_admin", File: "main", Prop: "passwordAdmin",
				Label: "Пароль администратора", Section: SectionAdmin,
				Kind: KindString, MaxLen: 128, Secret: true},
			{Key: "max_players", File: "main", Prop: "maxPlayers", Slots: true,
				Label: "Мест на сервере", Section: SectionGeneral,
				Kind: KindInt, Min: AtLeast(1), Max: AtMost(200)},
			{Key: "motd_interval", File: "main", Prop: "motdInterval",
				Label: "Интервал сообщений, с", Section: SectionGeneral,
				Kind: KindInt, Min: AtLeast(0), Max: AtMost(3600), Default: "5"},
			{Key: "verify_signatures", File: "main", Prop: "verifySignatures",
				Label: "Проверять подписи модов", Section: SectionPlayers,
				Kind: KindEnum, Default: "2", Options: []Option{
					{Value: "0", Label: "Не проверять"},
					{Value: "2", Label: "Проверять"},
				}},
			{Key: "battleye", File: "main", Prop: "BattlEye",
				Label: "Защита BattlEye", Section: SectionPlayers,
				Kind: KindBool, True: "1", False: "0", Default: "true"},
			{Key: "kick_duplicate", File: "main", Prop: "kickDuplicate",
				Label: "Выгонять при повторном входе", Section: SectionPlayers,
				Kind: KindBool, True: "1", False: "0", Default: "true"},
			{Key: "allowed_file_patching", File: "main", Prop: "allowedFilePatching",
				Label: "Разрешить изменённые файлы", Section: SectionPlayers,
				Kind: KindEnum, Default: "1", Options: []Option{
					{Value: "0", Label: "Запретить"},
					{Value: "1", Label: "Только заголовки"},
					{Value: "2", Label: "Разрешить"},
				}},
			{Key: "persistent", File: "main", Prop: "persistent",
				Label: "Сохранять миссию между сессиями", Section: SectionWorld,
				Kind: KindBool, True: "1", False: "0"},
			{Key: "vote_threshold", File: "main", Prop: "voteThreshold",
				Label: "Порог голосования", Section: SectionGameplay,
				Kind: KindFloat, Min: AtLeast(0.1), Max: AtMost(1), Default: "0.5"},
			{Key: "disable_von", File: "main", Prop: "disableVoN",
				Label: "Отключить голосовой чат", Section: SectionPlayers,
				Kind: KindBool, True: "1", False: "0"},

			{Key: "max_msg_send", File: "basic", Prop: "MaxMsgSend",
				Label: "Пакетов за кадр", Section: SectionPerformance,
				Kind: KindInt, Min: AtLeast(16), Max: AtMost(1024), Default: "128"},
			{Key: "min_bandwidth", File: "basic", Prop: "MinBandwidth",
				Label: "Минимальный канал, бит/с", Section: SectionPerformance,
				Kind: KindInt, Min: AtLeast(0), Max: AtMost(1073741824)},
			{Key: "max_bandwidth", File: "basic", Prop: "MaxBandwidth",
				Label: "Максимальный канал, бит/с", Section: SectionPerformance,
				Kind: KindInt, Min: AtLeast(0), Max: AtMost(2147483647)},
			{Key: "max_custom_file_size", File: "basic", Prop: "MaxCustomFileSize",
				Label: "Предел пользовательского файла, байт", Section: SectionPerformance,
				Kind: KindInt, Min: AtLeast(0), Max: AtMost(1073741824)},
		},
	})

	register(Profile{
		Key: "armaref",
		Files: []ConfigFile{
			{ID: "main", Path: "config.json", Format: FormatJSON, Title: "config.json",
				Template: "{\n}\n"},
			{ID: "startup", Path: StartupPath, Format: FormatArgs, Title: "Параметры запуска"},
		},
		Fields: []Field{
			{Key: "name", File: "main", Prop: "game.name",
				Label: "Название сервера", Section: SectionGeneral, Kind: KindString, MaxLen: 128},
			{Key: "password", File: "main", Prop: "game.password",
				Label: "Пароль для входа", Section: SectionGeneral,
				Kind: KindString, MaxLen: 128, Secret: true, Clearable: true},
			{Key: "password_admin", File: "main", Prop: "game.passwordAdmin",
				Label: "Пароль администратора", Section: SectionAdmin,
				Kind: KindString, MaxLen: 128, Secret: true},
			{Key: "max_players", File: "main", Prop: "game.maxPlayers", Slots: true,
				Label: "Мест на сервере", Section: SectionGeneral,
				Kind: KindInt, Min: AtLeast(1), Max: AtMost(128)},
			{Key: "scenario_id", File: "main", Prop: "game.scenarioId",
				Label: "Сценарий", Section: SectionWorld, Kind: KindString, MaxLen: 255,
				Hint: "Идентификатор вида {ECC61978EDCC2B5A}Missions/23_Campaign.conf"},
			{Key: "visible", File: "main", Prop: "game.visible",
				Label: "Показывать в общем списке", Section: SectionNetwork,
				Kind: KindBool, Default: "true"},
			{Key: "battleye", File: "main", Prop: "game.gameProperties.battlEye",
				Label: "Защита BattlEye", Section: SectionPlayers, Kind: KindBool, Default: "true"},
			{Key: "disable_third_person", File: "main", Prop: "game.gameProperties.disableThirdPerson",
				Label: "Только вид от первого лица", Section: SectionGameplay, Kind: KindBool},
			{Key: "von_disable_ui", File: "main", Prop: "game.gameProperties.VONDisableUI",
				Label: "Скрыть интерфейс голосового чата", Section: SectionPlayers, Kind: KindBool},
			{Key: "fast_validation", File: "main", Prop: "game.gameProperties.fastValidation",
				Label: "Быстрая проверка клиентов", Section: SectionPlayers,
				Kind: KindBool, Default: "true"},
			{Key: "server_max_view_distance", File: "main", Prop: "game.gameProperties.serverMaxViewDistance",
				Label: "Дальность прорисовки", Section: SectionPerformance,
				Kind: KindInt, Min: AtLeast(500), Max: AtMost(10000), Default: "1600"},
			{Key: "network_view_distance", File: "main", Prop: "game.gameProperties.networkViewDistance",
				Label: "Дальность синхронизации", Section: SectionPerformance,
				Kind: KindInt, Min: AtLeast(500), Max: AtMost(5000), Default: "1000"},
		},
	})

	register(Profile{
		Key: "empyrion",
		Files: []ConfigFile{
			{ID: "main", Path: "dedicated.yaml", Format: FormatYAMLFlat, Title: "dedicated.yaml"},
			{ID: "startup", Path: StartupPath, Format: FormatArgs, Title: "Параметры запуска"},
		},
		Fields: []Field{
			{Key: "srv_name", File: "main", Prop: "ServerConfig/Srv_Name",
				Label: "Название сервера", Section: SectionGeneral, Kind: KindString, MaxLen: 128},
			{Key: "srv_description", File: "main", Prop: "ServerConfig/Srv_Description",
				Label: "Описание", Section: SectionGeneral, Kind: KindText, MaxLen: 500, Clearable: true},
			{Key: "srv_password", File: "main", Prop: "ServerConfig/Srv_Password",
				Label: "Пароль для входа", Section: SectionGeneral,
				Kind: KindString, MaxLen: 128, Secret: true, Clearable: true},
			{Key: "srv_max_players", File: "main", Prop: "ServerConfig/Srv_MaxPlayers", Slots: true,
				Label: "Мест на сервере", Section: SectionGeneral,
				Kind: KindInt, Min: AtLeast(1), Max: AtMost(100)},
			{Key: "srv_public", File: "main", Prop: "ServerConfig/Srv_Public",
				Label: "Показывать в общем списке", Section: SectionNetwork,
				Kind: KindBool, Default: "true"},
			{Key: "eac_active", File: "main", Prop: "ServerConfig/EACActive",
				Label: "Защита EasyAntiCheat", Section: SectionPlayers, Kind: KindBool, Default: "true"},
			{Key: "tel_enabled", File: "main", Prop: "ServerConfig/Tel_Enabled",
				Label: "Управление по Telnet", Section: SectionAdmin, Kind: KindBool},
			{Key: "tel_pwd", File: "main", Prop: "ServerConfig/Tel_Pwd",
				Label: "Пароль Telnet", Section: SectionAdmin,
				Kind: KindString, MaxLen: 128, Secret: true, Clearable: true},
			{Key: "save_directory", File: "main", Prop: "ServerConfig/SaveDirectory",
				Label: "Каталог сохранений", Section: SectionWorld,
				Kind: KindString, MaxLen: 128, Default: "Saves"},
			{Key: "game_name", File: "main", Prop: "GameConfig/GameName",
				Label: "Имя игры", Section: SectionWorld, Kind: KindString, MaxLen: 128},
			{Key: "mode", File: "main", Prop: "GameConfig/Mode",
				Label: "Режим", Section: SectionGameplay, Kind: KindEnum, Default: "Survival",
				Options: []Option{
					{Value: "Survival", Label: "Выживание"},
					{Value: "Creative", Label: "Творческий"},
				}},
			{Key: "seed", File: "main", Prop: "GameConfig/Seed",
				Label: "Зерно генерации", Section: SectionWorld,
				Kind: KindString, MaxLen: 32, Clearable: true},
			{Key: "custom_scenario", File: "main", Prop: "GameConfig/CustomScenario",
				Label: "Сценарий", Section: SectionWorld,
				Kind: KindString, MaxLen: 128, Clearable: true},
		},
	})

	register(Profile{
		Key: "icarus",
		Files: []ConfigFile{
			{ID: "main", Path: "Icarus/Saved/Config/LinuxServer/ServerSettings.ini",
				Format: FormatINI, Title: "ServerSettings.ini"},
			{ID: "startup", Path: StartupPath, Format: FormatArgs, Title: "Параметры запуска"},
		},
		Note: "Файл настроек появляется после первого запуска сервера.",
		Fields: []Field{
			{Key: "session_name", File: "main", Prop: "/Script/Icarus.DedicatedServerSettings/SessionName",
				Label: "Название сервера", Section: SectionGeneral, Kind: KindString, MaxLen: 128},
			{Key: "join_password", File: "main", Prop: "/Script/Icarus.DedicatedServerSettings/JoinPassword",
				Label: "Пароль для входа", Section: SectionGeneral,
				Kind: KindString, MaxLen: 128, Secret: true, Clearable: true},
			{Key: "admin_password", File: "main", Prop: "/Script/Icarus.DedicatedServerSettings/AdminPassword",
				Label: "Пароль администратора", Section: SectionAdmin,
				Kind: KindString, MaxLen: 128, Secret: true},
			{Key: "max_players", File: "main", Prop: "/Script/Icarus.DedicatedServerSettings/MaxPlayers", Slots: true,
				Label: "Мест на сервере", Section: SectionGeneral,
				Kind: KindInt, Min: AtLeast(1), Max: AtMost(8)},
			{Key: "shutdown_if_not_joined", File: "main", Prop: "/Script/Icarus.DedicatedServerSettings/ShutdownIfNotJoinedFor",
				Label: "Выключать, если никто не зашёл, с", Section: SectionAdvanced,
				Kind: KindInt, Min: AtLeast(0), Max: AtMost(86400), Default: "300"},
			{Key: "shutdown_if_empty", File: "main", Prop: "/Script/Icarus.DedicatedServerSettings/ShutdownIfEmptyFor",
				Label: "Выключать при пустом сервере, с", Section: SectionAdvanced,
				Kind: KindInt, Min: AtLeast(0), Max: AtMost(86400), Default: "300"},
			{Key: "allow_non_admin_launch", File: "main", Prop: "/Script/Icarus.DedicatedServerSettings/AllowNonAdminsToLaunchProspects",
				Label: "Запуск экспедиций без прав администратора", Section: SectionGameplay,
				Kind: KindBool, True: "True", False: "False"},
			{Key: "allow_non_admin_delete", File: "main", Prop: "/Script/Icarus.DedicatedServerSettings/AllowNonAdminsToDeleteProspects",
				Label: "Удаление экспедиций без прав администратора", Section: SectionGameplay,
				Kind: KindBool, True: "True", False: "False"},
			{Key: "resume_prospect", File: "main", Prop: "/Script/Icarus.DedicatedServerSettings/ResumeProspect",
				Label: "Продолжать последнюю экспедицию", Section: SectionWorld,
				Kind: KindBool, True: "True", False: "False", Default: "true"},
			{Key: "last_prospect_name", File: "main", Prop: "/Script/Icarus.DedicatedServerSettings/LastProspectName",
				Label: "Имя экспедиции", Section: SectionWorld,
				Kind: KindString, MaxLen: 128, Clearable: true},
		},
	})

	register(Profile{
		Key: "mordhau",
		Files: []ConfigFile{
			{ID: "main", Path: "Mordhau/Saved/Config/LinuxServer/Game.ini",
				Format: FormatINI, Title: "Game.ini"},
			{ID: "engine", Path: "Mordhau/Saved/Config/LinuxServer/Engine.ini",
				Format: FormatINI, Title: "Engine.ini", RawOnly: true},
			{ID: "startup", Path: StartupPath, Format: FormatArgs, Title: "Параметры запуска"},
		},
		Note: "Файл настроек появляется после первого запуска сервера.",
		Fields: []Field{
			{Key: "server_name", File: "main", Prop: "/Script/Mordhau.MordhauGameMode/ServerName",
				Label: "Название сервера", Section: SectionGeneral, Kind: KindString, MaxLen: 128},
			{Key: "server_password", File: "main", Prop: "/Script/Mordhau.MordhauGameMode/ServerPassword",
				Label: "Пароль для входа", Section: SectionGeneral,
				Kind: KindString, MaxLen: 128, Secret: true, Clearable: true},
			{Key: "admin_password", File: "main", Prop: "/Script/Mordhau.MordhauGameMode/AdminPassword",
				Label: "Пароль администратора", Section: SectionAdmin,
				Kind: KindString, MaxLen: 128, Secret: true},
			{Key: "max_slots", File: "main", Prop: "/Script/Mordhau.MordhauGameMode/MaxSlots", Slots: true,
				Label: "Мест на сервере", Section: SectionGeneral,
				Kind: KindInt, Min: AtLeast(1), Max: AtMost(64)},
			{Key: "private_server", File: "main", Prop: "/Script/Mordhau.MordhauGameMode/bIsPrivateServer",
				Label: "Закрытый сервер", Section: SectionNetwork,
				Kind: KindBool, True: "True", False: "False"},
			{Key: "idle_kick_time", File: "main", Prop: "/Script/Mordhau.MordhauGameMode/IdleKickTime",
				Label: "Кик за бездействие, с", Section: SectionPlayers,
				Kind: KindInt, Min: AtLeast(0), Max: AtMost(3600), Default: "300"},
			{Key: "rcon_password", File: "main", Prop: "/Script/Mordhau.MordhauGameMode/RconPassword",
				Label: "Пароль RCON", Section: SectionAdmin,
				Kind: KindString, MaxLen: 128, Secret: true, Clearable: true},
			{Key: "rcon_port", File: "main", Prop: "/Script/Mordhau.MordhauGameMode/RconPort",
				Label: "Порт RCON", Section: SectionAdmin,
				Kind: KindInt, Min: AtLeast(1), Max: AtMost(65535), ReadOnly: true},
		},
	})

	// The Isle: ключи Game.ini у этой игры меняются от сборки к сборке, и по коду
	// репозитория их не проверить — образ конфиг не создаёт. Даём правильный путь
	// и сырой редактор; форму добавим, когда снимем ключи с живого сервера.
	// Показать поле, которое молча ничего не делает, было бы хуже.
	for _, key := range []string{"theisle", "isleevr"} {
		register(Profile{
			Key: key,
			Files: []ConfigFile{
				{ID: "main", Path: "TheIsle/Saved/Config/LinuxServer/Game.ini",
					Format: FormatINI, Title: "Game.ini"},
				{ID: "engine", Path: "TheIsle/Saved/Config/LinuxServer/Engine.ini",
					Format: FormatINI, Title: "Engine.ini", RawOnly: true},
				{ID: "startup", Path: StartupPath, Format: FormatArgs, Title: "Параметры запуска"},
			},
			Note: "Набор настроек The Isle заметно меняется между сборками, поэтому форма пока не описана — правьте Game.ini напрямую во вкладке «Конфиг».",
		})
	}

	register(Profile{
		Key: "hytale",
		Files: []ConfigFile{
			{ID: "startup", Path: StartupPath, Format: FormatArgs, Title: "Параметры запуска"},
		},
		Note: "Публичного выделенного сервера Hytale пока не существует, поэтому настраивать нечего.",
	})

	// Space Engineers: в каталоге игра значится как поддерживающая Linux, но её
	// собственная заметка об установке говорит, что выделенный сервер существует
	// только под Windows. Форму не описываем, пока это расхождение не решено.
	register(Profile{
		Key: "spaceeng",
		Files: []ConfigFile{
			{ID: "main", Path: "SpaceEngineers-Dedicated.cfg", Format: FormatXMLTags,
				Title: "SpaceEngineers-Dedicated.cfg"},
			{ID: "startup", Path: StartupPath, Format: FormatArgs, Title: "Параметры запуска"},
		},
		Note: "Выделенный сервер Space Engineers выпускается только под Windows — на Linux его работа не гарантирована.",
	})
}
