// Package gamesettings описывает настройки игровых серверов: какие у игры есть
// поля, куда они пишутся и что считается допустимым значением.
//
// Раньше это знание было размазано по трём независимым спискам: форма в панели
// (lib/game-settings/layouts.ts), фильтр «поддерживается ли игра»
// (lib/server-lifecycle.ts) и правила валидации в core-api. Списки никто не
// сверял, и они разошлись: у Minecraft бэкенд принимал 28 полей, а панель
// показывала 8, причём поле, которого не было в правилах, при сохранении молча
// выбрасывалось — с сообщением «Настройки сохранены».
//
// Здесь описание одно. Из него получаются и проверка значения, и схема формы,
// и ответ на вопрос, поддержана ли игра. Разойтись им больше негде.
package gamesettings

import "regexp"

// Format — как устроен файл конфигурации. От формата зависит пара «разобрать» и
// «применить изменения», но не набор полей: одна и та же игра может держать
// часть настроек в файле, а часть в аргументах запуска.
type Format string

const (
	FormatKVSpace    Format = "kv_space"   // `ключ значение` — server.cfg SA-MP, Commands.dat Unturned
	FormatSourceCfg  Format = "source_cfg" // `ключ "значение"` — GoldSrc и Source
	FormatProperties Format = "properties" // `ключ=значение` — server.properties, serverDZ.cfg
	FormatEnv        Format = "env"        // `КЛЮЧ=значение` — rust.env
	FormatINI        Format = "ini"        // `[Секция]` + `ключ=значение` — Unreal-игры
	FormatUEOption   Format = "ue_option"  // `OptionSettings=(a=1,b="x")` — Palworld
	FormatJSON       Format = "json"       // Factorio, V Rising, Enshrouded, SotF
	FormatXMLProps   Format = "xml_props"  // `<property name= value=/>` — 7 Days to Die
	FormatXMLTags    Format = "xml_tags"   // `<тег>значение</тег>` — MTA:SA
	FormatYAMLFlat   Format = "yaml_flat"  // построчный YAML — Empyrion
	FormatArgs       Format = "args"       // аргументы запуска, файла нет
	FormatUEQuery    Format = "ue_query"   // `Карта?Ключ=Значение` — Unreal-игры
)

// StartupPath — путь-заглушка для настроек, которые живут не в файле, а в строке
// аргументов запуска.
//
// Так устроен Valheim: файла у него нет вовсе. Раньше это был отдельный режим со
// своей веткой в обработчике; теперь это обычный «файл», и та же механика
// достаётся играм, где имя сессии и слоты задаются только аргументами — от ARK
// до Soulmask.
const StartupPath = "@startup"

// Kind — тип поля с точки зрения формы. Не путать с форматом файла: булево
// значение выглядит тумблером всегда, а в файл уедет как True, 1 или true —
// смотря что скажут поля True и False.
type Kind string

const (
	KindString Kind = "string"
	KindText   Kind = "text"
	KindInt    Kind = "int"
	KindFloat  Kind = "float"
	KindBool   Kind = "bool"
	KindEnum   Kind = "enum"
)

// Option — вариант для поля типа KindEnum. Label показывается человеку, Value
// уходит в файл.
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Field — одно настраиваемое значение.
type Field struct {
	// Key — имя поля в API. Стабильный контракт с панелью, snake_case. Не обязано
	// совпадать с именем в файле: у Minecraft ключ max_players, а в файле
	// max-players.
	Key string `json:"key"`

	// File — идентификатор файла из Profile.Files. Пусто означает первый файл
	// профиля.
	File string `json:"file,omitempty"`

	// Prop — имя в самом файле. Для форматов с вложенностью это путь:
	// "ServerSettings/PVPEnabled" для ini с секциями, "visibility.public" для
	// JSON. Для аргументов запуска — сам флаг, например "-name".
	Prop string `json:"-"`

	Label   string    `json:"label"`
	Hint    string    `json:"hint,omitempty"`
	Section SectionID `json:"section"`
	Kind    Kind      `json:"kind"`
	Options []Option  `json:"options,omitempty"`
	Default string    `json:"default,omitempty"`

	// Min и Max — указатели, потому что ноль здесь законная граница. В прежних
	// правилах они были обычными числами и проверялись как `rule.MinInt != 0`,
	// из-за чего нижняя граница «не меньше нуля» просто не работала: у
	// tf_bot_quota отрицательное значение проходило насквозь.
	Min *float64 `json:"min,omitempty"`
	Max *float64 `json:"max,omitempty"`

	MaxLen  int            `json:"max_len,omitempty"`
	Pattern *regexp.Regexp `json:"-"`

	// True и False — как булево записывается в этом конкретном файле. Panel
	// всегда присылает "true"/"false", а на диск уедет то, что понимает игра.
	True  string `json:"-"`
	False string `json:"-"`

	// AppliesLive — значение подхватывается без перезапуска. По умолчанию false:
	// подавляющее большинство конфигов читается один раз при старте, и честнее
	// предупредить лишний раз, чем умолчать. Флаг именно такой, а не Restart,
	// чтобы забытое поле по умолчанию требовало перезапуска, а не наоборот.
	AppliesLive bool `json:"applies_live,omitempty"`

	// ReadOnly — поле показывается, но не редактируется: порт назначает панель,
	// слоты приходят из тарифа.
	ReadOnly bool `json:"read_only,omitempty"`

	// Secret — значение не отдаётся наружу. Пароли RCON и администратора видел
	// любой, у кого есть право на просмотр настроек, хотя право на правку
	// отдельное.
	Secret bool `json:"secret,omitempty"`

	// Slots — поле отвечает за число игроков. При тарификации по слотам его
	// значение задаёт тариф, а не клиент.
	Slots bool `json:"slots,omitempty"`

	// Clearable — поле разрешено очищать. Для остальных пустая строка означает
	// «не трогать»: раньше форма отправляла все поля разом, пустое значение
	// оседало в базе и навсегда закрывало собой то, что реально лежит в конфиге.
	Clearable bool `json:"clearable,omitempty"`
}

