"use client";

import { useMemo } from "react";
import { cn } from "@/lib/utils";

type LegalBlock =
  | { type: "h2" | "h3" | "p"; text: string }
  | { type: "ul"; items: string[] };

function parseLegalText(body: string): LegalBlock[] {
  const blocks: LegalBlock[] = [];
  let paragraph: string[] = [];
  let list: string[] = [];
  const flushParagraph = () => {
    if (paragraph.length > 0) {
      blocks.push({ type: "p", text: paragraph.join(" ") });
      paragraph = [];
    }
  };
  const flushList = () => {
    if (list.length > 0) {
      blocks.push({ type: "ul", items: list });
      list = [];
    }
  };
  for (const raw of body.split("\n")) {
    const line = raw.trim();
    if (!line) {
      flushParagraph();
      flushList();
      continue;
    }
    if (line.startsWith("## ")) {
      flushParagraph();
      flushList();
      blocks.push({ type: "h3", text: line.slice(3) });
      continue;
    }
    if (line.startsWith("# ")) {
      flushParagraph();
      flushList();
      blocks.push({ type: "h2", text: line.slice(2) });
      continue;
    }
    if (line.startsWith("- ")) {
      flushParagraph();
      list.push(line.slice(2));
      continue;
    }
    flushList();
    paragraph.push(line);
  }
  flushParagraph();
  flushList();
  return blocks;
}

export function LegalText({ body, className }: { body: string; className?: string }) {
  const blocks = useMemo(() => parseLegalText(body), [body]);
  return (
    <div className={cn("space-y-3 text-[14.5px] leading-[1.65] text-foreground/90", className)}>
      {blocks.map((block, index) => {
        if (block.type === "ul") {
          return (
            <ul key={index} className="list-disc space-y-1 pl-6">
              {block.items.map((item, itemIndex) => (
                <li key={itemIndex}>{item}</li>
              ))}
            </ul>
          );
        }
        if (block.type === "h2") {
          return (
            <h2 key={index} className="pt-4 text-xl font-semibold text-foreground">
              {block.text}
            </h2>
          );
        }
        if (block.type === "h3") {
          return (
            <h3 key={index} className="pt-3 text-base font-semibold text-foreground">
              {block.text}
            </h3>
          );
        }
        return <p key={index}>{block.text}</p>;
      })}
    </div>
  );
}
