// Package notify — единый слой оповещений: что произошло, кому сообщить и
// какими каналами.
//
// До него оповещения были тремя несвязанными кусками. Уведомление внутри панели
// создавалось прямым INSERT из четырнадцати мест, каждое со своим написанием
// типа. Почта существовала в трёх независимых реализациях SMTP, две из которых —
// побайтовые копии друг друга. Каналы Telegram и Discord были в интерфейсе и в
// базе, но отправки не существовало ни строки.
//
// Здесь всё это сведено в одно место: событие описывается один раз, а куда оно
// уедет — решают справочник и настройки получателя.
package notify

import "sort"

// Kind — что именно произошло. Значение уходит в базу как есть, поэтому менять
// уже существующие строки нельзя: по ним отфильтрованы старые уведомления.
type Kind string

const (
	// Жизненный цикл сервера
	KindServerReady     Kind = "server.ready"
	KindServerFailed    Kind = "server.failed"
	KindServerDown      Kind = "server.down"
	KindServerExpiring  Kind = "server.expiring"
	KindServerSuspended Kind = "server.suspended"
	KindServerBlocked   Kind = "server.blocked"
	KindServerUnblocked Kind = "server.unblocked"
	KindServerRenewed   Kind = "server.renewed"
	KindServerDeleted   Kind = "server.deleted"
	KindServerMigrated  Kind = "server.migrated"
	KindBackupReady     Kind = "backup.ready"
	KindBackupFailed    Kind = "backup.failed"

	// Деньги
	KindPaymentReceived Kind = "payment.received"
	KindPaymentFailed   Kind = "payment.failed"
	KindPaymentRefunded Kind = "payment.refunded"
	KindBalanceLow      Kind = "balance.low"
	KindBonusGranted    Kind = "bonus.granted"

	// Поддержка
	KindSupportReply  Kind = "support.reply"
	KindSupportStatus Kind = "support.status"

	// Инфраструктура
	KindNodeMaintenance Kind = "node.maintenance"
	KindNodeOffline     Kind = "node.offline"
	KindDiskLow         Kind = "disk.low"

	// Безопасность
	KindNewLogin       Kind = "security.new_login"
	KindPasswordChange Kind = "security.password"
	KindTwoFactor      Kind = "security.2fa"

	// Система
	KindPanelUpdate Kind = "panel.update"
	KindAnnounce    Kind = "system.announce"
)

// Severity — насколько событие важно. Управляет цветом в панели и тем, попадает
// ли событие в каналы, помеченные «только важное».
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Group — раздел, в который событие попадает в интерфейсе.
type Group string

const (
	GroupServers  Group = "servers"
	GroupBilling  Group = "billing"
	GroupSupport  Group = "support"
	GroupSecurity Group = "security"
	GroupSystem   Group = "system"
)

var groupTitles = map[Group]string{
	GroupServers:  "Серверы",
	GroupBilling:  "Биллинг",
	GroupSupport:  "Поддержка",
	GroupSecurity: "Безопасность",
	GroupSystem:   "Система",
}

// Def — как событие показывается и куда доставляется по умолчанию.
//
// Раньше эти сведения приходилось угадывать из строки типа: справочник категорий
// знал ключ «server», а в базу писали «server.expiring», совпадений не было ни
// одного, и почти всё уведомление вырождалось в «Система» с общей иконкой.
// Здесь разойтись негде — тип и его описание объявлены рядом.
type Def struct {
	Kind     Kind
	Group    Group
	Icon     string   // имя иконки remixicon
	Severity Severity // важность по умолчанию, событие может её повысить

	// Channels — куда доставлять, если получатель не сказал иначе. Внутри панели
	// уведомление появляется всегда, здесь только внешние каналы.
	Channels []Channel

	// Digest — событие можно откладывать и слать пачкой. Для критичных ложно:
	// «сервер упал» через час бесполезно.
	Digest bool
}

