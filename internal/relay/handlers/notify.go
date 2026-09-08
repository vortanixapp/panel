package handlers

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/vortanix/vortanix/pkg/notify"
)

// Оповещения о том, что relay узнаёт первым.
//
// Именно сюда приходят события, о которых клиент до сих пор не узнавал ничем:
// сервер упал, установка не удалась, нода пропала со связи. Всё это записывалось
// в базу и публиковалось в Redis для живого экрана — то есть увидеть можно было,
// только если смотреть на панель в этот момент.
//
// Отправлять отсюда ничего не нужно: производитель события пишет запись и строки
// очереди, а разбором очереди занят воркер. Поэтому relay обходится одним
// подключением к базе и не знает ни про SMTP, ни про токен бота.

// notifyServerOwner отправляет событие владельцу сервера.
//
// Ошибку только логируем: оповещение — следствие уже случившегося, и ронять из-за
// него обработку сообщения от агента нельзя.
func (h *Handler) notifyServerOwner(ctx context.Context, db *pgxpool.Pool, tenantID, serverID string, e notify.Event) {
	r, err := notify.LoadServerOwner(ctx, db, tenantID, serverID)
	if err != nil {
		// Сервер без владельца — обычное дело для служебных машин.
		return
	}
	if _, err := notify.Dispatch(ctx, db, tenantID, r, e); err != nil {
		log.Printf("relay: оповещение %s по серверу %s: %v", e.Kind, serverID, err)
	}
}

// serverAction — кнопка со ссылкой на сервер.
//
// Абсолютная: то же уведомление уходит письмом и в Telegram, где относительный
// путь никуда не ведёт. Если адрес панели не задан, кнопки просто не будет.
func (h *Handler) serverAction(label, serverID, suffix string) *notify.Action {
	if h.panelURL == "" {
		return nil
	}
	return &notify.Action{Label: label, Href: h.panelURL + "/servers/" + serverID + suffix}
}

// notifyNodeOwners сообщает владельцам серверов локации, что она пропала.
//
// Адресуем владельцам, а не администратору: у клиента лежит именно его сервер, и
// узнать об этом он должен от нас, а не от игроков.
func (h *Handler) notifyNodeOwners(ctx context.Context, db *pgxpool.Pool, tenantID, nodeID, nodeName string) {
	rows, err := db.Query(ctx, `
		SELECT DISTINCT user_id::text
		FROM core.servers
		WHERE tenant_id = $1 AND node_id = $2 AND user_id IS NOT NULL
		  AND COALESCE(runtime_status, status) = 'running'
	`, tenantID, nodeID)
	if err != nil {
		return
	}
	defer rows.Close()

	owners := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			owners = append(owners, id)
		}
	}
	if len(owners) == 0 {
		return
	}

	name := nodeName
	if name == "" {
		name = "локация"
	}
	for _, userID := range owners {
		r, err := notify.LoadRecipient(ctx, db, tenantID, userID)
		if err != nil {
			continue
		}
		_, _ = notify.Dispatch(ctx, db, tenantID, r, notify.Event{
			Kind:  notify.KindNodeOffline,
			Title: "Локация недоступна",
			Body: "Связь с локацией «" + name + "» потеряна. Серверы на ней могут быть недоступны. " +
				"Мы уже разбираемся — отдельных действий от вас не требуется.",
			Meta: map[string]any{"node_id": nodeID, "node_name": name},
			// Разрыв связи бывает частым и коротким. Ключ не даёт слать одно и
			// то же при каждом переподключении агента.
			DedupeKey: "node.offline:" + nodeID,
		})
	}
}

// pendingServerOperation — по серверу прямо сейчас идёт операция питания.
//
// Признак берётся из того же кэша статуса, который пишет обработчик питания в
// core-api: он кладёт туда переходное состояние с коротким сроком жизни. Это
// удобно вдвойне — отдельного поля «идёт операция» в схеме нет, а истекающий
// ключ сам собой снимает защиту, если операция подвисла, и сверка по метрикам
// снова начинает чинить состояние.
func pendingServerOperation(ctx context.Context, rdb *redis.Client, serverID string) bool {
	if rdb == nil {
		return false
	}
	v, err := rdb.Get(ctx, "srv:"+serverID+":status").Result()
	if err != nil {
		return false
	}
	switch v {
	case "stopping", "starting", "reinstalling", "installing":
		return true
	}
	return false
}
