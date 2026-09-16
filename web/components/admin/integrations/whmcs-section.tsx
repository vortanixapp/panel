"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
  EmptyBlock,
  OneTimeSecret,
  TabHeader,
} from "@/components/admin/integrations/integrations-ui";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  createAdminAPIKey,
  downloadWhmcsModule,
  fetchAdminWhmcsServices,
  fetchAdminWhmcsSettings,
  saveAdminWhmcsSettings,
  type AdminWhmcsService,
  type AdminWhmcsSettings,
} from "@/lib/api";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { dateLocaleTag } from "@/lib/i18n";

const SETTINGS_KEY = ["admin-whmcs-settings"];
const SERVICES_KEY = ["admin-whmcs-services"];
const DEFAULT_SCOPE = "admin.whmcs.write";

function fmtDate(value: string | null): string {
  if (!value) return "—";
  const date = new Date(`${value}T00:00:00`);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleDateString(dateLocaleTag());
}

function hostOf(value: string | undefined): string {
  if (!value) return "";
  try {
    return new URL(value).host;
  } catch {
    return "";
  }
}

function statusVariant(
  status: AdminWhmcsService["status"]
): "secondary" | "outline" | "destructive" {
  if (status === "active") return "secondary";
  if (status === "suspended") return "destructive";
  return "outline";
}

