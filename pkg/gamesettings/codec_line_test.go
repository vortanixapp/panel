package gamesettings

import (
	"reflect"
	"strings"
	"testing"
)

// checkLines сравнивает результат построчно — так видно, какая именно строка
// конфига поехала, а не «две простыни не совпали».
func checkLines(t *testing.T, label, got, want string) {
	t.Helper()
	if got == want {
		return
	}
	g := strings.Split(got, "\n")
	w := strings.Split(want, "\n")
	n := len(g)
	if len(w) > n {
		n = len(w)
	}
	for i := 0; i < n; i++ {
		gl, wl := "<нет строки>", "<нет строки>"
		if i < len(g) {
			gl = g[i]
		}
		if i < len(w) {
			wl = w[i]
		}
		if gl != wl {
			t.Errorf("%s: строка %d: ожидалось %q, получено %q", label, i+1, wl, gl)
		}
	}
	if len(g) != len(w) {
		t.Errorf("%s: строк %d, ожидалось %d", label, len(g), len(w))
	}
}

const sampCfg = `echo Executing Server Config...
lanmode 0
rcon_password changeme
maxplayers 50
hostname SA-MP 0.3 Server
; адрес сайта проекта
weburl www.sa-mp.com

# сетевые настройки
onfoot_rate 40
password
`

func TestKVSpaceParse(t *testing.T) {
	c, _ := CodecFor(FormatKVSpace)
	got := c.Parse(sampCfg)
	want := map[string]string{
		"echo":          "Executing Server Config...",
		"lanmode":       "0",
		"rcon_password": "changeme",
		"maxplayers":    "50",
		"hostname":      "SA-MP 0.3 Server",
		"weburl":        "www.sa-mp.com",
		"onfoot_rate":   "40",
		// Ключ без значения — законная запись: пустой пароль. Прежняя регулярка
		// требовала пробел после ключа, и такая строка была не видна вовсе.
		"password": "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("разобрано %#v\nожидалось %#v", got, want)
	}
}

func TestKVSpaceStripsInlineComment(t *testing.T) {
	c, _ := CodecFor(FormatKVSpace)
	// Хвостовой комментарий попадал в значение: «50 ; слоты» вместо «50».
	got := c.Parse("maxplayers 50 ; слоты\n")
	if got["maxplayers"] != "50" {
		t.Errorf("значение %q, ожидалось 50", got["maxplayers"])
	}
	// И при этом обязан пережить правку — он объясняет именно эту строку.
	out, err := c.Apply("maxplayers 50 ; слоты\n", map[string]string{"maxplayers": "100"})
	if err != nil {
		t.Fatal(err)
	}
	checkLines(t, "хвостовой комментарий", out, "maxplayers 100 ; слоты\n")
}

