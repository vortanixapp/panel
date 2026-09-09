"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import {
  type AdminHostingPlanInput,
  createAdminHostingPlan,
  deleteAdminHostingPlan,
  fetchAdminHostingPlans,
  fetchAdminHostingServers,
  updateAdminHostingPlan,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";

type HostingPlan = {
  id: string;
  hosting_server_id: string;
  name: string;
  panel_package_name: string;
  price_monthly: number;
  active: boolean;
  disk_mb?: number;
  bandwidth_mb?: number;
  max_domains?: number;
  max_databases?: number;
  max_email_accounts?: number;
  has_ssl?: boolean;
  has_ssh?: boolean;
  has_cron?: boolean;
  has_backup?: boolean;
};

type HostingServer = { id: string; name: string };

const emptyForm: AdminHostingPlanInput = {
  hosting_server_id: "",
  name: "",
  panel_package_name: "",
  disk_mb: 1024,
  bandwidth_mb: 0,
  max_domains: 1,
  max_databases: 1,
  max_email_accounts: 5,
  has_ssl: true,
  has_ssh: false,
  has_cron: true,
  has_backup: true,
  price_monthly: 0,
  active: true,
};

export function HostingPlansPageContent() {
  const t = useT();
  const qc = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: ["admin-hosting-plans"],
    queryFn: async () => (await fetchAdminHostingPlans()).plans as HostingPlan[],
  });
  const { data: serversData } = useQuery({
    queryKey: ["admin-hosting-servers"],
    queryFn: async () => (await fetchAdminHostingServers()).servers as HostingServer[],
  });
  const plans = data ?? [];
  const servers = serversData ?? [];

  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<AdminHostingPlanInput>(emptyForm);

  const saveMut = useMutation({
    mutationFn: (d: AdminHostingPlanInput) =>
      d.id ? updateAdminHostingPlan(d.id, d) : createAdminHostingPlan(d),
    onSuccess: () => {
      toast.success(
        t(
          form.id ? "admin.hosting_plans.updated" : "admin.hosting_plans.created"
        )
      );
      setShowForm(false);
      setForm(emptyForm);
      qc.invalidateQueries({ queryKey: ["admin-hosting-plans"] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.hosting_plans.save_failed")),
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteAdminHostingPlan(id),
    onSuccess: () => {
      toast.success(t("admin.hosting_plans.deleted"));
      qc.invalidateQueries({ queryKey: ["admin-hosting-plans"] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.hosting.delete_failed")),
  });

  function startCreate() {
    setForm({ ...emptyForm, hosting_server_id: servers[0]?.id ?? "" });
    setShowForm(true);
  }

  function startEdit(p: HostingPlan) {
    setForm({
      id: p.id,
      hosting_server_id: p.hosting_server_id,
      name: p.name,
      panel_package_name: p.panel_package_name,
      disk_mb: p.disk_mb ?? emptyForm.disk_mb,
      bandwidth_mb: p.bandwidth_mb ?? emptyForm.bandwidth_mb,
      max_domains: p.max_domains ?? emptyForm.max_domains,
      max_databases: p.max_databases ?? emptyForm.max_databases,
      max_email_accounts: p.max_email_accounts ?? emptyForm.max_email_accounts,
      has_ssl: p.has_ssl ?? emptyForm.has_ssl,
      has_ssh: p.has_ssh ?? emptyForm.has_ssh,
      has_cron: p.has_cron ?? emptyForm.has_cron,
      has_backup: p.has_backup ?? emptyForm.has_backup,
      price_monthly: p.price_monthly,
      active: p.active,
    });
    setShowForm(true);
  }

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.name.trim() || !form.panel_package_name.trim() || !form.hosting_server_id) return;
    saveMut.mutate(form);
  }

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("admin.hosting_plans.title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.hosting_plans.subtitle")}
          </p>
        </div>
        <Button onClick={startCreate} disabled={servers.length === 0}>
          {t("admin.hosting_plans.add")}
        </Button>
      </div>
      {servers.length === 0 && (
        <p className="mb-4 text-sm text-amber-600">
          {t("admin.hosting_plans.no_servers")}
        </p>
      )}

      {showForm && (
        <form onSubmit={onSubmit} className="mb-6 space-y-4 rounded-lg border bg-card p-4">
          <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
            <div className="space-y-1">
              <Label>{t("common.server")}</Label>
              <Select
                value={form.hosting_server_id || undefined}
                onValueChange={(v) => setForm({ ...form, hosting_server_id: v })}
              >
                <SelectTrigger>
                  <SelectValue
                    placeholder={t("admin.hosting_plans.select_server")}
                  />
                </SelectTrigger>
                <SelectContent>
                  {servers.map((s) => (
                    <SelectItem key={s.id} value={s.id}>
                      {s.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1">
              <Label>{t("admin.hosting_plans.name")}</Label>
              <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required />
            </div>
            <div className="space-y-1 md:col-span-2">
              <Label>{t("admin.hosting_plans.package_name")}</Label>
              <Input
                value={form.panel_package_name}
                onChange={(e) => setForm({ ...form, panel_package_name: e.target.value })}
                placeholder={t("admin.hosting_plans.package_name_placeholder")}
                required
              />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.hosting_plans.disk")}</Label>
              <Input
                type="number"
                value={form.disk_mb ?? 0}
                onChange={(e) => setForm({ ...form, disk_mb: Number(e.target.value) })}
              />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.hosting_plans.bandwidth")}</Label>
              <Input
                type="number"
                value={form.bandwidth_mb ?? 0}
                onChange={(e) => setForm({ ...form, bandwidth_mb: Number(e.target.value) })}
              />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.hosting_plans.domains")}</Label>
              <Input
                type="number"
                value={form.max_domains ?? 0}
                onChange={(e) => setForm({ ...form, max_domains: Number(e.target.value) })}
              />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.hosting_plans.databases")}</Label>
              <Input
                type="number"
                value={form.max_databases ?? 0}
                onChange={(e) => setForm({ ...form, max_databases: Number(e.target.value) })}
              />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.hosting_plans.email_accounts")}</Label>
              <Input
                type="number"
                value={form.max_email_accounts ?? 0}
                onChange={(e) => setForm({ ...form, max_email_accounts: Number(e.target.value) })}
              />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.hosting_plans.price")}</Label>
              <Input
                type="number"
                step="0.01"
                value={form.price_monthly ?? 0}
                onChange={(e) => setForm({ ...form, price_monthly: Number(e.target.value) })}
              />
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-6">
            {(
              [
                ["has_ssl", "admin.hosting_plans.feature.ssl"],
                ["has_ssh", "admin.hosting_plans.feature.ssh"],
                ["has_cron", "admin.hosting_plans.feature.cron"],
                ["has_backup", "admin.hosting_plans.feature.backup"],
                ["active", "admin.hosting_plans.feature.active"],
              ] as const
            ).map(([key, labelKey]) => (
              <div key={key} className="flex items-center gap-2">
                <Switch checked={!!form[key]} onCheckedChange={(v) => setForm({ ...form, [key]: v })} />
                <Label className="!mb-0">{t(labelKey)}</Label>
              </div>
            ))}
          </div>
          <div className="flex gap-2">
            <Button type="submit" disabled={saveMut.isPending}>
              {form.id ? t("common.save") : t("common.add")}
            </Button>
            <Button type="button" variant="ghost" onClick={() => setShowForm(false)}>
              {t("common.cancel")}
            </Button>
          </div>
        </form>
      )}

      <div className="rounded-lg border bg-card">
        {isLoading ? (
          <div className="space-y-3 p-6">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
          </div>
        ) : plans.length === 0 ? (
          <div className="p-10 text-center text-sm text-muted-foreground">
            {t("admin.hosting_plans.empty")}
          </div>
        ) : (
          <div className="divide-y">
            {plans.map((p) => (
              <div key={p.id} className="flex items-center justify-between gap-4 p-4">
                <div>
                  <div className="font-medium">{p.name}</div>
                  <div className="text-xs text-muted-foreground">
                    {p.panel_package_name} ·{" "}
                    {t("admin.hosting_plans.price_per_month", {
                      price: p.price_monthly,
                    })}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {p.active ? (
                    <Badge className="bg-emerald-500/10 text-emerald-600 ring-1 ring-inset ring-emerald-500/20">
                      {t("admin.hosting.active")}
                    </Badge>
                  ) : (
                    <Badge variant="outline">{t("admin.hosting.inactive")}</Badge>
                  )}
                  <Button variant="outline" size="sm" onClick={() => startEdit(p)}>
                    {t("admin.infra.edit")}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => deleteMut.mutate(p.id)}
                    disabled={deleteMut.isPending}
                  >
                    {t("common.delete")}
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </PageShell>
  );
}
