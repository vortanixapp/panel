"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Trash2 } from "lucide-react";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import {
  createAdminGameVersion,
  deleteAdminGameVersion,
  fetchAdminGameEdit,
  updateAdminGame,
  type AdminGameVersion,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";

type FormState = {
  name: string;
  slug: string;
  description: string;
  code: string;
  query: string;
  minport: number;
  maxport: number;
  default_startup_params: string;
  status: boolean;
};

type VersionFormState = {
  name: string;
  source_type: "archive" | "steam" | "docker";
  url: string;
  steam_app_id: string;
  steam_branch: string;
  sort_order: number;
  is_active: boolean;
};

export function GameEditContent() {
  const t = useT();
  const params = useParams();
  const id = String(params?.id ?? "");
  const queryClient = useQueryClient();

  const [form, setForm] = useState<FormState>({
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
  const [vForm, setVForm] = useState<VersionFormState>({
    name: "",
    source_type: "archive",
    url: "",
    steam_app_id: "",
    steam_branch: "",
    sort_order: 0,
    is_active: true,
  });
  const [versions, setVersions] = useState<AdminGameVersion[]>([]);

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminGameEdit(id),
    queryFn: () => fetchAdminGameEdit(id),
    enabled: !!id,
  });

  useEffect(() => {
    if (!data?.game) return;
    const g = data.game;
    setForm({
      name: g.name || "",
      slug: g.slug || "",
      description: g.description || "",
      code: g.code || "",
      query: g.query || "",
      minport: g.minport ?? 1024,
      maxport: g.maxport ?? 65535,
      default_startup_params: g.default_startup_params || "",
      status: !!(g.status ?? g.is_active),
    });
    setVersions(g.versions ?? []);
  }, [data]);

  const saveMut = useMutation({
    mutationFn: () => updateAdminGame(id, form),
    onSuccess: () => {
      toast.success(t("common.saved"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminGames });
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminGameEdit(id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminGame(id) });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const addVersionMut = useMutation({
    mutationFn: () =>
      createAdminGameVersion(id, {
        name: vForm.name,
        source_type: vForm.source_type,
        url: vForm.source_type === "archive" ? vForm.url : undefined,
        steam_app_id:
          vForm.source_type === "steam" ? Number(vForm.steam_app_id) : undefined,
        steam_branch: vForm.steam_branch || undefined,
        is_active: vForm.is_active,
        sort_order: vForm.sort_order,
      }),
    onSuccess: () => {
      toast.success(t("admin.games.version_added"));
      setVForm({
        name: "",
        source_type: "archive",
        url: "",
        steam_app_id: "",
        steam_branch: "",
        sort_order: 0,
        is_active: true,
      });
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminGameEdit(id) });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const deleteVersionMut = useMutation({
    mutationFn: (versionId: string) => deleteAdminGameVersion(id, versionId),
    onSuccess: () => {
      toast.success(t("admin.games.version_deleted"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminGameEdit(id) });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  function setField<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((p) => ({ ...p, [key]: value }));
  }

  function setVField<K extends keyof VersionFormState>(
    key: K,
    value: VersionFormState[K]
  ) {
    setVForm((p) => ({ ...p, [key]: value }));
  }

  async function onSave(e: React.FormEvent) {
    e.preventDefault();
    await saveMut.mutateAsync();
  }

  async function onAddVersion(e: React.FormEvent) {
    e.preventDefault();
    if (vForm.source_type === "archive" && !vForm.url.trim()) {
      toast.error(t("admin.games.need_archive_url"));
      return;
    }
    if (vForm.source_type === "steam") {
      const appId = Number(vForm.steam_app_id);
      if (!appId || appId <= 0) {
        toast.error(t("admin.games.need_app_id"));
        return;
      }
    }
    await addVersionMut.mutateAsync();
  }

  if (isLoading) {
    return (
      <PageShell variant="admin">
        <Skeleton className="h-64 w-full" />
      </PageShell>
    );
  }

  if (!data?.game) {
    return (
      <PageShell variant="admin">
        <p className="py-20 text-center text-muted-foreground">
          {t("admin.games.not_found")}
        </p>
      </PageShell>
    );
  }

  const gameName = data.game.name;

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">
            {t("admin.games.edit_title", { name: gameName })}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.games.edit_subtitle")}
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" asChild>
            <Link href={`/admin/games/${id}`}>{t("admin.locations.view")}</Link>
          </Button>
          <Button variant="outline" asChild>
            <Link href="/admin/games">← {t("admin.tariffs.to_list")}</Link>
          </Button>
        </div>
      </div>

      <Tabs defaultValue="settings" className="space-y-4">
        <TabsList>
          <TabsTrigger value="settings">{t("common.settings")}</TabsTrigger>
          <TabsTrigger value="versions">{t("admin.games.versions")}</TabsTrigger>
        </TabsList>

        <TabsContent value="settings">
          <Card>
            <CardContent className="pt-6">
              <form onSubmit={onSave} className="space-y-4">
                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <Label htmlFor="edit-name">{t("common.name")}</Label>
                    <Input
                      id="edit-name"
                      value={form.name}
                      onChange={(e) => setField("name", e.target.value)}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-slug">Slug</Label>
                    <Input
                      id="edit-slug"
                      value={form.slug}
                      onChange={(e) => setField("slug", e.target.value)}
                      required
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="edit-desc">{t("common.description")}</Label>
                  <Textarea
                    id="edit-desc"
                    value={form.description}
                    onChange={(e) => setField("description", e.target.value)}
                    rows={4}
                  />
                </div>
                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <Label htmlFor="edit-code">{t("admin.games.code")}</Label>
                    <Input
                      id="edit-code"
                      value={form.code}
                      onChange={(e) => setField("code", e.target.value)}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-query">{t("admin.games.query")}</Label>
                    <Input
                      id="edit-query"
                      value={form.query}
                      onChange={(e) => setField("query", e.target.value)}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-minport">
                      {t("admin.games.minport")}
                    </Label>
                    <Input
                      id="edit-minport"
                      type="number"
                      min={1}
                      max={65535}
                      value={form.minport}
                      onChange={(e) => setField("minport", +e.target.value)}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-maxport">
                      {t("admin.games.maxport")}
                    </Label>
                    <Input
                      id="edit-maxport"
                      type="number"
                      min={1}
                      max={65535}
                      value={form.maxport}
                      onChange={(e) => setField("maxport", +e.target.value)}
                      required
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="edit-startup">
                    {t("admin.games.startup")}
                  </Label>
                  <Input
                    id="edit-startup"
                    value={form.default_startup_params}
                    onChange={(e) =>
                      setField("default_startup_params", e.target.value)
                    }
                    maxLength={512}
                  />
                </div>
                <div className="flex items-center gap-3">
                  <Switch
                    id="edit-status"
                    checked={form.status}
                    onCheckedChange={(v) => setField("status", v)}
                  />
                  <Label htmlFor="edit-status">{t("admin.games.enabled")}</Label>
                </div>
                <div className="flex justify-end gap-2 border-t pt-4">
                  <Button variant="outline" asChild>
                    <Link href="/admin/games">{t("common.cancel")}</Link>
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
        </TabsContent>

        <TabsContent value="versions">
          <div className="grid gap-4 lg:grid-cols-3">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">
                  {t("admin.games.add_version")}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <form onSubmit={onAddVersion} className="space-y-3">
                  <div className="space-y-2">
                    <Label>{t("admin.games.version_name")}</Label>
                    <Input
                      value={vForm.name}
                      onChange={(e) => setVField("name", e.target.value)}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label>{t("admin.games.source")}</Label>
                    <Select
                      value={vForm.source_type}
                      onValueChange={(v) =>
                        setVField("source_type", v as VersionFormState["source_type"])
                      }
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="archive">Archive (URL)</SelectItem>
                        <SelectItem value="steam">Steam (App ID)</SelectItem>
                      </SelectContent>
                    </Select>
                    <p className="text-xs text-muted-foreground">
                      {t("admin.games.source_hint")}
                    </p>
                  </div>
                  {vForm.source_type === "archive" && (
                    <div className="space-y-2">
                      <Label>{t("admin.games.archive_url")}</Label>
                      <Input
                        value={vForm.url}
                        onChange={(e) => setVField("url", e.target.value)}
                        required
                      />
                    </div>
                  )}
                  {vForm.source_type === "steam" && (
                    <div className="grid grid-cols-2 gap-3">
                      <div className="space-y-2">
                        <Label>Steam App ID</Label>
                        <Input
                          type="number"
                          min={1}
                          value={vForm.steam_app_id}
                          onChange={(e) => setVField("steam_app_id", e.target.value)}
                          required
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>Branch</Label>
                        <Input
                          value={vForm.steam_branch}
                          onChange={(e) => setVField("steam_branch", e.target.value)}
                        />
                      </div>
                    </div>
                  )}
                  <div className="grid grid-cols-2 gap-3">
                    <div className="space-y-2">
                      <Label>{t("admin.tariffs.col_position")}</Label>
                      <Input
                        type="number"
                        min={0}
                        value={vForm.sort_order}
                        onChange={(e) => setVField("sort_order", +e.target.value)}
                      />
                    </div>
                    <div className="flex items-end gap-3 pb-2">
                      <Switch
                        checked={vForm.is_active}
                        onCheckedChange={(v) => setVField("is_active", v)}
                      />
                      <Label>{t("admin.locations.active")}</Label>
                    </div>
                  </div>
                  <Button
                    type="submit"
                    className="w-full"
                    disabled={addVersionMut.isPending}
                  >
                    {addVersionMut.isPending
                      ? t("common.adding")
                      : t("admin.games.add_version")}
                  </Button>
                </form>
              </CardContent>
            </Card>

            <Card className="lg:col-span-2">
              <CardHeader>
                <CardTitle className="text-base">
                  {t("admin.games.versions_list")}
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {versions.length === 0 ? (
                  <p className="py-8 text-center text-sm text-muted-foreground">
                    {t("admin.games.versions_empty")}
                  </p>
                ) : (
                  versions.map((v) => (
                    <div
                      key={v.id}
                      className="flex flex-col gap-3 rounded-lg border bg-muted/30 p-4 sm:flex-row sm:items-center sm:justify-between"
                    >
                      <div>
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="font-medium">{v.name}</span>
                          <Badge
                            variant="secondary"
                            className={
                              v.is_active
                                ? "bg-emerald-500/10 text-emerald-600"
                                : undefined
                            }
                          >
                            {v.is_active
                              ? t("admin.locations.active")
                              : t("admin.locations.inactive")}
                          </Badge>
                          <span className="text-xs text-muted-foreground">
                            #{v.sort_order ?? 0}
                          </span>
                        </div>
                        <p className="mt-1 break-all text-xs text-muted-foreground">
                          {v.source_type === "steam" ? (
                            <>
                              app_id={v.steam_app_id ?? 0}
                              {v.steam_branch ? ` branch=${v.steam_branch}` : ""}
                            </>
                          ) : (
                            (v.archive_url || v.url || v.docker_image || "").slice(
                              0,
                              80
                            )
                          )}
                        </p>
                      </div>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="text-rose-500"
                        disabled={deleteVersionMut.isPending}
                        onClick={() => {
                          if (confirm(t("admin.games.version_delete_confirm"))) {
                            deleteVersionMut.mutate(v.id);
                          }
                        }}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  ))
                )}
              </CardContent>
            </Card>
          </div>
        </TabsContent>
      </Tabs>
    </PageShell>
  );
}
