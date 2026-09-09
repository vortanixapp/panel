"use client";

import * as React from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";

export function SettingsCard({
  title,
  description,
  action,
  className,
  children,
}: {
  title?: React.ReactNode;
  description?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <section
      className={cn(
        "rounded-2xl border bg-card px-5 py-5 sm:px-7 sm:py-6",
        className
      )}
    >
      {(title || action) && (
        <div className="mb-5 flex flex-wrap items-start justify-between gap-3">
          <div className="space-y-1">
            {title && (
              <div className="text-[15px] leading-none font-semibold">
                {title}
              </div>
            )}
            {description && (
              <div className="text-[13px] text-muted-foreground">
                {description}
              </div>
            )}
          </div>
          {action}
        </div>
      )}
      {children}
    </section>
  );
}

export function FieldGrid({
  cols = 2,
  className,
  children,
}: {
  cols?: 2 | 3 | 4;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "grid gap-x-5 gap-y-[18px]",
        cols === 2 && "sm:grid-cols-2",
        cols === 3 && "sm:grid-cols-2 lg:grid-cols-3",
        cols === 4 && "sm:grid-cols-2 lg:grid-cols-4",
        className
      )}
    >
      {children}
    </div>
  );
}

const controlClass =
  "h-[38px] rounded-lg text-[13px] shadow-none md:text-[13px]";

export function TextField({
  label,
  value,
  onChange,
  placeholder,
  type = "text",
  mono,
  span,
  accent,
  autoComplete = "off",
}: {
  label: string;
  value: string;
  onChange: (next: string) => void;
  placeholder?: string;
  type?: string;
  mono?: boolean;
  span?: 2 | 3 | 4 | "full";
  accent?: boolean;
  autoComplete?: string;
}) {
  return (
    <div className={cn("flex flex-col gap-[7px]", spanClass(span))}>
      <Label
        className={cn(
          "text-xs font-normal",
          accent ? "text-amber-500" : "text-muted-foreground"
        )}
      >
        {label}
      </Label>
      <Input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        autoComplete={autoComplete}
        className={cn(
          controlClass,
          mono && "font-mono",
          accent && "border-amber-500/40 focus-visible:border-amber-500"
        )}
      />
    </div>
  );
}

export function TextAreaField({
  label,
  value,
  onChange,
  placeholder,
  mono,
  span,
  rows = 3,
}: {
  label: string;
  value: string;
  onChange: (next: string) => void;
  placeholder?: string;
  mono?: boolean;
  span?: 2 | 3 | 4 | "full";
  rows?: number;
}) {
  return (
    <div className={cn("flex flex-col gap-[7px]", spanClass(span))}>
      <Label className="text-xs font-normal text-muted-foreground">
        {label}
      </Label>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        rows={rows}
        className={cn(
          "min-h-[84px] w-full resize-y rounded-lg border border-input bg-transparent px-3 py-2.5 text-[13px] leading-relaxed outline-none transition-[color,box-shadow] placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:bg-input/30",
          mono && "font-mono"
        )}
      />
    </div>
  );
}

export function SelectField({
  label,
  value,
  onChange,
  options,
  span,
}: {
  label: string;
  value: string;
  onChange: (next: string) => void;
  options: readonly string[] | readonly { value: string; label: string }[];
  span?: 2 | 3 | 4 | "full";
}) {
  const normalized = options.map((opt) =>
    typeof opt === "string" ? { value: opt, label: opt } : opt
  );
  return (
    <div className={cn("flex flex-col gap-[7px]", spanClass(span))}>
      <Label className="text-xs font-normal text-muted-foreground">
        {label}
      </Label>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="h-[38px] w-full rounded-lg border border-input bg-transparent px-2.5 text-[13px] outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:bg-input/30"
      >
        {normalized.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
    </div>
  );
}

export function ToggleRow({
  label,
  hint,
  checked,
  onCheckedChange,
  accent,
}: {
  label: string;
  hint?: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  accent?: boolean;
}) {
  return (
    <div
      className={cn(
        "flex items-center justify-between gap-5 rounded-xl border bg-muted/30 px-4 py-3.5",
        accent && "border-amber-500/30"
      )}
    >
      <div className="space-y-0.5">
        <div
          className={cn(
            "text-[13px] font-medium",
            accent && "text-amber-500"
          )}
        >
          {label}
        </div>
        {hint && (
          <div className="text-xs text-muted-foreground">{hint}</div>
        )}
      </div>
      <Switch checked={checked} onCheckedChange={onCheckedChange} />
    </div>
  );
}

export function Segmented<T extends string>({
  items,
  value,
  onChange,
  size = "md",
  className,
}: {
  items: readonly { id: T; label: string }[];
  value: T;
  onChange: (next: T) => void;
  size?: "sm" | "md";
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex gap-1 overflow-x-auto rounded-xl border bg-card p-1",
        size === "sm" && "rounded-lg border-0 bg-muted/60 p-0.5",
        className
      )}
    >
      {items.map((item) => (
        <button
          key={item.id}
          type="button"
          onClick={() => onChange(item.id)}
          className={cn(
            "shrink-0 grow rounded-[9px] px-3.5 text-[13px] whitespace-nowrap transition-colors",
            size === "md" ? "h-[34px]" : "h-[30px] rounded-md font-mono text-xs",
            value === item.id
              ? size === "md"
                ? "bg-primary font-medium text-primary-foreground"
                : "bg-background text-foreground shadow-xs"
              : "text-muted-foreground hover:text-foreground"
          )}
        >
          {item.label}
        </button>
      ))}
    </div>
  );
}

function spanClass(span?: 2 | 3 | 4 | "full") {
  if (!span) return undefined;
  if (span === "full") return "sm:col-span-full";
  if (span === 2) return "sm:col-span-2";
  if (span === 3) return "sm:col-span-2 lg:col-span-3";
  return "sm:col-span-2 lg:col-span-4";
}
