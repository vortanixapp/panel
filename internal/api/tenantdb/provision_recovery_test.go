package tenantdb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func registrySource(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(".", "registry.go"))
	if err != nil {
		t.Fatalf("не прочитать registry.go: %v", err)
	}
	return string(raw)
}

// funcBody возвращает текст функции от объявления до следующего объявления
// верхнего уровня.
func funcBody(t *testing.T, src, decl string) string {
	t.Helper()
	i := strings.Index(src, decl)
	if i < 0 {
		t.Fatalf("не найдено объявление %q", decl)
	}
	rest := src[i+len(decl):]
	if j := strings.Index(rest, "\nfunc "); j >= 0 {
		return rest[:j]
	}
	return rest
}

// Оборвавшееся заведение базы оставляло за собой половину работы: база с
// частью миграций, роль без прав — и никакой записи в реестре. Повтор такую
// базу принимал за готовую.
func TestBrokenProvisionCleansUpAfterItself(t *testing.T) {
	src := registrySource(t)

	body := funcBody(t, src, "func (r *Registry) ProvisionWithUser(")
	if !strings.Contains(body, "undoProvision") {
		t.Error("после неудачи за собой не убирают")
	}

	undo := funcBody(t, src, "func (r *Registry) undoProvision(")
	if !strings.Contains(undo, "DROP DATABASE IF EXISTS") {
		t.Error("недоделанная база остаётся с частью миграций")
	}
	if !strings.Contains(undo, "DROP ROLE IF EXISTS") {
		t.Error("роль недоделанной базы остаётся")
	}
	if strings.Index(undo, "DROP DATABASE IF EXISTS") > strings.Index(undo, "DROP ROLE IF EXISTS") {
		t.Error("роль удаляют раньше базы: DROP ROLE откажет, пока у роли есть CONNECT")
	}
	if !strings.Contains(undo, "context.WithoutCancel") {
		t.Error("уборка идёт на контексте вызова: истёкший таймаут отменит и её")
	}

	// Убирать разрешено только своё: база, жившая до вызова, может хранить
	// данные клиента.
	if !strings.Contains(undo, "done.db") || !strings.Contains(undo, "done.role") {
		t.Error("уборка не различает, что завела именно эта попытка")
	}
}

// Пустой пароль после оборвавшейся попытки означал, что доступ к своей базе
// клиент не получит никогда: прежний пароль не сохранил никто, а нового
// провизионирование не выдавало.
func TestPasswordIsReissuedAfterBrokenAttempt(t *testing.T) {
	src := registrySource(t)

	provision := funcBody(t, src, "func (r *Registry) ProvisionWithUser(")
	if !strings.Contains(provision, "FROM core.tenant_databases WHERE slug = $1") {
		t.Error("завершённое заведение не отличают от оборванного")
	}

	role := funcBody(t, src, "func (r *Registry) ensureRole(")
	if !strings.Contains(role, "if exists && registered") {
		t.Error("пароль молчат для любой существующей роли, а не только для состоявшегося арендатора")
	}
	if !strings.Contains(role, "ALTER ROLE") {
		t.Error("роли от оборванной попытки не назначают новый пароль")
	}
}
