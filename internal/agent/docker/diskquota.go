package docker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Дисковая квота игрового сервера.
//
// Место в тарифе было единственным ресурсом, который нигде не проверялся:
// disk_mb доезжал до агента и попадал только в статистику. Данные лежат в
// bind-монтировании, куда клиент пишет по SFTP, и залить туда можно было
// сколько угодно — вплоть до заполнения диска всей локации.
//
// Ограничиваем квотами проекта ext4: лимит держит ядро, запись за пределом
// падает с ENOSPC сразу, без опроса и без окна, в которое можно проскочить.
// Docker для этого не годится: --storage-opt ограничивает только пишущий слой
// контейнера, а не примонтированный каталог, и требует overlay2 поверх XFS.
//
// Квоты включены не на всякой ноде, поэтому отсутствие поддержки — не ошибка:
// сервер должен подняться и без лимита. Об этом пишем в журнал один раз, а не
// на каждый запуск: строка про неподдерживаемую ФС, повторённая тысячу раз,
// прячет настоящие сообщения.

const projectIDBase = 10000

var (
	quotaOnce    sync.Once
	quotaMount   string // точка монтирования с prjquota, пусто — не поддерживается
	quotaWarned  sync.Once
	projectIDMux sync.Mutex
)

// quotaMountpoint — точка монтирования каталога серверов, если она смонтирована
// с prjquota. Пусто означает «квот на этой ноде нет».
//
// Читаем /proc/mounts, а не спрашиваем findmnt: агент работает в контейнере на
// alpine, и findmnt там нет — вызов молча проваливался, определение возвращало
// «нет», и лимиты не проставлялись никому при живых и правильно смонтированных
// квотах. Файл же есть всегда и показывает внутри контейнера ровно то, что
// нужно: строку с prjquota для проброшенного каталога.
//
// Определяется один раз: перемонтирование на живой ноде требует перезапуска
// агента, и кэш здесь не устареет незаметно.
func quotaMountpoint() string {
	quotaOnce.Do(func() {
		raw, err := os.ReadFile("/proc/mounts")
		if err != nil {
			return
		}
		quotaMount = quotaMountFrom(string(raw), serversRoot())
	})
	return quotaMount
}

// quotaMountFrom ищет файловую систему, на которой лежит каталог, и проверяет
// её параметры. Вынесено отдельно, чтобы разбор проверялся без квот на машине.
func quotaMountFrom(mounts, dir string) string {
	// Сначала находим монтирование, которое отвечает за каталог, и только потом
	// смотрим его параметры. Наоборот нельзя: если снаружи корень с prjquota, а
	// внутрь примонтирован каталог без квот, проверка «есть ли среди подходящих
	// хоть одно с prjquota» ответила бы «да» — и агент считал бы место
	// ограниченным там, где его никто не ограничивает.
	target, opts := "", ""
	for _, line := range strings.Split(mounts, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		// Пробелы в путях /proc/mounts кодирует четырьмя символами \040.
		point := strings.ReplaceAll(fields[1], `\040`, " ")
		if !underMount(dir, point) {
			continue
		}
		// Вложенных монтирований может быть несколько; за каталог отвечает
		// самое глубокое, как и решает ядро.
		if len(point) < len(target) {
			continue
		}
		target, opts = point, fields[3]
	}

	if target == "" {
		return ""
	}
	for _, opt := range strings.Split(opts, ",") {
		if opt == "prjquota" {
			return target
		}
	}
	return ""
}

// underMount — лежит ли каталог на этой точке монтирования. Сравниваем по
// границе пути: иначе /var/lib/vortanix-old сошёл бы за /var/lib/vortanix.
func underMount(dir, target string) bool {
	if target == "/" {
		return true
	}
	if dir == target {
		return true
	}
	return strings.HasPrefix(dir, strings.TrimSuffix(target, "/")+"/")
}

func serversRoot() string {
	base := os.Getenv("VORTANIX_DATA_DIR")
	if base == "" {
		base = "/var/lib/vortanix/servers"
	}
	return base
}

// QuotaSupported говорит, ограничивается ли место на этой ноде.
func QuotaSupported() bool { return quotaMountpoint() != "" }

// ApplyDiskQuota выставляет лимит места каталогу сервера.
//
// Вызывается при каждом запуске контейнера: тариф мог смениться, а идти за
// этим отдельной командой значит оставить окно, в котором лимит уже другой, а
// на диске ещё старый.
func ApplyDiskQuota(ctx context.Context, serverID string, diskMB int) {
	if diskMB <= 0 {
		// Тариф без места — ограничивать нечего. Снимать уже выставленную
		// квоту тоже не будем: это молча превратило бы урезание тарифа в
		// безлимит.
		return
	}
	mount := quotaMountpoint()
	if mount == "" {
		quotaWarned.Do(func() {
			log.Printf("дисковые квоты недоступны: %s смонтирован без prjquota — место сервера не ограничивается",
				serversRoot())
		})
		return
	}

	dir := serverDataDir(serverID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("квота %s: каталог недоступен: %v", serverID, err)
		return
	}

	projID, err := ensureProjectID(ctx, dir)
	if err != nil {
		log.Printf("квота %s: не удалось назначить проект: %v", serverID, err)
		return
	}

	// Мягкий и жёсткий предел одинаковы: «мягкий» даёт льготный период, в
	// который клиент продолжает писать сверх тарифа, а нам нужен именно
	// потолок.
	blocks := strconv.Itoa(diskMB * 1024) // setquota считает в килобайтах
	cmd := exec.CommandContext(ctx, "setquota", "-P", strconv.FormatUint(uint64(projID), 10),
		blocks, blocks, "0", "0", mount)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("квота %s: setquota не отработал: %v: %s", serverID, err, strings.TrimSpace(string(out)))
		return
	}
}

