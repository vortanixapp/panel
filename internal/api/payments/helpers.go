package payments

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

var httpClient = &http.Client{Timeout: 25 * time.Second}

var ErrBadSignature = errors.New("подпись уведомления не сходится")

func notConfigured(code string) error {
	return fmt.Errorf("%s: не заполнены ключи в настройках провайдера", code)
}

func strCfg(cfg map[string]any, key string) string {
	if cfg == nil {
		return ""
	}
	v, ok := cfg[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "1"
		}
		return "0"
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func boolCfg(cfg map[string]any, key string) bool {
	switch strings.ToLower(strCfg(cfg, key)) {
	case "1", "true", "on", "yes":
		return true
	}
	return false
}

func urlCfg(cfg map[string]any, key, fallback string) string {
	if v := strings.TrimRight(strCfg(cfg, key), "/"); v != "" {
		return v
	}
	return strings.TrimRight(fallback, "/")
}

func digestHex(h hash.Hash, s string) string {
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func md5Hex(s string) string    { return digestHex(md5.New(), s) }
func sha1Hex(s string) string   { return digestHex(sha1.New(), s) }
func sha256Hex(s string) string { return digestHex(sha256.New(), s) }

func hmacSum(newHash func() hash.Hash, key, data []byte) []byte {
	mac := hmac.New(newHash, key)
	mac.Write(data)
	return mac.Sum(nil)
}

func hmacSHA256Hex(key string, data []byte) string {
	return hex.EncodeToString(hmacSum(sha256.New, []byte(key), data))
}

func hmacSHA256Base64(key string, data []byte) string {
	return base64.StdEncoding.EncodeToString(hmacSum(sha256.New, []byte(key), data))
}

func hmacSHA512Hex(key, data []byte) string {
	return hex.EncodeToString(hmacSum(sha512.New, key, data))
}

func secureEqualFold(expected, got string) bool {
	expected = strings.ToLower(strings.TrimSpace(expected))
	got = strings.ToLower(strings.TrimSpace(got))
	return expected != "" && hmac.Equal([]byte(expected), []byte(got))
}

func secureEqual(expected, got string) bool {
	expected = strings.TrimSpace(expected)
	got = strings.TrimSpace(got)
	return expected != "" && hmac.Equal([]byte(expected), []byte(got))
}

func roundCents(v float64) float64 {
	return math.Round(v*100) / 100
}

func money(v float64) string {
	return strconv.FormatFloat(roundCents(v), 'f', 2, 64)
}

var zeroDecimalCurrencies = map[string]bool{
	"BIF": true, "CLP": true, "DJF": true, "GNF": true, "JPY": true, "KMF": true,
	"KRW": true, "MGA": true, "PYG": true, "RWF": true, "UGX": true, "VND": true,
	"VUV": true, "XAF": true, "XOF": true, "XPF": true,
}

func minorUnits(v float64, currency string) int64 {
	if zeroDecimalCurrencies[strings.ToUpper(currency)] {
		return int64(math.Round(v))
	}
	return int64(math.Round(v * 100))
}

func fromMinorUnits(v float64, currency string) float64 {
	if zeroDecimalCurrencies[strings.ToUpper(currency)] {
		return v
	}
	return v / 100
}

func parseAmount(s string) float64 {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", ".")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func numberValue(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case json.Number:
		f, _ := t.Float64()
		return f
	case string:
		return parseAmount(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	}
	return 0
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func invoiceRef(in CheckoutInput) string {
	return strconv.FormatInt(in.InvoiceNo, 10)
}

func parseInvoiceNo(s string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func isUUID(s string) bool {
	_, err := uuid.Parse(strings.TrimSpace(s))
	return err == nil
}

func orderReference(n *Notification, ref string) {
	ref = strings.TrimSpace(ref)
	if isUUID(ref) {
		n.PaymentID = ref
		return
	}
	n.InvoiceNo = parseInvoiceNo(ref)
}

func metaPaymentID(meta map[string]any) string {
	for _, k := range []string{"payment_id", "vortanix_payment_id", "order_id"} {
		if v := strings.TrimSpace(fmt.Sprint(meta[k])); v != "" && v != "<nil>" {
			return v
		}
	}
	return ""
}

type reqOption func(*http.Request)

func withHeader(key, value string) reqOption {
	return func(r *http.Request) { r.Header.Set(key, value) }
}

func withBasicAuth(user, pass string) reqOption {
	return func(r *http.Request) { r.SetBasicAuth(user, pass) }
}

func withBearer(token string) reqOption {
	return withHeader("Authorization", "Bearer "+token)
}

func doRequest(ctx context.Context, method, endpoint, contentType string, body []byte, out any, opts ...reqOption) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil && contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for _, opt := range opts {
		opt(req)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(strings.TrimSpace(string(raw)), 300))
	}
	if out == nil || len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("ответ не разобран: %w (%s)", err, truncate(string(raw), 200))
	}
	return nil
}

func sendJSON(ctx context.Context, method, endpoint string, payload any, out any, opts ...reqOption) error {
	var body []byte
	switch p := payload.(type) {
	case nil:
	case []byte:
		body = p
	default:
		raw, err := json.Marshal(p)
		if err != nil {
			return err
		}
		body = raw
	}
	return doRequest(ctx, method, endpoint, "application/json", body, out, opts...)
}

func sendForm(ctx context.Context, endpoint string, form url.Values, out any, opts ...reqOption) error {
	return doRequest(ctx, http.MethodPost, endpoint, "application/x-www-form-urlencoded", []byte(form.Encode()), out, opts...)
}

func phpURLEncode(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "~", "%7E")
}

func formOf(action string, pairs ...string) *Form {
	f := &Form{Action: action, Method: http.MethodPost}
	for i := 0; i+1 < len(pairs); i += 2 {
		f.Fields = append(f.Fields, FormField{Name: pairs[i], Value: pairs[i+1]})
	}
	return f
}

func (f *Form) add(name, value string) {
	f.Fields = append(f.Fields, FormField{Name: name, Value: value})
}

func cfgOr(cfg map[string]any, key, fallback string) string {
	if v := strCfg(cfg, key); v != "" {
		return v
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" && v != "<nil>" {
			return v
		}
	}
	return ""
}

func withQuery(base string, q url.Values) string {
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + q.Encode()
}

func rootPath(raw string) string {
	u, err := url.Parse(raw)
	if err == nil && u.Host != "" && u.Path == "" {
		u.Path = "/"
		return u.String()
	}
	return raw
}

func isHexString(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}

func upper(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

func base64Text(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
