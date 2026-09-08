"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { AlertTriangle, Check, Copy, KeyRound, RefreshCw } from "lucide-react";

import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  bindAdminLicenseKey,
  fetchAdminLicense,
  refreshAdminLicense,
  type AdminLicenseInfo,
} from "@/lib/api";
import { cn } from "@/lib/utils";
import { localeTag, type TranslateFn } from "@/lib/i18n";
import { useT } from "@/hooks/use-translations";

export default function AdminLicensePage() {
  const t = useT();
  const queryClient = useQueryClient();
  const [bindOpen, setBindOpen] = useState(false);
  const [licenseKey, setLicenseKey] = useState("");
  const [copied, setCopied] = useState<string>("");

  const { data, isLoading } = useQuery({
    queryKey: ["admin-license"],
    queryFn: fetchAdminLicense,
    refetchInterval: 60_000,
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ["admin-license"] });
    void queryClient.invalidateQueries({ queryKey: ["license-state"] });
  };

  const refreshMutation = useMutation({
    mutationFn: refreshAdminLicense,
    onSuccess: () => {
      toast.success(t("admin.license.check_started"));
      setTimeout(invalidate, 2500);
    },
    onError: (err: unknown) =>
      toast.error(
        err instanceof Error ? err.message : t("admin.license.check_failed")
      ),
  });

  const bindMutation = useMutation({
    mutationFn: () => bindAdminLicenseKey(licenseKey.trim()),
    onSuccess: () => {
      toast.success(t("admin.license.key_bound"));
      setBindOpen(false);
      setLicenseKey("");
      invalidate();
    },
    onError: (err: unknown) =>
      toast.error(
        err instanceof Error ? err.message : t("admin.license.bind_failed")
      ),
  });

  const copy = (what: string, text: string) => {
    if (!text) return;
    void navigator.clipboard?.writeText(text).catch(() => {});
    setCopied(what);
    setTimeout(() => setCopied(""), 1400);
  };

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div className="space-y-1">
          <h1 className="text-2xl font-bold tracking-tight">
            {t("admin.license.title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.license.subtitle")}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => refreshMutation.mutate()}
            disabled={refreshMutation.isPending}
          >
            <RefreshCw
              className={cn("size-4", refreshMutation.isPending && "animate-spin")}
            />
            {t("layout.check_now")}
          </Button>
          <Button size="sm" onClick={() => setBindOpen(true)}>
            <KeyRound className="size-4" />
            {t("admin.license.bind_key")}
          </Button>
        </div>
      </div>

      {isLoading || !data ? (
        <div className="space-y-4">
          <Skeleton className="h-28 w-full" />
          <div className="grid gap-4 lg:grid-cols-[1.15fr_1fr]">
            <Skeleton className="h-72 w-full" />
            <Skeleton className="h-72 w-full" />
          </div>
        </div>
      ) : (
        <div className="space-y-4">
          {data.legacy && <LegacyNotice onBind={() => setBindOpen(true)} />}

          <StatusStrip data={data} />

          <div className="grid gap-4 lg:grid-cols-[1.15fr_1fr]">
            <UsageCard data={data} />
            <div className="flex flex-col gap-4">
              <InstallationCard data={data} copied={copied} onCopy={copy} />
              <DiagnosticsCard data={data} />
            </div>
          </div>
        </div>
      )}

      <BindKeyDialog
        open={bindOpen}
        onOpenChange={setBindOpen}
        value={licenseKey}
        onChange={setLicenseKey}
        pending={bindMutation.isPending}
        onSubmit={() => bindMutation.mutate()}
      />
    </PageShell>
  );
}

type Tone = {
  text: string;
  dot: string;
};

const TONES: Record<AdminLicenseInfo["state"], Tone> = {
  active: { text: "text-foreground", dot: "bg-foreground" },
  grace: { text: "text-amber-500", dot: "bg-amber-500" },
  read_only: { text: "text-red-400", dot: "bg-red-400" },
};

// Подписи храним ключами: запись читается на уровне модуля, и готовый текст
// застыл бы на языке, который стоял в момент загрузки страницы.
const MODE_TITLE_KEYS: Record<AdminLicenseInfo["state"], string> = {
  active: "admin.license.mode_active",
  grace: "admin.license.mode_grace",
  read_only: "admin.license.mode_read_only",
};

const MODE_TEXT_KEYS: Record<AdminLicenseInfo["state"], string> = {
  active: "admin.license.mode_active_desc",
  grace: "admin.license.mode_grace_desc",
  read_only: "admin.license.mode_read_only_desc",
};

function modeText(data: AdminLicenseInfo, t: TranslateFn): string {
  if (data.message) return data.message;
  return t(MODE_TEXT_KEYS[data.state]);
}

