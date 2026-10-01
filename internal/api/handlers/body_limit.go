package handlers

import (
	"net/http"
	"strings"
)

const (
	jsonBodyLimit   = 64 << 20
	publicBodyLimit = 1 << 20
)

func JSONBodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && r.Body != http.NoBody && !isMultipart(r.Header.Get("Content-Type")) {
			limit := int64(jsonBodyLimit)
			if isPublicAuthPath(r.URL.Path) {
				limit = publicBodyLimit
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}

func isMultipart(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "multipart/")
}

func isPublicAuthPath(path string) bool {
	return strings.HasPrefix(path, "/v1/auth/") || path == "/v1/tenants/bootstrap"
}
