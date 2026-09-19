package notify

import (
	"sort"

	"github.com/vortanixapp/panel/pkg/i18n"
)

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
	KindServerOwner     Kind = "server.owner_changed"
	KindServerExtended  Kind = "server.extended"
	KindBackupReady     Kind = "backup.ready"
	KindBackupFailed    Kind = "backup.failed"
	KindNodeMaintenance Kind = "node.maintenance"
	KindNodeOffline     Kind = "node.offline"
	KindNodeOnline      Kind = "node.online"
	KindAbuseNotice     Kind = "abuse.notice"

	KindPaymentReceived Kind = "payment.received"
	KindPaymentFailed   Kind = "payment.failed"
	KindPaymentRefunded Kind = "payment.refunded"
	KindRefundRejected  Kind = "payment.refund_rejected"
	KindBalanceLow      Kind = "balance.low"
	KindBonusGranted    Kind = "bonus.granted"

	KindSupportReply  Kind = "support.reply"
	KindSupportStatus Kind = "support.status"

	KindNewLogin       Kind = "security.new_login"
	KindPasswordChange Kind = "security.password"
	KindTwoFactor      Kind = "security.2fa"
	KindEmailChange    Kind = "security.email"
	KindSocialAccount  Kind = "security.social"
	KindRecoveryCode   Kind = "security.recovery"

	KindAnnounce Kind = "system.announce"

	KindStaffTicketNew    Kind = "staff.ticket_new"
	KindStaffTicketReply  Kind = "staff.ticket_reply"
	KindStaffAudit        Kind = "staff.audit"
	KindStaffNodeOffline  Kind = "staff.node_offline"
	KindStaffNodeOnline   Kind = "staff.node_online"
	KindDiskLow           Kind = "disk.low"
	KindPanelUpdate       Kind = "panel.update"
	KindPanelUpdateFailed Kind = "panel.update_failed"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeveritySuccess  Severity = "success"
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
	GroupStaff    Group = "staff"
)

type Def struct {
	Kind     Kind
	Group    Group
	Icon     string
	Severity Severity

	Channels []Channel

	Quiet    bool
	Required bool
}

var (
	channelsAll   = []Channel{ChannelEmail, ChannelTelegram, ChannelDiscord}
	channelsEmail = []Channel{ChannelEmail}
)

