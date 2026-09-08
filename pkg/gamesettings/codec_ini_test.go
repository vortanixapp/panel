package gamesettings

import "testing"

const pzIni = `# Project Zomboid server
PublicName=Мой сервер
PublicDescription=Добро пожаловать
MaxPlayers=32
PVP=true
ServerWelcomeMessage=Привет! a=b=c
`

func TestINIWithoutSections(t *testing.T) {
	c, _ := CodecFor(FormatINI)
	got := c.Parse(pzIni)
	if got["PublicName"] != "Мой сервер" {
		t.Errorf("PublicName = %q", got["PublicName"])
	}
	// Режем по первому знаку равенства: приветствие вполне содержит «=».
	if got["ServerWelcomeMessage"] != "Привет! a=b=c" {
		t.Errorf("ServerWelcomeMessage = %q", got["ServerWelcomeMessage"])
	}

	out, err := c.Apply(pzIni, map[string]string{"MaxPlayers": "64", "PVP": "false"})
	if err != nil {
		t.Fatal(err)
	}
	want := `# Project Zomboid server
PublicName=Мой сервер
PublicDescription=Добро пожаловать
MaxPlayers=64
PVP=false
ServerWelcomeMessage=Привет! a=b=c
`
	checkLines(t, "ini без секций", out, want)
}

const arkIni = `[/Script/Engine.GameSession]
MaxPlayers=70

[ServerSettings]
ServerPassword=секрет
ServerAdminPassword=админ
XPMultiplier=1.0
; комментарий
DifficultyOffset=0.2
`

func TestINISectionKeysUseLastSlash(t *testing.T) {
	c, _ := CodecFor(FormatINI)
	got := c.Parse(arkIni)
	// Секции Unreal сами содержат косые черты, поэтому имя ключа отделяется
	// по последней.
	if got["/Script/Engine.GameSession/MaxPlayers"] != "70" {
		t.Errorf("MaxPlayers = %q", got["/Script/Engine.GameSession/MaxPlayers"])
	}
	if got["ServerSettings/XPMultiplier"] != "1.0" {
		t.Errorf("XPMultiplier = %q", got["ServerSettings/XPMultiplier"])
	}

	section, key := splitINIKey("/Script/Engine.GameSession/MaxPlayers")
	if section != "/Script/Engine.GameSession" || key != "MaxPlayers" {
		t.Errorf("разбор ключа дал %q и %q", section, key)
	}
}

