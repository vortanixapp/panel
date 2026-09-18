"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { usePathname, useRouter } from "next/navigation";
import { ArrowDown, ArrowUp, Copy, Eye, EyeOff, Plus, RotateCcw, Trash2, X } from "lucide-react";
import { fetchI18n, setRequestBodyFilter } from "@/lib/api";
import {
  catalogText,
  i18nState,
  parseI18nPayload,
  rawMessage,
  setLocaleOverride,
  setMarkerEncoder,
  type BaseLocale,
  type I18nPayload,
} from "@/lib/i18n";
import { blockMeta, isBuiltinBlock } from "@/lib/site/blocks";
import {
  EDIT_PARAM,
  editorPreviewUrl,
  isMessage,
  type BlockAction,
  type FrameMessage,
  type ParentMessage,
} from "@/lib/site/bridge";
import { blockText, findBlock } from "@/lib/site/doc";
import { markText, stripMarkers } from "@/lib/site/markers";
import { setSiteOverride } from "@/lib/site/store";
import type { SiteZone } from "@/lib/site/types";
import { cn } from "@/lib/utils";
import {
  EMPTY_SCAN,
  WATCHED_ATTRS,
  createScanner,
  isOverlayNode,
  phraseAt,
  type Phrase,
  type ScanResult,
} from "@/components/site/editor-bridge/scanner";

type StateMessage = Extract<ParentMessage, { type: "vx:state" }>;

type Hit =
  | { kind: "phrase"; phrase: Phrase }
  | { kind: "field"; element: Element; page: string; zone: SiteZone; block: string; path: string; multiline: boolean }
  | { kind: "insert"; page: string; zone: SiteZone; index: number }
  | { kind: "section"; element: Element; page: string; section: string }
  | { kind: "block"; element: Element; page: string; zone: SiteZone; block: string };

type EditTarget =
  | { kind: "phrase"; key: string; rect: DOMRect }
  | { kind: "field"; page: string; zone: SiteZone; block: string; path: string; multiline: boolean };

type Source = { base: BaseLocale; messages: Record<string, string> };

const SCROLL_KEYS = new Set(["ArrowUp", "ArrowDown", "PageUp", "PageDown", "Home", "End", " "]);

function currentPath(): string {
  const params = new URLSearchParams(window.location.search);
  params.delete(EDIT_PARAM);
  const query = params.toString();
  return window.location.pathname + (query ? `?${query}` : "");
}

function fill(text: string, params?: Record<string, string | number>): string {
  if (!params) return text;
  return text.replace(/\{(\w+)\}/g, (match, name: string) => (params[name] === undefined ? match : String(params[name])));
}

function blockElement(id: string): Element | null {
  return document.querySelector(`[data-vx-block="${CSS.escape(id)}"]`);
}

function frameInfo(element: Element): { page: string; zone: SiteZone } | null {
  const frame = element.closest("[data-vx-block]");
  const page = frame?.getAttribute("data-vx-page");
  const zone = frame?.getAttribute("data-vx-zone") as SiteZone | null;
  return page && zone ? { page, zone } : null;
}

function blockIndex(element: Element): number {
  const page = element.getAttribute("data-vx-page") ?? "";
  const zone = element.getAttribute("data-vx-zone") ?? "";
  const siblings = Array.from(
    document.querySelectorAll(`[data-vx-block][data-vx-page="${CSS.escape(page)}"][data-vx-zone="${CSS.escape(zone)}"]`)
  );
  return Math.max(0, siblings.indexOf(element));
}

function unionRect(range: Range): DOMRect | null {
  try {
    const rect = range.getBoundingClientRect();
    return rect.width || rect.height ? rect : null;
  } catch {
    return null;
  }
}

