import type { SettingsTab } from "./types";

export const SETTINGS_TABS: { id: SettingsTab; labelKey: string }[] = [
  { id: "main", labelKey: "admin.settings.tab.main" },
  { id: "mail", labelKey: "admin.settings.tab.mail" },
  { id: "social", labelKey: "admin.settings.tab.social" },
  { id: "dockerhub", labelKey: "admin.settings.tab.dockerhub" },
  { id: "telegram", labelKey: "admin.settings.tab.telegram" },
  { id: "files", labelKey: "admin.settings.tab.files" },
];

export const MAIL_MAILERS = ["smtp", "log", "array"] as const;

export const MAIL_SCHEMES = [
  { value: "", labelKey: "admin.settings.mail.scheme_auto" },
  { value: "smtp", labelKey: "admin.settings.mail.scheme_starttls" },
  { value: "smtps", labelKey: "admin.settings.mail.scheme_ssl" },
] as const;

export const OAUTH_PROVIDERS = [
  { name: "Google", prefix: "services.google", slug: "google" },
  { name: "Discord", prefix: "services.discord", slug: "discord" },
  { name: "VK", prefix: "services.vkontakte", slug: "vk" },
] as const;

export const FILES_STORAGE_DRIVERS = [
  { id: "ftp", label: "FTP" },
  { id: "sftp", label: "SFTP" },
  { id: "s3", label: "S3" },
] as const;

export const SETTINGS_KEY_MAP: Record<string, string> = {
  "app.site.description": "site_description",
  "app.site.domain": "site_domain",
  "app.site.ip": "site_ip",
  "app.site.subnet": "site_subnet",
  "dockerhub.username": "dockerhub_username",
  "dockerhub.token": "dockerhub_token",
  "services.recaptcha.site_key": "recaptcha_site_key",
  "services.recaptcha.secret_key": "recaptcha_secret_key",
  "auth.require_verified_email": "require_verified_email",
  "vtx_mail.server_status_notifications": "server_status_notifications",
  "mail.default": "mail_mailer",
  "mail.mailers.smtp.scheme": "mail_scheme",
  "mail.mailers.smtp.host": "mail_host",
  "mail.mailers.smtp.port": "mail_port",
  "mail.mailers.smtp.username": "mail_username",
  "mail.mailers.smtp.password": "mail_password",
  "mail.from.address": "mail_from_address",
  "mail.from.name": "mail_from_name",
  "services.google.client_id": "google_client_id",
  "services.google.client_secret": "google_client_secret",
  "services.google.redirect": "google_redirect_uri",
  "services.discord.client_id": "discord_client_id",
  "services.discord.client_secret": "discord_client_secret",
  "services.discord.redirect": "discord_redirect_uri",
  "services.vkontakte.client_id": "vk_client_id",
  "services.vkontakte.client_secret": "vk_client_secret",
  "services.vkontakte.redirect": "vk_redirect_uri",
  "telegram.notifications.enabled": "telegram_notifications_enabled",
  "telegram.notifications.bot_token": "telegram_bot_token",
  "telegram.notifications.bot_username": "telegram_bot_username",
  "telegram.notifications.admin_chat_id": "telegram_admin_chat_id",
  "files.storage.driver": "files_storage_driver",
  "files.storage.url": "files_storage_url",
  "files.storage.ftp.host": "files_storage_ftp_host",
  "files.storage.ftp.port": "files_storage_ftp_port",
  "files.storage.ftp.username": "files_storage_ftp_username",
  "files.storage.ftp.password": "files_storage_ftp_password",
  "files.storage.ftp.root": "files_storage_ftp_root",
  "files.storage.ftp.passive": "files_storage_ftp_passive",
  "files.storage.ftp.ssl": "files_storage_ftp_ssl",
  "files.storage.ftp.timeout": "files_storage_ftp_timeout",
  "files.storage.s3.key": "files_storage_s3_key",
  "files.storage.s3.secret": "files_storage_s3_secret",
  "files.storage.s3.region": "files_storage_s3_region",
  "files.storage.s3.bucket": "files_storage_s3_bucket",
  "files.storage.s3.endpoint": "files_storage_s3_endpoint",
  "files.storage.s3.use_path_style_endpoint":
    "files_storage_s3_use_path_style_endpoint",
  "files.storage.sftp.host": "files_storage_sftp_host",
  "files.storage.sftp.port": "files_storage_sftp_port",
  "files.storage.sftp.username": "files_storage_sftp_username",
  "files.storage.sftp.password": "files_storage_sftp_password",
  "files.storage.sftp.root": "files_storage_sftp_root",
  "files.storage.sftp.timeout": "files_storage_sftp_timeout",
  "files.storage.path.avatars": "files_storage_path_avatars",
  "files.storage.path.plugins": "files_storage_path_plugins",
  "files.storage.path.maps": "files_storage_path_maps",
  "files.storage.path.support": "files_storage_path_support",
};

export function isSettingsTab(value: string | null): value is SettingsTab {
  return !!value && SETTINGS_TABS.some((tab) => tab.id === value);
}
