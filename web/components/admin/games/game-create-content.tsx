"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { createAdminGame } from "@/lib/api";
import { useT } from "@/hooks/use-translations";

export function GameCreateContent() {
  const t = useT();
  const router = useRouter();
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({
    name: "",
    slug: "",
    description: "",
    code: "",
    query: "",
    minport: 1024,
    maxport: 65535,
    default_startup_params: "",
    status: true,
  });

  function setField<K extends keyof typeof form>(key: K, value: (typeof form)[K]) {
    setForm((p) => ({ ...p, [key]: value }));
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      await createAdminGame(form);
      toast.success(t("admin.games.created"));
      router.push("/admin/games");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("admin.games.create_failed")
      );
    } finally {
      setSaving(false);
    }
  }

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("admin.games.new_title")}</h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.games.new_subtitle")}
          </p>
        </div>
        <Button variant="outline" asChild>
          <Link href="/admin/games">← {t("common.back")}</Link>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">
            {t("admin.games.params")}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="name">{t("admin.games.name")}</Label>
                <Input
                  id="name"
                  value={form.name}
                  onChange={(e) => setField("name", e.target.value)}
                  required
                  placeholder="Counter-Strike 1.6"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="slug">{t("admin.games.slug")}</Label>
                <Input
                  id="slug"
                  value={form.slug}
                  onChange={(e) => setField("slug", e.target.value)}
                  required
                  placeholder="cs16"
                />
              </div>
              <div className="space-y-2 md:col-span-2">
                <Label htmlFor="description">{t("common.description")}</Label>
                <Textarea
                  id="description"
                  value={form.description}
                  onChange={(e) => setField("description", e.target.value)}
                  rows={4}
                  placeholder={t("admin.games.description_placeholder")}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="code">{t("admin.games.code")}</Label>
                <Input
                  id="code"
                  value={form.code}
                  onChange={(e) => setField("code", e.target.value)}
                  required
                  placeholder="cstrike"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="query">{t("admin.games.query")}</Label>
                <Input
                  id="query"
                  value={form.query}
                  onChange={(e) => setField("query", e.target.value)}
                  required
                  placeholder="goldsource"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="minport">{t("admin.games.minport")}</Label>
                <Input
                  id="minport"
                  type="number"
                  min={1}
                  max={65535}
                  value={form.minport}
                  onChange={(e) => setField("minport", +e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="maxport">{t("admin.games.maxport")}</Label>
                <Input
                  id="maxport"
                  type="number"
                  min={1}
                  max={65535}
                  value={form.maxport}
                  onChange={(e) => setField("maxport", +e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2 md:col-span-2">
                <Label htmlFor="startup">{t("admin.games.startup")}</Label>
                <Input
                  id="startup"
                  value={form.default_startup_params}
                  onChange={(e) => setField("default_startup_params", e.target.value)}
                  maxLength={512}
                  placeholder="-game cstrike +ip {{ip}} +port {{port}}"
                />
                <p className="text-xs text-muted-foreground">
                  {t("admin.games.startup_hint")}
                </p>
              </div>
              <div className="flex items-center gap-3 md:col-span-2">
                <Switch
                  id="status"
                  checked={form.status}
                  onCheckedChange={(v) => setField("status", v)}
                />
                <Label htmlFor="status">{t("admin.games.enabled")}</Label>
              </div>
            </div>
            <div className="flex justify-end gap-2 border-t pt-4">
              <Button variant="outline" asChild>
                <Link href="/admin/games">{t("common.cancel")}</Link>
              </Button>
              <Button type="submit" disabled={saving}>
                {saving ? t("common.creating") : t("admin.games.create")}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </PageShell>
  );
}