// ensureProjectID возвращает идентификатор проекта каталога, назначая новый,
// если его ещё нет.
//
// Идентификатор не выводим из UUID сервера: 32 бита на несколько тысяч
// серверов дают заметную вероятность совпадения, а два сервера с одним
// проектом делили бы одну квоту. Источник истины — сама файловая система:
// уже назначенный идентификатор читается с каталога, новый берётся на единицу
// больше максимального среди соседей.
func ensureProjectID(ctx context.Context, dir string) (uint32, error) {
	projectIDMux.Lock()
	defer projectIDMux.Unlock()

	if id := readProjectID(ctx, dir); id != 0 {
		return id, nil
	}

	next := uint32(projectIDBase)
	entries, err := os.ReadDir(serversRoot())
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if id := readProjectID(ctx, filepath.Join(serversRoot(), e.Name())); id >= next {
				next = id + 1
			}
		}
	}

	id := strconv.FormatUint(uint64(next), 10)

	// Два прохода, а не один рекурсивный с +P.
	//
	// +P — наследование проекта — атрибут только для каталогов: на обычном
	// файле ядро отвечает «operation not supported», рекурсивный вызов падает
	// на первом же файле и возвращает ошибку. Из-за этого квота не
	// проставлялась вовсе — проверено на боевой ноде, где каталог сервера полон
	// файлов игры.
	//
	// Сначала сам каталог: проект плюс наследование, чтобы всё созданное после
	// попадало под учёт автоматически.
	if out, err := exec.CommandContext(ctx, "chattr", "-p", id, "+P", dir).CombinedOutput(); err != nil {
		return 0, fmt.Errorf("chattr +P: %w: %s", err, strings.TrimSpace(string(out)))
	}

	// Затем содержимое: каталог уже может быть непустым (перенос ноды,
	// восстановление копии, включение квот на живой локации), и файлы без
	// проекта не попали бы под учёт. Здесь без +P.
	if out, err := exec.CommandContext(ctx, "chattr", "-R", "-p", id, dir).CombinedOutput(); err != nil {
		// Не возвращаем ошибку: каталог уже помечен и лимит выставить можно, а
		// отдельные файлы, которым проект не сменить, лишь останутся вне учёта.
		log.Printf("квота: не весь %s помечен проектом %s: %v: %s",
			dir, id, err, strings.TrimSpace(string(out)))
	}
	return next, nil
}

func readProjectID(ctx context.Context, dir string) uint32 {
	out, err := exec.CommandContext(ctx, "lsattr", "-pd", dir).Output()
	if err != nil {
		return 0
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) == 0 {
		return 0
	}
	id, err := strconv.ParseUint(fields[0], 10, 32)
	if err != nil {
		return 0
	}
	return uint32(id)
}

// DiskQuotaUsedMB — занятое место по данным квоты.
//
// Точнее и дешевле du: ядро уже ведёт счёт, а обход дерева на сервере с сотней
// тысяч файлов занимает секунды и бьёт по диску на каждый опрос статистики.
// Ноль означает «квоты нет» — вызывающий тогда считает по-старому.
func DiskQuotaUsedMB(ctx context.Context, serverID string) int {
	mount := quotaMountpoint()
	if mount == "" {
		return 0
	}
	dir := serverDataDir(serverID)
	projID := readProjectID(ctx, dir)
	if projID == 0 {
		return 0
	}
	out, err := exec.CommandContext(ctx, "repquota", "-P", "-O", "csv", mount).Output()
	if err != nil {
		return 0
	}
	return parseRepquotaUsedMB(string(out), projID)
}

// parseRepquotaUsedMB достаёт занятые килобайты нужного проекта из вывода
// repquota в csv. Вынесено отдельно, чтобы разбор проверялся без квот на машине.
func parseRepquotaUsedMB(out string, projID uint32) int {
	// Раскладка repquota -O csv:
	//   Project,BlockStatus,FileStatus,BlockUsed,BlockSoftLimit,BlockHardLimit,…
	// Занятое место — четвёртая колонка, в килобайтах. Берём её по номеру, а не
	// «первое похожее на число»: у проекта с нулевым расходом так подхватился бы
	// первый попавшийся столбец и расход показался бы взятым с потолка.
	const (
		colProject   = 0
		colBlockUsed = 3
	)
	want := strconv.FormatUint(uint64(projID), 10)
	for _, line := range strings.Split(out, "\n") {
		cols := strings.Split(strings.TrimSpace(line), ",")
		if len(cols) <= colBlockUsed {
			continue
		}
		if strings.TrimSpace(cols[colProject]) != want {
			continue
		}
		kb, err := strconv.Atoi(strings.TrimSpace(cols[colBlockUsed]))
		if err != nil || kb < 0 {
			return 0
		}
		return kb / 1024
	}
	return 0
}
