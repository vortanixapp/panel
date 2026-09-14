package mail

import (
	"fmt"
	"net/smtp"
	"strings"

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
		from, to, subject, body))
	auth := smtp.PlainAuth("", c.User, c.Pass, c.Host)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

func VerificationBody(brand mailtpl.Brand, verifyURL string) string {
	return mailtpl.Render(brand, mailtpl.Message{
		Title:       "Подтвердите email",
		Body:        mailtpl.Paragraphs("Чтобы завершить регистрацию, подтвердите адрес электронной почты."),
		ActionLabel: "Подтвердить email",
		ActionURL:   verifyURL,
	})
}

func PasswordResetBody(brand mailtpl.Brand, resetURL string) string {
	return mailtpl.Render(brand, mailtpl.Message{
		Title: "Восстановление пароля",
		Body: mailtpl.Paragraphs("Вы запросили восстановление пароля.\n\n" +
			"Ссылка действует два часа. Если вы не запрашивали восстановление, " +
			"просто удалите это письмо — пароль останется прежним."),
		ActionLabel: "Задать новый пароль",
		ActionURL:   resetURL,
	})
}

func TestBody(brand mailtpl.Brand, to string) string {
	return mailtpl.Render(brand, mailtpl.Message{
		Title: "Тестовое письмо",
		Body: mailtpl.Paragraphs("Письмо отправлено на " + to + ".\n\n" +
			"Так выглядят письма панели: логотип и акцентный цвет берутся из раздела «Оформление»."),
	})
}