function Box({ rect, tone, label }: { rect: DOMRect; tone: "hover" | "selected" | "section"; label?: string }) {
  const above = rect.top > 22;
  return (
    <div
      className={cn(
        "absolute rounded-[4px] border-2",
        tone === "selected" && "border-indigo-500",
        tone === "hover" && "border-dashed border-indigo-400",
        tone === "section" && "border-dashed border-amber-500"
      )}
      style={{ left: rect.left - 3, top: rect.top - 3, width: rect.width + 6, height: rect.height + 6 }}
    >
      {label ? (
        <span
          className={cn(
            "absolute left-[-2px] rounded px-1.5 py-0.5 text-[10.5px] leading-4 font-medium whitespace-nowrap text-white shadow",
            tone === "section" ? "bg-amber-500" : "bg-indigo-500",
            above ? "-top-[21px]" : "top-0"
          )}
        >
          {label}
        </span>
      ) : null}
    </div>
  );
}

function ToolButton({
  title,
  onClick,
  danger,
  children,
}: {
  title: string;
  onClick: () => void;
  danger?: boolean;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      title={title}
      aria-label={title}
      onClick={onClick}
      className={cn(
        "flex size-7 items-center justify-center rounded-md text-white transition-colors",
        danger ? "hover:bg-red-500" : "hover:bg-white/20"
      )}
    >
      {children}
    </button>
  );
}

function TextPopover({
  anchor,
  title,
  hint,
  value,
  multiline,
  labels,
  onChange,
  onReset,
  onClose,
}: {
  anchor: DOMRect;
  title: string;
  hint?: string;
  value: string;
  multiline: boolean;
  labels: { done: string; reset: string; close: string };
  onChange: (value: string) => void;
  onReset: () => string;
  onClose: () => void;
}) {
  const [draft, setDraft] = useState(value);
  const width = Math.min(460, window.innerWidth - 24);
  const left = Math.min(Math.max(12, anchor.left), window.innerWidth - width - 12);
  const height = multiline ? 230 : 170;
  const fitsBelow = anchor.bottom + 10 + height < window.innerHeight;
  const top = fitsBelow ? anchor.bottom + 10 : Math.max(12, anchor.top - 10 - height);

  return (
    <div
      className="pointer-events-auto absolute rounded-xl border border-border bg-popover p-3 text-popover-foreground shadow-2xl"
      style={{ left, top, width }}
    >
      <div className="mb-2 flex items-center gap-2">
        <span className="min-w-0 flex-1 truncate font-mono text-[11px] text-muted-foreground">{title}</span>
        <button
          type="button"
          aria-label={labels.close}
          onClick={onClose}
          className="rounded-md p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <X className="size-3.5" />
        </button>
      </div>
      <textarea
        autoFocus
        value={draft}
        rows={multiline ? 5 : 3}
        onChange={(event) => {
          setDraft(event.target.value);
          onChange(event.target.value);
        }}
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            event.preventDefault();
            onClose();
          } else if (event.key === "Enter" && (event.ctrlKey || event.metaKey || (!multiline && !event.shiftKey))) {
            event.preventDefault();
            onClose();
          }
        }}
        className="w-full resize-y rounded-lg border border-input bg-background px-2.5 py-2 text-sm leading-relaxed outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/25"
      />
      {hint ? <p className="mt-1.5 text-[11px] leading-snug text-muted-foreground">{hint}</p> : null}
      <div className="mt-2.5 flex items-center justify-end gap-2">
        <button
          type="button"
          onClick={() => setDraft(onReset())}
          className="inline-flex h-8 items-center gap-1.5 rounded-md px-2.5 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <RotateCcw className="size-3.5" />
          {labels.reset}
        </button>
        <button
          type="button"
          onClick={onClose}
          className="inline-flex h-8 items-center rounded-md bg-indigo-600 px-3 text-xs font-medium text-white transition-colors hover:bg-indigo-500"
        >
          {labels.done}
        </button>
      </div>
    </div>
  );
}

