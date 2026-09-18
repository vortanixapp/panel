"use client";

import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import { Ban, ImageIcon, Loader2, Trash2, Upload } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { resolveImage } from "@/components/site/blocks/shared";
import { useEditor } from "@/components/site/editor/editor-context";
import { uploadTemplateAsset } from "@/lib/site/api";
import { SITE_ICONS, SITE_ICON_NAMES, siteIcon } from "@/lib/site/icons";
import { withText } from "@/lib/site/text";
import type { LText } from "@/lib/site/types";
import { safeImageUrl, safeUrl, safeVideoUrl } from "@/lib/site/url";
import { cn } from "@/lib/utils";

const UPLOAD_LIMIT = 5 * 1024 * 1024;

export function FieldShell({
  label,
  hint,
  error,
  action,
  htmlFor,
  children,
}: {
  label: string;
  hint?: ReactNode;
  error?: string;
  action?: ReactNode;
  htmlFor?: string;
  children: ReactNode;
}) {
  return (
    <div className="space-y-1.5">
      <div className="flex min-h-5 items-center justify-between gap-2">
        <Label htmlFor={htmlFor} className="text-xs font-medium">
          {label}
        </Label>
        {action}
      </div>
      {children}
      {error ? (
        <p className="text-[11px] leading-snug text-destructive">{error}</p>
      ) : hint ? (
        <p className="text-[11px] leading-snug text-muted-foreground">{hint}</p>
      ) : null}
    </div>
  );
}

export function LTextField({
  label,
  value,
  onChange,
  multiline,
  placeholder,
  maxLength,
}: {
  label: string;
  value: LText | undefined;
  onChange: (next: LText | undefined) => void;
  multiline?: boolean;
  placeholder?: string;
  maxLength?: number;
}) {
  const { locale, defaultLocale, languages, t } = useEditor();
  const id = useId();
  const current = value?.[locale] ?? "";
  const others = languages.filter((lang) => lang.code !== locale && value?.[lang.code]?.trim());
  const fallback = !current && locale !== defaultLocale ? (value?.[defaultLocale] ?? "") : "";
  const hint = others.length ? t("template.field.other_languages", { list: others.map((lang) => lang.name).join(", ") }) : undefined;
  return (
    <FieldShell label={label} hint={hint} htmlFor={id}>
      {multiline ? (
        <Textarea
          id={id}
          value={current}
          rows={4}
          maxLength={maxLength}
          placeholder={fallback || placeholder}
          onChange={(event) => onChange(withText(value, locale, event.target.value))}
          className="min-h-20 text-sm"
        />
      ) : (
        <Input
          id={id}
          value={current}
          maxLength={maxLength}
          placeholder={fallback || placeholder}
          onChange={(event) => onChange(withText(value, locale, event.target.value))}
          className="h-8 text-sm"
        />
      )}
    </FieldShell>
  );
}

function useDraft(value: string) {
  const [draft, setDraft] = useState(value);
  const last = useRef(value);
  useEffect(() => {
    if (value !== last.current) {
      last.current = value;
      setDraft(value);
    }
  }, [value]);
  return [draft, setDraft, last] as const;
}

export function UrlField({
  label,
  value,
  onChange,
  placeholder,
  kind = "link",
}: {
  label: string;
  value: string;
  onChange: (next: string) => void;
  placeholder?: string;
  kind?: "link" | "video";
}) {
  const { t, pageLinks } = useEditor();
  const listId = useId();
  const [draft, setDraft, last] = useDraft(value);
  const check = kind === "video" ? safeVideoUrl : safeUrl;
  const valid = check(draft.trim());
  return (
    <FieldShell
      label={label}
      error={valid ? undefined : t(kind === "video" ? "template.field.video_invalid" : "template.field.url_invalid")}
      hint={t(kind === "video" ? "template.field.video_hint" : "template.field.url_hint")}
    >
      <Input
        list={kind === "link" ? listId : undefined}
        value={draft}
        placeholder={placeholder ?? (kind === "video" ? "https://youtu.be/…" : "/page, https://…")}
        onChange={(event) => {
          const next = event.target.value;
          setDraft(next);
          if (check(next.trim())) {
            last.current = next.trim();
            onChange(next.trim());
          }
        }}
        className={cn("h-8 font-mono text-xs", !valid && "border-destructive focus-visible:ring-destructive/30")}
      />
      {kind === "link" ? (
        <datalist id={listId}>
          {pageLinks.map((link) => (
            <option key={link.url} value={link.url}>
              {link.label}
            </option>
          ))}
        </datalist>
      ) : null}
    </FieldShell>
  );
}

