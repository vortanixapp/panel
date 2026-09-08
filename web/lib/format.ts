export function formatAmount(value: unknown, fractionDigits = 2): string {
  const num = Number(value);
  if (!Number.isFinite(num)) {
    return (0).toFixed(fractionDigits);
  }
  return num.toFixed(fractionDigits);
}