func TestKVSpaceApplyKeepsEverythingElse(t *testing.T) {
	c, _ := CodecFor(FormatKVSpace)
	out, err := c.Apply(sampCfg, map[string]string{
		"hostname":   "Мой сервер | RP",
		"maxplayers": "100",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `echo Executing Server Config...
lanmode 0
rcon_password changeme
maxplayers 100
hostname Мой сервер | RP
; адрес сайта проекта
weburl www.sa-mp.com

# сетевые настройки
onfoot_rate 40
password
`
	checkLines(t, "правка на месте", out, want)
}

func TestKVSpaceAppendsMissingWithoutBlankLines(t *testing.T) {
	c, _ := CodecFor(FormatKVSpace)
	out, _ := c.Apply("hostname Сервер\n", map[string]string{"worldtime": "12", "announce": "1"})
	// Порядок дописывания устойчивый, лишних пустых строк не появляется.
	want := "hostname Сервер\nannounce 1\nworldtime 12\n"
	checkLines(t, "дописывание", out, want)

	for i := 0; i < 20; i++ {
		again, _ := c.Apply("hostname Сервер\n", map[string]string{"worldtime": "12", "announce": "1"})
		if again != out {
			t.Fatalf("порядок неустойчив:\n%q\n%q", out, again)
		}
	}
}

func TestKVSpaceRoundTripAndIdempotency(t *testing.T) {
	c, _ := CodecFor(FormatKVSpace)
	changes := map[string]string{"hostname": "Мой сервер", "maxplayers": "64", "announce": "1"}

	once, err := c.Apply(sampCfg, changes)
	if err != nil {
		t.Fatal(err)
	}
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

func TestKVSpaceKeyMatchIgnoresCaseKeepsSpelling(t *testing.T) {
	c, _ := CodecFor(FormatKVSpace)
	out, _ := c.Apply("HostName Старое\n", map[string]string{"hostname": "Новое"})
	// Написание ключа в файле не наше дело — меняем только значение.
	checkLines(t, "регистр ключа", out, "HostName Новое\n")
}

func TestSourceCfgQuoting(t *testing.T) {
	c, _ := CodecFor(FormatSourceCfg)
	in := `hostname "Старое имя"
mp_timelimit 30
sv_downloadurl "http://fastdl.example.com/"
// комментарий
sv_password ""
`
	got := c.Parse(in)
	if got["hostname"] != "Старое имя" {
		t.Errorf("hostname = %q", got["hostname"])
	}
	if got["mp_timelimit"] != "30" {
		t.Errorf("mp_timelimit = %q", got["mp_timelimit"])
	}
	// Двойная косая внутри адреса не комментарий: перед ней нет пробела.
	if got["sv_downloadurl"] != "http://fastdl.example.com/" {
		t.Errorf("sv_downloadurl = %q", got["sv_downloadurl"])
	}
	if got["sv_password"] != "" {
		t.Errorf("sv_password = %q", got["sv_password"])
	}

	out, _ := c.Apply(in, map[string]string{
		"hostname":     "Новое имя",
		"mp_timelimit": "45",
		"sv_password":  "секрет",
	})
	want := `hostname "Новое имя"
mp_timelimit 45
sv_downloadurl "http://fastdl.example.com/"
// комментарий
sv_password "секрет"
`
	checkLines(t, "конфиг Source", out, want)
}

func TestSourceCfgInlineCommentNotConfusedWithURL(t *testing.T) {
	c, _ := CodecFor(FormatSourceCfg)
	got := c.Parse("sv_downloadurl http://x/y // быстрая загрузка\n")
	if got["sv_downloadurl"] != "http://x/y" {
		t.Errorf("значение %q", got["sv_downloadurl"])
	}
}

func TestSourceCfgClearsValue(t *testing.T) {
	c, _ := CodecFor(FormatSourceCfg)
	// Пустое значение раньше пропускалось при записи, и снять пароль было нельзя.
	out, _ := c.Apply(`sv_password "секрет"`+"\n", map[string]string{"sv_password": ""})
	checkLines(t, "очистка пароля", out, `sv_password ""`+"\n")
	if v, ok := c.Parse(out)["sv_password"]; !ok || v != "" {
		t.Errorf("после очистки %q, %v", v, ok)
	}
}

func TestPropertiesKeepsPhysicalKeys(t *testing.T) {
	c, _ := CodecFor(FormatProperties)
	in := `#Minecraft server properties
#Mon Sep 01 12:00:00 UTC 2026
motd=Vortanix
max-players=20
rcon.password=секрет
query.port=25565
level-seed=
`
	got := c.Parse(in)
	// Ключи ровно как в файле: никаких max_players и rcon_password.
	for k, want := range map[string]string{
		"motd": "Vortanix", "max-players": "20",
		"rcon.password": "секрет", "query.port": "25565", "level-seed": "",
	} {
		if got[k] != want {
			t.Errorf("%s = %q, ожидалось %q", k, got[k], want)
		}
	}
	if _, bad := got["max_players"]; bad {
		t.Error("кодек не должен переводить имена ключей")
	}

	out, _ := c.Apply(in, map[string]string{"max-players": "64", "motd": ""})
	want := `#Minecraft server properties
#Mon Sep 01 12:00:00 UTC 2026
motd=
max-players=64
rcon.password=секрет
query.port=25565
level-seed=
`
	checkLines(t, "server.properties", out, want)
}

func TestPropertiesValueMayContainEquals(t *testing.T) {
	c, _ := CodecFor(FormatProperties)
	// Режем по первому знаку равенства: приветствие вполне может содержать «=».
	got := c.Parse("motd=a=b=c\n")
	if got["motd"] != "a=b=c" {
		t.Errorf("motd = %q", got["motd"])
	}
}

func TestEnvKeepsCommentsAndOrder(t *testing.T) {
	c, _ := CodecFor(FormatEnv)
	in := `# настройки Rust
RUST_HOSTNAME=Мой сервер
RUST_MAXPLAYERS=100
RUST_SEED=12345
`
	// Прежний код пересобирал rust.env целиком: комментарии и порядок терялись.
	out, _ := c.Apply(in, map[string]string{"RUST_MAXPLAYERS": "200"})
	want := `# настройки Rust
RUST_HOSTNAME=Мой сервер
RUST_MAXPLAYERS=200
RUST_SEED=12345
`
	checkLines(t, "rust.env", out, want)
}

func TestLineCodecsPreserveCRLF(t *testing.T) {
	c, _ := CodecFor(FormatProperties)
	out, _ := c.Apply("motd=Старое\r\nmax-players=20\r\n", map[string]string{"motd": "Новое"})
	if !strings.Contains(out, "\r\n") {
		t.Errorf("перевод строки Windows потерян: %q", out)
	}
	if strings.Contains(strings.ReplaceAll(out, "\r\n", ""), "\r") {
		t.Errorf("остались одиночные возвраты каретки: %q", out)
	}
}

func TestLineCodecsEmptyInput(t *testing.T) {
	for _, f := range []Format{FormatKVSpace, FormatSourceCfg, FormatProperties, FormatEnv} {
		c, _ := CodecFor(f)
		if got := c.Parse(""); len(got) != 0 {
			t.Errorf("%s: на пустом входе разобрано %#v", f, got)
		}
		out, err := c.Apply("", map[string]string{"key": "значение"})
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		if v := c.Parse(out)["key"]; v != "значение" {
			t.Errorf("%s: после записи в пустой файл получено %q", f, v)
		}
	}
}
