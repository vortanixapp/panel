package mail

import (
	"fmt"
	"net/smtp"
	"strings"
)

type Config struct {
	Host      string
	Port      string
	User      string
	Pass      string
	From      string
	DevExpose bool
}

func (c Config) Enabled() bool {
	return strings.TrimSpace(c.Host) != ""
}

func (c Config) Send(to, subject, body string) error {
	if !c.Enabled() {
		return fmt.Errorf("smtp not configured")
	}
	addr := c.Host + ":" + c.Port
	from := c.From
	if from == "" {
		from = c.User
	}
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body))
	auth := smtp.PlainAuth("", c.User, c.Pass, c.Host)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

func VerificationBody(verifyURL string) string {
	return fmt.Sprintf(`<p>Подтвердите email, перейдя по ссылке:</p><p><a href="%s">Подтвердить email</a></p>`, verifyURL)
}

func PasswordResetBody(resetURL string) string {
	return fmt.Sprintf(
		`<p>Вы запросили восстановление пароля в панели Vortanix.</p>`+
			`<p><a href="%s">Задать новый пароль</a></p>`+
			`<p>Ссылка действует два часа. Если вы не запрашивали восстановление, `+
			`просто удалите это письмо — пароль останется прежним.</p>`,
		resetURL)
}