function StatusStrip({ data }: { data: AdminLicenseInfo }) {
  const t = useT();
  const tone = TONES[data.state];
  const expiry = expiryFields(data, t);

  return (
    <div className="rounded-xl border bg-card p-6">
      <div className="grid gap-6 md:grid-cols-[minmax(220px,1.2fr)_auto_1fr_auto_1fr] md:items-center">
        <div className="space-y-2.5">
          <div className="flex items-center gap-2.5">
            <span className={cn("size-2 rounded-full", tone.dot)} />
            <span className={cn("text-lg font-semibold tracking-tight", tone.text)}>
              {t(MODE_TITLE_KEYS[data.state])}
            </span>
          </div>
          <p className="text-[13px] leading-relaxed text-muted-foreground">
            {modeText(data, t)}
          </p>
        </div>

        <Divider />

        <Field label={t("admin.license.plan")}>
          <div className="text-lg font-semibold uppercase">{data.plan || "—"}</div>
          <div className="text-xs text-muted-foreground">
            {data.tenant_name} · {data.tenant_slug}
          </div>
        </Field>

        <Divider />

        <Field label={expiry.label}>
          <div className={cn("text-lg font-semibold", expiry.tone)}>{expiry.value}</div>
          <div className="text-xs text-muted-foreground">{expiry.hint}</div>
        </Field>
      </div>
    </div>
  );
}

function Divider() {
  return <div className="hidden w-px self-stretch bg-border md:block" />;
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1.5">
      <div className="font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      {children}
    </div>
  );
}

function expiryFields(
  data: AdminLicenseInfo,
  t: TranslateFn
): {
  label: string;
  value: string;
  hint: string;
  tone: string;
} {
  if (data.state === "grace" && data.grace_until) {
    return {
      label: t("admin.license.grace_until"),
      value: relative(data.grace_until),
      hint: absolute(data.grace_until),
      tone: "text-amber-500",
    };
  }

  const expires = data.license_expires_at;
  if (!expires) {
    return {
      label: t("admin.license.term"),
      value: t("admin.license.perpetual"),
      hint: t("admin.license.perpetual_hint"),
      tone: "text-foreground",
    };
  }

  const past = new Date(expires).getTime() < Date.now();
  return {
    label: past ? t("admin.license.expired") : t("admin.license.valid_until"),
    value: past ? relative(expires) : absolute(expires, { dateOnly: true }),
    hint: past ? absolute(expires) : remaining(expires, t),
    tone: past ? "text-red-400" : "text-foreground",
  };
}

function UsageCard({ data }: { data: AdminLicenseInfo }) {
  const t = useT();
  const rows = [
    { label: t("layout.nodes"), used: data.usage.nodes, limit: data.limits.max_nodes },
    {
      label: t("common.servers"),
      used: data.usage.servers,
      limit: data.limits.max_servers,
    },
    {
      label: t("admin.license.admins"),
      used: data.usage.admins,
      limit: data.limits.max_admins,
    },
  ];

  return (
    <div className="flex flex-col gap-5 rounded-xl border bg-card p-6">
      <div className="flex items-baseline justify-between">
        <h2 className="text-[15px] font-semibold">{t("admin.license.usage_title")}</h2>
        <span className="font-mono text-[11px] text-muted-foreground">
          {t("admin.license.usage_hint")}
        </span>
      </div>

      <div className="flex flex-col gap-4">
        {rows.map((row) => (
          <UsageRow key={row.label} {...row} />
        ))}
      </div>

      <div className="flex items-center justify-between border-t pt-4 text-[13px]">
        <span>{t("admin.license.api_rate")}</span>
        <span className="font-mono">
          {data.limits.api_rpm > 0
            ? t("admin.license.api_rpm", { rpm: data.limits.api_rpm })
            : t("admin.license.api_unlimited")}
        </span>
      </div>
    </div>
  );
}

function UsageRow({ label, used, limit }: { label: string; used: number; limit: number }) {
  const unlimited = limit < 0;
  const unset = limit === 0;
  const percent = unlimited || unset ? 0 : Math.min(100, Math.round((used / limit) * 100));

  return (
    <div className="space-y-2">
      <div className="flex justify-between text-[13px]">
        <span>{label}</span>
        <span className="font-mono">
          {used}
          <span className="text-muted-foreground">
            {unlimited ? " / ∞" : unset ? "" : ` / ${limit}`}
          </span>
        </span>
      </div>
      <div className="h-1 overflow-hidden rounded-full bg-muted">
        <div
          className={cn(
            "h-full transition-[width]",
            percent >= 100 ? "bg-red-400" : percent >= 80 ? "bg-amber-500" : "bg-foreground",
          )}
          style={{ width: `${percent}%` }}
        />
      </div>
    </div>
  );
}

function InstallationCard({
  data,
  copied,
  onCopy,
}: {
  data: AdminLicenseInfo;
  copied: string;
  onCopy: (what: string, text: string) => void;
}) {
  const t = useT();
  return (
    <div className="flex flex-col gap-4 rounded-xl border bg-card p-6">
      <h2 className="text-[15px] font-semibold">{t("admin.license.install_title")}</h2>

      <Field label="Installation ID">
        <div className="flex items-center gap-2">
          <span className="truncate font-mono text-xs">{data.installation_id || "—"}</span>
          <CopyButton
            done={copied === "install"}
            onClick={() => onCopy("install", data.installation_id)}
          />
        </div>
      </Field>

      <Field label={t("admin.license.key")}>
        <div className="flex items-center gap-2">
          <span className="font-mono text-xs">
            {data.key_hint
              ? `VRTX-••••-${data.key_hint}`
              : t("admin.license.key_unbound")}
          </span>
        </div>
      </Field>
    </div>
  );
}

