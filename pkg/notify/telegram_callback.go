package notify

import (
	"strings"

	"github.com/vortanixapp/panel/pkg/i18n"
)

const (
	CBMenu        = "m"
	CBServers     = "l"
	CBServer      = "s"
	CBPower       = "p"
	CBConsole     = "c"
	CBQuick       = "cq"
	CBCustom      = "cu"
	CBLogs        = "lg"
	CBPlayers     = "pl"
	CBMetrics     = "mt"
	CBBackup      = "b"
	CBBackupMake  = "bc"
	CBBackupList  = "bl"
	CBBackupAsk   = "br"
	CBBackupApply = "bx"
	CBWipe        = "w"
	CBWipeNow     = "wn"
	CBWipeRun     = "wr"
	CBWipeToggle  = "wt"
	CBWipeSkip    = "ws"
	CBWipeCancel  = "wc"
	CBBalance     = "bal"
	CBSettings    = "set"
	CBUnlink      = "ul"
	CBUnlinkApply = "ux"
	CBCancel      = "x"
)

func Callback(verb string, args ...string) string {
	if len(args) == 0 {
		return verb
	}
	return verb + ":" + strings.Join(args, ":")
}

func ParseCallback(data string) (verb string, args []string) {
	parts := strings.Split(data, ":")
	return parts[0], parts[1:]
}

type Button struct {
	Text string `json:"text"`
	Data string `json:"data"`
}

func ControlButtons(l i18n.Localizer, kind Kind, serverID string) []Button {
	if serverID == "" {
		return nil
	}
	var out []Button
	switch kind {
	case KindServerDown, KindServerFailed:
		out = append(out,
			Button{Text: l.T("notify.btn.restart"), Data: Callback(CBPower, serverID, "r")},
			Button{Text: l.T("notify.btn.logs"), Data: Callback(CBLogs, serverID)},
		)
	case KindBackupFailed:
		out = append(out, Button{Text: l.T("notify.btn.backup"), Data: Callback(CBBackupMake, serverID)})
	case KindWipeFailed:
		out = append(out, Button{Text: l.T("notify.btn.start"), Data: Callback(CBPower, serverID, "s")})
	}
	switch kind {
	case KindServerReady, KindServerDown, KindServerFailed, KindBackupReady, KindBackupFailed,
		KindWipeDone, KindWipeFailed, KindServerExpiring:
		out = append(out, Button{Text: l.T("notify.btn.manage"), Data: Callback(CBServer, serverID)})
	}
	return out
}