export default function EditorBridge() {
  const router = useRouter();
  const pathname = usePathname();
  const [state, setState] = useState<StateMessage | null>(null);
  const [payload, setPayload] = useState<{ locale: string; data: I18nPayload } | null>(null);
  const [scan, setScan] = useState<ScanResult>(EMPTY_SCAN);
  const [hover, setHover] = useState<Hit | null>(null);
  const [editing, setEditing] = useState<EditTarget | null>(null);
  const [, setTick] = useState(0);
  const scanRef = useRef(scan);
  const scannerRef = useRef<ReturnType<typeof createScanner> | null>(null);
  const scrollRef = useRef<string | null>(null);
  const inventoryRef = useRef("");
  const mode = state?.mode ?? "preview";
  const hasState = state !== null;

  useEffect(() => {
    scanRef.current = scan;
  }, [scan]);

  const post = useCallback((message: FrameMessage) => {
    window.parent.postMessage(message, window.location.origin);
  }, []);

  useEffect(() => {
    setRequestBodyFilter(stripMarkers);
    const onMessage = (event: MessageEvent) => {
      if (event.origin !== window.location.origin || event.source !== window.parent) return;
      if (!isMessage<ParentMessage>(event.data)) return;
      const message = event.data;
      if (message.type === "vx:state") {
        setState(message);
      } else if (message.type === "vx:navigate") {
        router.push(editorPreviewUrl(message.path));
      } else if (message.type === "vx:scroll") {
        const element = blockElement(message.block);
        if (element) {
          scrollRef.current = null;
          element.scrollIntoView({ behavior: "smooth", block: "center" });
        } else {
          scrollRef.current = message.block;
        }
      }
    };
    window.addEventListener("message", onMessage);
    post({ type: "vx:ready", path: currentPath() });
    return () => {
      window.removeEventListener("message", onMessage);
      setRequestBodyFilter(null);
      setMarkerEncoder(null);
      setLocaleOverride(null);
      setSiteOverride(null);
    };
  }, [post, router]);

  useEffect(() => {
    post({ type: "vx:path", path: currentPath() });
    inventoryRef.current = "";
    setEditing(null);
    setHover(null);
  }, [pathname, post]);

  useEffect(() => {
    if (!state) return;
    setSiteOverride({
      document: state.document,
      mode: state.mode,
      viewer: state.viewer,
      selected: state.selected,
    });
    const pending = scrollRef.current;
    if (!pending) return;
    const frame = window.requestAnimationFrame(() => {
      const element = blockElement(pending);
      if (element) {
        scrollRef.current = null;
        element.scrollIntoView({ behavior: "smooth", block: "center" });
      }
    });
    return () => window.cancelAnimationFrame(frame);
  }, [state]);

  const selectedId = state?.selected ?? null;
  useEffect(() => {
    setEditing((current) => {
      if (!current) return current;
      if (current.kind === "field" && current.block === selectedId) return current;
      return null;
    });
  }, [selectedId]);

  const locale = state?.locale;
  useEffect(() => {
    if (!locale || payload?.locale === locale) return;
    let cancelled = false;
    fetchI18n(locale)
      .then((raw) => {
        if (!cancelled) setPayload({ locale, data: parseI18nPayload(raw, locale) });
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [locale, payload?.locale]);

  const textsJson = payload && state ? JSON.stringify(state.document.texts?.[payload.locale] ?? {}) : "{}";

  const source = useMemo<Source | null>(() => {
    if (!payload) return null;
    const messages = { ...payload.data.messages };
    for (const [key, value] of Object.entries(JSON.parse(textsJson) as Record<string, string>)) {
      if (value.trim()) messages[key] = value;
      else delete messages[key];
    }
    return { base: payload.data.base, messages };
  }, [payload, textsJson]);

  useEffect(() => {
    if (!hasState || !payload || !source) return;
    setMarkerEncoder(mode === "edit" ? markText : null);
    setLocaleOverride({ ...i18nState(payload.data), messages: source.messages });
  }, [hasState, payload, source, mode]);

  const label = useCallback(
    (key: string, params?: Record<string, string | number>) =>
      fill((source ? rawMessage(source, key) : catalogText("ru", key)) ?? key, params),
    [source]
  );

  const reportInventory = useCallback(
    (result: ScanResult) => {
      const keys = [...new Set([...result.phrases.map((p) => p.key), ...result.attrs.map((a) => a.key)])];
      const sections = Array.from(document.querySelectorAll("[data-vx-section]"))
        .map((element) => ({
          id: element.getAttribute("data-vx-section") ?? "",
          hidden: element.hasAttribute("data-vx-hidden"),
        }))
        .filter((section) => section.id);
      const path = currentPath();
      const json = JSON.stringify([path, keys, sections]);
      if (json === inventoryRef.current) return;
      inventoryRef.current = json;
      post({ type: "vx:inventory", path, keys, sections });
    },
    [post]
  );

  useEffect(() => {
    if (mode !== "edit") {
      scannerRef.current?.reset();
      setScan(EMPTY_SCAN);
      return;
    }
    const scanner = scannerRef.current ?? createScanner();
    scannerRef.current = scanner;
    let frame = 0;
    const options: MutationObserverInit = {
      childList: true,
      subtree: true,
      characterData: true,
      attributes: true,
      attributeFilter: [...WATCHED_ATTRS, "data-vx-hidden", "data-vx-section"],
    };
    const observer = new MutationObserver((records) => {
      if (records.some((record) => !isOverlayNode(record.target))) schedule();
    });
    const run = () => {
      frame = 0;
      observer.disconnect();
      const result = scanner.scan();
      observer.observe(document.body, options);
      setScan(result);
      reportInventory(result);
    };
    const schedule = () => {
      if (!frame) frame = window.requestAnimationFrame(run);
    };
    observer.observe(document.body, options);
    schedule();
    return () => {
      observer.disconnect();
      if (frame) window.cancelAnimationFrame(frame);
    };
  }, [mode, reportInventory]);

  useEffect(() => {
    let frame = 0;
    const bump = () => {
      if (!frame) {
        frame = window.requestAnimationFrame(() => {
          frame = 0;
          setTick((tick) => tick + 1);
        });
      }
    };
    window.addEventListener("scroll", bump, true);
    window.addEventListener("resize", bump);
    return () => {
      window.removeEventListener("scroll", bump, true);
      window.removeEventListener("resize", bump);
      if (frame) window.cancelAnimationFrame(frame);
    };
  }, []);

  const hitTest = useCallback((x: number, y: number): Hit | null => {
    const element = document.elementFromPoint(x, y);
    if (!element || isOverlayNode(element)) return null;
    const field = element.closest("[data-vx-field]");
    if (field) {
      const [block, path] = (field.getAttribute("data-vx-field") ?? "").split("|");
      const info = frameInfo(field);
      if (block && path && info) {
        return { kind: "field", element: field, block, path, ...info, multiline: field.hasAttribute("data-vx-multiline") };
      }
    }
    const phrase = phraseAt(scanRef.current.phrases, x, y, element);
    if (phrase) return { kind: "phrase", phrase };
    const insert = element.closest("[data-vx-insert]");
    if (insert) {
      return {
        kind: "insert",
        page: insert.getAttribute("data-vx-page") ?? "",
        zone: (insert.getAttribute("data-vx-zone") ?? "blocks") as SiteZone,
        index: Number(insert.getAttribute("data-vx-index") ?? 0),
      };
    }
    const section = element.closest("[data-vx-section]");
    if (section) {
      return {
        kind: "section",
        element: section,
        page: section.getAttribute("data-vx-page") ?? "",
        section: section.getAttribute("data-vx-section") ?? "",
      };
    }
    const block = element.closest("[data-vx-block]");
    if (block) {
      return {
        kind: "block",
        element: block,
        page: block.getAttribute("data-vx-page") ?? "",
        zone: (block.getAttribute("data-vx-zone") ?? "blocks") as SiteZone,
        block: block.getAttribute("data-vx-block") ?? "",
      };
    }
    return null;
  }, []);

  const activate = useCallback(
    (hit: Hit | null) => {
      if (!hit) {
        setEditing(null);
        post({ type: "vx:deselect" });
        return;
      }
      switch (hit.kind) {
        case "phrase": {
          const rect = unionRect(hit.phrase.range);
          if (rect) setEditing({ kind: "phrase", key: hit.phrase.key, rect });
          return;
        }
        case "field":
          setEditing({ kind: "field", page: hit.page, zone: hit.zone, block: hit.block, path: hit.path, multiline: hit.multiline });
          post({ type: "vx:select", page: hit.page, zone: hit.zone, block: hit.block });
          return;
        case "insert":
          setEditing(null);
          post({ type: "vx:insert", page: hit.page, zone: hit.zone, index: hit.index });
          return;
        case "section":
          setEditing(null);
          post({ type: "vx:select-section", page: hit.page, section: hit.section });
          return;
        case "block":
          setEditing(null);
          post({ type: "vx:select", page: hit.page, zone: hit.zone, block: hit.block });
      }
    },
    [post]
  );

  useEffect(() => {
    if (mode !== "edit") {
      setHover(null);
      setEditing(null);
      return;
    }
    let frame = 0;
    let last: [number, number] | null = null;
    const onMove = (event: PointerEvent) => {
      if (isOverlayNode(event.target as Node)) return;
      last = [event.clientX, event.clientY];
      if (!frame) {
        frame = window.requestAnimationFrame(() => {
          frame = 0;
          if (last) setHover(hitTest(last[0], last[1]));
        });
      }
    };
    const onLeave = () => setHover(null);
    const swallow = (event: Event) => {
      if (isOverlayNode(event.target as Node)) return;
      event.preventDefault();
      event.stopPropagation();
    };
    const onClick = (event: MouseEvent) => {
      if (isOverlayNode(event.target as Node)) return;
      event.preventDefault();
      event.stopPropagation();
      activate(hitTest(event.clientX, event.clientY));
    };
    const onKey = (event: KeyboardEvent) => {
      const mod = event.ctrlKey || event.metaKey;
      const key = event.key.toLowerCase();
      if (mod && key === "s") {
        event.preventDefault();
        post({ type: "vx:history", action: "save" });
        return;
      }
      if (isOverlayNode(event.target as Node)) return;
      if (mod && (key === "z" || key === "y")) {
        event.preventDefault();
        post({ type: "vx:history", action: key === "y" || event.shiftKey ? "redo" : "undo" });
        return;
      }
      if (event.key === "Escape") {
        setEditing(null);
        post({ type: "vx:deselect" });
      }
      if (SCROLL_KEYS.has(event.key)) return;
      event.preventDefault();
      event.stopPropagation();
    };
    const blocked = ["pointerdown", "mousedown", "pointerup", "mouseup", "dblclick", "auxclick", "contextmenu", "submit", "dragstart", "touchstart"];
    document.addEventListener("pointermove", onMove, true);
    document.addEventListener("click", onClick, true);
    document.addEventListener("keydown", onKey, true);
    for (const type of blocked) document.addEventListener(type, swallow, { capture: true, passive: false });
    document.documentElement.addEventListener("mouseleave", onLeave);
    return () => {
      document.removeEventListener("pointermove", onMove, true);
      document.removeEventListener("click", onClick, true);
      document.removeEventListener("keydown", onKey, true);
      for (const type of blocked) document.removeEventListener(type, swallow, { capture: true });
      document.documentElement.removeEventListener("mouseleave", onLeave);
      if (frame) window.cancelAnimationFrame(frame);
    };
  }, [mode, hitTest, activate, post]);

  useEffect(() => {
    if (mode !== "preview") return;
    const onClick = (event: MouseEvent) => {
      const anchor = (event.target as Element | null)?.closest?.("a[href]");
      if (!anchor) return;
      const href = anchor.getAttribute("href") ?? "";
      let url: URL;
      try {
        url = new URL(href, window.location.href);
      } catch {
        return;
      }
      if (url.origin === window.location.origin) return;
      event.preventDefault();
      window.open(url.toString(), "_blank", "noopener,noreferrer");
    };
    document.addEventListener("click", onClick, true);
    return () => document.removeEventListener("click", onClick, true);
  }, [mode]);

  if (mode !== "edit" || !state) return null;

  const blockLabel = (id: string, fallback: string) => {
    const type = document.querySelector(`[data-vx-block="${CSS.escape(id)}"]`)?.getAttribute("data-vx-type") ?? fallback;
    const meta = blockMeta(type);
    return meta ? label(meta.titleKey) : label("template.bridge.block");
  };
  const sectionLabel = (id: string) => {
    const text = label(`template.section.${id}`);
    return text === `template.section.${id}` ? label("template.bridge.section") : text;
  };

  const hoverBox = (() => {
    if (!hover) return null;
    switch (hover.kind) {
      case "phrase": {
        const rect = unionRect(hover.phrase.range);
        return rect ? <Box rect={rect} tone="hover" label={label("template.bridge.text")} /> : null;
      }
      case "field":
        return <Box rect={hover.element.getBoundingClientRect()} tone="hover" label={label("template.bridge.field")} />;
      case "section":
        return <Box rect={hover.element.getBoundingClientRect()} tone="section" label={sectionLabel(hover.section)} />;
      case "block":
        return hover.block === state.selected ? null : (
          <Box rect={hover.element.getBoundingClientRect()} tone="hover" label={blockLabel(hover.block, "")} />
        );
      default:
        return null;
    }
  })();

  const hoverBlockElement =
    hover && hover.kind !== "insert"
      ? hover.kind === "phrase"
        ? (hover.phrase.range.commonAncestorContainer instanceof Element
            ? hover.phrase.range.commonAncestorContainer
            : hover.phrase.range.commonAncestorContainer.parentElement
          )?.closest("[data-vx-block]") ?? null
        : hover.element.closest("[data-vx-block]")
      : null;

  const insertHandles = (() => {
    if (!hoverBlockElement) return null;
    const rect = hoverBlockElement.getBoundingClientRect();
    const page = hoverBlockElement.getAttribute("data-vx-page") ?? "";
    const zone = (hoverBlockElement.getAttribute("data-vx-zone") ?? "blocks") as SiteZone;
    const index = blockIndex(hoverBlockElement);
    const center = rect.left + rect.width / 2;
    return (
      <>
        {[
          { at: rect.top, index },
          { at: rect.bottom, index: index + 1 },
        ].map((handle) =>
          handle.at > 0 && handle.at < window.innerHeight ? (
            <button
              key={handle.index}
              type="button"
              title={label("template.bridge.insert")}
              aria-label={label("template.bridge.insert")}
              onClick={() => post({ type: "vx:insert", page, zone, index: handle.index })}
              className="pointer-events-auto absolute flex size-6 -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full bg-indigo-600 text-white shadow-lg ring-2 ring-white transition-transform hover:scale-110"
              style={{ left: center, top: handle.at }}
            >
              <Plus className="size-3.5" />
            </button>
          ) : null
        )}
      </>
    );
  })();

  const selected = (() => {
    const id = state.selected;
    if (!id) return null;
    if (id.startsWith("section:")) {
      const sectionId = id.slice("section:".length);
      const element = document.querySelector(`[data-vx-section="${CSS.escape(sectionId)}"]`);
      if (!element) return null;
      const rect = element.getBoundingClientRect();
      const hidden = element.hasAttribute("data-vx-hidden");
      const page = element.getAttribute("data-vx-page") ?? "";
      return (
        <>
          <Box rect={rect} tone="section" label={sectionLabel(sectionId)} />
          <div
            className="pointer-events-auto absolute flex gap-0.5 rounded-lg bg-amber-500 p-0.5 shadow-lg"
            style={{ left: Math.max(4, rect.right - 36), top: Math.max(4, rect.top + 6) }}
          >
            <ToolButton
              title={hidden ? label("template.bridge.show") : label("template.bridge.hide")}
              onClick={() => post({ type: "vx:section", page, section: sectionId, hidden: !hidden })}
            >
              {hidden ? <Eye className="size-4" /> : <EyeOff className="size-4" />}
            </ToolButton>
          </div>
        </>
      );
    }
    const element = blockElement(id);
    if (!element) return null;
    const rect = element.getBoundingClientRect();
    if (rect.bottom < 0 || rect.top > window.innerHeight) return null;
    const info = { page: element.getAttribute("data-vx-page") ?? "", zone: (element.getAttribute("data-vx-zone") ?? "blocks") as SiteZone };
    const block = findBlock(state.document, info, id);
    const builtin = block ? isBuiltinBlock(block.type) : false;
    const act = (action: BlockAction) => post({ type: "vx:block-action", ...info, block: id, action });
    const toolbarWidth = builtin ? 128 : 160;
    return (
      <>
        <Box rect={rect} tone="selected" label={block ? label(blockMeta(block.type)?.titleKey ?? "template.bridge.block") : undefined} />
        <div
          className="pointer-events-auto absolute flex gap-0.5 rounded-lg bg-indigo-600 p-0.5 shadow-lg"
          style={{
            left: Math.min(window.innerWidth - toolbarWidth - 6, Math.max(4, rect.right - toolbarWidth)),
            top: Math.min(window.innerHeight - 40, Math.max(4, rect.top + 6)),
          }}
        >
          <ToolButton title={label("template.bridge.up")} onClick={() => act("up")}>
            <ArrowUp className="size-4" />
          </ToolButton>
          <ToolButton title={label("template.bridge.down")} onClick={() => act("down")}>
            <ArrowDown className="size-4" />
          </ToolButton>
          <ToolButton
            title={block?.hidden ? label("template.bridge.show") : label("template.bridge.hide")}
            onClick={() => act(block?.hidden ? "show" : "hide")}
          >
            {block?.hidden ? <Eye className="size-4" /> : <EyeOff className="size-4" />}
          </ToolButton>
          {builtin ? null : (
            <ToolButton title={label("template.bridge.duplicate")} onClick={() => act("duplicate")}>
              <Copy className="size-4" />
            </ToolButton>
          )}
          <ToolButton title={label("template.bridge.remove")} onClick={() => act("remove")} danger>
            <Trash2 className="size-4" />
          </ToolButton>
        </div>
      </>
    );
  })();

  const popover = (() => {
    if (!editing || !source) return null;
    const labels = { done: label("template.bridge.done"), reset: label("template.bridge.reset"), close: label("common.close") };
    if (editing.kind === "phrase") {
      const candidates = scan.phrases.filter((phrase) => phrase.key === editing.key);
      let anchor = editing.rect;
      let best = Infinity;
      for (const phrase of candidates) {
        const rect = unionRect(phrase.range);
        if (!rect) continue;
        const distance = Math.abs(rect.top - editing.rect.top) + Math.abs(rect.left - editing.rect.left);
        if (distance < best) {
          best = distance;
          anchor = rect;
        }
      }
      const value = rawMessage(source, editing.key) ?? "";
      const params = [...new Set(Array.from(value.matchAll(/\{(\w+)\}/g), (m) => `{${m[1]}}`))];
      return (
        <TextPopover
          key={`phrase:${editing.key}`}
          anchor={anchor}
          title={editing.key}
          hint={params.length ? label("template.bridge.params_hint", { params: params.join(", ") }) : undefined}
          value={value}
          multiline={value.includes("\n") || value.length > 90}
          labels={labels}
          onChange={(next) => post({ type: "vx:text", locale: state.locale, key: editing.key, value: next })}
          onReset={() => {
            post({ type: "vx:text", locale: state.locale, key: editing.key, value: "" });
            return catalogText(source.base, editing.key) ?? "";
          }}
          onClose={() => setEditing(null)}
        />
      );
    }
    const element = document.querySelector(`[data-vx-field="${CSS.escape(`${editing.block}|${editing.path}`)}"]`);
    const block = findBlock(state.document, { page: editing.page, zone: editing.zone }, editing.block);
    if (!element || !block) return null;
    return (
      <TextPopover
        key={`field:${editing.block}|${editing.path}`}
        anchor={element.getBoundingClientRect()}
        title={label("template.bridge.field_title", { name: label(blockMeta(block.type)?.titleKey ?? "template.bridge.block") })}
        value={blockText(block, editing.path, state.locale)}
        multiline={editing.multiline}
        labels={labels}
        onChange={(next) =>
          post({
            type: "vx:field",
            page: editing.page,
            zone: editing.zone,
            block: editing.block,
            path: editing.path,
            locale: state.locale,
            value: next,
          })
        }
        onReset={() => {
          post({
            type: "vx:field",
            page: editing.page,
            zone: editing.zone,
            block: editing.block,
            path: editing.path,
            locale: state.locale,
            value: "",
          });
          return "";
        }}
        onClose={() => setEditing(null)}
      />
    );
  })();

  return createPortal(
    <div data-vx-overlay="" className="pointer-events-none fixed inset-0 z-[2147483000]">
      {hoverBox}
      {selected}
      {insertHandles}
      {popover}
    </div>,
    document.body
  );
}