export function WhmcsSection() {
  const t = useT();
  const qc = useQueryClient();

  const settingsQuery = useQuery({ queryKey: SETTINGS_KEY, queryFn: fetchAdminWhmcsSettings });
  const servicesQuery = useQuery({ queryKey: SERVICES_KEY, queryFn: fetchAdminWhmcsServices });

  const [loaded, setLoaded] = useState(false);
  const [url, setUrl] = useState("");
  const [orderUrl, setOrderUrl] = useState("");
  const [ordersOnly, setOrdersOnly] = useState(false);
  const [issuedKey, setIssuedKey] = useState("");

  const applySettings = (data: AdminWhmcsSettings) => {
    setUrl(data.url);
    setOrderUrl(data.order_url);
    setOrdersOnly(data.orders_only);
  };

  useEffect(() => {
    if (loaded || !settingsQuery.data) return;
    applySettings(settingsQuery.data);
    setLoaded(true);
  }, [loaded, settingsQuery.data]);

  const saveMut = useMutation({
    mutationFn: () =>
      saveAdminWhmcsSettings({ url, order_url: orderUrl, orders_only: ordersOnly }),
    onSuccess: (data) => {
      qc.setQueryData(SETTINGS_KEY, data);
      applySettings(data);
      toast.success(t("admin.integrations.whmcs.saved"));
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.integrations.whmcs.save_failed")),
  });

  const keyMut = useMutation({
    mutationFn: () =>
      createAdminAPIKey({
        name: "WHMCS",
        scopes: [settingsQuery.data?.scope || DEFAULT_SCOPE],
      }),
    onSuccess: (res) => {
      setIssuedKey(res.key);
      void qc.invalidateQueries({ queryKey: ["admin-api-keys"] });
      void qc.invalidateQueries({ queryKey: SETTINGS_KEY });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.integrations.key_create_failed")),
  });

  const downloadMut = useMutation({
    mutationFn: downloadWhmcsModule,
    onError: (e: Error) =>
      toast.error(e.message || t("admin.integrations.whmcs.download_failed")),
  });

  const settings = settingsQuery.data;
  const services = servicesQuery.data?.services ?? [];
  const panelHost =
    hostOf(settings?.panel_url) ||
    (typeof window !== "undefined" ? window.location.host : "");
  const billingHost = hostOf(settings?.url);
  const connected = Boolean(settings?.url) && (settings?.active_keys ?? 0) > 0;

  return (
    <div className="flex flex-col gap-4">
      <TabHeader
        title={t("admin.integrations.whmcs.title")}
        description={t("admin.integrations.whmcs.subtitle")}
      />

      {issuedKey ? (
        <OneTimeSecret
          title={t("admin.integrations.key_once")}
          value={issuedKey}
          hint={t("admin.integrations.whmcs.key_hint")}
          onDismiss={() => setIssuedKey("")}
        />
      ) : null}

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <StatTile
          label={t("common.status")}
          value={
            settingsQuery.isLoading
              ? "…"
              : connected
                ? t("admin.integrations.whmcs.state_connected")
                : t("admin.integrations.whmcs.state_not_connected")
          }
          tone={connected ? "ok" : "muted"}
        />
        <StatTile
          label={t("admin.integrations.whmcs.url")}
          value={billingHost || t("admin.integrations.whmcs.not_set")}
          mono={!!billingHost}
        />
        <StatTile
          label={t("admin.integrations.whmcs.stat_keys")}
          value={String(settings?.active_keys ?? 0)}
        />
        <StatTile
          label={t("admin.integrations.whmcs.services_title")}
          value={String(settings?.services.total ?? 0)}
          sub={
            settings
              ? t("admin.integrations.whmcs.services_short", {
                  active: settings.services.active,
                  suspended: settings.services.suspended,
                })
              : undefined
          }
        />
      </div>

      <div className="grid items-start gap-4 lg:grid-cols-2">
        <section className="flex flex-col gap-4 rounded-xl border bg-card p-5">
          <h3 className="text-[15px] leading-none font-semibold">
            {t("admin.integrations.whmcs.setup_title")}
          </h3>
          <ol className="flex flex-col gap-4">
            <Step index={1} text={t("admin.integrations.whmcs.step_download")}>
              <div className="flex flex-wrap items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={downloadMut.isPending}
                  onClick={() => downloadMut.mutate()}
                >
                  {t("admin.integrations.whmcs.download")}
                </Button>
                {settings?.module_version ? (
                  <span className="text-xs text-muted-foreground">
                    {t("admin.integrations.whmcs.module_version", {
                      version: settings.module_version,
                    })}
                  </span>
                ) : null}
              </div>
            </Step>
            <Step
              index={2}
              text={t("admin.integrations.whmcs.step_key", {
                scope: settings?.scope || DEFAULT_SCOPE,
              })}
            >
              <div className="flex flex-wrap items-center gap-2">
                <Button size="sm" disabled={keyMut.isPending} onClick={() => keyMut.mutate()}>
                  {t("admin.integrations.whmcs.issue_key")}
                </Button>
                <span className="text-xs text-muted-foreground">
                  {t("admin.integrations.whmcs.active_keys", {
                    count: settings?.active_keys ?? 0,
                  })}
                </span>
              </div>
            </Step>
            <Step
              index={3}
              text={t("admin.integrations.whmcs.step_server", { host: panelHost || "—" })}
            />
            <Step index={4} text={t("admin.integrations.whmcs.step_product")}>
              <p className="text-xs text-muted-foreground">
                {t("admin.integrations.whmcs.step_product_options")}{" "}
                <code className="font-mono">
                  game, location, version, slots, cpu_cores, ram_gb, disk_gb, server_name
                </code>
              </p>
            </Step>
          </ol>
        </section>

        <section className="flex flex-col gap-4 rounded-xl border bg-card p-5">
          <h3 className="text-[15px] leading-none font-semibold">
            {t("admin.integrations.whmcs.settings_title")}
          </h3>
          {settingsQuery.isLoading ? (
            <Skeleton className="h-48 w-full" />
          ) : (
            <form
              className="flex flex-col gap-4"
              onSubmit={(e) => {
                e.preventDefault();
                saveMut.mutate();
              }}
            >
              <div className="space-y-1.5">
                <Label htmlFor="whmcs-url">{t("admin.integrations.whmcs.url")}</Label>
                <Input
                  id="whmcs-url"
                  value={url}
                  onChange={(e) => setUrl(e.target.value)}
                  placeholder="https://billing.example.com"
                />
                <p className="text-xs text-muted-foreground">
                  {t("admin.integrations.whmcs.url_hint")}
                </p>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="whmcs-order-url">
                  {t("admin.integrations.whmcs.order_url")}
                </Label>
                <Input
                  id="whmcs-order-url"
                  value={orderUrl}
                  onChange={(e) => setOrderUrl(e.target.value)}
                  placeholder="https://billing.example.com/cart.php?gid=1"
                />
                <p className="text-xs text-muted-foreground">
                  {t("admin.integrations.whmcs.order_url_hint")}
                </p>
              </div>
              <label className="flex cursor-pointer items-start gap-2.5 rounded-lg border p-3 text-sm">
                <Checkbox
                  className="mt-0.5"
                  checked={ordersOnly}
                  onCheckedChange={(value) => setOrdersOnly(value === true)}
                />
                <span>
                  {t("admin.integrations.whmcs.orders_only")}
                  <span className="mt-0.5 block text-xs text-muted-foreground">
                    {t("admin.integrations.whmcs.orders_only_hint")}
                  </span>
                </span>
              </label>
              <div>
                <Button type="submit" disabled={saveMut.isPending}>
                  {t("common.save")}
                </Button>
              </div>
            </form>
          )}
        </section>
      </div>

      <section className="overflow-hidden rounded-xl border bg-card">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b px-5 py-3.5">
          <h3 className="text-[15px] leading-none font-semibold">
            {t("admin.integrations.whmcs.services_title")}
          </h3>
          {settings ? (
            <span className="text-xs text-muted-foreground">
              {t("admin.integrations.whmcs.services_summary", settings.services)}
            </span>
          ) : null}
        </div>
        {servicesQuery.isLoading ? (
          <div className="p-4">
            <Skeleton className="h-16 w-full" />
          </div>
        ) : services.length === 0 ? (
          <EmptyBlock>{t("admin.integrations.whmcs.services_empty")}</EmptyBlock>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[640px] text-sm">
              <thead>
                <tr className="border-b text-left text-xs text-muted-foreground">
                  <th className="px-5 py-2.5 font-medium">
                    {t("admin.integrations.whmcs.col_service")}
                  </th>
                  <th className="px-5 py-2.5 font-medium">
                    {t("admin.integrations.whmcs.col_client")}
                  </th>
                  <th className="px-5 py-2.5 font-medium">
                    {t("admin.integrations.whmcs.col_server")}
                  </th>
                  <th className="px-5 py-2.5 font-medium">{t("common.status")}</th>
                  <th className="px-5 py-2.5 font-medium">
                    {t("admin.integrations.whmcs.col_due")}
                  </th>
                </tr>
              </thead>
              <tbody>
                {services.map((service) => (
                  <tr key={service.service_id} className="border-b last:border-0">
                    <td className="px-5 py-2.5">
                      <span className="font-mono">#{service.service_id}</span>
                      {service.product ? (
                        <div className="text-xs text-muted-foreground">{service.product}</div>
                      ) : null}
                    </td>
                    <td className="px-5 py-2.5">{service.email || `#${service.client_id}`}</td>
                    <td className="px-5 py-2.5">
                      {service.server_id ? (
                        <Link
                          href={`/admin/servers/${service.server_id}`}
                          className="text-primary underline-offset-2 hover:underline"
                        >
                          {service.server_name || service.server_id}
                        </Link>
                      ) : (
                        <span className="text-muted-foreground">
                          {t("admin.integrations.whmcs.no_server")}
                        </span>
                      )}
                    </td>
                    <td className="px-5 py-2.5">
                      <Badge variant={statusVariant(service.status)}>
                        {t(`servers.whmcs.status_${service.status}`)}
                      </Badge>
                    </td>
                    <td className="px-5 py-2.5 whitespace-nowrap">
                      {fmtDate(service.next_due_date)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}

function StatTile({
  label,
  value,
  sub,
  tone,
  mono,
}: {
  label: string;
  value: string;
  sub?: string;
  tone?: "ok" | "muted";
  mono?: boolean;
}) {
  return (
    <div className="flex min-w-0 flex-col gap-1 rounded-xl border bg-card px-4 py-3.5">
      <span className="truncate text-xs text-muted-foreground">{label}</span>
      <span
        className={cn(
          "flex items-center gap-2 truncate text-base font-semibold",
          mono && "font-mono text-sm",
          tone === "muted" && "text-muted-foreground"
        )}
        title={value}
      >
        {tone ? (
          <span
            className={cn(
              "size-2 shrink-0 rounded-full",
              tone === "ok" ? "bg-emerald-500" : "bg-muted-foreground/50"
            )}
          />
        ) : null}
        <span className="truncate">{value}</span>
      </span>
      {sub ? <span className="truncate text-xs text-muted-foreground">{sub}</span> : null}
    </div>
  );
}

function Step({
  index,
  text,
  children,
}: {
  index: number;
  text: string;
  children?: React.ReactNode;
}) {
  return (
    <li className="flex gap-3">
      <span className="flex size-6 shrink-0 items-center justify-center rounded-full border text-xs font-medium tabular-nums">
        {index}
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-2 pt-0.5 text-sm">
        <p className="leading-relaxed">{text}</p>
        {children}
      </div>
    </li>
  );
}
