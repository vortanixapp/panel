package mail

import (
	"fmt"
	"net/smtp"
	"strings"
)

type Config struct {
	Host string
	Port string
	User string
	Pass string
	From string
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
