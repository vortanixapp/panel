export const ACCOUNTING_REGIONS = ["cis", "eu", "europe", "us"];

export function parseAccountingRegions(value: unknown): string[] {
  if (!Array.isArray(value)) return ["cis"];
  const items = ACCOUNTING_REGIONS.filter((region) => value.includes(region));
  return items.length > 0 ? items : ["cis"];
}
