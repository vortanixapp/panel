"use client";

import { Fragment } from "react";
import { cn } from "@/lib/utils";

type Block =
  | { kind: "p"; text: string }
  | { kind: "ul"; items: string[] }
  | { kind: "h"; text: string };

const LIST_ITEM = /^\s*(?:[-*•—–]|\d+[.)])\s+/;
const HEADING = /^#{1,6}\s+(.*)$/;
const INLINE = /(`[^`]+`|\*\*[^*]+\*\*|\[[^\]]+\]\(https?:\/\/[^\s)]+\)|https?:\/\/[^\s<>()]+)/g;

export function parseNotes(text: string): Block[] {
  const blocks: Block[] = [];
  let paragraph: string[] = [];
  let list: string[] = [];
  const flushParagraph = () => {
    if (paragraph.length) blocks.push({ kind: "p", text: paragraph.join(" ") });
    paragraph = [];
  };
  const flushList = () => {
    if (list.length) blocks.push({ kind: "ul", items: list });
    list = [];
  };
  for (const raw of text.replace(/\r\n?/g, "\n").split("\n")) {
    const line = raw.trim();
    if (!line) {
      flushParagraph();
      flushList();
      continue;
    }
    const heading = HEADING.exec(line);
    if (heading) {
      flushParagraph();
      flushList();
      blocks.push({ kind: "h", text: heading[1] });
      continue;
    }
    if (LIST_ITEM.test(raw)) {
      flushParagraph();
      list.push(raw.replace(LIST_ITEM, "").trim());
      continue;
    }
    if (list.length && /^\s{2,}/.test(raw)) {
      list[list.length - 1] += ` ${line}`;
      continue;
    }
    flushList();
    paragraph.push(line);
  }
  flushParagraph();
  flushList();
  return blocks;
}

function renderInline(text: string): React.ReactNode[] {
  const out: React.ReactNode[] = [];
  let last = 0;
  let key = 0;
  for (const match of text.matchAll(INLINE)) {
    const index = match.index ?? 0;
    const token = match[0];
    if (index > last) out.push(text.slice(last, index));
    if (token.startsWith("`")) {
      out.push(
        <code key={key++} className="rounded bg-muted px-1 py-0.5 font-mono text-[0.92em]">
          {token.slice(1, -1)}
        </code>
      );
    } else if (token.startsWith("**")) {
      out.push(
        <strong key={key++} className="font-semibold text-foreground">
          {token.slice(2, -2)}
        </strong>
      );
    } else {
      const link = /^\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)$/.exec(token);
      const href = link ? link[2] : token;
      out.push(
        <a
          key={key++}
          href={href}
          target="_blank"
          rel="noreferrer noopener"
          className="text-primary underline-offset-4 hover:underline"
        >
          {link ? link[1] : token}
        </a>
      );
    }
    last = index + token.length;
  }
  if (last < text.length) out.push(text.slice(last));
  return out;
}

export function notesTitle(notes: string): string {
  const first = parseNotes(notes)[0];
  if (!first) return "";
  if (first.kind === "ul") return "";
  return first.text;
}

export function ReleaseNotes({
  text,
  skipTitle,
  className,
}: {
  text: string;
  skipTitle?: boolean;
  className?: string;
}) {
  let blocks = parseNotes(text);
  if (skipTitle && blocks[0] && blocks[0].kind !== "ul") blocks = blocks.slice(1);
  if (blocks.length === 0) return null;
  return (
    <div className={cn("space-y-2.5 text-[13.5px] leading-relaxed text-muted-foreground", className)}>
      {blocks.map((block, index) => (
        <Fragment key={index}>
          {block.kind === "h" && (
            <h4 className="pt-1 text-[13.5px] font-semibold text-foreground">{renderInline(block.text)}</h4>
          )}
          {block.kind === "p" && <p>{renderInline(block.text)}</p>}
          {block.kind === "ul" && (
            <ul className="space-y-1.5">
              {block.items.map((item, itemIndex) => (
                <li key={itemIndex} className="flex gap-2.5">
                  <span className="mt-[0.6em] size-1.5 shrink-0 rounded-full bg-muted-foreground/50" />
                  <span>{renderInline(item)}</span>
                </li>
              ))}
            </ul>
          )}
        </Fragment>
      ))}
    </div>
  );
}
