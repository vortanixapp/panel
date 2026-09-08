package handlers

import (
	"strings"
	"testing"
)

// Агент прежних версий не разбирает новые команды, и обработчика неизвестной
// команды у него нет: он отвечает «выполнено» и сообщает статус по умолчанию —
// «остановлен». Между выкаткой панели и обновлением агентов на нодах такой
// ответ нельзя принимать за успех.
func TestPortsSyncRejectsOldAgent(t *testing.T) {
	body := funcBody(t, readSource(t, "server_sync.go"), "pushPortsState")

	if !strings.Contains(body, `res["ports"]`) {
		t.Error("ответ агента не проверяется на признак новой версии")
	}
	if !strings.Contains(body, "errAgentTooOld") {
		t.Error("устаревший агент не отличается от обычного отказа")
	}
	if !strings.Contains(body, "restoreServerStatus") {
		t.Error("ложный статус «остановлен» остаётся в базе")
	}
	if !strings.Contains(body, "prevStatus") {
		t.Error("прежний статус не запоминается до отправки команды")
	}
}

func TestUpdateGameRejectsOldAgent(t *testing.T) {
	body := funcBody(t, readSource(t, "server_lifecycle.go"), "UpdateServerGame")

	if !strings.Contains(body, `result["status"]`) {
		t.Error("ответ агента не проверяется: панель показала бы «обновляется» впустую")
	}
	if !strings.Contains(body, "restoreServerStatus") {
		t.Error("ложный статус «остановлен» остаётся в базе")
	}
	if !strings.Contains(body, "StatusConflict") {
		t.Error("устаревший агент должен отвечать 409, а не выглядеть сбоем связи")
	}
}

// Статус возвращается ровно тот, что был: пустые значения не затирают колонки.
func TestRestoreServerStatusKeepsPreviousValues(t *testing.T) {
	body := funcBody(t, readSource(t, "server_sync.go"), "restoreServerStatus")

	if !strings.Contains(body, "NULLIF") {
		t.Error("пустой прежний статус запишется вместо настоящего")
	}
	if !strings.Contains(body, "tenant_id = $2") {
		t.Error("восстановление не ограничено арендатором")
	}
}
