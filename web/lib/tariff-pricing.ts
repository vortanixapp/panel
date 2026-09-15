export type TariffPricingFields = {
  billing_type?: string;
  cpu_cores?: number | null;
  ram_gb?: number | null;
  disk_gb?: number | null;
  cpu_min?: number | null;
  cpu_max?: number | null;
  cpu_step?: number | null;
  ram_min?: number | null;
  ram_max?: number | null;
  ram_step?: number | null;
  disk_min?: number | null;
  disk_max?: number | null;
  disk_step?: number | null;
  min_slots?: number | null;
  max_slots?: number | null;
  price_from?: number | null;
  price_monthly?: number | null;
  discounts?: unknown;
};

export type TariffRange = {
  min: number;
  max: number;
  step: number;
  default: number;
};

function num(value: unknown): number {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

export function tariffRange(
  tariff: TariffPricingFields | undefined,
  field: "cpu" | "ram" | "disk",
  fallback: number
): TariffRange {
  if (!tariff) return { min: fallback, max: fallback, step: 1, default: fallback };
  const fixedRaw =
    field === "cpu" ? tariff.cpu_cores : field === "ram" ? tariff.ram_gb : tariff.disk_gb;
  const fixed = num(fixedRaw) > 0 ? num(fixedRaw) : fallback;
  const lo = num(tariff[`${field}_min` as keyof TariffPricingFields]);
  const hi = num(tariff[`${field}_max` as keyof TariffPricingFields]);
  const step = Math.max(1, num(tariff[`${field}_step` as keyof TariffPricingFields]) || 1);
  if (tariff.billing_type !== "resources" || (lo <= 0 && hi <= 0)) {
    return { min: fixed, max: fixed, step, default: fixed };
  }
  const min = lo > 0 ? lo : fixed;
  const max = Math.max(hi > 0 ? hi : Math.max(fixed, min), min);
  return { min, max, step, default: Math.min(Math.max(fixed, min), max) };
}

export function slotRange(
  tariff: TariffPricingFields | undefined,
  fallback: number
): TariffRange {
  if (!tariff) return { min: fallback, max: fallback, step: 1, default: fallback };
  const min = Math.max(1, num(tariff.min_slots) || 1);
  const max = Math.max(num(tariff.max_slots) || min, min);
  return { min, max, step: 1, default: min };
}

export function fitRange(range: TariffRange, value: number): number {
  let next = value > 0 ? value : range.default;
  next = Math.min(Math.max(next, range.min), range.max);
  if (range.step > 1) {
    next = range.min + Math.round((next - range.min) / range.step) * range.step;
    while (next > range.max) next -= range.step;
    next = Math.max(next, range.min);
  }
  return next;
}

export function periodDiscount(tariff: TariffPricingFields | undefined, days: number): number {
  const discounts = tariff?.discounts;
  if (!discounts || typeof discounts !== "object" || Array.isArray(discounts)) return 0;
  const percent = num((discounts as Record<string, unknown>)[String(days)]);
  return Math.min(100, Math.max(0, percent));
}

export function periodCost(tariff: TariffPricingFields | undefined, days: number): number | null {
  const monthly = tariff?.price_from ?? tariff?.price_monthly;
  if (monthly == null) return null;
  const cost = (num(monthly) * days) / 30;
  return cost - (cost * periodDiscount(tariff, days)) / 100;
}

export function allowedPeriods(periods: number[] | undefined, fallback: readonly number[]): number[] {
  const list = (periods ?? []).filter((days) => days > 0);
  return list.length > 0 ? list : [...fallback];
}
