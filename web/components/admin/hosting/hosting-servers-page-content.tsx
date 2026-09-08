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
  type AdminHostingServerInput,
  createAdminHostingServer,
  deleteAdminHostingServer,
  fetchAdminHostingServer,
  fetchAdminHostingServers,
  updateAdminHostingServer,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";

type HostingServer = {
  id: string;
  name: string;
  panel_type: string;
  host: string;
  active: boolean;
};

// value уходит в базу и остаётся как есть; подпись хранится ключом, потому что
// список считается один раз при загрузке модуля и иначе застыл бы на одном языке.
const PANEL_TYPES = [
  { value: "cpanel", labelKey: "admin.hosting_servers.panel.cpanel" },
  { value: "fastpanel", labelKey: "admin.hosting_servers.panel.fastpanel" },
  { value: "ispmanager", labelKey: "admin.hosting_servers.panel.ispmanager" },
  { value: "plesk", labelKey: "admin.hosting_servers.panel.plesk" },
];

const emptyForm: AdminHostingServerInput = {
  name: "",
  hostname: "",
  ip_address: "",
  port: 2087,
  panel_type: "cpanel",
  api_url: "",
  api_username: "",
  api_token: "",
  use_ssl: true,
  max_accounts: 0,
  active: true,
  description: "",
};

export function HostingServersPageContent() {
  const t = useT();
  const qc = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: ["admin-hosting-servers"],
    queryFn: async () => (await fetchAdminHostingServers()).servers as HostingServer[],
  });
  const servers = data ?? [];

  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<AdminHostingServerInput>(emptyForm);

  const saveMut = useMutation({
    mutationFn: (d: AdminHostingServerInput) =>
      d.id ? updateAdminHostingServer(d.id, d) : createAdminHostingServer(d),
    onSuccess: () => {
      toast.success(
        t(
          form.id
            ? "admin.hosting_servers.updated"
            : "admin.hosting_servers.created"
        )
      );
      setShowForm(false);
      setForm(emptyForm);
      qc.invalidateQueries({ queryKey: ["admin-hosting-servers"] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.hosting_servers.save_failed")),
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteAdminHostingServer(id),
    onSuccess: () => {
      toast.success(t("admin.hosting_servers.deleted"));
      qc.invalidateQueries({ queryKey: ["admin-hosting-servers"] });
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.hosting.delete_failed")),
  });

  function startCreate() {
    setForm(emptyForm);
    setShowForm(true);
  }

  // Токен с сервера не возвращается: пустое поле означает «оставить прежний».
  async function startEdit(id: string) {
    try {
      const s = await fetchAdminHostingServer(id);
      setForm({
        id: s.id,
        name: s.name ?? "",
        hostname: s.hostname ?? "",
        ip_address: s.ip_address ?? "",
        port: s.port ?? 2087,
        panel_type: s.panel_type ?? "cpanel",
        api_url: s.api_url ?? "",
        api_username: s.api_username ?? "",
        api_token: "",
        use_ssl: s.use_ssl ?? true,
        max_accounts: s.max_accounts ?? 0,
        active: s.active ?? true,
        description: s.description ?? "",
      });
      setShowForm(true);
    } catch (e) {
      toast.error(
        (e as Error).message || t("admin.hosting_servers.load_failed")
      );
    }
  }

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.name.trim() || !form.hostname.trim() || !form.api_url.trim()) return;
    saveMut.mutate(form);
  }

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("admin.hosting_servers.title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.hosting_servers.subtitle")}
          </p>
        </div>
        <Button onClick={startCreate}>{t("admin.hosting_servers.add")}</Button>
      </div>

      {showForm && (
        <form onSubmit={onSubmit} className="mb-6 space-y-4 rounded-lg border bg-card p-4">
          <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
            <div className="space-y-1">
              <Label>{t("common.name")}</Label>
              <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.hosting_servers.panel_type")}</Label>
              <Select value={form.panel_type} onValueChange={(v) => setForm({ ...form, panel_type: v })}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {PANEL_TYPES.map((p) => (
                    <SelectItem key={p.value} value={p.value}>
                      {t(p.labelKey)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                {t("admin.hosting_servers.api_hint")}
              </p>
            </div>
            <div className="space-y-1">
              <Label>Hostname</Label>
              <Input
                value={form.hostname}
                onChange={(e) => setForm({ ...form, hostname: e.target.value })}
                placeholder="node1.example.com"
                required
              />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.hosting_servers.ip")}</Label>
              <Input value={form.ip_address} onChange={(e) => setForm({ ...form, ip_address: e.target.value })} />
            </div>
            <div className="space-y-1">
              <Label>API URL</Label>
              <Input
                value={form.api_url}
                onChange={(e) => setForm({ ...form, api_url: e.target.value })}
                placeholder="https://node1.example.com:2087"
                required
              />
            </div>
            <div className="space-y-1">
              <Label>API username</Label>
              <Input
                value={form.api_username}
                onChange={(e) => setForm({ ...form, api_username: e.target.value })}
                placeholder="root"
              />
            </div>
            <div className="space-y-1 md:col-span-2">
              <Label>API token</Label>
              <Input
                type="password"
                value={form.api_token}
                onChange={(e) => setForm({ ...form, api_token: e.target.value })}
                placeholder={
                  form.id ? t("admin.hosting_servers.token_keep") : "WHM API token"
                }
              />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.hosting_servers.max_accounts")}</Label>
              <Input
                type="number"
                value={form.max_accounts ?? 0}
                onChange={(e) => setForm({ ...form, max_accounts: Number(e.target.value) })}
              />
            </div>
          </div>
          <div className="flex items-center gap-6">
            <div className="flex items-center gap-2">
              <Switch checked={!!form.use_ssl} onCheckedChange={(v) => setForm({ ...form, use_ssl: v })} />
              <Label className="!mb-0">SSL</Label>
            </div>
            <div className="flex items-center gap-2">
              <Switch checked={!!form.active} onCheckedChange={(v) => setForm({ ...form, active: v })} />
              <Label className="!mb-0">{t("admin.hosting.active")}</Label>
            </div>
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
        ) : servers.length === 0 ? (
          <div className="p-10 text-center text-sm text-muted-foreground">
            {t("admin.hosting_servers.empty")}
          </div>
        ) : (
          <div className="divide-y">
            {servers.map((s) => (
              <div key={s.id} className="flex items-center justify-between gap-4 p-4">
                <div>
                  <div className="font-medium">{s.name}</div>
                  <div className="text-xs text-muted-foreground">
                    {s.host} · {s.panel_type}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {s.active ? (
                    <Badge className="bg-emerald-500/10 text-emerald-600 ring-1 ring-inset ring-emerald-500/20">
                      {t("admin.hosting.active")}
                    </Badge>
                  ) : (
                    <Badge variant="outline">{t("admin.hosting.inactive")}</Badge>
                  )}
                  <Button variant="outline" size="sm" onClick={() => startEdit(s.id)}>
                    {t("admin.infra.edit")}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => deleteMut.mutate(s.id)}
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
