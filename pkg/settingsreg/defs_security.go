package settingsreg

import (
	"strconv"
	"strings"
)

const (
	GroupSecurity = "security"
)

var (
	AuthAccessTTL = def(Setting{
		Key: "auth.access_ttl_min", Group: GroupSecurity, Section: "tokens",
		Kind: KindInt, Default: "60", Min: 5, Max: 1440, Unit: UnitMin,
	})
	AuthRefreshTTLDefault = def(Setting{
		Key: "auth.refresh_ttl_default_days", Group: GroupSecurity, Section: "tokens",
		Kind: KindInt, Default: "7", Min: 1, Max: 90, Unit: UnitDay,
	})
	AuthRefreshTTLSession = def(Setting{
		Key: "auth.refresh_ttl_session_hours", Group: GroupSecurity, Section: "tokens",
		Kind: KindInt, Default: "24", Min: 1, Max: 168, Unit: UnitHour,
	})
	AuthRefreshTTLRemember = def(Setting{
		Key: "auth.refresh_ttl_remember_days", Group: GroupSecurity, Section: "tokens",
		Kind: KindInt, Default: "30", Min: 1, Max: 365, Unit: UnitDay,
	})
	AuthRefreshGrace = def(Setting{
		Key: "auth.refresh_grace_sec", Group: GroupSecurity, Section: "tokens",
		Kind: KindInt, Default: "30", Min: 0, Max: 300, Unit: UnitSec,
	})
	AuthMaxSavedLogins = def(Setting{
		Key: "auth.saved_logins_max", Group: GroupSecurity, Section: "tokens",
		Kind: KindInt, Default: "6", Min: 1, Max: 12, Unit: UnitCount,
	})
	AuthTwoFAChallengeTTL = def(Setting{
		Key: "auth.twofa_challenge_ttl_min", Group: GroupSecurity, Section: "tokens",
		Kind: KindInt, Default: "5", Min: 1, Max: 30, Unit: UnitMin,
	})

	AuthResetLinkTTL = def(Setting{
		Key: "auth.reset_link_ttl_min", Group: GroupSecurity, Section: "links",
		Kind: KindInt, Default: "120", Min: 10, Max: 4320, Unit: UnitMin,
	})
	AuthRegisterConfirmTTL = def(Setting{
		Key: "auth.register_confirm_ttl_hours", Group: GroupSecurity, Section: "links",
		Kind: KindInt, Default: "24", Min: 1, Max: 168, Unit: UnitHour,
	})
	AuthEmailChangeTTL = def(Setting{
		Key: "auth.email_change_ttl_hours", Group: GroupSecurity, Section: "links",
		Kind: KindInt, Default: "24", Min: 1, Max: 168, Unit: UnitHour,
	})
	AuthEmailVerifyTTL = def(Setting{
		Key: "auth.email_verify_ttl_min", Group: GroupSecurity, Section: "links",
		Kind: KindInt, Default: "60", Min: 10, Max: 4320, Unit: UnitMin,
	})
	AuthExistsMailInterval = def(Setting{
		Key: "auth.exists_mail_interval_min", Group: GroupSecurity, Section: "links",
		Kind: KindInt, Default: "60", Min: 1, Max: 1440, Unit: UnitMin,
	})

	AuthRegistrationEnabled = def(Setting{
		Key: "auth.registration_enabled", Group: GroupSecurity, Section: "access",
		Kind: KindBool, Default: "1", Public: true,
	})
	AuthRegisterConfirmEmail = def(Setting{
		Key: "auth.register_confirm_email", Group: GroupSecurity, Section: "access",
		Kind: KindBool, Default: "1",
	})
	AuthPasswordMinLength = def(Setting{
		Key: "auth.password_min_length", Group: GroupSecurity, Section: "access",
		Kind: KindInt, Default: "8", Min: 6, Max: 64, Unit: UnitCount, Public: true,
	})
	AuthPersonalTokenLimit = def(Setting{
		Key: "auth.personal_token_limit", Group: GroupSecurity, Section: "access",
		Kind: KindInt, Default: "10", Min: 1, Max: 100, Unit: UnitCount,
	})

	AuthLoginAttempts = def(Setting{
		Key: "auth.login_attempts", Group: GroupSecurity, Section: "attempts",
		Kind: KindInt, Default: "15", Min: 3, Max: 200, Unit: UnitCount,
	})
	AuthLoginWindow = def(Setting{
		Key: "auth.login_window_min", Group: GroupSecurity, Section: "attempts",
		Kind: KindInt, Default: "1", Min: 1, Max: 60, Unit: UnitMin,
	})
	AuthRegisterPerHour = def(Setting{
		Key: "auth.register_per_hour", Group: GroupSecurity, Section: "attempts",
		Kind: KindInt, Default: "20", Min: 1, Max: 500, Unit: UnitCount,
	})
	AuthRegisterDupPerHour = def(Setting{
		Key: "auth.register_dup_per_hour", Group: GroupSecurity, Section: "attempts",
		Kind: KindInt, Default: "5", Min: 1, Max: 100, Unit: UnitCount,
	})
	AuthRegisterMailPerHour = def(Setting{
		Key: "auth.register_mail_per_hour", Group: GroupSecurity, Section: "attempts",
		Kind: KindInt, Default: "3", Min: 1, Max: 50, Unit: UnitCount,
	})
	AuthForgotAttempts = def(Setting{
		Key: "auth.forgot_attempts", Group: GroupSecurity, Section: "attempts",
		Kind: KindInt, Default: "5", Min: 1, Max: 100, Unit: UnitCount,
	})
	AuthResetAttempts = def(Setting{
		Key: "auth.reset_attempts", Group: GroupSecurity, Section: "attempts",
		Kind: KindInt, Default: "10", Min: 1, Max: 100, Unit: UnitCount,
	})
	AuthTwoFAIPAttempts = def(Setting{
		Key: "auth.twofa_ip_attempts", Group: GroupSecurity, Section: "attempts",
		Kind: KindInt, Default: "20", Min: 3, Max: 200, Unit: UnitCount,
	})
	AuthTwoFAChallengeTries = def(Setting{
		Key: "auth.twofa_challenge_tries", Group: GroupSecurity, Section: "attempts",
		Kind: KindInt, Default: "5", Min: 1, Max: 20, Unit: UnitCount,
	})
	AuthTwoFAUserAttempts = def(Setting{
		Key: "auth.twofa_user_attempts", Group: GroupSecurity, Section: "attempts",
		Kind: KindInt, Default: "10", Min: 3, Max: 100, Unit: UnitCount,
	})
	AuthAccountAttempts = def(Setting{
		Key: "auth.account_attempts", Group: GroupSecurity, Section: "attempts",
		Kind: KindInt, Default: "10", Min: 3, Max: 100, Unit: UnitCount,
	})
	AuthSocialAttempts = def(Setting{
		Key: "auth.social_attempts", Group: GroupSecurity, Section: "attempts",
		Kind: KindInt, Default: "20", Min: 3, Max: 200, Unit: UnitCount,
	})
	AuthConfirmAttempts = def(Setting{
		Key: "auth.confirm_attempts", Group: GroupSecurity, Section: "attempts",
		Kind: KindInt, Default: "30", Min: 3, Max: 300, Unit: UnitCount,
	})
	AuthAttemptsWindow = def(Setting{
		Key: "auth.attempts_window_min", Group: GroupSecurity, Section: "attempts",
		Kind: KindInt, Default: "15", Min: 1, Max: 240, Unit: UnitMin,
	})

	AuthIPBlockWindow = def(Setting{
		Key: "auth.ip_block_window_min", Group: GroupSecurity, Section: "ipblock",
		Kind: KindInt, Default: "15", Min: 1, Max: 1440, Unit: UnitMin,
	})
	AuthIPBlockThreshold = def(Setting{
		Key: "auth.ip_block_threshold", Group: GroupSecurity, Section: "ipblock",
		Kind: KindInt, Default: "10", Min: 3, Max: 500, Unit: UnitCount,
	})
	AuthIPBlockDuration = def(Setting{
		Key: "auth.ip_block_duration_min", Group: GroupSecurity, Section: "ipblock",
		Kind: KindInt, Default: "30", Min: 1, Max: 43200, Unit: UnitMin,
	})

	SecurityWriteRPM = def(Setting{
		Key: "security.write_rpm", Group: GroupSecurity, Section: "requests",
		Kind: KindInt, Default: "600", Min: 30, Max: 100000, Unit: UnitCount,
	})
)

func (s *Setting) SetDefault(value string) {
	norm, err := s.Normalize(strings.TrimSpace(value))
	if err != nil {
		return
	}
	s.Default = norm
}

func (s *Setting) SetDefaultInt(n int) {
	s.SetDefault(strconv.Itoa(n))
}
