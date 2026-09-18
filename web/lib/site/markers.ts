const OPEN = "\u206A";
const CLOSE = "\u206B";
const DIGITS = ["\u2061", "\u2062", "\u2063", "\u2064"];
const WIDTH = 7;

export const MARK_TEST = /[\u206A\u206B]/;
export const MARK_RE = /([\u206A\u206B])([\u2061-\u2064]{7})/g;
const STRIP_RE = /[\u206A\u206B\u2061-\u2064]/g;

const keys: string[] = [];
const ids = new Map<string, number>();

function encodeId(id: number): string {
  let out = "";
  let rest = id;
  for (let i = 0; i < WIDTH; i++) {
    out = DIGITS[rest % 4] + out;
    rest = Math.floor(rest / 4);
  }
  return out;
}

export function decodeId(code: string): number {
  let n = 0;
  for (const ch of code) n = n * 4 + DIGITS.indexOf(ch);
  return n;
}

export function isOpenMark(ch: string): boolean {
  return ch === OPEN;
}

export function markText(key: string, text: string): string {
  let id = ids.get(key);
  if (id === undefined) {
    id = keys.length;
    keys.push(key);
    ids.set(key, id);
  }
  const code = encodeId(id);
  return OPEN + code + text + CLOSE + code;
}

export function markedKey(id: number): string | undefined {
  return keys[id];
}

export function stripMarkers(text: string): string {
  return text.replace(STRIP_RE, "");
}
