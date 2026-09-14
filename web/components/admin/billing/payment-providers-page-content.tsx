"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, ChevronDown, Copy, Search } from "lucide-react";
import { toast } from "sonner";

import {
  FieldGrid,
  Segmented,
  SelectField,
  SettingsCard,
  TextAreaField,
  TextField,
  ToggleRow,
} from "@/components/admin/settings/settings-ui";
import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import {
  fetchAdminPaymentProviders,
  updateAdminPaymentProvider,
  updateAdminPaymentSettings,
  type AdminPaymentProvider,
  type AdminPaymentProvidersData,
  type AdminPaymentSettings,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

type Filter = "all" | "enabled" | "disabled";

const FILTERS: { id: Filter; labelKey: string }[] = [
  { id: "all", labelKey: "admin.psp.filter.all" },
  { id: "enabled", labelKey: "admin.psp.filter.enabled" },
  { id: "disabled", labelKey: "admin.psp.filter.disabled" },
];

function formatPercent(value: number): string {
  return String(Number((value || 0).toFixed(2)));
}

function initialConfig(provider: AdminPaymentProvider): Record<string, string> {
  const out: Record<string, string> = {};
  for (const field of provider.fields) {
    if (field.type === "password") {
      out[field.key] = "";
      continue;
    }
    const stored = provider.config[field.key] ?? "";
    out[field.key] = stored || (field.type === "select" || field.type === "checkbox" ? field.default ?? "" : "");
  }
  return out;
}

function rowVersion(provider: AdminPaymentProvider): string {
  return [
    provider.key,
    provider.enabled,
    provider.fee_percent,
    JSON.stringify(provider.config),
    JSON.stringify(provider.secrets),
  ].join("|");
}

export function PaymentProvidersPageContent() {
  const t = useT();
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<Filter>("all");
  const [open, setOpen] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminPaymentProviders,
    queryFn: fetchAdminPaymentProviders,
  });

  const providers = useMemo(() => data?.providers ?? [], [data?.providers]);
  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return providers.filter((p) => {
      if (filter === "enabled" && !p.enabled) return false;
      if (filter === "disabled" && p.enabled) return false;
      return !q || p.name.toLowerCase().includes(q) || p.key.includes(q);
    });
  }, [providers, query, filter]);
  const enabledCount = providers.filter((p) => p.enabled).length;

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-6 pb-10">
        <div className="space-y-1.5">
          <h1 className="text-[26px] leading-none font-bold tracking-tight">
            {t("admin.psp.title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.psp.subtitle")}
          </p>
        </div>

        {isLoading || !data ? (
          <div className="space-y-4">
            <Skeleton className="h-40 w-full" />
            <Skeleton className="h-96 w-full" />
          </div>
        ) : (
          <>
            <PaymentSettingsCard
              key={`${data.settings.default_currency}|${data.settings.fx_fee_percent}`}
              settings={data.settings}
            />

            <SettingsCard
              title={t("admin.psp.list_title")}
              description={t("admin.psp.list_description", {
                enabled: enabledCount,
                total: providers.length,
              })}
            >
              <div className="mb-4 flex flex-wrap items-center gap-3">
                <div className="relative min-w-[220px] flex-1">
                  <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                    placeholder={t("admin.psp.search")}
                    className="h-[38px] rounded-lg pl-9 text-[13px] shadow-none md:text-[13px]"
                  />
                </div>
                <Segmented
                  size="sm"
                  items={FILTERS.map((f) => ({ id: f.id, label: t(f.labelKey) }))}
                  value={filter}
                  onChange={setFilter}
                />
              </div>

              <div className="space-y-2.5">
                {visible.length === 0 && (
                  <div className="rounded-xl border border-dashed px-4 py-6 text-center text-[13px] text-muted-foreground">
                    {t("admin.psp.empty")}
                  </div>
                )}
                {visible.map((provider) => (
                  <ProviderRow
                    key={rowVersion(provider)}
                    provider={provider}
                    open={open === provider.key}
                    onToggleOpen={() =>
                      setOpen((cur) => (cur === provider.key ? null : provider.key))
                    }
                    onRequireOpen={() => setOpen(provider.key)}
                  />
                ))}
              </div>
            </SettingsCard>
          </>
        )}
      </div>
    </PageShell>
  );
}