func TestINIApplyKeepsSectionsAndComments(t *testing.T) {
	c, _ := CodecFor(FormatINI)
	out, err := c.Apply(arkIni, map[string]string{
		"ServerSettings/XPMultiplier":           "3.0",
		"/Script/Engine.GameSession/MaxPlayers": "100",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `[/Script/Engine.GameSession]
MaxPlayers=100

[ServerSettings]
ServerPassword=секрет
ServerAdminPassword=админ
XPMultiplier=3.0
; комментарий
DifficultyOffset=0.2
`
	checkLines(t, "ini с секциями", out, want)
}

func TestINIAppendsIntoOwnSection(t *testing.T) {
	c, _ := CodecFor(FormatINI)
	// Новый ключ обязан попасть внутрь своей секции, а не в конец файла под
	// чужой заголовок.
	out, _ := c.Apply(arkIni, map[string]string{"/Script/Engine.GameSession/bAllow": "True"})
	want := `[/Script/Engine.GameSession]
MaxPlayers=70
bAllow=True

[ServerSettings]
ServerPassword=секрет
ServerAdminPassword=админ
XPMultiplier=1.0
; комментарий
DifficultyOffset=0.2
`
	checkLines(t, "дописывание в секцию", out, want)
}

func TestINICreatesMissingSection(t *testing.T) {
	c, _ := CodecFor(FormatINI)
	out, _ := c.Apply(arkIni, map[string]string{"NewSection/Key": "1"})
	if got := c.Parse(out)["NewSection/Key"]; got != "1" {
		t.Errorf("новая секция не создана: %q\n%s", got, out)
	}
	// Прежние секции не пострадали.
	if got := c.Parse(out)["ServerSettings/XPMultiplier"]; got != "1.0" {
		t.Errorf("старое значение поехало: %q", got)
	}
}

func TestINIRoundTripAndIdempotency(t *testing.T) {
	c, _ := CodecFor(FormatINI)
	changes := map[string]string{
		"ServerSettings/ServerPassword": "новый",
		"ServerSettings/NewKey":         "значение",
	}
	once, _ := c.Apply(arkIni, changes)
	twice, _ := c.Apply(once, changes)
	if once != twice {
		t.Errorf("не идемпотентно:\n%q\n%q", once, twice)
	}
	got := c.Parse(once)
	for k, v := range changes {
		if got[k] != v {
			t.Errorf("%s = %q, ожидалось %q", k, got[k], v)
		}
	}
}

const palworldIni = `[/Script/Pal.PalGameWorldSettings]
OptionSettings=(Difficulty=None,DayTimeSpeedRate=1.000000,ServerName="Мой сервер",ServerPassword="",AdminPassword="x",ServerPlayerMaxNum=32,bIsPvP=False,DeathPenalty=All)
`

func TestUEOptionParse(t *testing.T) {
	c, _ := CodecFor(FormatUEOption)
	got := c.Parse(palworldIni)
	for k, want := range map[string]string{
		"Difficulty": "None", "DayTimeSpeedRate": "1.000000",
		"ServerName": "Мой сервер", "ServerPassword": "", "AdminPassword": "x",
		"ServerPlayerMaxNum": "32", "bIsPvP": "False", "DeathPenalty": "All",
	} {
		if got[k] != want {
			t.Errorf("%s = %q, ожидалось %q", k, got[k], want)
		}
	}
}

// Прежняя регулярка `OptionSettings=\(([^)]+)\)` обрывалась на первой
// закрывающей скобке, и имя сервода со скобкой ломало разбор и запись.
func TestUEOptionHandlesParenthesesInValue(t *testing.T) {
	c, _ := CodecFor(FormatUEOption)
	in := `OptionSettings=(ServerName="Сервер (RU)",ServerPlayerMaxNum=32,bIsPvP=True)` + "\n"
	got := c.Parse(in)
	if got["ServerName"] != "Сервер (RU)" {
		t.Errorf("ServerName = %q", got["ServerName"])
	}
	// Параметры после скобки не должны пропадать.
	if got["ServerPlayerMaxNum"] != "32" || got["bIsPvP"] != "True" {
		t.Errorf("параметры после скобки потеряны: %#v", got)
	}

	out, err := c.Apply(in, map[string]string{"ServerPlayerMaxNum": "16"})
	if err != nil {
		t.Fatal(err)
	}
	again := c.Parse(out)
	if again["ServerName"] != "Сервер (RU)" || again["ServerPlayerMaxNum"] != "16" {
		t.Errorf("после правки %#v\n%s", again, out)
	}
}

func TestUEOptionApplyKeepsOtherParams(t *testing.T) {
	c, _ := CodecFor(FormatUEOption)
	out, err := c.Apply(palworldIni, map[string]string{
		"ServerName":         "Новое имя",
		"ServerPlayerMaxNum": "16",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `[/Script/Pal.PalGameWorldSettings]
OptionSettings=(Difficulty=None,DayTimeSpeedRate=1.000000,ServerName="Новое имя",ServerPassword="",AdminPassword="x",ServerPlayerMaxNum=16,bIsPvP=False,DeathPenalty=All)
`
	checkLines(t, "Palworld", out, want)
}

// Пустой файл раньше приводил к тому, что выписывались ВСЕ известные ключи со
// значениями по умолчанию — нулями. Palworld ждёт там True/False, а ExpRate=0
// останавливает прогресс: сервер поднимался с настройками, которых никто не
// задавал.
func TestUEOptionEmptyContentWritesOnlyRequested(t *testing.T) {
	c, _ := CodecFor(FormatUEOption)
	out, err := c.Apply("", map[string]string{"ServerName": "Новый", "bIsPvP": "True"})
	if err != nil {
		t.Fatal(err)
	}
	got := c.Parse(out)
	if len(got) != 2 {
		t.Errorf("записано %d параметров, ожидалось 2: %#v", len(got), got)
	}
	if got["ServerName"] != "Новый" || got["bIsPvP"] != "True" {
		t.Errorf("получено %#v", got)
	}
	for _, forbidden := range []string{"ExpRate", "bUseAuth", "DeathPenalty"} {
		if _, bad := got[forbidden]; bad {
			t.Errorf("выдуман параметр %s, которого не просили", forbidden)
		}
	}
}

func TestUEOptionQuotingFollowsSource(t *testing.T) {
	c, _ := CodecFor(FormatUEOption)
	// Строка была в кавычках — остаётся в кавычках; число было без — остаётся без.
	out, _ := c.Apply(palworldIni, map[string]string{
		"ServerName":         "Другое",
		"ServerPlayerMaxNum": "8",
	})
	if !contains(out, `ServerName="Другое"`) {
		t.Errorf("строка потеряла кавычки: %s", out)
	}
	if !contains(out, `ServerPlayerMaxNum=8`) {
		t.Errorf("число получило кавычки: %s", out)
	}
}

func TestUEOptionIdempotency(t *testing.T) {
	c, _ := CodecFor(FormatUEOption)
	changes := map[string]string{"ServerName": "Мир", "NewOption": "1"}
	once, _ := c.Apply(palworldIni, changes)
	twice, _ := c.Apply(once, changes)
	if once != twice {
		t.Errorf("не идемпотентно:\n%q\n%q", once, twice)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
