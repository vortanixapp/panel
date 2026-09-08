package gamesettings

import (
	"fmt"
	"sort"
	"testing"

	"github.com/vortanix/vortanix/pkg/gamecatalog"
)

// notYetCovered — игры с Linux-сервером, для которых профиль ещё не написан.
//
// Список существует, чтобы незакрытая работа была видна в коде, а не только в
// голове. Он должен ТОЛЬКО СОКРАЩАТЬСЯ: добавили профиль — убрали строку.
// Добавлять сюда новую игру допустимо лишь как осознанное «пока не сделано»,
// и тест ниже следит, чтобы запись нельзя было забыть удалить.
// Список пуст: профиль есть у каждой игры каталога с Linux-сервером. Оставлен
// как есть, чтобы новую игру можно было завести с честной пометкой «пока не
// сделано», а не молча без настроек.
var notYetCovered = map[string]string{}

// untypedOnPurpose — игры, у которых есть профиль с путями к файлам, но нет
// описанной формы. Это осознанное решение, а не забытая работа.
//
// Показать поле, которое молча ничего не делает, хуже, чем показать сырой
// редактор: клиент считает настройку применённой и не понимает, почему сервер
// ведёт себя иначе.
var untypedOnPurpose = map[string]string{
	"theisle":  "набор ключей Game.ini заметно меняется между сборками",
	"isleevr":  "набор ключей Game.ini заметно меняется между сборками",
	"hytale":   "публичного выделенного сервера не существует",
	"spaceeng": "выделенный сервер выпускается только под Windows",
}

// У игры либо есть описанная форма, либо есть запись, объясняющая почему её нет.
func TestUntypedProfilesAreExplained(t *testing.T) {
	for _, p := range All() {
		if len(p.Fields) > 0 {
			continue
		}
		if _, known := untypedOnPurpose[p.Key]; !known {
			t.Errorf("у профиля %s нет полей и нет объяснения в untypedOnPurpose", p.Key)
		}
		if p.Note == "" {
			t.Errorf("профиль %s без полей обязан объяснять это клиенту в Note", p.Key)
		}
	}
	for key := range untypedOnPurpose {
		if Typed(key) {
			t.Errorf("у %s появилась форма — уберите из untypedOnPurpose", key)
		}
	}
}

