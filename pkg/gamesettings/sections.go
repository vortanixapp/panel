package gamesettings

// Разделы вкладки «Настройки».
//
// Словарь закрытый: профиль игры не может завести свой раздел, это проверяется
// тестом. Смысл ограничения в том, чтобы одинаковое по смыслу называлось
// одинаково — «Мир» у Minecraft и «Мир» у Rust должны быть одним разделом, а не
// двумя похожими надписями. Порядок разделов в панели задаётся здесь, а не
// порядком полей в профиле: у всех игр вкладки идут в одной последовательности.
type SectionID string

const (
	SectionGeneral     SectionID = "general"
	SectionGameplay    SectionID = "gameplay"
	SectionWorld       SectionID = "world"
	SectionRates       SectionID = "rates"
	SectionPlayers     SectionID = "players"
	SectionAdmin       SectionID = "admin"
	SectionNetwork     SectionID = "network"
	SectionPerformance SectionID = "performance"
	SectionSaves       SectionID = "saves"
	SectionMods        SectionID = "mods"
	SectionAdvanced    SectionID = "advanced"

	// Системные разделы. Их не объявляет ни один профиль — панель добавляет их
	// сама: «Конфиг» показывает объявленные профилем файлы как есть, «Параметры
	// запуска» есть у любой игры, даже у той, для которой полей ещё не описано.
	SectionRaw     SectionID = "raw"
	SectionStartup SectionID = "startup"
)

// Section — раздел с человеческим заголовком.
type Section struct {
	ID    SectionID `json:"id"`
	Title string    `json:"title"`
	Hint  string    `json:"hint,omitempty"`
}

// sectionOrder задаёт и состав словаря, и порядок вкладок.
var sectionOrder = []Section{
	{ID: SectionGeneral, Title: "Основное", Hint: "Имя сервера, пароль входа, число мест"},
	{ID: SectionGameplay, Title: "Игровой процесс", Hint: "Режим, сложность, правила боя"},
	{ID: SectionWorld, Title: "Мир", Hint: "Карта, зерно генерации, ход времени"},
	{ID: SectionRates, Title: "Рейты и прогресс", Hint: "Множители опыта, добычи и крафта"},
	{ID: SectionPlayers, Title: "Игроки и доступ", Hint: "Белый список, проверка подлинности, автокик"},
	{ID: SectionAdmin, Title: "Администрирование", Hint: "Пароль администратора, RCON, телнет"},
	{ID: SectionNetwork, Title: "Сеть", Hint: "Видимость в списке серверов, служебные порты"},
	{ID: SectionPerformance, Title: "Производительность", Hint: "Дальность прорисовки, частота тиков, потоки"},
	{ID: SectionSaves, Title: "Сохранения", Hint: "Частота автосохранения и число копий"},
	{ID: SectionMods, Title: "Моды", Hint: "Подключённые модификации"},
	{ID: SectionAdvanced, Title: "Дополнительно"},
	{ID: SectionRaw, Title: "Конфиг", Hint: "Правка файлов настроек напрямую"},
	{ID: SectionStartup, Title: "Параметры запуска", Hint: "Аргументы командной строки сервера"},
}

var sectionByID = func() map[SectionID]Section {
	m := make(map[SectionID]Section, len(sectionOrder))
	for _, s := range sectionOrder {
		m[s.ID] = s
	}
	return m
}()

// Sections возвращает словарь разделов в порядке показа.
func Sections() []Section {
	out := make([]Section, len(sectionOrder))
	copy(out, sectionOrder)
	return out
}

// SectionKnown сообщает, есть ли такой раздел в словаре.
func SectionKnown(id SectionID) bool {
	_, ok := sectionByID[id]
	return ok
}

// SectionTitle возвращает заголовок раздела. Для неизвестного отдаёт сам
// идентификатор: тест не даст такому просочиться, а падать из-за надписи глупо.
func SectionTitle(id SectionID) string {
	if s, ok := sectionByID[id]; ok {
		return s.Title
	}
	return string(id)
}

// SectionRank — позиция раздела в порядке показа. Профилю не нужно думать о
// порядке: он перечисляет поля как удобно, а раскладывает их панель.
func SectionRank(id SectionID) int {
	for i, s := range sectionOrder {
		if s.ID == id {
			return i
		}
	}
	return len(sectionOrder)
}
