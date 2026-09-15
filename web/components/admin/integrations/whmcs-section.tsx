"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

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

  return (
    <div className="mb-8">
      <h2 className="text-lg font-semibold">{t("admin.integrations.whmcs.title")}</h2>
      <p className="mb-3 text-sm text-muted-foreground">
        {t("admin.integrations.whmcs.subtitle")}
      </p>

      {issuedKey ? (
        <div className="mb-4 rounded-lg border border-amber-500/40 bg-amber-500/5 p-4">
          <div className="text-sm font-medium">{t("admin.integrations.key_once")}</div>
          <code className="mt-2 block break-all rounded bg-muted p-2 font-mono text-xs">
            {issuedKey}
          </code>
          <p className="mt-2 text-xs text-muted-foreground">
            {t("admin.integrations.whmcs.key_hint")}
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-3"
            onClick={() => setIssuedKey("")}
          >
            {t("admin.integrations.copied")}
          </Button>
        </div>
      ) : null}

      <div className="mb-4 grid gap-4 lg:grid-cols-2">
        <div className="rounded-lg border bg-card p-4">
          <ol className="list-decimal space-y-4 pl-5 text-sm">
            <li>
              <p>{t("admin.integrations.whmcs.step_download")}</p>
              <div className="mt-2 flex flex-wrap items-center gap-2">
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
            </li>
            <li>
              <p>
                {t("admin.integrations.whmcs.step_key", {
                  scope: settings?.scope || DEFAULT_SCOPE,
                })}
              </p>
              <div className="mt-2 flex flex-wrap items-center gap-2">
                <Button size="sm" disabled={keyMut.isPending} onClick={() => keyMut.mutate()}>
                  {t("admin.integrations.whmcs.issue_key")}
                </Button>
                <span className="text-xs text-muted-foreground">
                  {t("admin.integrations.whmcs.active_keys", {
                    count: settings?.active_keys ?? 0,
                  })}
                </span>
              </div>
            </li>
            <li>
              <p>{t("admin.integrations.whmcs.step_server", { host: panelHost || "—" })}</p>
            </li>
            <li>
              <p>{t("admin.integrations.whmcs.step_product")}</p>
              <p className="mt-1 text-xs text-muted-foreground">
                {t("admin.integrations.whmcs.step_product_options")}{" "}
                <code className="font-mono">
                  game, location, version, slots, cpu_cores, ram_gb, disk_gb, server_name
                </code>
              </p>
            </li>
          </ol>
        </div>

        <div className="space-y-3 rounded-lg border bg-card p-4">
          <div className="text-sm font-medium">
            {t("admin.integrations.whmcs.settings_title")}
          </div>
          {settingsQuery.isLoading ? (
            <Skeleton className="h-40 w-full" />
          ) : (
            <>
              <div className="space-y-1">
                <Label>{t("admin.integrations.whmcs.url")}</Label>
                <Input
                  value={url}
                  onChange={(e) => setUrl(e.target.value)}
                  placeholder="https://billing.example.com"
                />
                <p className="text-xs text-muted-foreground">
                  {t("admin.integrations.whmcs.url_hint")}
                </p>
              </div>
              <div className="space-y-1">
                <Label>{t("admin.integrations.whmcs.order_url")}</Label>
                <Input
                  value={orderUrl}
                  onChange={(e) => setOrderUrl(e.target.value)}
                  placeholder="https://billing.example.com/cart.php?gid=1"
                />
                <p className="text-xs text-muted-foreground">
                  {t("admin.integrations.whmcs.order_url_hint")}
                </p>
              </div>
              <label className="flex items-start gap-2 text-sm">
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
              <Button onClick={() => saveMut.mutate()} disabled={saveMut.isPending}>
                {t("common.save")}
              </Button>
            </>
          )}
        </div>
      </div>

      <div className="rounded-lg border bg-card">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b p-4">
          <span className="text-sm font-medium">
            {t("admin.integrations.whmcs.services_title")}
          </span>
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
          <div className="p-8 text-center text-sm text-muted-foreground">
            {t("admin.integrations.whmcs.services_empty")}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-xs text-muted-foreground">
                  <th className="px-4 py-2 text-left font-normal">
                    {t("admin.integrations.whmcs.col_service")}
                  </th>
                  <th className="px-4 py-2 text-left font-normal">
                    {t("admin.integrations.whmcs.col_client")}
                  </th>
                  <th className="px-4 py-2 text-left font-normal">
                    {t("admin.integrations.whmcs.col_server")}
                  </th>
                  <th className="px-4 py-2 text-left font-normal">{t("common.status")}</th>
                  <th className="px-4 py-2 text-left font-normal">
                    {t("admin.integrations.whmcs.col_due")}
                  </th>
                </tr>
              </thead>
              <tbody>
                {services.map((service) => (
                  <tr key={service.service_id} className="border-t">
                    <td className="px-4 py-2">
                      <span className="font-mono">#{service.service_id}</span>
                      {service.product ? (
                        <div className="text-xs text-muted-foreground">{service.product}</div>
                      ) : null}
                    </td>
                    <td className="px-4 py-2">{service.email || `#${service.client_id}`}</td>
                    <td className="px-4 py-2">
                      {service.server_id ? (
                        <Link
                          href={`/admin/servers/${service.server_id}`}
                          className="underline-offset-2 hover:underline"
                        >
                          {service.server_name || service.server_id}
                        </Link>
                      ) : (
                        <span className="text-muted-foreground">
                          {t("admin.integrations.whmcs.no_server")}
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-2">
                      <Badge variant={statusVariant(service.status)}>
                        {t(`servers.whmcs.status_${service.status}`)}
                      </Badge>
                    </td>
                    <td className="px-4 py-2">{fmtDate(service.next_due_date)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