var defs = map[Kind]Def{
	KindServerReady: {Group: GroupServers, Icon: "ri-check-line", Severity: SeverityInfo,
		Channels: []Channel{ChannelEmail}},
	KindServerFailed: {Group: GroupServers, Icon: "ri-error-warning-line", Severity: SeverityCritical,
		Channels: []Channel{ChannelEmail, ChannelTelegram, ChannelDiscord}},
	KindServerDown: {Group: GroupServers, Icon: "ri-shut-down-line", Severity: SeverityCritical,
		Channels: []Channel{ChannelEmail, ChannelTelegram, ChannelDiscord}},
	KindServerExpiring: {Group: GroupServers, Icon: "ri-time-line", Severity: SeverityWarning,
		Channels: []Channel{ChannelEmail, ChannelTelegram}},
	KindServerSuspended: {Group: GroupServers, Icon: "ri-pause-circle-line", Severity: SeverityCritical,
		Channels: []Channel{ChannelEmail, ChannelTelegram, ChannelDiscord}},
	// Блокировка администратором: сервер остановлен и управление закрыто.
	// Молчать об этом нельзя — иначе владелец видит нерабочий сервер и не
	// знает ни причины, ни к кому идти.
	KindServerBlocked: {Group: GroupServers, Icon: "ri-lock-line", Severity: SeverityCritical,
		Channels: []Channel{ChannelEmail, ChannelTelegram, ChannelDiscord}},
	KindServerUnblocked: {Group: GroupServers, Icon: "ri-lock-unlock-line", Severity: SeverityInfo,
		Channels: []Channel{ChannelEmail, ChannelTelegram}},
	KindServerRenewed: {Group: GroupServers, Icon: "ri-refresh-line", Severity: SeverityInfo,
		Channels: []Channel{ChannelEmail}, Digest: true},
	KindServerDeleted: {Group: GroupServers, Icon: "ri-delete-bin-line", Severity: SeverityWarning,
		Channels: []Channel{ChannelEmail}},
	KindServerMigrated: {Group: GroupServers, Icon: "ri-server-line", Severity: SeverityInfo,
		Channels: []Channel{ChannelEmail}, Digest: true},
	KindBackupReady: {Group: GroupServers, Icon: "ri-archive-line", Severity: SeverityInfo,
		Digest: true},
	KindBackupFailed: {Group: GroupServers, Icon: "ri-archive-line", Severity: SeverityWarning,
		Channels: []Channel{ChannelEmail}},

	KindPaymentReceived: {Group: GroupBilling, Icon: "ri-bank-card-line", Severity: SeverityInfo,
		Channels: []Channel{ChannelEmail}},
	KindPaymentFailed: {Group: GroupBilling, Icon: "ri-bank-card-line", Severity: SeverityWarning,
		Channels: []Channel{ChannelEmail, ChannelTelegram}},
	KindPaymentRefunded: {Group: GroupBilling, Icon: "ri-refund-2-line", Severity: SeverityInfo,
		Channels: []Channel{ChannelEmail}},
	KindBalanceLow: {Group: GroupBilling, Icon: "ri-wallet-line", Severity: SeverityWarning,
		Channels: []Channel{ChannelEmail, ChannelTelegram}},
	KindBonusGranted: {Group: GroupBilling, Icon: "ri-gift-line", Severity: SeverityInfo,
		Digest: true},

	KindSupportReply: {Group: GroupSupport, Icon: "ri-customer-service-line", Severity: SeverityInfo,
		Channels: []Channel{ChannelEmail, ChannelTelegram}},
	KindSupportStatus: {Group: GroupSupport, Icon: "ri-customer-service-line", Severity: SeverityInfo,
		Digest: true},

	KindNodeMaintenance: {Group: GroupSystem, Icon: "ri-tools-line", Severity: SeverityWarning,
		Channels: []Channel{ChannelEmail}},
	KindNodeOffline: {Group: GroupSystem, Icon: "ri-cloud-off-line", Severity: SeverityCritical,
		Channels: []Channel{ChannelEmail, ChannelTelegram, ChannelDiscord}},
	KindDiskLow: {Group: GroupSystem, Icon: "ri-hard-drive-2-line", Severity: SeverityWarning,
		Channels: []Channel{ChannelEmail, ChannelTelegram}},

	// Безопасность идёт почтой всегда и не откладывается в дайджест: смысл
	// такого письма в том, чтобы владелец успел среагировать.
	KindNewLogin: {Group: GroupSecurity, Icon: "ri-login-circle-line", Severity: SeverityWarning,
		Channels: []Channel{ChannelEmail}},
	KindPasswordChange: {Group: GroupSecurity, Icon: "ri-lock-line", Severity: SeverityWarning,
		Channels: []Channel{ChannelEmail}},
	KindTwoFactor: {Group: GroupSecurity, Icon: "ri-shield-keyhole-line", Severity: SeverityWarning,
		Channels: []Channel{ChannelEmail}},

	KindPanelUpdate: {Group: GroupSystem, Icon: "ri-download-2-line", Severity: SeverityInfo,
		Digest: true},
	KindAnnounce: {Group: GroupSystem, Icon: "ri-megaphone-line", Severity: SeverityInfo},
}

func init() {
	// Ключ карты и поле Kind обязаны совпадать: иначе DefFor вернёт описание
	// чужого события, и разницу заметят не сразу.
	for k, d := range defs {
		d.Kind = k
		defs[k] = d
	}
}

// DefFor возвращает описание события.
//
// Для неизвестного типа отдаётся запасной вариант, а не пустая структура: старые
// уведомления в базе хранят типы, которых в справочнике уже может не быть, и
// показать их всё равно нужно.
func DefFor(kind Kind) Def {
	if d, ok := defs[kind]; ok {
		return d
	}
	return Def{Kind: kind, Group: GroupSystem, Icon: "ri-information-line", Severity: SeverityInfo}
}

// Known сообщает, объявлено ли событие в справочнике.
func Known(kind Kind) bool {
	_, ok := defs[kind]
	return ok
}

// Kinds возвращает все объявленные события, отсортированные для устойчивости.
func Kinds() []Kind {
	out := make([]Kind, 0, len(defs))
	for k := range defs {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// GroupTitle — человеческое название раздела.
func GroupTitle(g Group) string {
	if t, ok := groupTitles[g]; ok {
		return t
	}
	return groupTitles[GroupSystem]
}

// Groups возвращает разделы в порядке показа.
func Groups() []Group {
	return []Group{GroupServers, GroupBilling, GroupSupport, GroupSecurity, GroupSystem}
}