function PaymentSettingsCard({ settings }: { settings: AdminPaymentSettings }) {
  const t = useT();
  const queryClient = useQueryClient();
  const [currency, setCurrency] = useState(settings.default_currency);
  const [fxFee, setFxFee] = useState(formatPercent(settings.fx_fee_percent));

  const dirty =
    currency !== settings.default_currency ||
    fxFee.trim() !== formatPercent(settings.fx_fee_percent);

  const mutation = useMutation({
    mutationFn: () =>
      updateAdminPaymentSettings({ default_currency: currency, fx_fee_percent: fxFee }),
    onSuccess: () => {
      toast.success(t("admin.psp.settings_saved"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminPaymentProviders });
    },
    onError: (e: Error) => toast.error(e.message || t("common.save_failed")),
  });

  return (
    <SettingsCard
      title={t("admin.psp.settings_title")}
      description={t("admin.psp.settings_description")}
      action={
        <Button
          type="button"
          className="h-8 px-4 text-[13px]"
          disabled={!dirty || mutation.isPending}
          onClick={() => mutation.mutate()}
        >
          {mutation.isPending ? t("common.saving") : t("common.save")}
        </Button>
      }
    >
      <FieldGrid cols={2}>
        <SelectField
          label={t("admin.psp.default_currency")}
          value={currency}
          onChange={setCurrency}
          options={settings.currencies}
        />
        <TextField
          mono
          label={t("admin.psp.fx_fee")}
          value={fxFee}
          onChange={setFxFee}
          placeholder="0"
        />
      </FieldGrid>
      <p className="mt-3 text-[12.5px] text-muted-foreground">
        {t("admin.psp.fx_fee_hint")}
      </p>
    </SettingsCard>
  );
}

