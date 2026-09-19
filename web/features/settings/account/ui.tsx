"use client";

import { useEffect, useState } from "react";
import { Loader2, RotateCcw } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";

export function SettingsSection({
  id,
  title,
  description,
  action,
  danger,
  className,
  children,
}: {
  id?: string;
  title?: React.ReactNode;
  description?: React.ReactNode;
  action?: React.ReactNode;
  danger?: boolean;
  className?: string;
  children?: React.ReactNode;
}) {
  return (
    <section
      id={id}
      className={cn(
        "scroll-mt-24 rounded-2xl border bg-card p-5 sm:p-6",
        danger ? "border-destructive/40" : "border-border",
        className
      )}
    >
      {(title || action) && (
        <div className={cn("flex flex-wrap items-start justify-between gap-3", children != null && "mb-5")}>
          <div className="min-w-0 space-y-1">
            {title && (
              <h3 className={cn("text-[15px] leading-tight font-semibold", danger && "text-destructive")}>
                {title}
              </h3>
            )}
            {description && (
              <p className="text-[13px] leading-relaxed text-muted-foreground">{description}</p>
            )}
          </div>
          {action && <div className="flex flex-wrap items-center gap-2">{action}</div>}
        </div>
      )}
      {children}
    </section>
  );
}

export function QueryState({
  isLoading,
  isError,
  error,
  onRetry,
  rows = 2,
  children,
}: {
  isLoading: boolean;
  isError: boolean;
  error?: unknown;
  onRetry: () => void;
  rows?: number;
  children: React.ReactNode;
}) {
  const t = useT();
  if (isLoading) {
    return (
      <div className="space-y-2.5">
        {Array.from({ length: rows }, (_, i) => (
          <Skeleton key={i} className="h-14 w-full rounded-xl" />
        ))}
      </div>
    );
  }
  if (isError) {
    return (
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-[13px] text-destructive">
        <span>{error instanceof Error && error.message ? error.message : t("settings.common.load_failed")}</span>
        <Button size="sm" variant="outline" onClick={onRetry}>
          <RotateCcw className="size-3.5" />
          {t("settings.common.retry")}
        </Button>
      </div>
    );
  }
  return <>{children}</>;
}

export function Field({
  label,
  hint,
  htmlFor,
  className,
  children,
}: {
  label: React.ReactNode;
  hint?: React.ReactNode;
  htmlFor?: string;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div className={cn("space-y-1.5", className)}>
      <label htmlFor={htmlFor} className="text-[13px] font-medium">
        {label}
      </label>
      {children}
      {hint && <p className="text-[12px] leading-relaxed text-muted-foreground">{hint}</p>}
    </div>
  );
}

export function FormActions({
  dirty,
  pending,
  saveLabel,
  onReset,
}: {
  dirty: boolean;
  pending: boolean;
  saveLabel?: string;
  onReset?: () => void;
}) {
  const t = useT();
  return (
    <div className="mt-5 flex flex-wrap items-center gap-2.5">
      <Button type="submit" disabled={!dirty || pending}>
        {pending && <Loader2 className="size-4 animate-spin" />}
        {pending ? t("common.saving") : saveLabel ?? t("common.save")}
      </Button>
      {dirty && onReset && (
        <Button type="button" variant="ghost" disabled={pending} onClick={onReset}>
          {t("settings.common.reset")}
        </Button>
      )}
      {dirty && !pending && (
        <span className="text-[12.5px] text-amber-600 dark:text-amber-500">{t("settings.common.unsaved")}</span>
      )}
    </div>
  );
}

export function useUnsavedGuard(dirty: boolean) {
  useEffect(() => {
    if (!dirty) return;
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      e.returnValue = "";
    };
    window.addEventListener("beforeunload", handler);
    return () => window.removeEventListener("beforeunload", handler);
  }, [dirty]);
}

export function useCopy() {
  const t = useT();
  const [copied, setCopied] = useState("");
  const copy = async (value: string, key = value) => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(key);
      toast.success(t("settings.common.copied"));
      window.setTimeout(() => setCopied((c) => (c === key ? "" : c)), 1600);
    } catch {
      toast.error(t("settings.common.copy_failed"));
    }
  };
  return { copied, copy };
}

export function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleString(localeTag(), {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function formatDate(iso: string | null | undefined): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleDateString(localeTag(), { day: "numeric", month: "long", year: "numeric" });
}

export function errorText(err: unknown, fallback: string): string {
  return err instanceof Error && err.message ? err.message : fallback;
}

export function StatusDot({ tone }: { tone: "ok" | "warn" | "bad" | "off" }) {
  return (
    <span
      className={cn(
        "inline-block size-2 shrink-0 rounded-full",
        tone === "ok" && "bg-emerald-500",
        tone === "warn" && "bg-amber-500",
        tone === "bad" && "bg-rose-500",
        tone === "off" && "bg-muted-foreground/40"
      )}
    />
  );
}
