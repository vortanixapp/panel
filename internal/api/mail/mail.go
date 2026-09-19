package mail

import (
	"fmt"
	"mime"
	"net/smtp"
	"strings"

	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/mailtpl"
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
		from, to, mime.QEncoding.Encode("utf-8", subject), body))
	auth := smtp.PlainAuth("", c.User, c.Pass, c.Host)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

func VerificationEmail(l i18n.Localizer, brand mailtpl.Brand, verifyURL string) (subject, body string) {
	return l.T("mail.verify.subject"), mailtpl.Render(brand, mailtpl.Message{
		Title:       l.T("mail.verify.title"),
		Body:        mailtpl.Paragraphs(l.T("mail.verify.body")),
		ActionLabel: l.T("mail.verify.action"),
		ActionURL:   verifyURL,
	})
}

func PasswordResetEmail(l i18n.Localizer, brand mailtpl.Brand, resetURL string) (subject, body string) {
	return l.T("mail.reset.subject"), mailtpl.Render(brand, mailtpl.Message{
		Title:       l.T("mail.reset.title"),
		Body:        mailtpl.Paragraphs(l.T("mail.reset.body")),
		ActionLabel: l.T("mail.reset.action"),
		ActionURL:   resetURL,
	})
}

func EmailChangeEmail(l i18n.Localizer, brand mailtpl.Brand, email, confirmURL string) (subject, body string) {
	return l.T("mail.email_change.subject"), mailtpl.Render(brand, mailtpl.Message{
		Title:       l.T("mail.email_change.title"),
		Body:        mailtpl.Paragraphs(l.T("mail.email_change.body", i18n.Params{"email": email})),
		ActionLabel: l.T("mail.email_change.action"),
		ActionURL:   confirmURL,
	})
}

func TestEmail(l i18n.Localizer, brand mailtpl.Brand, to string) (subject, body string) {
	return l.T("mail.test.subject"), mailtpl.Render(brand, mailtpl.Message{
		Title: l.T("mail.test.title"),
		Body:  mailtpl.Paragraphs(l.T("mail.test.body", i18n.Params{"email": to})),
	})
}
