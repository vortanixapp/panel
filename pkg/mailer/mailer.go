package mailer

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

const (
	DefaultPort    = "587"
	DefaultTimeout = 20 * time.Second

	MailerSMTP  = "smtp"
	MailerLog   = "log"
	MailerArray = "array"
)

var ErrNotConfigured = errors.New("почта не настроена: не задан адрес SMTP-сервера")

var tlsConfigFor = func(host string) *tls.Config {
	return &tls.Config{ServerName: host}
}

type Config struct {
	Mailer      string
	Host        string
	Port        string
	Scheme      string
	User        string
	Pass        string
	FromAddress string
	FromName    string
	Timeout     time.Duration
	DevExpose   bool
}

type Message struct {
	To      string
	Subject string
	HTML    string
	Text    string
}

func (c Config) mailer() string {
	switch strings.ToLower(strings.TrimSpace(c.Mailer)) {
	case MailerLog:
		return MailerLog
	case MailerArray:
		return MailerArray
	default:
		return MailerSMTP
	}
}

func (c Config) MailerName() string {
	return c.mailer()
}

func (c Config) Silent() bool {
	return c.mailer() != MailerSMTP
}

func (c Config) Configured() bool {
	return strings.TrimSpace(c.Host) != ""
}

func (c Config) Enabled() bool {
	return c.Configured() && !c.Silent()
}

func (c Config) port() string {
	if p := strings.TrimSpace(c.Port); p != "" {
		return p
	}
	return DefaultPort
}

func (c Config) implicitTLS() bool {
	switch strings.ToLower(strings.TrimSpace(c.Scheme)) {
	case "smtps", "ssl", "tls_implicit":
		return true
	}
	return c.port() == "465"
}

func (c Config) From() string {
	from := strings.TrimSpace(c.FromAddress)
	if from == "" {
		from = strings.TrimSpace(c.User)
	}
	if from == "" && c.Configured() {
		from = "no-reply@" + strings.TrimSpace(c.Host)
	}
	return from
}

func (c Config) fromHeader() string {
	from := c.From()
	name := strings.TrimSpace(c.FromName)
	if name == "" {
		return from
	}
	return mime.QEncoding.Encode("utf-8", name) + " <" + from + ">"
}

func (c Config) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return DefaultTimeout
}

func ValidAddress(address string) bool {
	address = strings.TrimSpace(address)
	if address == "" || strings.ContainsAny(address, "\r\n") {
		return false
	}
	parsed, err := mail.ParseAddress(address)
	return err == nil && parsed.Address == address
}

