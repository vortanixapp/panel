"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import {
  fetchAdminLocationEdit,
  testAdminLocationSSH,
  testAdminLocationSSHBody,
  updateAdminLocation,
  type MysqlInstance,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";

type FormState = {
  code: string;
  name: string;
  country: string;
  city: string;
  region: string;
  description: string;
  ip_address: string;
  ip_pool: string;
  ssh_host: string;
  ssh_user: string;
  ssh_password: string;
  ssh_port: number;
  sort_order: number;
  is_active: boolean;
  mysql_instances: string;
  docker_images: string;
  phpmyadmin_port: number;
};

export function LocationEditContent() {
  const t = useT();
  const params = useParams();
  const id = String(params?.id ?? "");
  const router = useRouter();
  const queryClient = useQueryClient();
  const [mysqlInstances, setMysqlInstances] = useState<MysqlInstance[]>([]);
  const [sshTesting, setSshTesting] = useState(false);
  const [form, setForm] = useState<FormState>({
    code: "",
    name: "",
    country: "",
    city: "",
    region: "",
    description: "",
    ip_address: "",
    ip_pool: "",
    ssh_host: "",
    ssh_user: "",
    ssh_password: "",
    ssh_port: 22,
    sort_order: 0,
    is_active: true,
    mysql_instances: "",
    docker_images: "",
    phpmyadmin_port: 8081,
  });

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminLocationEdit(id),
    queryFn: () => fetchAdminLocationEdit(id),
    enabled: !!id,
  });

  useEffect(() => {
    if (!data?.location) return;
    const l = data.location as Record<string, unknown>;
    const instances = Array.isArray(l.mysql_instances)
      ? (l.mysql_instances as MysqlInstance[])
      : [];
    setMysqlInstances(instances);
    const ipPool = Array.isArray(l.ip_pool)
      ? JSON.stringify(l.ip_pool)
      : String(l.ip_pool ?? "");
    setForm({
      code: String(l.code ?? ""),
      name: String(l.name ?? ""),
      country: String(l.country ?? ""),
      city: String(l.city ?? ""),
      region: String(l.region ?? ""),
      description: String(l.description ?? ""),
      ip_address: String(l.ip_address ?? ""),
      ip_pool: ipPool,
      ssh_host: String(l.ssh_host ?? ""),
      ssh_user: String(l.ssh_user ?? ""),
      ssh_password: "",
      ssh_port: Number(l.ssh_port ?? 22),
      sort_order: Number(l.sort_order ?? 0),
      is_active: Boolean(l.is_active ?? true),
      mysql_instances: instances.length ? JSON.stringify(instances, null, 2) : "",
      docker_images: Array.isArray(l.docker_images)
        ? JSON.stringify(l.docker_images, null, 2)
        : "",
      phpmyadmin_port: Number(l.phpmyadmin_port ?? 8081),
    });
  }, [data]);

  const saveMut = useMutation({
    mutationFn: () => updateAdminLocation(id, form),
    onSuccess: async () => {
      toast.success(t("common.saved"));
      await queryClient.invalidateQueries({ queryKey: queryKeys.adminLocations });
      await queryClient.invalidateQueries({ queryKey: queryKeys.adminLocation(id) });
      router.push(`/admin/locations/${id}`);
    },
    onError: (e: Error) => toast.error(e.message),
  });

  function setField<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((p) => ({ ...p, [key]: value }));
  }

  async function testSSH() {
    setSshTesting(true);
    try {
      const res = form.ssh_password.trim()
        ? await testAdminLocationSSHBody({
            ssh_host: form.ssh_host,
            ssh_user: form.ssh_user,
            ssh_password: form.ssh_password,
            ssh_port: form.ssh_port,
          })
        : await testAdminLocationSSH(id);
      if (res.ok) {
        toast.success(
          res.output ? `SSH OK: ${res.output}` : t("admin.location.ssh_ok")
        );
      } else {
        toast.error(res.error || t("admin.location.svc.ssh_error"));
      }
    } catch (e) {
      toast.error(
        e instanceof Error ? e.message : t("admin.location.svc.ssh_error")
      );
    } finally {
      setSshTesting(false);
    }
  }

  if (isLoading) {
    return (
      <PageShell variant="admin">
        <Skeleton className="h-24 w-full" />
      </PageShell>
    );
  }

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="mb-2 flex items-center gap-2">
            <Badge variant="outline" className="font-mono">
              {form.code}
            </Badge>
            <Badge variant={form.is_active ? "secondary" : "outline"}>
              {form.is_active
                ? t("admin.locations.active")
                : t("admin.locations.inactive")}
            </Badge>
          </div>
          <h1 className="text-2xl font-bold">
            {t("admin.location.edit_title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.location.edit_subtitle")}
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" asChild>
            <Link href={`/admin/locations/${id}`}>
              {t("admin.locations.view")}
            </Link>
          </Button>
          <Button variant="outline" asChild>
            <Link href="/admin/locations">{t("admin.tariffs.to_list")}</Link>
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">
            {t("admin.location.params")}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              saveMut.mutate();
            }}
            className="space-y-4"
          >
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label>{t("admin.location.code")}</Label>
                <Input value={form.code} onChange={(e) => setField("code", e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label>{t("common.name")}</Label>
                <Input value={form.name} onChange={(e) => setField("name", e.target.value)} required />
              </div>
            </div>
            <div className="grid gap-4 md:grid-cols-3">
              <div className="space-y-2">
                <Label>{t("admin.location.region")}</Label>
                <Input value={form.region} onChange={(e) => setField("region", e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>{t("admin.location.city")}</Label>
                <Input value={form.city} onChange={(e) => setField("city", e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>{t("admin.location.country")}</Label>
                <Input value={form.country} onChange={(e) => setField("country", e.target.value)} />
              </div>
            </div>
            <div className="space-y-2">
              <Label>{t("common.description")}</Label>
              <Textarea value={form.description} onChange={(e) => setField("description", e.target.value)} rows={3} />
            </div>
            <div className="space-y-2">
              <Label>{t("admin.location.ip_for_players")}</Label>
              <Input value={form.ip_address} onChange={(e) => setField("ip_address", e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label>{t("admin.location.ip_pool_json")}</Label>
              <Textarea value={form.ip_pool} onChange={(e) => setField("ip_pool", e.target.value)} rows={2} />
            </div>

            <Tabs defaultValue="main">
              <TabsList>
                <TabsTrigger value="main">
                  {t("admin.settings.tab.main")}
                </TabsTrigger>
                <TabsTrigger value="mysql">MySQL</TabsTrigger>
              </TabsList>
              <TabsContent value="main" className="space-y-4 pt-4">
                <div className="space-y-2">
                  <Label>{t("admin.location.ssh_access")}</Label>
                  <div className="grid gap-3 md:grid-cols-4">
                    <Input placeholder="host" value={form.ssh_host} onChange={(e) => setField("ssh_host", e.target.value)} />
                    <Input placeholder="user" value={form.ssh_user} onChange={(e) => setField("ssh_user", e.target.value)} />
                    <Input type="password" placeholder={t("admin.location.new_password")} value={form.ssh_password} onChange={(e) => setField("ssh_password", e.target.value)} />
                    <Input type="number" value={form.ssh_port} onChange={(e) => setField("ssh_port", Number(e.target.value))} />
                  </div>
                  <p className="text-xs text-muted-foreground">
                    {t("admin.location.password_hidden")}
                  </p>
                  <Button type="button" variant="secondary" size="sm" disabled={sshTesting} onClick={() => testSSH()}>
                    {sshTesting
                      ? t("admin.settings.storage.testing")
                      : t("admin.location.test_ssh")}
                  </Button>
                </div>
                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <Label>{t("admin.location.sort_order_label")}</Label>
                    <Input type="number" value={form.sort_order} onChange={(e) => setField("sort_order", Number(e.target.value))} />
                  </div>
                  <div className="flex items-center gap-3 pt-6">
                    <Switch checked={form.is_active} onCheckedChange={(v) => setField("is_active", v)} />
                    <Label>{t("admin.location.is_active")}</Label>
                  </div>
                </div>
              </TabsContent>
              <TabsContent value="mysql" className="space-y-4 pt-4">
                <p className="text-xs text-muted-foreground">
                  {t("admin.location.pma_port", {
                    port: form.phpmyadmin_port,
                  })}
                </p>
                <div className="w-full overflow-x-auto rounded-md border">
                  <table className="min-w-[760px] w-full text-xs">
                    <thead className="bg-muted/50">
                      <tr>
                        <th className="px-3 py-2 text-left">key</th>
                        <th className="px-3 py-2 text-left">engine</th>
                        <th className="px-3 py-2 text-left">version</th>
                        <th className="px-3 py-2 text-left">port</th>
                        <th className="px-3 py-2 text-left">container</th>
                        <th className="px-3 py-2 text-left">enabled</th>
                      </tr>
                    </thead>
                    <tbody>
                      {mysqlInstances.length === 0 ? (
                        <tr>
                          <td colSpan={6} className="px-3 py-3 text-muted-foreground">
                            {t("admin.location.no_instances")}
                          </td>
                        </tr>
                      ) : (
                        mysqlInstances.map((inst, i) => (
                          <tr key={i} className="border-t">
                            <td className="px-3 py-2 font-mono">{inst.key}</td>
                            <td className="px-3 py-2">{inst.engine}</td>
                            <td className="px-3 py-2">{inst.version}</td>
                            <td className="px-3 py-2">{inst.port}</td>
                            <td className="px-3 py-2 font-mono">{inst.container}</td>
                            <td className="px-3 py-2">{inst.enabled !== false ? "yes" : "no"}</td>
                          </tr>
                        ))
                      )}
                    </tbody>
                  </table>
                </div>
                <div className="space-y-2">
                  <Label>{t("admin.location.config_json")}</Label>
                  <Textarea
                    value={form.mysql_instances}
                    onChange={(e) => setField("mysql_instances", e.target.value)}
                    rows={8}
                  />
                </div>
              </TabsContent>
            </Tabs>

            <div className="flex justify-end gap-2 pt-4">
              <Button type="button" variant="outline" asChild>
                <Link href={`/admin/locations/${id}`}>{t("common.cancel")}</Link>
              </Button>
              <Button type="submit" disabled={saveMut.isPending}>
                {saveMut.isPending
                  ? t("common.saving")
                  : t("common.save_changes")}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </PageShell>
  );
}
