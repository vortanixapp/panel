package handlers

import (
	"net/http"
	"regexp"
	"strings"
)

func (h *Handler) licenseGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		st := h.licenseFor(r.Context())
		if !st.BlocksWrites() || allowedInReadOnly(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		writeCodedError(w, http.StatusLocked, "license_read_only",
			"панель работает в режиме только для чтения: "+st.Reason)
	})
}

var readOnlyAllowlist = []string{
	"/v1/auth/",
	"/v1/me",
	"/v1/account/",
	"/v1/email/verification-notification",
	"/v1/license/",
	"/v1/admin/license",
	"/v1/locale/",
	"/v1/notifications/",
}

var serverRuntimePaths = regexp.MustCompile(
	`^/v1/servers/[^/]+/(power|restart|start|stop|kill|console|console-ticket|console-command)`)

func allowedInReadOnly(path string) bool {
	for _, prefix := range readOnlyAllowlist {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return serverRuntimePaths.MatchString(path)
}