var defs = map[Kind]Def{
	KindServerReady:     {Group: GroupServers, Icon: "ri-check-line", Severity: SeveritySuccess, Channels: channelsAll},
	KindServerFailed:    {Group: GroupServers, Icon: "ri-error-warning-line", Severity: SeverityCritical, Channels: channelsAll},
	KindServerDown:      {Group: GroupServers, Icon: "ri-shut-down-line", Severity: SeverityCritical, Channels: channelsAll},
	KindServerExpiring:  {Group: GroupServers, Icon: "ri-time-line", Severity: SeverityWarning, Channels: channelsAll},
	KindServerSuspended: {Group: GroupServers, Icon: "ri-pause-circle-line", Severity: SeverityCritical, Channels: channelsAll},
	KindServerBlocked:   {Group: GroupServers, Icon: "ri-lock-line", Severity: SeverityCritical, Channels: channelsAll},
	KindServerUnblocked: {Group: GroupServers, Icon: "ri-lock-unlock-line", Severity: SeveritySuccess, Channels: channelsAll},
	KindServerRenewed:   {Group: GroupServers, Icon: "ri-refresh-line", Severity: SeveritySuccess, Channels: channelsEmail, Quiet: true},
	KindServerDeleted:   {Group: GroupServers, Icon: "ri-delete-bin-line", Severity: SeverityWarning, Channels: channelsAll},
	KindServerMigrated:  {Group: GroupServers, Icon: "ri-server-line", Severity: SeverityInfo, Channels: channelsEmail, Quiet: true},
	KindServerOwner:     {Group: GroupServers, Icon: "ri-user-shared-line", Severity: SeverityInfo, Channels: channelsEmail},
	KindServerExtended:  {Group: GroupServers, Icon: "ri-calendar-check-line", Severity: SeveritySuccess, Channels: channelsEmail, Quiet: true},
	KindBackupReady:     {Group: GroupServers, Icon: "ri-archive-line", Severity: SeveritySuccess, Quiet: true},
	KindBackupFailed:    {Group: GroupServers, Icon: "ri-archive-line", Severity: SeverityWarning, Channels: channelsAll},
	KindNodeMaintenance: {Group: GroupServers, Icon: "ri-tools-line", Severity: SeverityWarning, Channels: channelsAll},
	KindNodeOffline:     {Group: GroupServers, Icon: "ri-cloud-off-line", Severity: SeverityCritical, Channels: channelsAll},
	KindNodeOnline:      {Group: GroupServers, Icon: "ri-cloud-line", Severity: SeveritySuccess, Channels: channelsAll},
	KindAbuseNotice:     {Group: GroupServers, Icon: "ri-alarm-warning-line", Severity: SeverityWarning, Channels: channelsAll, Required: true},

	KindPaymentReceived: {Group: GroupBilling, Icon: "ri-bank-card-line", Severity: SeveritySuccess, Channels: channelsEmail},
	KindPaymentFailed:   {Group: GroupBilling, Icon: "ri-bank-card-line", Severity: SeverityWarning, Channels: channelsAll},
	KindPaymentRefunded: {Group: GroupBilling, Icon: "ri-refund-2-line", Severity: SeverityInfo, Channels: channelsEmail},
	KindRefundRejected:  {Group: GroupBilling, Icon: "ri-refund-2-line", Severity: SeverityWarning, Channels: channelsEmail},
	KindBalanceLow:      {Group: GroupBilling, Icon: "ri-wallet-line", Severity: SeverityWarning, Channels: channelsAll},
	KindBonusGranted:    {Group: GroupBilling, Icon: "ri-gift-line", Severity: SeveritySuccess, Quiet: true},

	KindSupportReply:  {Group: GroupSupport, Icon: "ri-customer-service-line", Severity: SeverityInfo, Channels: channelsAll},
	KindSupportStatus: {Group: GroupSupport, Icon: "ri-customer-service-line", Severity: SeverityInfo, Quiet: true},

	KindNewLogin:       {Group: GroupSecurity, Icon: "ri-login-circle-line", Severity: SeverityWarning, Channels: channelsAll, Required: true},
	KindPasswordChange: {Group: GroupSecurity, Icon: "ri-lock-password-line", Severity: SeverityWarning, Channels: channelsAll, Required: true},
	KindTwoFactor:      {Group: GroupSecurity, Icon: "ri-shield-keyhole-line", Severity: SeverityWarning, Channels: channelsAll, Required: true},
	KindEmailChange:    {Group: GroupSecurity, Icon: "ri-mail-settings-line", Severity: SeverityWarning, Channels: channelsAll, Required: true},
	KindSocialAccount:  {Group: GroupSecurity, Icon: "ri-links-line", Severity: SeverityWarning, Channels: channelsAll, Required: true},
	KindRecoveryCode:   {Group: GroupSecurity, Icon: "ri-key-2-line", Severity: SeverityWarning, Channels: channelsAll, Required: true},

	KindAnnounce: {Group: GroupSystem, Icon: "ri-megaphone-line", Severity: SeverityInfo},

	KindStaffTicketNew:    {Group: GroupStaff, Icon: "ri-inbox-unarchive-line", Severity: SeverityInfo, Channels: channelsAll},
	KindStaffTicketReply:  {Group: GroupStaff, Icon: "ri-chat-3-line", Severity: SeverityInfo, Channels: channelsAll},
	KindStaffAudit:        {Group: GroupStaff, Icon: "ri-shield-user-line", Severity: SeverityWarning},
	KindStaffNodeOffline:  {Group: GroupStaff, Icon: "ri-cloud-off-line", Severity: SeverityCritical, Channels: channelsAll},
	KindStaffNodeOnline:   {Group: GroupStaff, Icon: "ri-cloud-line", Severity: SeveritySuccess, Channels: channelsAll},
	KindDiskLow:           {Group: GroupStaff, Icon: "ri-hard-drive-2-line", Severity: SeverityWarning, Channels: channelsAll},
	KindPanelUpdate:       {Group: GroupStaff, Icon: "ri-download-2-line", Severity: SeverityInfo, Channels: channelsEmail},
	KindPanelUpdateFailed: {Group: GroupStaff, Icon: "ri-error-warning-line", Severity: SeverityCritical, Channels: channelsAll},
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

func KindsOf(g Group) []Kind {
	out := []Kind{}
	for _, k := range Kinds() {
		if defs[k].Group == g {
			out = append(out, k)
		}
	}
	return out
}

func GroupLabel(g Group) i18n.Msg {
	for _, known := range Groups() {
		if known == g {
			return i18n.Key("notify.group." + string(g))
		}
	}
	return i18n.Key("notify.group." + string(GroupSystem))
}

func Groups() []Group {
	return []Group{GroupServers, GroupBilling, GroupSupport, GroupSecurity, GroupSystem, GroupStaff}
}

func KnownGroup(g Group) bool {
	for _, known := range Groups() {
		if known == g {
			return true
		}
	}
	return false
}

func GroupsFor(staff bool) []Group {
	out := []Group{}
	for _, g := range Groups() {
		if g == GroupStaff && !staff {
			continue
		}
		out = append(out, g)
	}
	return out
}

func GroupChannels(g Group) []Channel {
	used := map[Channel]bool{}
	for _, k := range KindsOf(g) {
		for _, c := range defs[k].Channels {
			used[c] = true
		}
	}
	out := []Channel{}
	for _, c := range ExternalChannels() {
		if used[c] {
			out = append(out, c)
		}
	}
	return out
}

func GroupLocked(g Group, c Channel) bool {
	if c != ChannelEmail {
		return false
	}
	found := false
	for _, k := range KindsOf(g) {
		d := defs[k]
		if !hasChannel(d.Channels, c) {
			continue
		}
		if !d.Required {
			return false
		}
		found = true
	}
	return found
}

func hasChannel(list []Channel, c Channel) bool {
	for _, item := range list {
		if item == c {
			return true
		}
	}
	return false
}
