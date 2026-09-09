"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";
import { CheckCircle2, Copy } from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { createAdminLocation, testAdminLocationSSHBody } from "@/lib/api";
import { useT } from "@/hooks/use-translations";

type Step = 1 | 2 | 3;

export function LocationCreateContent() {
  const t = useT();
  const router = useRouter();
  const [step, setStep] = useState<Step>(1);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [sshOk, setSshOk] = useState(false);
  const [created, setCreated] = useState<{ id: string; agent_token: string } | null>(null);
  const [form, setForm] = useState({
    code: "",
    name: "",
    region: "",
    city: "",
    country: "",
    description: "",
    ip_address: "",
    ip_pool: "",
    ssh_host: "",
    ssh_user: "root",
    ssh_password: "",
    ssh_port: 22,
    sort_order: 0,
    is_active: true,
  });

  function setField<K extends keyof typeof form>(key: K, value: (typeof form)[K]) {
    setForm((p) => ({ ...p, [key]: value }));
    if (key.startsWith("ssh_")) setSshOk(false);
  }

  async function testSSH() {
    setTesting(true);
    try {
      const res = await testAdminLocationSSHBody({
        ssh_host: form.ssh_host,
        ssh_user: form.ssh_user,
        ssh_password: form.ssh_password,
        ssh_port: form.ssh_port,
      });
      if (res.ok) {
        setSshOk(true);
        toast.success(
          res.output ? `SSH OK: ${res.output}` : t("admin.location.ssh_ok")
        );
      } else {
        setSshOk(false);
        toast.error(res.error || t("admin.location.svc.ssh_error"));
      }
    } catch (e) {
      setSshOk(false);
      toast.error(
        e instanceof Error ? e.message : t("admin.location.svc.ssh_error")
      );
    } finally {
      setTesting(false);
    }
  }

  async function onSubmit() {
    setSaving(true);
    try {
      let ip_pool: unknown = form.ip_pool;
      if (form.ip_pool.trim().startsWith("[")) {
        ip_pool = JSON.parse(form.ip_pool);
      } else if (form.ip_pool.trim()) {
        ip_pool = form.ip_pool.split(",").map((s) => s.trim()).filter(Boolean);
      } else {
        ip_pool = [];
      }
      const host = form.ssh_host.trim() || form.ip_address.trim();
      const res = await createAdminLocation({
        ...form,
        ssh_host: host || form.ssh_host,
        ip_pool,
      });
      setCreated({ id: res.id, agent_token: res.agent_token });
      setStep(3);
      toast.success(t("admin.location.created"));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("common.error"));
    } finally {
      setSaving(false);
    }
  }

  async function copyEnv() {
    if (!created) return;
    const text = `AGENT_TOKEN=${created.agent_token}\nNODE_ID=${created.id}\n`;
    await navigator.clipboard.writeText(text);
    toast.success(t("admin.location.env_copied"));
  }

  if (step === 3 && created) {
    return (
      <PageShell variant="admin">
        <Card className="w-full">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <CheckCircle2 className="h-5 w-5 text-emerald-500" />
              {t("admin.location.created")}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-sm text-muted-foreground">
              {t("admin.location.created_hint")}
            </p>
            <div className="rounded-md border p-3 space-y-2 text-sm">
              <div>
                <span className="text-muted-foreground">NODE_ID: </span>
                <code className="text-xs break-all">{created.id}</code>
              </div>
              <div>
                <span className="text-muted-foreground">AGENT_TOKEN: </span>
                <code className="text-xs break-all">{created.agent_token}</code>
              </div>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button variant="outline" onClick={() => copyEnv()}>
                <Copy className="mr-2 h-4 w-4" />
                {t("admin.location.copy_env")}
              </Button>
              <Button asChild>
                <Link href={`/admin/locations/${created.id}/setup`}>
                  {t("admin.locations.ssh_setup")}
                </Link>
              </Button>
              <Button variant="secondary" asChild>
                <Link href={`/admin/locations/${created.id}`}>
                  {t("admin.location.to_location")}
                </Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      </PageShell>
    );
  }

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("admin.location.new_title")}</h1>
        </div>
        <Button variant="outline" asChild>
          <Link href="/admin/locations">← {t("common.back")}</Link>
        </Button>
      </div>

      {step === 1 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">
              {t("admin.settings.tab.main")}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label>{t("admin.location.code")}</Label>
                <Input
                  value={form.code}
                  onChange={(e) => setField("code", e.target.value)}
                  placeholder="eu-msk-01"
                  required
                />
              </div>
              <div className="space-y-2">
                <Label>{t("common.name")}</Label>
                <Input
                  value={form.name}
                  onChange={(e) => setField("name", e.target.value)}
                  placeholder="Testing Node"
                  required
                />
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
              <Textarea
                value={form.description}
                onChange={(e) => setField("description", e.target.value)}
                rows={2}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("admin.location.public_ip")}</Label>
              <Input
                value={form.ip_address}
                onChange={(e) => setField("ip_address", e.target.value)}
                placeholder="203.0.113.10"
              />
            </div>
            <div className="flex items-center gap-3">
              <Switch checked={form.is_active} onCheckedChange={(v) => setField("is_active", v)} />
              <Label>{t("admin.locations.active")}</Label>
            </div>
            <div className="flex justify-end">
              <Button
                type="button"
                disabled={!form.code.trim() || !form.name.trim()}
                onClick={() => setStep(2)}
              >
                {t("admin.location.next_ssh")}
              </Button>
            </div>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">
              {t("admin.location.ssh_card_title")}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-sm text-muted-foreground">
              {t("admin.location.ssh_card_hint")}
            </p>
            <div className="grid gap-3 md:grid-cols-4">
              <div className="space-y-2 md:col-span-2">
                <Label>Host</Label>
                <Input
                  placeholder={t("admin.location.node_ip_placeholder")}
                  value={form.ssh_host || form.ip_address}
                  onChange={(e) => setField("ssh_host", e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label>User</Label>
                <Input value={form.ssh_user} onChange={(e) => setField("ssh_user", e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>Port</Label>
                <Input
                  type="number"
                  value={form.ssh_port}
                  onChange={(e) => setField("ssh_port", Number(e.target.value))}
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label>Password</Label>
              <Input
                type="password"
                value={form.ssh_password}
                onChange={(e) => setField("ssh_password", e.target.value)}
              />
            </div>
            <div className="flex flex-wrap gap-2">
              <Button type="button" variant="secondary" disabled={testing} onClick={() => testSSH()}>
                {testing
                  ? t("admin.settings.storage.testing")
                  : t("admin.location.test_ssh")}
              </Button>
              {sshOk ? (
                <span className="flex items-center text-sm text-emerald-500">
                  <CheckCircle2 className="mr-1 h-4 w-4" />
                  {t("admin.location.connection_ok")}
                </span>
              ) : null}
            </div>
            <div className="flex justify-between gap-2 pt-4 border-t">
              <Button type="button" variant="outline" onClick={() => setStep(1)}>
                {t("common.back")}
              </Button>
              <Button
                type="button"
                disabled={saving || !(form.ssh_host.trim() || form.ip_address.trim())}
                onClick={() => onSubmit()}
              >
                {saving
                  ? t("common.creating")
                  : t("admin.location.create_action")}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </PageShell>
  );
}