function CopyButton({ done, onClick }: { done: boolean; onClick: () => void }) {
  const t = useT();
  return (
    <Button
      variant="outline"
      size="sm"
      className="h-6 shrink-0 px-2 font-mono text-[11px]"
      onClick={onClick}
    >
      {done ? <Check className="size-3" /> : <Copy className="size-3" />}
      {done ? t("admin.license.copy_done") : t("admin.license.copy")}
    </Button>
  );
}

function DiagnosticsCard({ data }: { data: AdminLicenseInfo }) {
  const t = useT();
  return (
    <div className="flex flex-col gap-3 rounded-xl border bg-card p-6">
      <h2 className="text-[15px] font-semibold">{t("admin.license.diag_title")}</h2>

      <Row label={t("admin.license.diag_last")}>
        {data.last_verified_at
          ? relative(data.last_verified_at)
          : t("admin.license.diag_never")}
      </Row>
      <Row label={t("admin.license.diag_revision")}>#{data.revision}</Row>
      <Row
        label={t("admin.license.diag_error")}
        tone={data.last_error ? "text-red-400" : undefined}
      >
        {data.last_error || "—"}
      </Row>
    </div>
  );
}

function Row({
  label,
  tone,
  children,
}: {
  label: string;
  tone?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex justify-between gap-4 text-[13px]">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span className={cn("text-right font-mono break-all", tone)}>{children}</span>
    </div>
  );
}

function LegacyNotice({ onBind }: { onBind: () => void }) {
  const t = useT();
  return (
    <div className="flex flex-wrap items-center gap-3 rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3">
      <AlertTriangle className="size-4 shrink-0 text-amber-500" />
      <p className="flex-1 text-[13px] leading-relaxed">
        {t("admin.license.legacy_notice")}
      </p>
      <Button variant="outline" size="sm" onClick={onBind}>
        {t("admin.license.legacy_bind")}
      </Button>
    </div>
  );
}

function BindKeyDialog({
  open,
  onOpenChange,
  value,
  onChange,
  pending,
  onSubmit,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  value: string;
  onChange: (value: string) => void;
  pending: boolean;
  onSubmit: () => void;
}) {
  const t = useT();
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t("admin.license.bind_key")}</DialogTitle>
          <DialogDescription>
            {t("admin.license.bind_dialog_desc")}
          </DialogDescription>
        </DialogHeader>

        <Input
          autoFocus
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="VRTX-XXXXX-XXXXX-XXXXX-XXXXX"
          className="font-mono"
          onKeyDown={(e) => {
            if (e.key === "Enter" && value.trim() && !pending) onSubmit();
          }}
        />

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button onClick={onSubmit} disabled={pending || !value.trim()}>
            {pending ? t("admin.license.binding") : t("admin.license.legacy_bind")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

const rtf = new Intl.RelativeTimeFormat("ru", { numeric: "auto" });

function relative(iso: string): string {
  const diffMs = new Date(iso).getTime() - Date.now();
  if (Number.isNaN(diffMs)) return "—";

  const units: Array<[Intl.RelativeTimeFormatUnit, number]> = [
    ["second", 1000],
    ["minute", 60_000],
    ["hour", 3_600_000],
    ["day", 86_400_000],
  ];

  let unit: Intl.RelativeTimeFormatUnit = "day";
  let size = 86_400_000;
  for (const [candidate, ms] of units) {
    unit = candidate;
    size = ms;
    if (Math.abs(diffMs) < ms * 60) break;
  }
  return rtf.format(Math.round(diffMs / size), unit);
}

function absolute(iso: string, opts?: { dateOnly?: boolean }): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleString(localeTag(), {
    day: "numeric",
    month: "short",
    year: "numeric",
    ...(opts?.dateOnly ? {} : { hour: "2-digit", minute: "2-digit" }),
  });
}

function remaining(iso: string, t: TranslateFn): string {
  const days = Math.ceil((new Date(iso).getTime() - Date.now()) / 86_400_000);
  if (Number.isNaN(days) || days < 0) return "—";
  if (days === 0) return t("admin.license.expires_today");
  // Русская форма слова зависит от числа, поэтому выбираем её здесь,
  // а не держим три готовые фразы в словаре.
  const unitKey =
    days % 10 === 1 && days % 100 !== 11
      ? "admin.license.day_one"
      : days % 10 >= 2 && days % 10 <= 4 && (days % 100 < 10 || days % 100 >= 20)
        ? "admin.license.day_few"
        : "admin.license.day_many";
  return t("admin.license.remaining", { days, unit: t(unitKey) });
}