function ProviderRow({
  provider,
  open,
  onToggleOpen,
  onRequireOpen,
}: {
  provider: AdminPaymentProvider;
  open: boolean;
  onToggleOpen: () => void;
  onRequireOpen: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const baseline = useMemo(() => initialConfig(provider), [provider]);
  const [config, setConfig] = useState<Record<string, string>>(baseline);
  const [fee, setFee] = useState(formatPercent(provider.fee_percent));
  const [copied, setCopied] = useState(false);

  const dirty =
    fee.trim() !== formatPercent(provider.fee_percent) ||
    Object.keys(config).some((k) => (config[k] ?? "") !== (baseline[k] ?? ""));

  const applyResult = (updated: AdminPaymentProvider) => {
    queryClient.setQueryData<AdminPaymentProvidersData>(
      queryKeys.adminPaymentProviders,
      (prev) =>
        prev
          ? {
              ...prev,
              providers: prev.providers.map((p) => (p.key === updated.key ? updated : p)),
            }
          : prev
    );
  };

  const toggle = useMutation({
    mutationFn: (enabled: boolean) =>
      updateAdminPaymentProvider(
        provider.key,
        dirty ? { enabled, fee_percent: fee, config } : { enabled }
      ),
    onSuccess: (res, enabled) => {
      applyResult(res.provider);
      toast.success(
        t(enabled ? "admin.psp.enabled_toast" : "admin.psp.disabled_toast", {
          name: provider.name,
        })
      );
    },
    onError: (e: Error) => {
      toast.error(e.message || t("admin.psp.status_failed"));
      onRequireOpen();
    },
  });

  const save = useMutation({
    mutationFn: () => updateAdminPaymentProvider(provider.key, { fee_percent: fee, config }),
    onSuccess: (res) => {
      applyResult(res.provider);
      toast.success(t("admin.psp.config_saved"));
    },
    onError: (e: Error) => toast.error(e.message || t("admin.psp.config_failed")),
  });

  const setField = (key: string, value: string) =>
    setConfig((prev) => ({ ...prev, [key]: value }));

  const copyWebhook = () => {
    if (!provider.webhook_url) return;
    void navigator.clipboard.writeText(provider.webhook_url).then(() => {
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    });
  };

  const currencyLabel = (() => {
    switch (provider.currency_mode) {
      case "fixed":
        return t("admin.psp.currency_fixed", { currency: provider.currency });
      case "setting":
        return t("admin.psp.currency_setting", { currency: provider.currency });
      case "account":
        return provider.currency
          ? t("admin.psp.currency_account", { currency: provider.currency })
          : t("admin.psp.currency_account_unknown");
      default:
        return t("admin.psp.currency_wallet");
    }
  })();

  return (
    <div
      className={cn(
        "rounded-xl border bg-muted/30",
        provider.enabled && "border-primary/40"
      )}
    >
      <div className="flex flex-wrap items-center justify-between gap-4 px-5 py-4">
        <button
          type="button"
          onClick={onToggleOpen}
          className="flex min-w-0 flex-1 items-center gap-3 text-left"
        >
          <ChevronDown
            className={cn(
              "size-4 shrink-0 text-muted-foreground transition-transform",
              open && "rotate-180"
            )}
          />
          <span className="min-w-0">
            <span className="flex flex-wrap items-center gap-2">
              <span className="truncate text-sm font-semibold">{provider.name}</span>
              {!provider.configured && (
                <Badge className="bg-amber-500/10 text-amber-600 ring-1 ring-amber-500/20 ring-inset hover:bg-amber-500/10">
                  {t("admin.psp.needs_setup")}
                </Badge>
              )}
              {provider.unreadable && (
                <Badge variant="destructive">{t("admin.psp.unreadable")}</Badge>
              )}
              {provider.fee_percent > 0 && (
                <Badge variant="outline" className="font-mono">
                  {t("admin.psp.fee_badge", { fee: formatPercent(provider.fee_percent) })}
                </Badge>
              )}
            </span>
            <span className="block truncate font-mono text-[11px] text-muted-foreground">
              {provider.key} · {currencyLabel}
            </span>
          </span>
        </button>
        <label className="flex items-center gap-3.5">
          <span className="text-xs text-muted-foreground">
            {provider.enabled ? t("admin.psp.enabled") : t("admin.psp.disabled")}
          </span>
          <Switch
            checked={provider.enabled}
            disabled={toggle.isPending}
            onCheckedChange={(checked) => toggle.mutate(checked)}
          />
        </label>
      </div>

      {open && (
        <div className="space-y-5 border-t px-5 py-5">
          {provider.webhook_url && (
            <div className="space-y-2">
              <div className="text-xs text-muted-foreground">
                {t("admin.psp.webhook_url")}
              </div>
              <div className="flex items-center gap-2">
                <code className="min-w-0 flex-1 truncate rounded-lg border bg-background px-3 py-2 font-mono text-[12.5px]">
                  {provider.webhook_url}
                </code>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="h-[34px] gap-1.5 text-[13px]"
                  onClick={copyWebhook}
                >
                  {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
                  {copied ? t("admin.psp.copied") : t("admin.psp.copy")}
                </Button>
              </div>
              <p className="text-[12px] text-muted-foreground">
                {t("admin.psp.webhook_hint")}
              </p>
            </div>
          )}

          {provider.manual && (
            <p className="text-[12.5px] text-muted-foreground">
              {t("admin.psp.manual_hint")}
            </p>
          )}

          <FieldGrid cols={3}>
            <TextField
              mono
              label={t("admin.psp.fee")}
              value={fee}
              onChange={setFee}
              placeholder="0"
            />
            {provider.fields.map((field) => {
              const label = field.required ? `${field.label} *` : field.label;
              const value = config[field.key] ?? "";
              if (field.type === "checkbox") {
                return (
                  <div key={field.key} className="sm:col-span-2 lg:col-span-3">
                    <ToggleRow
                      label={field.label}
                      checked={value === "1"}
                      onCheckedChange={(checked) => setField(field.key, checked ? "1" : "0")}
                    />
                  </div>
                );
              }
              if (field.type === "select" && field.options?.length) {
                return (
                  <SelectField
                    key={field.key}
                    label={label}
                    value={value || field.default || ""}
                    onChange={(v) => setField(field.key, v)}
                    options={field.options}
                  />
                );
              }
              if (field.type === "textarea") {
                return (
                  <TextAreaField
                    key={field.key}
                    span={3}
                    mono
                    label={label}
                    value={value}
                    onChange={(v) => setField(field.key, v)}
                  />
                );
              }
              return (
                <TextField
                  key={field.key}
                  mono
                  type={field.type === "password" ? "password" : "text"}
                  label={label}
                  value={value}
                  onChange={(v) => setField(field.key, v)}
                  placeholder={
                    field.type === "password"
                      ? provider.secrets[field.key]
                        ? t("admin.psp.secret_saved")
                        : ""
                      : field.default ?? ""
                  }
                  autoComplete={field.type === "password" ? "new-password" : "off"}
                />
              );
            })}
          </FieldGrid>

          <div className="flex flex-wrap items-center justify-between gap-3">
            <span className="text-xs text-muted-foreground">{t("admin.psp.fee_hint")}</span>
            <div className="flex items-center gap-2.5">
              <Button
                type="button"
                variant="ghost"
                className="h-9 text-[13px] text-muted-foreground"
                disabled={!dirty || save.isPending}
                onClick={() => {
                  setConfig(baseline);
                  setFee(formatPercent(provider.fee_percent));
                }}
              >
                {t("admin.settings.discard")}
              </Button>
              <Button
                type="button"
                className="h-9 px-5 text-[13px]"
                disabled={!dirty || save.isPending}
                onClick={() => save.mutate()}
              >
                {save.isPending ? t("common.saving") : t("common.save")}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
