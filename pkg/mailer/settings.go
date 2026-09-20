package mailer

import "strings"

const (
	SettingMailer      = "mail.default"
	SettingHost        = "mail.mailers.smtp.host"
	SettingPort        = "mail.mailers.smtp.port"
	SettingScheme      = "mail.mailers.smtp.scheme"
	SettingUser        = "mail.mailers.smtp.username"
	SettingPassword    = "mail.mailers.smtp.password"
	SettingFromAddress = "mail.from.address"
	SettingFromName    = "mail.from.name"
)

func FromSettings(base Config, values map[string]string) Config {
	cfg := base
	if host := strings.TrimSpace(values[SettingHost]); host != "" {
		cfg.Host = host
		cfg.Port = strings.TrimSpace(values[SettingPort])
		cfg.User = strings.TrimSpace(values[SettingUser])
		cfg.Pass = values[SettingPassword]
		cfg.Scheme = strings.TrimSpace(values[SettingScheme])
	}
	if from := strings.TrimSpace(values[SettingFromAddress]); from != "" {
		cfg.FromAddress = from
	}
	if name := strings.TrimSpace(values[SettingFromName]); name != "" {
		cfg.FromName = name
	}
	if name := strings.TrimSpace(values[SettingMailer]); name != "" {
		cfg.Mailer = name
	}
	return cfg
}
