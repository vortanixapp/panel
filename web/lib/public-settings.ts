import { injectedBranding } from "@/lib/appearance";
import { parseAccountingRegions } from "@/lib/accounting-regions";

export type PublicSettings = {
  registrationEnabled: boolean;
  passwordMinLength: number;
  rentMaxBatch: number;
  topupMin: number;
  topupMax: number;
  topupDefault: number;
  topupPresets: number[];
  avatarMb: number;
  siteAssetMb: number;
  supportAttachmentMb: number;
  supportAttachmentCount: number;
  supportBodyMax: number;
  pollScalePercent: number;
  staleMs: number;
  consoleBufferLines: number;
  toastMs: number;
  pageSize: number;
  feedPageSize: number;
  activityPageSize: number;
  searchDebounceMs: number;
  cookieNoticeDays: number;
  accountingRegions: string[];
};

export const DEFAULT_PUBLIC_SETTINGS: PublicSettings = {
  registrationEnabled: true,
  passwordMinLength: 8,
  rentMaxBatch: 5,
  topupMin: 0,
  topupMax: 0,
  topupDefault: 1000,
  topupPresets: [500, 1000, 2000, 5000, 10000],
  avatarMb: 4,
  siteAssetMb: 5,
  supportAttachmentMb: 8,
  supportAttachmentCount: 5,
  supportBodyMax: 20000,
  pollScalePercent: 100,
  staleMs: 5000,
  consoleBufferLines: 4000,
  toastMs: 4000,
  pageSize: 10,
  feedPageSize: 30,
  activityPageSize: 100,
  searchDebounceMs: 300,
  cookieNoticeDays: 0,
  accountingRegions: ["cis"],
};

type Raw = Record<string, unknown> | null | undefined;

function int(raw: Raw, key: string, fallback: number, min: number, max: number): number {
  const value = raw?.[key];
  if (typeof value !== "number" || !Number.isFinite(value)) return fallback;
  return Math.min(max, Math.max(min, Math.round(value)));
}

function bool(raw: Raw, key: string, fallback: boolean): boolean {
  const value = raw?.[key];
  return typeof value === "boolean" ? value : fallback;
}

function list(raw: Raw, key: string, fallback: number[]): number[] {
  const value = raw?.[key];
  if (!Array.isArray(value)) return fallback;
  const items = value.filter((n): n is number => typeof n === "number" && Number.isFinite(n) && n > 0);
  return items.length > 0 ? items : fallback;
}

export function resolvePublicSettings(raw: Raw): PublicSettings {
  const d = DEFAULT_PUBLIC_SETTINGS;
  return {
    registrationEnabled: bool(raw, "auth.registration_enabled", d.registrationEnabled),
    passwordMinLength: int(raw, "auth.password_min_length", d.passwordMinLength, 6, 64),
    rentMaxBatch: int(raw, "billing.rent_max_batch", d.rentMaxBatch, 1, 50),
    topupMin: int(raw, "billing.topup_min", d.topupMin, 0, 100000000),
    topupMax: int(raw, "billing.topup_max", d.topupMax, 0, 100000000),
    topupDefault: int(raw, "billing.topup_default", d.topupDefault, 1, 100000000),
    topupPresets: list(raw, "billing.topup_presets", d.topupPresets),
    avatarMb: int(raw, "uploads.avatar_mb", d.avatarMb, 1, 20),
    siteAssetMb: int(raw, "uploads.site_asset_mb", d.siteAssetMb, 1, 32),
    supportAttachmentMb: int(raw, "support.attachment_mb", d.supportAttachmentMb, 1, 64),
    supportAttachmentCount: int(raw, "support.attachment_count", d.supportAttachmentCount, 1, 20),
    supportBodyMax: int(raw, "support.body_max", d.supportBodyMax, 500, 100000),
    pollScalePercent: int(raw, "ui.poll_scale_percent", d.pollScalePercent, 25, 1000),
    staleMs: int(raw, "ui.stale_ms", d.staleMs, 1000, 120000),
    consoleBufferLines: int(raw, "ui.console_buffer_lines", d.consoleBufferLines, 500, 20000),
    toastMs: int(raw, "ui.toast_ms", d.toastMs, 1000, 20000),
    pageSize: int(raw, "ui.page_size", d.pageSize, 5, 100),
    feedPageSize: int(raw, "ui.feed_page_size", d.feedPageSize, 10, 200),
    activityPageSize: int(raw, "ui.activity_page_size", d.activityPageSize, 20, 500),
    searchDebounceMs: int(raw, "ui.search_debounce_ms", d.searchDebounceMs, 100, 1500),
    cookieNoticeDays: int(raw, "ui.cookie_notice_days", d.cookieNoticeDays, 0, 3650),
    accountingRegions: parseAccountingRegions(raw?.["accounting.regions"]),
  };
}

let cachedRaw: Raw = undefined;
let cachedSettings: PublicSettings | null = null;

export function publicSettings(): PublicSettings {
  const raw = injectedBranding()?.settings;
  if (cachedSettings && raw === cachedRaw) return cachedSettings;
  cachedRaw = raw;
  cachedSettings = resolvePublicSettings(raw);
  return cachedSettings;
}

export function pollMs(baseMs: number): number {
  const percent = publicSettings().pollScalePercent;
  return Math.max(1000, Math.round((baseMs * percent) / 100));
}
