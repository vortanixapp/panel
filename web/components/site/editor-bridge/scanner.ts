import { MARK_RE, MARK_TEST, decodeId, isOpenMark, markedKey, stripMarkers } from "@/lib/site/markers";

export type Phrase = { key: string; range: Range };
export type AttrPhrase = { key: string; element: Element; attr: string };
export type ScanResult = { phrases: Phrase[]; attrs: AttrPhrase[] };

export const WATCHED_ATTRS = ["title", "placeholder", "aria-label", "alt"];

export const EMPTY_SCAN: ScanResult = { phrases: [], attrs: [] };

export function isOverlayNode(node: Node | null | undefined): boolean {
  if (!node) return false;
  const element = node instanceof Element ? node : node.parentElement;
  return Boolean(element?.closest("[data-vx-overlay]"));
}

export function isEditorUi(node: Node | null | undefined): boolean {
  if (!node) return false;
  const element = node instanceof Element ? node : node.parentElement;
  return Boolean(element?.closest("[data-vx-ui]"));
}

type Pending = { id: number; start: [Text, number]; end: [Text, number] };

function collectText(touched: Set<Node>): Phrase[] {
  const phrases: Phrase[] = [];
  const stack: { id: number; node: Text; offset: number }[] = [];
  const pending: Pending[] = [];
  const edits: [Text, string][] = [];

  const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  for (let node = walker.nextNode() as Text | null; node; node = walker.nextNode() as Text | null) {
    const data = node.data;
    if (!MARK_TEST.test(data)) continue;
    touched.add(node);
    let out = "";
    let last = 0;
    MARK_RE.lastIndex = 0;
    for (let match = MARK_RE.exec(data); match; match = MARK_RE.exec(data)) {
      out += stripMarkers(data.slice(last, match.index));
      last = match.index + match[0].length;
      const id = decodeId(match[2]);
      if (isOpenMark(match[1])) {
        stack.push({ id, node, offset: out.length });
        continue;
      }
      for (let i = stack.length - 1; i >= 0; i--) {
        if (stack[i].id !== id) continue;
        const [open] = stack.splice(i, 1);
        pending.push({ id, start: [open.node, open.offset], end: [node, out.length] });
        break;
      }
    }
    out += stripMarkers(data.slice(last));
    edits.push([node, out]);
  }

  for (const [node, text] of edits) {
    if (node.data !== text) node.data = text;
  }

  for (const item of pending) {
    const key = markedKey(item.id);
    if (!key || isOverlayNode(item.start[0]) || isEditorUi(item.start[0])) continue;
    try {
      const range = document.createRange();
      range.setStart(item.start[0], Math.min(item.start[1], item.start[0].length));
      range.setEnd(item.end[0], Math.min(item.end[1], item.end[0].length));
      if (!range.collapsed) phrases.push({ key, range });
    } catch {
      continue;
    }
  }
  return phrases;
}

function collectAttrs(): AttrPhrase[] {
  const attrs: AttrPhrase[] = [];
  const selector = WATCHED_ATTRS.map((attr) => `[${attr}]`).join(",");
  document.body.querySelectorAll(selector).forEach((element) => {
    for (const attr of WATCHED_ATTRS) {
      const value = element.getAttribute(attr);
      if (!value || !MARK_TEST.test(value)) continue;
      MARK_RE.lastIndex = 0;
      const match = MARK_RE.exec(value);
      if (match && isOpenMark(match[1]) && !isOverlayNode(element) && !isEditorUi(element)) {
        const key = markedKey(decodeId(match[2]));
        if (key) attrs.push({ key, element, attr });
      }
      element.setAttribute(attr, stripMarkers(value));
    }
  });
  return attrs;
}

function alive(phrase: Phrase, touched: Set<Node>): boolean {
  const { startContainer, endContainer } = phrase.range;
  if (!startContainer.isConnected || !endContainer.isConnected || phrase.range.collapsed) return false;
  return !touched.has(startContainer) && !touched.has(endContainer);
}

function byPosition(a: Phrase, b: Phrase): number {
  try {
    return a.range.compareBoundaryPoints(Range.START_TO_START, b.range);
  } catch {
    return 0;
  }
}

export function createScanner() {
  let phrases: Phrase[] = [];
  let attrs: AttrPhrase[] = [];
  return {
    scan(): ScanResult {
      const touched = new Set<Node>();
      const fresh = collectText(touched);
      phrases = [...phrases.filter((phrase) => alive(phrase, touched)), ...fresh].sort(byPosition);
      const freshAttrs = collectAttrs();
      const replaced = (item: AttrPhrase) =>
        freshAttrs.some((next) => next.element === item.element && next.attr === item.attr);
      attrs = [...attrs.filter((item) => item.element.isConnected && !replaced(item)), ...freshAttrs];
      if (MARK_TEST.test(document.title)) document.title = stripMarkers(document.title);
      return { phrases: [...phrases], attrs: [...attrs] };
    },
    reset() {
      phrases = [];
      attrs = [];
    },
  };
}

export function phraseAt(phrases: Phrase[], x: number, y: number, target: Element): Phrase | null {
  let best: Phrase | null = null;
  let bestArea = Infinity;
  for (const phrase of phrases) {
    let hit = false;
    try {
      if (!phrase.range.intersectsNode(target)) continue;
      for (const rect of phrase.range.getClientRects()) {
        if (x >= rect.left - 2 && x <= rect.right + 2 && y >= rect.top - 2 && y <= rect.bottom + 2) {
          hit = true;
          break;
        }
      }
    } catch {
      continue;
    }
    if (!hit) continue;
    const box = phrase.range.getBoundingClientRect();
    const area = box.width * box.height;
    if (area < bestArea) {
      best = phrase;
      bestArea = area;
    }
  }
  return best;
}
