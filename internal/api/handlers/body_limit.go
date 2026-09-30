package handlers

import (
	"net/http"
	"strings"
)

const jsonBodyLimit = 64 << 20

func JSONBodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
			r.Body = http.MaxBytesReader(w, r.Body, jsonBodyLimit)
		}
		next.ServeHTTP(w, r)
	})
}
