package mail

import (
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/mailer"
	"github.com/vortanixapp/panel/pkg/mailtpl"
)

type Letter struct {
	Subject     string
	Title       string
	Body        string
	ActionLabel string
	ActionURL   string
}

func (l Letter) Message(brand mailtpl.Brand) mailer.Message {
	m := mailtpl.Message{
		Title:       l.Title,
		Body:        mailtpl.Paragraphs(l.Body),
		ActionLabel: l.ActionLabel,
		ActionURL:   l.ActionURL,
	}
	return mailer.Message{
		Subject: l.Subject,
		HTML:    mailtpl.Render(brand, m),
		Text:    mailtpl.PlainText(brand, m),
	}
}

func letter(l i18n.Localizer, prefix, actionURL string, params i18n.Params) Letter {
	out := Letter{
		Subject:   l.T(prefix + ".subject"),
		Title:     l.T(prefix+".title", params),
		Body:      l.T(prefix+".body", params),
		ActionURL: actionURL,
	}
	if actionURL != "" {
		out.ActionLabel = l.T(prefix + ".action")
	}
	return out
}

func VerificationEmail(l i18n.Localizer, brand mailtpl.Brand, verifyURL string) mailer.Message {
	return letter(l, "mail.verify", verifyURL, nil).Message(brand)
}

func PasswordResetEmail(l i18n.Localizer, brand mailtpl.Brand, resetURL string) mailer.Message {
	return letter(l, "mail.reset", resetURL, nil).Message(brand)
}

func EmailChangeEmail(l i18n.Localizer, brand mailtpl.Brand, email, confirmURL string) mailer.Message {
	return letter(l, "mail.email_change", confirmURL, i18n.Params{"email": email}).Message(brand)
}

func TestEmail(l i18n.Localizer, brand mailtpl.Brand, to string) mailer.Message {
	return letter(l, "mail.test", "", i18n.Params{"email": to}).Message(brand)
}
