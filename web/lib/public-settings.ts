import { injectedBranding } from "@/lib/appearance";

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
  };
}

export function publicSettings(): PublicSettings {
  const raw = injectedBranding()?.settings;
  return resolvePublicSettings(raw);
}
