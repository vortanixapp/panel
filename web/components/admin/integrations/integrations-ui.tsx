"use client";

import { Copy } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

export function fmtDateTime(iso: string | null): string {
  if (!iso) return "—";
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString(localeTag());
}

export function TabHeader({
  title,
  description,
  action,
}: {
  title: string;
  description?: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div className="min-w-0 max-w-3xl space-y-1">
        <h2 className="text-lg leading-tight font-semibold">{title}</h2>
        {description ? (
          <p className="text-sm text-muted-foreground">{description}</p>
        ) : null}
      </div>
      {action ? <div className="flex shrink-0 flex-wrap gap-2">{action}</div> : null}
    </div>
  );
}

export function OneTimeSecret({
  title,
  value,
  hint,
  onDismiss,
}: {
  title: string;
  value: string;
  hint?: React.ReactNode;
  onDismiss: () => void;
}) {
  const t = useT();

  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      toast.success(t("common.copied"));
    } catch {
      toast.error(t("common.copy_failed"));
    }
  }

  return (
    <div className="flex flex-col gap-3 rounded-xl border border-amber-500/40 bg-amber-500/5 p-4">
      <div className="text-sm font-medium">{title}</div>
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <code className="min-w-0 flex-1 rounded-md border bg-background px-3 py-2 font-mono text-xs break-all">
          {value}
        </code>
        <div className="flex shrink-0 gap-2">
          <Button size="sm" onClick={() => void copy()}>
            <Copy className="mr-1.5 size-3.5" />
            {t("common.copy")}
          </Button>
          <Button size="sm" variant="outline" onClick={onDismiss}>
            {t("admin.integrations.copied")}
          </Button>
        </div>
      </div>
      {hint ? <div className="text-xs text-muted-foreground">{hint}</div> : null}
    </div>
  );
}

export function EmptyBlock({ children }: { children: React.ReactNode }) {
  return (
    <div className="px-6 py-10 text-center text-sm text-muted-foreground">
      {children}
    </div>
  );
}
