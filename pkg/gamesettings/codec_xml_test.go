package gamesettings

import (
	"strings"
	"testing"
)

const sevenDaysXMLFile = `<?xml version="1.0" encoding="UTF-8"?>
<ServerSettings>
    <!-- Конфигурация 7 Days to Die -->
    <property name="ServerName" value="Мой сервер"/>
    <property name="ServerPassword" value=""/>
    <property name="ServerMaxPlayerCount" value="8"/>
    <property name="GameWorld" value="Navezgane"/>
    <property name="HideCommandExecutionLog" value="0"/>
</ServerSettings>
`

func TestXMLPropsParse(t *testing.T) {
	c, _ := CodecFor(FormatXMLProps)
	got := c.Parse(sevenDaysXMLFile)
	for k, want := range map[string]string{
		"ServerName": "Мой сервер", "ServerPassword": "",
		"ServerMaxPlayerCount": "8", "GameWorld": "Navezgane",
		"HideCommandExecutionLog": "0",
	} {
		if got[k] != want {
			t.Errorf("%s = %q, ожидалось %q", k, got[k], want)
		}
	}
}

// Раньше файл пересобирался через xml.Marshal: пропадали декларация,
// комментарии и отступы, а корневой тег становился именем Go-типа
// <sevenDaysXML> — игра такой конфиг не читает и не стартует.
func TestXMLPropsApplyKeepsDocumentIntact(t *testing.T) {
	c, _ := CodecFor(FormatXMLProps)
	out, err := c.Apply(sevenDaysXMLFile, map[string]string{"ServerName": "Новое имя"})
	if err != nil {
		t.Fatal(err)
	}
	want := `<?xml version="1.0" encoding="UTF-8"?>
<ServerSettings>
    <!-- Конфигурация 7 Days to Die -->
    <property name="ServerName" value="Новое имя"/>
    <property name="ServerPassword" value=""/>
    <property name="ServerMaxPlayerCount" value="8"/>
    <property name="GameWorld" value="Navezgane"/>
    <property name="HideCommandExecutionLog" value="0"/>
</ServerSettings>
`
	checkLines(t, "7d2d", out, want)
	if strings.Contains(out, "sevenDaysXML") {
		t.Error("корень назван именем Go-типа")
	}
}

func TestXMLPropsAppendsWithSameIndent(t *testing.T) {
	c, _ := CodecFor(FormatXMLProps)
	out, _ := c.Apply(sevenDaysXMLFile, map[string]string{"WorldGenSeed": "зерно"})
	if !strings.Contains(out, `    <property name="WorldGenSeed" value="зерно"/>`) {
		t.Errorf("новое свойство без отступа или отсутствует:\n%s", out)
	}
	if !strings.Contains(out, "</ServerSettings>") {
		t.Error("закрывающий тег потерян")
	}
	if c.Parse(out)["ServerName"] != "Мой сервер" {
		t.Error("прежние свойства пострадали")
	}
}

// Обход nameMap шёл по карте, поэтому и порядок дописывания, и то, какая правка
// потеряется, менялись от запуска к запуску.
func TestXMLPropsApplyIsDeterministic(t *testing.T) {
	c, _ := CodecFor(FormatXMLProps)
	changes := map[string]string{
		"ServerName": "Имя", "GameName": "Игра", "WorldGenSeed": "зерно", "GameWorld": "Мир",
	}
	first, err := c.Apply(sevenDaysXMLFile, changes)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		out, _ := c.Apply(sevenDaysXMLFile, changes)
		if out != first {
			t.Fatalf("результат неустойчив на итерации %d:\n%q\n%q", i, first, out)
		}
	}
	got := c.Parse(first)
	for k, v := range changes {
		if got[k] != v {
			t.Errorf("%s = %q, ожидалось %q", k, got[k], v)
		}
	}
}

func TestXMLPropsIgnoresComments(t *testing.T) {
	c, _ := CodecFor(FormatXMLProps)
	in := `<ServerSettings>
    <!-- <property name="ServerName" value="из комментария"/> -->
    <property name="ServerName" value="настоящее"/>
</ServerSettings>
`
	if got := c.Parse(in)["ServerName"]; got != "настоящее" {
		t.Errorf("значение взято из комментария: %q", got)
	}
	out, _ := c.Apply(in, map[string]string{"ServerName": "новое"})
	if !strings.Contains(out, `<!-- <property name="ServerName" value="из комментария"/> -->`) {
		t.Errorf("комментарий пострадал:\n%s", out)
	}
	if got := c.Parse(out)["ServerName"]; got != "новое" {
		t.Errorf("после правки %q", got)
	}
}