export function ImageField({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (next: string) => void;
}) {
  const { t } = useEditor();
  const [busy, setBusy] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const [draft, setDraft, last] = useDraft(value.startsWith("branding/") ? "" : value);
  const src = value ? resolveImage(value) : "";
  const valid = safeImageUrl(draft.trim());

  const upload = async (file: File) => {
    if (file.size > UPLOAD_LIMIT) {
      toast.error(t("template.field.upload_too_big"));
      return;
    }
    setBusy(true);
    try {
      const result = await uploadTemplateAsset(file);
      onChange(result.path);
      setDraft("");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("template.field.upload_failed"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <FieldShell label={label} error={valid ? undefined : t("template.field.url_invalid")}>
      <div className="overflow-hidden rounded-lg border bg-muted/40">
        {src ? (
          <img src={src} alt="" className="max-h-44 w-full object-contain" />
        ) : (
          <div className="flex h-24 items-center justify-center text-muted-foreground">
            <ImageIcon className="size-6 opacity-60" />
          </div>
        )}
      </div>
      <div className="flex flex-wrap gap-2">
        <Button type="button" size="sm" variant="outline" disabled={busy} onClick={() => inputRef.current?.click()}>
          {busy ? <Loader2 className="size-3.5 animate-spin" /> : <Upload className="size-3.5" />}
          {t("template.field.upload")}
        </Button>
        {value ? (
          <Button type="button" size="sm" variant="ghost" onClick={() => onChange("")}>
            <Trash2 className="size-3.5" />
            {t("template.field.remove")}
          </Button>
        ) : null}
      </div>
      <input
        ref={inputRef}
        type="file"
        accept="image/png,image/jpeg,image/webp,image/gif,image/svg+xml"
        className="hidden"
        onChange={(event) => {
          const file = event.target.files?.[0];
          event.target.value = "";
          if (file) void upload(file);
        }}
      />
      <Input
        value={draft}
        placeholder={t("template.field.image_url")}
        onChange={(event) => {
          const next = event.target.value;
          setDraft(next);
          if (safeImageUrl(next.trim())) {
            last.current = next.trim();
            onChange(next.trim());
          }
        }}
        className="h-8 font-mono text-xs"
      />
    </FieldShell>
  );
}

export function IconField({
  label,
  value,
  onChange,
  placeholder,
}: {
  label: string;
  value: string;
  onChange: (next: string) => void;
  placeholder?: string;
}) {
  const { t } = useEditor();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const shown = value || placeholder || "";
  const Current = siteIcon(shown);
  const needle = query.trim().toLowerCase();
  const names = needle ? SITE_ICON_NAMES.filter((name) => name.includes(needle)) : SITE_ICON_NAMES;
  return (
    <FieldShell label={label}>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button type="button" variant="outline" size="sm" className="h-8 w-full justify-start gap-2 font-normal">
            {Current ? <Current className="size-4" /> : <Ban className="size-4 text-muted-foreground" />}
            <span className="truncate text-xs">
              {value
                ? value
                : placeholder
                  ? t("template.field.icon_default", { name: placeholder })
                  : t("template.field.no_icon")}
            </span>
          </Button>
        </PopoverTrigger>
        <PopoverContent align="start" className="w-[296px] p-2">
          <Input
            autoFocus
            value={query}
            placeholder={t("template.field.icon_search")}
            onChange={(event) => setQuery(event.target.value)}
            className="h-8 text-xs"
          />
          <div className="mt-2 grid max-h-60 grid-cols-8 gap-1 overflow-y-auto pr-1">
            <button
              type="button"
              title={t("template.field.no_icon")}
              onClick={() => {
                onChange("");
                setOpen(false);
              }}
              className={cn(
                "flex size-8 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent",
                !value && "bg-accent"
              )}
            >
              <Ban className="size-4" />
            </button>
            {names.map((name) => {
              const Icon = SITE_ICONS[name];
              return (
                <button
                  key={name}
                  type="button"
                  title={name}
                  onClick={() => {
                    onChange(name);
                    setOpen(false);
                  }}
                  className={cn(
                    "flex size-8 items-center justify-center rounded-md transition-colors hover:bg-accent",
                    value === name && "bg-primary text-primary-foreground hover:bg-primary"
                  )}
                >
                  <Icon className="size-4" />
                </button>
              );
            })}
          </div>
        </PopoverContent>
      </Popover>
    </FieldShell>
  );
}

export function BoolField({
  label,
  hint,
  checked,
  onChange,
}: {
  label: string;
  hint?: string;
  checked: boolean;
  onChange: (next: boolean) => void;
}) {
  const id = useId();
  return (
    <div className="flex items-start justify-between gap-3 rounded-lg border bg-muted/30 px-3 py-2.5">
      <div className="min-w-0 space-y-0.5">
        <Label htmlFor={id} className="text-xs font-medium">
          {label}
        </Label>
        {hint ? <p className="text-[11px] leading-snug text-muted-foreground">{hint}</p> : null}
      </div>
      <Switch id={id} checked={checked} onCheckedChange={onChange} />
    </div>
  );
}

export function ChoiceField({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: string;
  options: { value: string; label: string }[];
  onChange: (next: string) => void;
}) {
  return (
    <FieldShell label={label}>
      <div className="flex flex-wrap gap-1 rounded-lg bg-muted/60 p-0.5">
        {options.map((option) => (
          <button
            key={option.value}
            type="button"
            onClick={() => onChange(option.value)}
            className={cn(
              "h-7 grow rounded-md px-2.5 text-xs whitespace-nowrap transition-colors",
              value === option.value
                ? "bg-background font-medium text-foreground shadow-xs"
                : "text-muted-foreground hover:text-foreground"
            )}
          >
            {option.label}
          </button>
        ))}
      </div>
    </FieldShell>
  );
}