// Все игры, влезающие в малую ноду, обязаны иметь типизированные поля: это те
// двадцать, что включены у клиентов по умолчанию, и оставлять их на голом
// редакторе файлов нельзя.
func TestSmallNodeGamesAreTyped(t *testing.T) {
	missing := []string{}
	for _, g := range gamecatalog.All() {
		if !gamecatalog.FitsSmallNode(g.Key) || !gamecatalog.LinuxSupported(g.Key) {
			continue
		}
		if !Typed(g.Key) {
			missing = append(missing, g.Key)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("нет описанных полей: %v", missing)
	}
}

// Каждая игра каталога с Linux-сервером должна иметь профиль либо честную
// запись в notYetCovered. Иначе новая игра тихо появилась бы в каталоге без
// настроек, и узнали бы об этом от клиента.
func TestEveryLinuxGameHasProfileOrKnownGap(t *testing.T) {
	missing := []string{}
	for _, g := range gamecatalog.All() {
		if !gamecatalog.LinuxSupported(g.Key) {
			continue
		}
		if Supported(g.Key) {
			continue
		}
		if _, known := notYetCovered[g.Key]; known {
			continue
		}
		missing = append(missing, g.Key)
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("нет ни профиля, ни записи в notYetCovered: %v", missing)
	}
}

// Запись в notYetCovered обязана устаревать вместе с работой: как только у игры
// появился профиль, строка должна исчезнуть. Иначе список превратится в
// декорацию и перестанет показывать реальный остаток.
func TestNotYetCoveredHasNoStaleEntries(t *testing.T) {
	stale := []string{}
	for key := range notYetCovered {
		if Supported(key) {
			stale = append(stale, key)
		}
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		t.Fatalf("профиль уже есть, уберите из notYetCovered: %v", stale)
	}
}

// В notYetCovered не должно быть игр, которых нет в каталоге или у которых нет
// Linux-сервера: первое — опечатка, второе — лишняя работа.
func TestNotYetCoveredMatchesCatalog(t *testing.T) {
	for key := range notYetCovered {
		if _, ok := gamecatalog.Resolve(key); !ok {
			t.Errorf("игры %q нет в каталоге", key)
			continue
		}
		if !gamecatalog.LinuxSupported(key) {
			t.Errorf("у игры %q нет Linux-сервера, профиль ей не нужен", key)
		}
	}
}

// Профиль не может завести раздел вне словаря: иначе у одной игры появится
// «Мир», а у другой «Игровой мир», и вкладки перестанут быть узнаваемыми.
func TestProfilesUseKnownSections(t *testing.T) {
	for _, p := range All() {
		for _, f := range p.Fields {
			if !SectionKnown(f.Section) {
				t.Errorf("%s: поле %s ссылается на неизвестный раздел %q", p.Key, f.Key, f.Section)
			}
			if f.Section == SectionRaw || f.Section == SectionStartup {
				t.Errorf("%s: поле %s заняло системный раздел %q — их добавляет панель",
					p.Key, f.Key, f.Section)
			}
		}
	}
}

func TestProfileIntegrity(t *testing.T) {
	for _, p := range All() {
		seenField := map[string]bool{}
		seenTarget := map[string]bool{}
		seenFile := map[string]bool{}
		slots := 0

		if len(p.Files) == 0 {
			t.Errorf("%s: профиль без файлов — настройкам некуда писаться", p.Key)
		}
		for _, file := range p.Files {
			if file.ID == "" {
				t.Errorf("%s: файл без идентификатора", p.Key)
			}
			if seenFile[file.ID] {
				t.Errorf("%s: файл %q объявлен дважды", p.Key, file.ID)
			}
			seenFile[file.ID] = true
			if file.Path == "" {
				t.Errorf("%s: файл %q без пути", p.Key, file.ID)
			}
		}

		for _, f := range p.Fields {
			if f.Key == "" || f.Label == "" {
				t.Errorf("%s: поле без ключа или подписи: %+v", p.Key, f)
				continue
			}
			if seenField[f.Key] {
				t.Errorf("%s: ключ %q повторяется", p.Key, f.Key)
			}
			seenField[f.Key] = true

			if _, ok := p.FileByID(f.File); !ok {
				t.Errorf("%s: поле %s ссылается на несуществующий файл %q", p.Key, f.Key, f.File)
			}
			if f.Prop == "" {
				t.Errorf("%s: поле %s не знает, куда писаться (пустой Prop)", p.Key, f.Key)
			} else {
				target := f.File + "\x00" + f.Prop
				if seenTarget[target] {
					t.Errorf("%s: два поля пишут в одно место %s/%s", p.Key, f.File, f.Prop)
				}
				seenTarget[target] = true
			}

			switch f.Kind {
			case KindEnum:
				if len(f.Options) < 2 {
					t.Errorf("%s: поле %s списочное, но вариантов меньше двух", p.Key, f.Key)
				}
				if f.Default != "" && !hasOption(f.Options, f.Default) {
					t.Errorf("%s: значение по умолчанию %q поля %s вне списка вариантов",
						p.Key, f.Default, f.Key)
				}
			case KindBool:
				// True и False пустыми быть могут — тогда пишется true/false, —
				// но задавать только одно из них бессмысленно и почти наверняка
				// означает забытую вторую половину.
				if (f.True == "") != (f.False == "") {
					t.Errorf("%s: у булева поля %s задана только половина написания (%q/%q)",
						p.Key, f.Key, f.True, f.False)
				}
			case KindInt, KindFloat:
				if f.Min != nil && f.Max != nil && *f.Min > *f.Max {
					t.Errorf("%s: у поля %s нижняя граница больше верхней", p.Key, f.Key)
				}
			}

			if f.Slots {
				slots++
			}
			if f.Secret && f.Kind != KindString {
				t.Errorf("%s: секретное поле %s должно быть строкой", p.Key, f.Key)
			}
		}

		if slots > 1 {
			t.Errorf("%s: полей со Slots должно быть не больше одного, найдено %d", p.Key, slots)
		}
	}
}

// Значение по умолчанию должно проходить собственную валидацию профиля. Иначе
// форма покажет значение, которое нельзя сохранить.
func TestProfileDefaultsAreValid(t *testing.T) {
	for _, p := range All() {
		for _, f := range p.Fields {
			if f.Default == "" || f.ReadOnly {
				continue
			}
			if _, err := Validate(f, f.Default); err != nil {
				t.Errorf("%s: значение по умолчанию поля %s не проходит проверку: %v",
					p.Key, f.Key, err)
			}
		}
	}
}

func hasOption(options []Option, value string) bool {
	for _, o := range options {
		if o.Value == value {
			return true
		}
	}
	return false
}

// Страховка от опечатки в register: одна игра — один профиль.
func TestNoGameServedByTwoProfiles(t *testing.T) {
	owner := map[string]string{}
	for _, p := range All() {
		for _, key := range p.Games() {
			norm := gamecatalog.Normalize(key)
			if prev, busy := owner[norm]; busy {
				t.Errorf("игру %s обслуживают сразу %s и %s", norm, prev, p.Key)
			}
			owner[norm] = p.Key
		}
	}
}

func TestSectionsDictionaryIsOrdered(t *testing.T) {
	seen := map[SectionID]bool{}
	for i, s := range Sections() {
		if s.Title == "" {
			t.Errorf("раздел %q без заголовка", s.ID)
		}
		if seen[s.ID] {
			t.Errorf("раздел %q повторяется в словаре", s.ID)
		}
		seen[s.ID] = true
		if got := SectionRank(s.ID); got != i {
			t.Errorf("%s: SectionRank = %d, ожидалось %d", s.ID, got, i)
		}
	}
	// Системные разделы обязаны быть последними: панель дописывает их после
	// разделов игры.
	all := Sections()
	last := []SectionID{all[len(all)-2].ID, all[len(all)-1].ID}
	if last[0] != SectionRaw || last[1] != SectionStartup {
		t.Errorf("в конце словаря %v, ожидались raw и startup", last)
	}
	if SectionKnown("выдуманный") {
		t.Error("словарь разделов должен быть закрытым")
	}
}

func TestSectionTitleFallsBackToID(t *testing.T) {
	if got := SectionTitle("нетакого"); got != "нетакого" {
		t.Errorf("SectionTitle для неизвестного раздела = %q", got)
	}
}

func TestGroupBySectionFollowsDictionaryOrder(t *testing.T) {
	p := Profile{
		Key:   "проверка",
		Files: []ConfigFile{{ID: "main", Path: "server.cfg", Format: FormatSourceCfg}},
		Fields: []Field{
			{Key: "c", Label: "c", Section: SectionPerformance, Kind: KindInt, Prop: "c"},
			{Key: "a", Label: "a", Section: SectionGeneral, Kind: KindString, Prop: "a"},
			{Key: "b", Label: "b", Section: SectionGameplay, Kind: KindBool, Prop: "b"},
		},
	}
	groups := p.GroupBySection()
	got := make([]string, 0, len(groups))
	for _, g := range groups {
		got = append(got, string(g.Section.ID))
	}
	want := fmt.Sprint([]string{"general", "gameplay", "performance"})
	if fmt.Sprint(got) != want {
		t.Errorf("порядок разделов %v, ожидался %v", got, want)
	}
}
