"use client";

import { useEffect, useRef, useState } from "react";
import { Bold, Code2, Eraser, Heading3, Italic, Link2, List, ListOrdered, Pilcrow } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { sanitizeCustomHtml, sanitizeRichHtml } from "@/components/site/sanitize";
import { useEditor } from "@/components/site/editor/editor-context";
import { safeUrl } from "@/lib/site/url";
import { cn } from "@/lib/utils";

type Command = "bold" | "italic" | "insertUnorderedList" | "insertOrderedList" | "h3" | "p" | "removeFormat";

export function RichTextField({
  value,
  onChange,
  mode = "rich",
}: {
  value: string;
  onChange: (next: string) => void;
  mode?: "rich" | "html";
}) {
  const { t } = useEditor();
  const [source, setSource] = useState(mode === "html");
  const [code, setCode] = useState(value);
  const [linkOpen, setLinkOpen] = useState(false);
  const [link, setLink] = useState("https://");
  const editorRef = useRef<HTMLDivElement>(null);
  const emitted = useRef(value);
  const savedRange = useRef<Range | null>(null);
  const clean = mode === "html" ? sanitizeCustomHtml : sanitizeRichHtml;

  useEffect(() => {
    if (value === emitted.current) return;
    emitted.current = value;
    setCode(value);
    if (editorRef.current) editorRef.current.innerHTML = clean(value);
  }, [value, clean]);

  useEffect(() => {
    if (!source && editorRef.current) editorRef.current.innerHTML = clean(emitted.current);
  }, [source, clean]);

  const emit = (html: string) => {
    emitted.current = html;
    setCode(html);
    onChange(html);
  };

  const sync = () => {
    if (!editorRef.current) return;
    emit(editorRef.current.innerHTML);
  };

  const saveSelection = () => {
    const selection = window.getSelection();
    if (selection && selection.rangeCount > 0 && editorRef.current?.contains(selection.anchorNode)) {
      savedRange.current = selection.getRangeAt(0).cloneRange();
    }
  };

  const restoreSelection = () => {
    const range = savedRange.current;
    const selection = window.getSelection();
    if (!range || !selection) return;
    selection.removeAllRanges();
    selection.addRange(range);
  };

  const run = (command: Command) => {
    editorRef.current?.focus();
    restoreSelection();
    if (command === "h3" || command === "p") {
      document.execCommand("formatBlock", false, command === "h3" ? "h3" : "p");
    } else {
      document.execCommand(command, false);
    }
    sync();
  };

  const applyLink = () => {
    const url = link.trim();
    setLinkOpen(false);
    if (!url || !safeUrl(url)) return;
    editorRef.current?.focus();
    restoreSelection();
    document.execCommand("createLink", false, url);
    sync();
  };

  const tools: { id: Command | "link"; icon: typeof Bold; label: string }[] = [
    { id: "bold", icon: Bold, label: t("template.rich.bold") },
    { id: "italic", icon: Italic, label: t("template.rich.italic") },
    { id: "link", icon: Link2, label: t("template.rich.link") },
    { id: "insertUnorderedList", icon: List, label: t("template.rich.bullets") },
    { id: "insertOrderedList", icon: ListOrdered, label: t("template.rich.numbers") },
    { id: "h3", icon: Heading3, label: t("template.rich.heading") },
    { id: "p", icon: Pilcrow, label: t("template.rich.paragraph") },
    { id: "removeFormat", icon: Eraser, label: t("template.rich.clear") },
  ];

  return (
    <div className="overflow-hidden rounded-lg border bg-background">
      <div className="flex flex-wrap items-center gap-0.5 border-b bg-muted/40 p-1">
        {!source &&
          mode === "rich" &&
          tools.map((tool) => (
            <button
              key={tool.id}
              type="button"
              title={tool.label}
              aria-label={tool.label}
              onMouseDown={(event) => {
                event.preventDefault();
                saveSelection();
              }}
              onClick={() => {
                if (tool.id === "link") setLinkOpen((open) => !open);
                else run(tool.id);
              }}
              className="flex size-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-background hover:text-foreground"
            >
              <tool.icon className="size-3.5" />
            </button>
          ))}
        {mode === "rich" ? (
          <button
            type="button"
            onClick={() => setSource((current) => !current)}
            className={cn(
              "ml-auto inline-flex h-7 items-center gap-1.5 rounded-md px-2 text-[11px] font-medium transition-colors",
              source ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:bg-background hover:text-foreground"
            )}
          >
            <Code2 className="size-3.5" />
            HTML
          </button>
        ) : null}
      </div>
      {linkOpen && !source ? (
        <div className="flex gap-1.5 border-b p-1.5">
          <Input
            autoFocus
            value={link}
            onChange={(event) => setLink(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                event.preventDefault();
                applyLink();
              } else if (event.key === "Escape") {
                setLinkOpen(false);
              }
            }}
            className="h-7 font-mono text-xs"
          />
          <button
            type="button"
            onClick={applyLink}
            className="h-7 rounded-md bg-primary px-2.5 text-xs font-medium text-primary-foreground"
          >
            {t("template.rich.apply")}
          </button>
        </div>
      ) : null}
      {source ? (
        <Textarea
          value={code}
          rows={10}
          spellCheck={false}
          onChange={(event) => emit(event.target.value)}
          className="min-h-48 rounded-none border-0 font-mono text-xs focus-visible:ring-0"
        />
      ) : (
        <div
          ref={editorRef}
          contentEditable
          suppressContentEditableWarning
          onInput={sync}
          onBlur={() => {
            saveSelection();
            if (editorRef.current) {
              const cleaned = clean(editorRef.current.innerHTML);
              if (cleaned !== editorRef.current.innerHTML) editorRef.current.innerHTML = cleaned;
              emit(cleaned);
            }
          }}
          onKeyUp={saveSelection}
          onMouseUp={saveSelection}
          className="vx-article min-h-40 px-3 py-2.5 text-sm outline-none"
        />
      )}
    </div>
  );
}
