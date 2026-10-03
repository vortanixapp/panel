export type LogLevel = "error" | "warn" | "info";

export type LogLine = {
  n: number;
  text: string;
  level: LogLevel;
  time?: string;
};

export type ServerLogStore = {
  lines: string[];
  times: string[];
  first: number;
  lastTime: string;
};

const ANSI = /\u001b\[[0-9;?]*[ -/]*[@-~]/g;
const ERROR = /\b(error|fatal|exception|crash(ed)?|panic|segfault|failed|critical)\b/i;
const WARN = /\b(warn(ing)?)\b/i;

export function stripAnsi(text: string): string {
  return text.replace(ANSI, "");
}

export function detectLevel(text: string): LogLevel {
  if (ERROR.test(text)) return "error";
  if (WARN.test(text)) return "warn";
  return "info";
}

export function parseLogLines(store: ServerLogStore | undefined): LogLine[] {
  if (!store) return [];
  return store.lines.map((line, index) => {
    const text = stripAnsi(line).replace(/\r$/, "");
    return {
      n: store.first + index + 1,
      text,
      level: detectLevel(text),
      time: store.times[index],
    };
  });
}

export function mergeLogs(
  prev: ServerLogStore | undefined,
  res: { lines: string[]; times?: string[]; incremental?: boolean },
  tail: number
): ServerLogStore {
  const times = res.times && res.times.length === res.lines.length ? res.times : [];
  if (!prev || !res.incremental) {
    return {
      lines: res.lines,
      times,
      first: 0,
      lastTime: times[times.length - 1] ?? "",
    };
  }
  if (res.lines.length === 0) return prev;
  const withTimes = times.length === res.lines.length && prev.times.length === prev.lines.length;
  let lines = prev.lines.concat(res.lines);
  let allTimes = withTimes ? prev.times.concat(times) : [];
  let first = prev.first;
  const overflow = lines.length - tail;
  if (overflow > 0) {
    lines = lines.slice(overflow);
    allTimes = allTimes.slice(overflow);
    first += overflow;
  }
  return {
    lines,
    times: allTimes,
    first,
    lastTime: withTimes ? times[times.length - 1] : "",
  };
}

export function splitByQuery(text: string, query: string): { part: string; match: boolean }[] {
  if (!query) return [{ part: text, match: false }];
  const lower = text.toLowerCase();
  const needle = query.toLowerCase();
  const out: { part: string; match: boolean }[] = [];
  let from = 0;
  while (from < text.length) {
    const at = lower.indexOf(needle, from);
    if (at === -1) break;
    if (at > from) out.push({ part: text.slice(from, at), match: false });
    out.push({ part: text.slice(at, at + needle.length), match: true });
    from = at + needle.length;
  }
  if (from < text.length) out.push({ part: text.slice(from), match: false });
  return out.length ? out : [{ part: text, match: false }];
}

export function downloadTextFile(name: string, content: string) {
  const url = URL.createObjectURL(new Blob([content], { type: "text/plain;charset=utf-8" }));
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}
