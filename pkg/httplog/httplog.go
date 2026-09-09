package httplog

import (
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

var secretQueryKeys = map[string]bool{
	"token":         true,
	"ticket":        true,
	"access_token":  true,
	"refresh_token": true,
	"api_token":     true,
	"password":      true,
	"secret":        true,
	"key":           true,
	"code":          true,
}

var secretPathPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^(/v1/email/verify/[^/]+/)[^/]+$`),
}

func Redact(uri string) string {
	parsed, err := url.ParseRequestURI(uri)
	if err != nil {
		if i := strings.IndexByte(uri, '?'); i >= 0 {
			return uri[:i] + "?redacted"
		}
		return uri
	}

	path := parsed.Path
	for _, re := range secretPathPatterns {
		if re.MatchString(path) {
			path = re.ReplaceAllString(path, "${1}redacted")
			break
		}
	}

	if parsed.RawQuery == "" {
		return path
	}
	values := parsed.Query()
	for key := range values {
		if secretQueryKeys[strings.ToLower(key)] {
			values.Set(key, "redacted")
		}
	}
	return path + "?" + values.Encode()
}

func Logger(next http.Handler) http.Handler {
	return middleware.RequestLogger(&formatter{})(next)
}

type formatter struct{}

func (f *formatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return &entry{
		method:    r.Method,
		uri:       scheme + "://" + r.Host + Redact(r.RequestURI),
		proto:     r.Proto,
		remote:    r.RemoteAddr,
		requestID: middleware.GetReqID(r.Context()),
	}
}

type entry struct {
	method    string
	uri       string
	proto     string
	remote    string
	requestID string
}

func (e *entry) Write(status, bytes int, _ http.Header, elapsed time.Duration, _ any) {
	prefix := ""
	if e.requestID != "" {
		prefix = "[" + e.requestID + "] "
	}
	log.Printf("%s\"%s %s %s\" from %s - %d %dB in %s",
		prefix, e.method, e.uri, e.proto, e.remote, status, bytes, elapsed)
}

func (e *entry) Panic(v any, stack []byte) {
	log.Printf("panic: %+v\n%s", v, stack)
}