func TestXMLPropsDuplicateFirstWins(t *testing.T) {
	c, _ := CodecFor(FormatXMLProps)
	in := `<ServerSettings><property name="ServerName" value="A"/><property name="ServerName" value="B"/></ServerSettings>`
	if got := c.Parse(in)["ServerName"]; got != "A" {
		t.Errorf("разбор взял %q, ожидалось первое вхождение A", got)
	}
	out, _ := c.Apply(in, map[string]string{"ServerName": "C"})
	// Записали в первое — читаем оттуда же. Раньше эти стороны расходились.
	if got := c.Parse(out)["ServerName"]; got != "C" {
		t.Errorf("записали C, прочиталось %q", got)
	}
}

func TestXMLPropsEscaping(t *testing.T) {
	c, _ := CodecFor(FormatXMLProps)
	out, _ := c.Apply(sevenDaysXMLFile, map[string]string{"ServerName": `Ка"вы"чки & <тег>`})
	if !strings.Contains(out, `value="Ка&quot;вы&quot;чки &amp; &lt;тег&gt;"`) {
		t.Errorf("экранирование неверно:\n%s", out)
	}
	if got := c.Parse(out)["ServerName"]; got != `Ка"вы"чки & <тег>` {
		t.Errorf("обратный разбор дал %q", got)
	}
}

func TestXMLPropsRejectsForeignRoot(t *testing.T) {
	c, _ := CodecFor(FormatXMLProps)
	// Чужой файл лучше не тронуть вовсе, чем принять за конфиг 7 Days to Die и
	// переписать поверх.
	if _, err := c.Apply(`<Whatever><property name="X" value="1"/></Whatever>`, map[string]string{"X": "2"}); err == nil {
		t.Fatal("ожидалась ошибка на чужом корне")
	}
}

const mtaConf = `<config>
    <!-- <password>из комментария</password> -->
    <servername>Мой сервер</servername>
    <maxplayers>32</maxplayers>
    <password></password>
    <fpslimit>36</fpslimit>
</config>
`

func TestXMLTagsParse(t *testing.T) {
	c, _ := CodecFor(FormatXMLTags)
	got := c.Parse(mtaConf)
	if got["servername"] != "Мой сервер" || got["maxplayers"] != "32" || got["fpslimit"] != "36" {
		t.Errorf("разобрано %#v", got)
	}
	// Комментарий не должен подменять настоящий тег.
	if got["password"] != "" {
		t.Errorf("password = %q, значение взято из комментария", got["password"])
	}
}

func TestXMLTagsApplyKeepsCommentsAndOrder(t *testing.T) {
	c, _ := CodecFor(FormatXMLTags)
	out, err := c.Apply(mtaConf, map[string]string{"servername": "Новое", "maxplayers": "64"})
	if err != nil {
		t.Fatal(err)
	}
	want := `<config>
    <!-- <password>из комментария</password> -->
    <servername>Новое</servername>
    <maxplayers>64</maxplayers>
    <password></password>
    <fpslimit>36</fpslimit>
</config>
`
	checkLines(t, "MTA", out, want)
}

func TestXMLTagsHandlesAttributes(t *testing.T) {
	c, _ := CodecFor(FormatXMLTags)
	// Прежний разбор регулярками тег с атрибутом просто не находил.
	in := `<config><servername lang="ru">Имя</servername></config>`
	if got := c.Parse(in)["servername"]; got != "Имя" {
		t.Errorf("тег с атрибутом не разобран: %q", got)
	}
	out, _ := c.Apply(in, map[string]string{"servername": "Другое"})
	if !strings.Contains(out, `<servername lang="ru">Другое</servername>`) {
		t.Errorf("атрибут потерян: %s", out)
	}
}

func TestXMLTagsAppendsBeforeClosing(t *testing.T) {
	c, _ := CodecFor(FormatXMLTags)
	out, _ := c.Apply(mtaConf, map[string]string{"httpport": "22005"})
	if !strings.Contains(out, "    <httpport>22005</httpport>\n</config>") {
		t.Errorf("новый тег дописан неверно:\n%s", out)
	}
}

func TestXMLTagsIdempotency(t *testing.T) {
	c, _ := CodecFor(FormatXMLTags)
	changes := map[string]string{"servername": "Мир", "httpport": "22005"}
	once, _ := c.Apply(mtaConf, changes)
	twice, _ := c.Apply(once, changes)
	if once != twice {
		t.Errorf("не идемпотентно:\n%q\n%q", once, twice)
	}
}
