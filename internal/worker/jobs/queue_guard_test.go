package jobs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func workerSource(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(".", name))
	if err != nil {
		t.Fatalf("не прочитать %s: %v", name, err)
	}
	return string(raw)
}

// Ядовитая задача бралась первой после каждого перезапуска и валила обработчик
// по кругу: предела попыток не было, а счётчик attempts, заведённый в схеме,
// воркер не трогал вовсе.
func TestEveryClaimCountsAttemptsAndStopsAtLimit(t *testing.T) {
	files := []string{
		"provision.go", "backup.go", "mailing.go", "daemon_ops.go",
		"nodesetup.go", "migrate.go", "node_bulk.go", "backup_offsite.go",
	}
	for _, name := range files {
		src := workerSource(t, name)

		claims := strings.Count(src, "SET status = 'running'")
		counted := strings.Count(src, "attempts = attempts + 1")
		if claims != counted {
			t.Errorf("%s: захватов %d, а попытки считаются в %d — задача будет браться вечно",
				name, claims, counted)
		}

		// Считаем именно выборки очереди: «SET remote_status = 'pending'» из
		// соседних таблиц к захвату задач отношения не имеет.
		pending := strings.Count(src, "AND status = 'pending'")
		guarded := strings.Count(src, "AND status = 'pending' AND attempts < 5")
		if pending != guarded {
			t.Errorf("%s: выборок задач %d, с пределом попыток %d", name, pending, guarded)
		}
	}
}

// Задача, взятая в работу перед падением процесса, оставалась в running
// навсегда, и очередь арендатора вставала молча — при живом /health.
func TestStaleRunningJobsAreReclaimed(t *testing.T) {
	src := workerSource(t, "queue_guard.go")

	if !strings.Contains(src, "status = 'pending'") {
		t.Error("брошенные задачи не возвращаются в очередь")
	}
	if !strings.Contains(src, "attempts >= $2") {
		t.Error("исчерпавшие попытки не уводятся в отказ и вернутся снова")
	}
	if !strings.Contains(src, "updated_at < now()") {
		t.Error("нет признака простоя: живая долгая задача будет отобрана у обработчика")
	}
}

// Цикл уборки должен подниматься на каждой базе арендатора, иначе очередь
// клиента остаётся без сторожа.
func TestStaleSweeperRunsForEveryTenant(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "cmd", "vortanix-worker", "tenants.go"))
	if err != nil {
		t.Fatalf("не прочитать tenants.go: %v", err)
	}
	if !strings.Contains(string(raw), "StaleJobsLoop") {
		t.Error("сторож очереди не запускается вместе с остальными циклами")
	}
}
