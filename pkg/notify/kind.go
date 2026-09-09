package notify

import "sort"

type Kind string

const (
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

	KindPaymentReceived Kind = "payment.received"
	KindPaymentFailed   Kind = "payment.failed"
	KindPaymentRefunded Kind = "payment.refunded"
	KindBalanceLow      Kind = "balance.low"
	KindBonusGranted    Kind = "bonus.granted"

	KindSupportReply  Kind = "support.reply"
	KindSupportStatus Kind = "support.status"

	KindNodeMaintenance Kind = "node.maintenance"
	KindNodeOffline     Kind = "node.offline"
	KindDiskLow         Kind = "disk.low"

	KindNewLogin       Kind = "security.new_login"
	KindPasswordChange Kind = "security.password"
	KindTwoFactor      Kind = "security.2fa"

	KindPanelUpdate Kind = "panel.update"
	KindAnnounce    Kind = "system.announce"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

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

type Def struct {
	Kind     Kind
	Group    Group
	Icon     string
	Severity Severity

	Channels []Channel

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
	for k, d := range defs {
		d.Kind = k
		defs[k] = d
	}
}

func DefFor(kind Kind) Def {
	if d, ok := defs[kind]; ok {
		return d
	}
	return Def{Kind: kind, Group: GroupSystem, Icon: "ri-information-line", Severity: SeverityInfo}
}

func Known(kind Kind) bool {
	_, ok := defs[kind]
	return ok
}

func Kinds() []Kind {
	out := make([]Kind, 0, len(defs))
	for k := range defs {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func GroupTitle(g Group) string {
	if t, ok := groupTitles[g]; ok {
		return t
	}
	return groupTitles[GroupSystem]
}

func Groups() []Group {
	return []Group{GroupServers, GroupBilling, GroupSupport, GroupSecurity, GroupSystem}
}