func (c Config) Send(ctx context.Context, m Message) error {
	if !c.Configured() {
		return ErrNotConfigured
	}
	to := strings.TrimSpace(m.To)
	if !ValidAddress(to) {
		return fmt.Errorf("адрес получателя выглядит неверно: %q", m.To)
	}
	from := c.From()
	if !ValidAddress(from) {
		return fmt.Errorf("адрес отправителя выглядит неверно: %q", from)
	}

	done := make(chan error, 1)
	go func() { done <- c.deliver(Message{To: to, Subject: m.Subject, HTML: m.HTML, Text: m.Text}, from) }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c Config) deliver(m Message, from string) error {
	host := strings.TrimSpace(c.Host)
	addr := net.JoinHostPort(host, c.port())
	timeout := c.timeout()

	conn, err := (&net.Dialer{Timeout: timeout}).Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("не удалось подключиться к %s: %w", addr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))

	secure := false
	if c.implicitTLS() {
		tlsConn := tls.Client(conn, tlsConfigFor(host))
		if err := tlsConn.Handshake(); err != nil {
			_ = conn.Close()
			return fmt.Errorf("не удалось установить TLS с %s: %w", addr, err)
		}
		conn, secure = tlsConn, true
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("сервер %s не ответил приветствием: %w", addr, err)
	}
	defer func() { _ = client.Close() }()

	if !secure {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsConfigFor(host)); err != nil {
				return fmt.Errorf("сервер %s не принял STARTTLS: %w", addr, err)
			}
			secure = true
		}
	}

	if user := strings.TrimSpace(c.User); user != "" {
		if !secure {
			return fmt.Errorf("сервер %s не поддерживает шифрование, а вход по логину без него небезопасен: включите TLS или порт 465", addr)
		}
		if err := client.Auth(authFor(client, user, c.Pass, host)); err != nil {
			return fmt.Errorf("сервер %s не принял логин и пароль: %w", addr, err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("сервер %s отклонил отправителя %s: %w", addr, from, err)
	}
	if err := client.Rcpt(m.To); err != nil {
		return fmt.Errorf("сервер %s отклонил получателя %s: %w", addr, m.To, err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("сервер %s не принял письмо: %w", addr, err)
	}
	if _, err := w.Write(c.build(m, from)); err != nil {
		_ = w.Close()
		return fmt.Errorf("письмо не передано на %s: %w", addr, err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("сервер %s не принял письмо: %w", addr, err)
	}
	return client.Quit()
}

func authFor(client *smtp.Client, user, pass, host string) smtp.Auth {
	mechanisms := ""
	if ok, list := client.Extension("AUTH"); ok {
		mechanisms = strings.ToUpper(list)
	}
	if strings.Contains(mechanisms, "PLAIN") || mechanisms == "" {
		return smtp.PlainAuth("", user, pass, host)
	}
	if strings.Contains(mechanisms, "LOGIN") {
		return loginAuth{user: user, pass: pass, host: host}
	}
	if strings.Contains(mechanisms, "CRAM-MD5") {
		return smtp.CRAMMD5Auth(user, pass)
	}
	return smtp.PlainAuth("", user, pass, host)
}

type loginAuth struct {
	user string
	pass string
	host string
}

func (a loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS {
		return "", nil, errors.New("вход LOGIN доступен только по защищённому соединению")
	}
	if server.Name != a.host {
		return "", nil, errors.New("сервер представился другим именем")
	}
	return "LOGIN", nil, nil
}

func (a loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	switch strings.ToLower(strings.TrimSpace(string(fromServer))) {
	case "username:":
		return []byte(a.user), nil
	case "password:":
		return []byte(a.pass), nil
	}
	return nil, fmt.Errorf("неожиданный запрос сервера при входе: %q", string(fromServer))
}

func (c Config) build(m Message, from string) []byte {
	boundary := "vtx" + randomHex(16)
	subject := mime.QEncoding.Encode("utf-8", strings.ReplaceAll(strings.TrimSpace(m.Subject), "\n", " "))
	html := strings.TrimSpace(m.HTML)
	text := strings.TrimSpace(m.Text)

	var b strings.Builder
	b.WriteString("From: " + c.fromHeader() + "\r\n")
	b.WriteString("To: " + m.To + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("Message-ID: <" + randomHex(16) + "@" + messageIDHost(from) + ">\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")

	if text == "" || html == "" {
		body, contentType := html, "text/html; charset=UTF-8"
		if html == "" {
			body, contentType = text, "text/plain; charset=UTF-8"
		}
		b.WriteString("Content-Type: " + contentType + "\r\n")
		b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		b.WriteString(normalizeCRLF(body))
		return []byte(b.String())
	}

	b.WriteString(`Content-Type: multipart/alternative; boundary="` + boundary + `"` + "\r\n\r\n")
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(normalizeCRLF(text) + "\r\n\r\n")
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(normalizeCRLF(html) + "\r\n\r\n")
	b.WriteString("--" + boundary + "--\r\n")
	return []byte(b.String())
}

func messageIDHost(from string) string {
	if at := strings.LastIndex(from, "@"); at >= 0 && at+1 < len(from) {
		return from[at+1:]
	}
	return "localhost"
}

func normalizeCRLF(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\n", "\r\n")
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strings.Repeat("0", n*2)
	}
	return hex.EncodeToString(b)
}
