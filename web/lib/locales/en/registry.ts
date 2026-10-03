export const registry = {
  "admin.settings.tab.security": "Security",
  "admin.settings.tab.billing": "Billing",
  "admin.settings.tab.servers": "Servers",
  "admin.settings.tab.notifications": "Notifications",
  "admin.settings.tab.retention": "Retention",
  "admin.settings.tab.nodes": "Nodes",
  "admin.settings.tab.uploads": "Uploads",
  "admin.settings.tab.interface": "Interface",

  "admin.settings.reg.group.security":
    "Session and link lifetimes, sign-in attempt limits, IP blocking and the password policy. Values apply immediately, no restart needed.",
  "admin.settings.reg.group.notifications": "",
  "admin.settings.reg.group.retention": "",
  "admin.settings.reg.group.nodes": "",
  "admin.settings.reg.group.uploads": "",
  "admin.settings.reg.group.interface": "",

  "admin.settings.reg.empty": "No settings in this section yet",
  "admin.settings.reg.reset_field": "Default",
  "admin.settings.reg.reset_all": "Reset section",
  "admin.settings.reg.reset_all_confirm":
    "Restore every setting in this section to its default? Changes apply after you save.",
  "admin.settings.reg.range": "{min} to {max} {unit} · default {default}",
  "admin.settings.reg.default": "default {default}",
  "admin.settings.reg.error.number": "Enter a whole number",
  "admin.settings.reg.error.list": "Enter numbers separated by commas",
  "admin.settings.reg.error.range": "Allowed range is {min} to {max}",
  "admin.settings.reg.error.length": "At most {max} characters",

  "admin.settings.reg.unit.ms": "ms",
  "admin.settings.reg.unit.sec": "sec",
  "admin.settings.reg.unit.min": "min",
  "admin.settings.reg.unit.hour": "h",
  "admin.settings.reg.unit.day": "days",
  "admin.settings.reg.unit.mb": "MB",
  "admin.settings.reg.unit.percent": "%",
  "admin.settings.reg.unit.count": "pcs",

  "admin.settings.reg.section.security.tokens": "Sessions and tokens",
  "admin.settings.reg.section.security.tokens.description":
    "Lifetime of access and refresh tokens. Tokens already issued keep their previous expiry.",
  "admin.settings.reg.section.security.links": "Email links",
  "admin.settings.reg.section.security.access": "Registration and passwords",
  "admin.settings.reg.section.security.attempts": "Attempt limits",
  "admin.settings.reg.section.security.attempts.description":
    "How many times an action can be tried within the window from one IP or for one account.",
  "admin.settings.reg.section.security.ipblock": "Automatic IP blocking",
  "admin.settings.reg.section.security.ipblock.description":
    "An IP is blocked when it accumulates too many failed sign-ins within the window.",
  "admin.settings.reg.section.security.requests": "Request rate",

  "admin.settings.reg.auth.access_ttl_min": "Access token lifetime",
  "admin.settings.reg.auth.access_ttl_min.hint": "Refreshed automatically when it expires",
  "admin.settings.reg.auth.refresh_ttl_default_days": "Default refresh token lifetime",
  "admin.settings.reg.auth.refresh_ttl_session_hours": "Session without “remember me”",
  "admin.settings.reg.auth.refresh_ttl_remember_days": "Session with “remember me”",
  "admin.settings.reg.auth.refresh_grace_sec": "Refresh token reuse window",
  "admin.settings.reg.auth.refresh_grace_sec.hint":
    "Covers a race between two tabs; after the window, replaying an old token ends the session",
  "admin.settings.reg.auth.saved_logins_max": "Remembered accounts in the switcher",
  "admin.settings.reg.auth.twofa_challenge_ttl_min": "Time to enter the 2FA code",

  "admin.settings.reg.auth.reset_link_ttl_min": "Password reset link",
  "admin.settings.reg.auth.register_confirm_ttl_hours": "Registration confirmation",
  "admin.settings.reg.auth.email_change_ttl_hours": "Email change confirmation",
  "admin.settings.reg.auth.email_verify_ttl_min": "Address verification",
  "admin.settings.reg.auth.exists_mail_interval_min": "“Address already taken” email no more often than",

  "admin.settings.reg.auth.registration_enabled": "Allow registration",
  "admin.settings.reg.auth.registration_enabled.hint":
    "Turn off to close new account creation, including social sign-in",
  "admin.settings.reg.auth.register_confirm_email": "Confirm registration by email",
  "admin.settings.reg.auth.register_confirm_email.hint":
    "Only works when outgoing mail is configured",
  "admin.settings.reg.auth.password_min_length": "Minimum password length",
  "admin.settings.reg.auth.password_min_length.hint": "Maximum is 72 bytes, the bcrypt limit",
  "admin.settings.reg.auth.personal_token_limit": "Personal API tokens per user",

  "admin.settings.reg.auth.login_attempts": "Sign-in attempts",
  "admin.settings.reg.auth.login_window_min": "Sign-in attempts window",
  "admin.settings.reg.auth.register_per_hour": "Registrations per hour",
  "admin.settings.reg.auth.register_dup_per_hour": "Taken email retries per hour",
  "admin.settings.reg.auth.register_mail_per_hour": "Confirmation emails per address per hour",
  "admin.settings.reg.auth.forgot_attempts": "Password reset requests",
  "admin.settings.reg.auth.reset_attempts": "Attempts to set a new password",
  "admin.settings.reg.auth.twofa_ip_attempts": "2FA attempts per IP",
  "admin.settings.reg.auth.twofa_challenge_tries": "Code attempts per sign-in",
  "admin.settings.reg.auth.twofa_user_attempts": "2FA attempts per account",
  "admin.settings.reg.auth.account_attempts": "Password change and 2FA codes",
  "admin.settings.reg.auth.social_attempts": "Social sign-in and email change",
  "admin.settings.reg.auth.confirm_attempts": "Registration confirmation",
  "admin.settings.reg.auth.attempts_window_min": "Window for the other limits",

  "admin.settings.reg.auth.ip_block_window_min": "Failure counting window",
  "admin.settings.reg.auth.ip_block_threshold": "Failed sign-in threshold",
  "admin.settings.reg.auth.ip_block_duration_min": "Block duration",

  "admin.settings.reg.security.write_rpm": "Write requests per minute",
  "admin.settings.reg.security.write_rpm.hint":
    "Per user or API key: POST, PUT, PATCH and DELETE",
};
