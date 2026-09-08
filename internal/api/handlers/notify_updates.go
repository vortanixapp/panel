package handlers

import (
	"context"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vortanix/vortanix/pkg/notify"
)

// NotifyPanelUpdate сообщает владельцу панели, что вышла новая версия.
//
// Событие panel.update было объявлено в справочнике, но не отправлялось: новую
// версию было видно только тому, кто сам открыл раздел обновлений. Вызывается
// обновлятором лицензии в момент, когда сервис назвал новую целевую версию;
// база приходит параметром, потому что вызов идёт не из запроса и выбрать пул
// по арендатору из контекста нельзя.
func (h *Handler) NotifyPanelUpdate(ctx context.Context, db *pgxpool.Pool, target, notes string) {
	target = strings.TrimSpace(target)
	if db == nil || target == "" {
		return
	}
	// Обновляться незачем, если панель уже на этой версии: сервис лицензий
	// присылает цель и тем установкам, которые её давно накатили.
	if target == envOr("VORTANIX_VERSION", "dev") {
		return
	}

	rows, err := db.Query(ctx, `
		SELECT id::text, tenant_id::text FROM core.users
		WHERE role IN ('owner', 'admin')
	`)
	if err != nil {
		log.Printf("оповещение о версии %s: получатели не найдены: %v", target, err)
		return
	}
	defer rows.Close()

	type staff struct{ userID, tenantID string }
	var list []staff
	for rows.Next() {
		var s staff
		if rows.Scan(&s.userID, &s.tenantID) == nil && s.tenantID != "" {
			list = append(list, s)
		}
	}

	body := "Доступна версия панели " + target + "."
	if strings.TrimSpace(notes) != "" {
		body += "\n\n" + excerpt(notes, 500)
	}
	body += "\n\nОбновление ставится кнопкой в разделе «Обновления»."

	for _, s := range list {
		rec, err := notify.LoadRecipient(ctx, db, s.tenantID, s.userID)
		if err != nil {
			log.Printf("оповещение о версии %s: получатель %s: %v", target, s.userID, err)
			continue
		}
		if _, err := notify.Dispatch(ctx, db, s.tenantID, rec, notify.Event{
			Kind:   notify.KindPanelUpdate,
			Title:  "Доступно обновление панели",
			Body:   body,
			Action: h.panelAction("Открыть обновления", "/admin/updates"),
			Meta:   map[string]any{"target_version": target},
			// Одна версия — одно письмо: сверка с сервисом лицензий идёт по
			// расписанию, и без ключа оповещение повторялось бы при каждом
			// перезапуске панели.
			DedupeKey: "panel.update:" + target,
		}); err != nil {
			log.Printf("оповещение о версии %s пользователю %s: %v", target, s.userID, err)
		}
	}
}
