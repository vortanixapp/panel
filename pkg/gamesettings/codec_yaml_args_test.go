package gamesettings

import (
	"reflect"
	"testing"
)

const empyrionYAML = `# конфигурация Empyrion
ServerConfig:
  Srv_Name: Мой сервер
  Srv_Password: ''
  Srv_MaxPlayers: 8
  Srv_Public: true
  # телнет для администрирования
  Tel_Enabled: false

GameConfig:
  GameName: Default
  Mode: Survival
`

func TestYAMLFlatParse(t *testing.T) {
	c, _ := CodecFor(FormatYAMLFlat)
	got := c.Parse(empyrionYAML)
	for k, want := range map[string]string{
		"ServerConfig/Srv_Name":       "Мой сервер",
		"ServerConfig/Srv_Password":   "",
		"ServerConfig/Srv_MaxPlayers": "8",
		"ServerConfig/Srv_Public":     "true",
		"ServerConfig/Tel_Enabled":    "false",
		"GameConfig/GameName":         "Default",
		"GameConfig/Mode":             "Survival",
	} {
		if got[k] != want {
			t.Errorf("%s = %q, ожидалось %q", k, got[k], want)
		}
	}
}

func TestYAMLFlatApplyKeepsCommentsAndIndent(t *testing.T) {
	c, _ := CodecFor(FormatYAMLFlat)
	out, err := c.Apply(empyrionYAML, map[string]string{
		"ServerConfig/Srv_Name":       "Новый сервер",
		"ServerConfig/Srv_MaxPlayers": "16",
		"GameConfig/Mode":             "Creative",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `# конфигурация Empyrion
ServerConfig:
  Srv_Name: Новый сервер
  Srv_Password: ''
  Srv_MaxPlayers: 16
  Srv_Public: true
  # телнет для администрирования
  Tel_Enabled: false

GameConfig:
  GameName: Default
  Mode: Creative
`
	checkLines(t, "empyrion", out, want)
}

func TestYAMLFlatQuotesWhenNeeded(t *testing.T) {
	c, _ := CodecFor(FormatYAMLFlat)
	// Двоеточие в значении без кавычек YAML прочитает как вложенный ключ.
	out, _ := c.Apply(empyrionYAML, map[string]string{"ServerConfig/Srv_Name": "Сервер: РУ"})
	if got := c.Parse(out)["ServerConfig/Srv_Name"]; got != "Сервер: РУ" {
		t.Errorf("значение с двоеточием поехало: %q\n%s", got, out)
	}
	// Значение, которое было в кавычках, в них и остаётся.
	out2, _ := c.Apply(empyrionYAML, map[string]string{"ServerConfig/Srv_Password": "секрет"})
	if got := c.Parse(out2)["ServerConfig/Srv_Password"]; got != "секрет" {
		t.Errorf("пароль поехал: %q", got)
	}
}

func TestYAMLFlatAppendsIntoSection(t *testing.T) {
	c, _ := CodecFor(FormatYAMLFlat)
	out, _ := c.Apply(empyrionYAML, map[string]string{"GameConfig/Seed": "12345"})
	if got := c.Parse(out)["GameConfig/Seed"]; got != "12345" {
		t.Errorf("новый ключ не записан: %q\n%s", got, out)
	}
	// Чужая секция не пострадала.
	if got := c.Parse(out)["ServerConfig/Srv_Name"]; got != "Мой сервер" {
		t.Errorf("соседняя секция поехала: %q", got)
	}
}

func TestYAMLFlatIdempotency(t *testing.T) {
	c, _ := CodecFor(FormatYAMLFlat)
	changes := map[string]string{"ServerConfig/Srv_Name": "Мир", "GameConfig/Seed": "7"}
	once, _ := c.Apply(empyrionYAML, changes)
	twice, _ := c.Apply(once, changes)
	if once != twice {
		t.Errorf("не идемпотентно:\n%q\n%q", once, twice)
	}
}

func TestArgsCodecParsesEveryFlag(t *testing.T) {
	c, _ := CodecFor(FormatArgs)
	got := c.Parse(`-name "Мой мир" -world Дом -crossplay -public 1`)
	want := map[string]string{
		"name": "Мой мир", "world": "Дом",
		// Переключатель без значения — «включено».
		"crossplay": "", "public": "1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("разобрано %#v\nожидалось %#v", got, want)
	}
}

func TestArgsCodecApplyRoundTrip(t *testing.T) {
	c, _ := CodecFor(FormatArgs)
	out, err := c.Apply(`-name Старое -world Дом`, map[string]string{"name": "Новое имя"})
	if err != nil {
		t.Fatal(err)
	}
	got := c.Parse(out)
	if got["name"] != "Новое имя" || got["world"] != "Дом" {
		t.Errorf("получено %#v из %q", got, out)
	}
}

func TestAllFormatsHaveCodec(t *testing.T) {
	// Формат без кодека означает игру, у которой настройки не читаются и не
	// пишутся, — и узнали бы мы об этом от клиента.
	for _, f := range []Format{
		FormatKVSpace, FormatSourceCfg, FormatProperties, FormatEnv,
		FormatINI, FormatUEOption, FormatJSON, FormatXMLProps,
		FormatXMLTags, FormatYAMLFlat, FormatArgs,
	} {
		if _, ok := CodecFor(f); !ok {
			t.Errorf("для формата %s нет кодека", f)
		}
	}
}

func TestUnknownFormatIsAnError(t *testing.T) {
	f := ConfigFile{ID: "main", Path: "x", Format: Format("выдуманный")}
	if _, err := ParseFile(f, "x"); err == nil {
		t.Error("ожидалась ошибка для неизвестного формата")
	}
	if _, err := ApplyFile(f, "x", map[string]string{"a": "b"}); err == nil {
		t.Error("ожидалась ошибка для неизвестного формата")
	}
}

func TestApplyFileUsesTemplateForEmptyContent(t *testing.T) {
	// Большинство образов конфиг не создают: он появляется только после первого
	// запуска игры. Заготовка избавляет от необходимости выдумывать структуру.
	f := ConfigFile{
		ID: "main", Path: "server.properties", Format: FormatProperties,
		Template: "#Minecraft server properties\nmotd=Vortanix\n",
	}
	out, err := ApplyFile(f, "", map[string]string{"max-players": "20"})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := ParseFile(f, out)
	if got["motd"] != "Vortanix" || got["max-players"] != "20" {
		t.Errorf("получено %#v из %q", got, out)
	}
}

func TestApplyFileWithoutChangesIsNoOp(t *testing.T) {
	f := ConfigFile{ID: "main", Path: "server.cfg", Format: FormatSourceCfg}
	const in = "hostname \"Сервер\"\n"
	out, err := ApplyFile(f, in, nil)
	if err != nil || out != in {
		t.Errorf("пустой набор изменений поменял файл: %q, %v", out, err)
	}
}
