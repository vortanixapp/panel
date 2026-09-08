package docker

import "testing"

// Расход берётся из колонки по номеру, а не «первое похожее на число». У
// проекта с нулевым расходом вольный разбор подхватывал бы соседний столбец, и
// клиенту показывалось бы занятое место, взятое с потолка.
func TestParseRepquotaUsedMB(t *testing.T) {
	const out = `Project,BlockStatus,FileStatus,BlockUsed,BlockSoftLimit,BlockHardLimit,BlockGrace,FileUsed,FileSoftLimit,FileHardLimit,FileGrace
10000,ok,ok,1572864,2097152,2097152,,412,0,0,
10001,ok,ok,0,524288,524288,,1,0,0,
10002,+,ok,2097152,2097152,2097152,7days,904,0,0,
`

	if got := parseRepquotaUsedMB(out, 10000); got != 1536 {
		t.Errorf("1572864 КБ это 1536 МБ, получено %d", got)
	}
	// Нулевой расход — это ноль, а не первое число из строки лимитов.
	if got := parseRepquotaUsedMB(out, 10001); got != 0 {
		t.Errorf("расход пустого проекта должен быть 0, получено %d", got)
	}
	if got := parseRepquotaUsedMB(out, 10002); got != 2048 {
		t.Errorf("проект за пределом: ожидалось 2048 МБ, получено %d", got)
	}
	// Неизвестный проект не должен подхватывать чужую строку.
	if got := parseRepquotaUsedMB(out, 99999); got != 0 {
		t.Errorf("чужой проект: ожидался 0, получено %d", got)
	}
}

func TestParseRepquotaHandlesGarbage(t *testing.T) {
	cases := []string{
		"",
		"мусор без запятых",
		"10000,ok",              // колонок меньше, чем нужно
		"10000,ok,ok,не-число,", // расход не разбирается
	}
	for _, c := range cases {
		if got := parseRepquotaUsedMB(c, 10000); got != 0 {
			t.Errorf("на входе %q ожидался 0, получено %d", c, got)
		}
	}
}

// Тариф без места не должен приводить к попытке выставить квоту: нулевой лимит
// в setquota означает «безлимит», и вызов был бы не просто лишним, а снял бы
// уже выставленный предел.
func TestApplyDiskQuotaIgnoresZero(t *testing.T) {
	t.Setenv("VORTANIX_DATA_DIR", t.TempDir())
	// Проверяем именно то, что функция возвращается сразу: на машине без квот
	// любой другой путь ушёл бы в exec и упал бы или наследил в журнале.
	ApplyDiskQuota(t.Context(), "srv-1", 0)
}

func TestDiskLimitMB(t *testing.T) {
	if got := diskLimitMB(map[string]any{"disk_mb": float64(2048)}); got != 2048 {
		t.Errorf("disk_mb float64: получено %d", got)
	}
	if got := diskLimitMB(map[string]any{"disk_mb": 4096}); got != 4096 {
		t.Errorf("disk_mb int: получено %d", got)
	}
	// Тариф без места — ноль, а не значение по умолчанию: придумывать за
	// администратора лимит, которого он не задавал, нельзя.
	if got := diskLimitMB(map[string]any{"memory_mb": 1024}); got != 0 {
		t.Errorf("без disk_mb ожидался 0, получено %d", got)
	}
	if got := diskLimitMB(nil); got != 0 {
		t.Errorf("nil: ожидался 0, получено %d", got)
	}
}

// Право писать мимо квоты должно сниматься явно. Docker его по умолчанию не
// выдаёт, но дисковый лимит — обещание клиенту, и держаться оно должно на нашем
// требовании: набор по умолчанию у демона может измениться с версией.
func TestBuildRunArgsDropsQuotaBypass(t *testing.T) {
	t.Setenv("VORTANIX_DATA_DIR", t.TempDir())

	args := buildRunArgs("srv-1", "cs2", map[string]any{"memory_mb": 1024, "disk_mb": 2048},
		"vortanix/cs2:latest", 27015, "")

	dropped := false
	for _, v := range argValues(args, "--cap-drop") {
		if v == "SYS_RESOURCE" {
			dropped = true
		}
	}
	if !dropped {
		t.Errorf("SYS_RESOURCE не снят — root в контейнере обойдёт дисковую квоту; args=%v", args)
	}
}

// Определение квот идёт по /proc/mounts, а не по findmnt: в образе агента
// (alpine) findmnt нет, вызов молча проваливался, и лимиты не проставлялись
// никому при живых и правильно смонтированных квотах.
func TestQuotaMountFrom(t *testing.T) {
	// Строка ровно в том виде, в каком её видит агент на боевой ноде.
	const real = `overlay / overlay rw,relatime,lowerdir=/x 0 0
/dev/sda1 /var/lib/docker ext4 rw,relatime 0 0
/dev/loop0 /var/lib/vortanix/servers ext4 rw,relatime,prjquota 0 0
proc /proc proc rw,nosuid 0 0
`
	if got := quotaMountFrom(real, "/var/lib/vortanix/servers"); got != "/var/lib/vortanix/servers" {
		t.Errorf("квоты не распознаны: получено %q", got)
	}
	// Каталог сервера лежит внутри точки монтирования — тоже под квотой.
	if got := quotaMountFrom(real, "/var/lib/vortanix/servers/abc"); got != "/var/lib/vortanix/servers" {
		t.Errorf("вложенный каталог: получено %q", got)
	}

	const noQuota = `/dev/sda1 / ext4 rw,relatime 0 0
`
	if got := quotaMountFrom(noQuota, "/var/lib/vortanix/servers"); got != "" {
		t.Errorf("без prjquota ожидалась пустая строка, получено %q", got)
	}

	// Похожий по началу путь не должен сходить за нужный: иначе чужая
	// файловая система выдавалась бы за квотируемую.
	const lookalike = `/dev/loop0 /var/lib/vortanix-old ext4 rw,prjquota 0 0
`
	if got := quotaMountFrom(lookalike, "/var/lib/vortanix/servers"); got != "" {
		t.Errorf("совпадение по префиксу без границы пути: получено %q", got)
	}

	// Более глубокое монтирование перекрывает внешнее: за каталог отвечает оно.
	const nested = `/dev/sda1 / ext4 rw,prjquota 0 0
/dev/loop0 /var/lib/vortanix/servers ext4 rw,relatime 0 0
`
	if got := quotaMountFrom(nested, "/var/lib/vortanix/servers"); got != "" {
		t.Errorf("вложенное монтирование без квот должно перекрывать корневое, получено %q", got)
	}

	// Пробел в пути /proc/mounts кодирует четырьмя символами.
	const spaced = "/dev/loop0 /var/lib/my\\040dir ext4 rw,prjquota 0 0\n"
	if got := quotaMountFrom(spaced, "/var/lib/my dir"); got != "/var/lib/my dir" {
		t.Errorf("путь с пробелом: получено %q", got)
	}
}
