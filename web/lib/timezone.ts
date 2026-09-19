const STORAGE_KEY = "vx-timezone";

let installed = false;
let current = "";

type DateLocaleMethod = (
  this: Date,
  locales?: Intl.LocalesArgument,
  options?: Intl.DateTimeFormatOptions
) => string;

function nativeFormat() {
  return (globalThis as { __vxNativeDateTimeFormat?: typeof Intl.DateTimeFormat }).__vxNativeDateTimeFormat ?? Intl.DateTimeFormat;
}

export function isValidTimeZone(zone: string): boolean {
  if (!zone) return false;
  try {
    new (nativeFormat())("en-US", { timeZone: zone });
    return true;
  } catch {
    return false;
  }
}

export function deviceTimeZone(): string {
  try {
    return new (nativeFormat())().resolvedOptions().timeZone || "UTC";
  } catch {
    return "UTC";
  }
}

function withZone(options?: Intl.DateTimeFormatOptions): Intl.DateTimeFormatOptions | undefined {
  if (!current || options?.timeZone) return options;
  return { ...(options ?? {}), timeZone: current };
}

function install() {
  if (installed || typeof window === "undefined") return;
  installed = true;
  const Native = Intl.DateTimeFormat;
  (globalThis as { __vxNativeDateTimeFormat?: typeof Intl.DateTimeFormat }).__vxNativeDateTimeFormat = Native;
  const Patched = function (locales?: Intl.LocalesArgument, options?: Intl.DateTimeFormatOptions) {
    return new Native(locales, withZone(options));
  } as unknown as typeof Intl.DateTimeFormat;
  Object.defineProperty(Patched, "prototype", { value: Native.prototype });
  Object.defineProperty(Patched, "supportedLocalesOf", { value: Native.supportedLocalesOf.bind(Native) });
  Intl.DateTimeFormat = Patched;

  const methods = ["toLocaleString", "toLocaleDateString", "toLocaleTimeString"] as const;
  for (const name of methods) {
    const native = Date.prototype[name] as DateLocaleMethod;
    const patched: DateLocaleMethod = function (locales, options) {
      return native.call(this, locales, withZone(options));
    };
    Object.defineProperty(Date.prototype, name, { value: patched, configurable: true, writable: true });
  }
}

export function displayTimeZone(): string {
  return current || deviceTimeZone();
}

export function setDisplayTimeZone(zone: string) {
  if (typeof window === "undefined") return;
  install();
  current = isValidTimeZone(zone) ? zone : "";
  try {
    if (current) localStorage.setItem(STORAGE_KEY, current);
    else localStorage.removeItem(STORAGE_KEY);
  } catch {
    return;
  }
}

export function restoreDisplayTimeZone() {
  if (typeof window === "undefined") return;
  try {
    const stored = localStorage.getItem(STORAGE_KEY) ?? "";
    if (stored) setDisplayTimeZone(stored);
  } catch {
    return;
  }
}

export function timeZoneOffsetLabel(zone: string, at = new Date()): string {
  try {
    const parts = new (nativeFormat())("en-US", { timeZone: zone, timeZoneName: "shortOffset" }).formatToParts(at);
    const name = parts.find((p) => p.type === "timeZoneName")?.value ?? "GMT";
    return name.replace("GMT", "UTC") === "UTC" ? "UTC+0" : name.replace("GMT", "UTC");
  } catch {
    return "UTC";
  }
}

export function timeZoneOffsetMinutes(zone: string, at = new Date()): number {
  const label = timeZoneOffsetLabel(zone, at);
  const m = /UTC([+-])(\d{1,2})(?::(\d{2}))?/.exec(label);
  if (!m) return 0;
  const minutes = Number(m[2]) * 60 + Number(m[3] ?? 0);
  return m[1] === "-" ? -minutes : minutes;
}

export function listTimeZones(): string[] {
  const intl = Intl as unknown as { supportedValuesOf?: (key: string) => string[] };
  const zones = typeof intl.supportedValuesOf === "function" ? intl.supportedValuesOf("timeZone") : [];
  const list = zones.length > 0 ? [...zones] : ["UTC", "Europe/Moscow", "Europe/Kyiv", "Europe/Minsk", "Asia/Almaty"];
  if (!list.includes("UTC")) list.unshift("UTC");
  return list;
}