// ConfigFile — файл, в котором живут настройки игры.
type ConfigFile struct {
	// ID — короткое имя для ссылок из Field.File и из адресов API: main, rcon,
	// engine, game, startup.
	ID string `json:"id"`

	// Path — путь относительно каталога данных сервера. Допускает подстановку
	// значения другого поля в фигурных скобках:
	// "Zomboid/Server/{servername}.ini".
	Path string `json:"path"`

	Format Format `json:"format"`
	Title  string `json:"title"`

	// Template — заготовка на случай, когда файла ещё нет. Большинство образов
	// конфиг не создают: он появляется только после первого запуска игры.
	Template string `json:"-"`

	// MaxBytes — предел для редактора. Ноль означает значение по умолчанию.
	// Нужен, чтобы распухший Game.ini не клал вкладку.
	MaxBytes int `json:"max_bytes,omitempty"`

	// RawOnly — файл доступен только в сыром виде, типизированных полей у него
	// нет. Таковы списки администраторов и банов.
	RawOnly bool `json:"raw_only,omitempty"`
}

// Profile — настройки одной игры.
type Profile struct {
	// Key — ключ игры из gamecatalog.
	Key string `json:"key"`

	// SharedBy — другие игры каталога, у которых настройки те же. Пять сборок
	// Minecraft Java отличаются загрузчиком, а не server.properties.
	SharedBy []string `json:"shared_by,omitempty"`

	Files  []ConfigFile `json:"files"`
	Fields []Field      `json:"fields"`

	// Note — пояснение в шапке вкладки. Место, где честно сказать про
	// особенность игры: у Satisfactory имя и пароль администратора задаются в
	// клиенте при подключении, и в панели их не будет никогда.
	Note string `json:"note,omitempty"`
}

// DefaultMaxBytes — предел содержимого для сырого редактора.
const DefaultMaxBytes = 256 << 10

// MaxFileBytes — предел файла с учётом умолчания.
func (f ConfigFile) MaxFileBytes() int {
	if f.MaxBytes > 0 {
		return f.MaxBytes
	}
	return DefaultMaxBytes
}

// IsStartup — настройки этого «файла» лежат в аргументах запуска.
func (f ConfigFile) IsStartup() bool {
	return f.Path == StartupPath
}

// FileByID возвращает файл профиля по идентификатору. Пустой идентификатор
// означает первый файл — так поле может не указывать File, когда файл один.
func (p Profile) FileByID(id string) (ConfigFile, bool) {
	if id == "" {
		if len(p.Files) == 0 {
			return ConfigFile{}, false
		}
		return p.Files[0], true
	}
	for _, f := range p.Files {
		if f.ID == id {
			return f, true
		}
	}
	return ConfigFile{}, false
}

// FieldByKey возвращает поле профиля по ключу API.
func (p Profile) FieldByKey(key string) (Field, bool) {
	for _, f := range p.Fields {
		if f.Key == key {
			return f, true
		}
	}
	return Field{}, false
}

// Games — все ключи каталога, которые обслуживает профиль.
func (p Profile) Games() []string {
	return append([]string{p.Key}, p.SharedBy...)
}

// BoolLiteral переводит значение формы в то, что понимает файл игры.
func (f Field) BoolLiteral(on bool) string {
	if on {
		if f.True != "" {
			return f.True
		}
		return "true"
	}
	if f.False != "" {
		return f.False
	}
	return "false"
}

func floatPtr(v float64) *float64 { return &v }

// Range — короткая запись границ для описания полей.
func Range(min, max float64) (*float64, *float64) {
	return floatPtr(min), floatPtr(max)
}

// AtLeast — только нижняя граница. Отдельная функция нужна как раз потому, что
// ноль здесь значимое значение и «не задано» им не выразить.
func AtLeast(min float64) *float64 { return floatPtr(min) }

// AtMost — только верхняя граница.
func AtMost(max float64) *float64 { return floatPtr(max) }
