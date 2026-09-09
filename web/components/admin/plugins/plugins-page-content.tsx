"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import {
  PluginActionsEditor,
  emptyPluginAction,
} from "@/components/admin/plugins/plugin-actions-editor";
import {
  createAdminPlugin,
  deleteAdminPlugin,
  fetchAdminGames,
  fetchAdminPlugins,
  uploadAdminPluginArchive,
  uploadAdminPluginImage,
  type AdminPlugin,
  type AdminPluginAction,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";

type FormState = {
  id?: string;
  slug: string;
  name: string;
  category: string;
  version: string;
  description: string;
  install_path: string;
  all_games: boolean;
  supported_games: string[];
  file_actions: AdminPluginAction[];
  uninstall_actions: AdminPluginAction[];
  restart_required: boolean;
  active: boolean;
};

const EMPTY: FormState = {
  slug: "",
  name: "",
  category: "",
  version: "",
  description: "",
  install_path: "",
  all_games: true,
  supported_games: [],
  file_actions: [],
  uninstall_actions: [],
  restart_required: false,
  active: true,
};

const NO_CATEGORY = "__no_category__";

function formatSize(bytes: number, t: TranslateFn): string {
  if (!bytes) return "—";
  if (bytes < 1024 * 1024)
    return `${Math.round(bytes / 1024)} ${t("admin.plugins.unit_kb")}`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} ${t("admin.infra.unit_mb")}`;
}

export function PluginsPageContent() {
  const t = useT();
  const qc = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: ["admin-plugins"],
    queryFn: async () => (await fetchAdminPlugins()).plugins,
  });
  const { data: gamesData } = useQuery({
    queryKey: ["admin-games-for-plugins"],
    queryFn: async () => (await fetchAdminGames()).games,
  });
  const plugins = useMemo(() => data ?? [], [data]);
  const games = gamesData ?? [];

  const [form, setForm] = useState<FormState | null>(null);
  const [archiveFile, setArchiveFile] = useState<File | null>(null);
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [search, setSearch] = useState("");

  const grouped = useMemo(() => {
    const q = search.trim().toLowerCase();
    const filtered = q
      ? plugins.filter(
          (p) =>
            p.name.toLowerCase().includes(q) ||
            p.slug.toLowerCase().includes(q) ||
            p.category.toLowerCase().includes(q)
        )
      : plugins;
    const map = new Map<string, AdminPlugin[]>();
    for (const p of filtered) {
      const key = p.category || NO_CATEGORY;
      map.set(key, [...(map.get(key) ?? []), p]);
    }
    return [...map.entries()].sort((a, b) => a[0].localeCompare(b[0], "ru"));
  }, [plugins, search]);

  const save = useMutation({
    mutationFn: async (state: FormState) => {
      const res = await createAdminPlugin({
        id: state.id,
        slug: state.slug,
        name: state.name,
        category: state.category,
        version: state.version,
        description: state.description,
        install_path: state.install_path,
        all_games: state.all_games,
        supported_games: state.supported_games,
        file_actions: state.file_actions,
        uninstall_actions: state.uninstall_actions,
        restart_required: state.restart_required,
        active: state.active,
      });
      const id = state.id ?? res.id;
      if (!id) return;
      if (archiveFile) await uploadAdminPluginArchive(id, archiveFile);
      if (imageFile) await uploadAdminPluginImage(id, imageFile);
    },
    onSuccess: () => {
      toast.success(
        form?.id ? t("admin.plugins.saved") : t("admin.plugins.created")
      );
      setForm(null);
      setArchiveFile(null);
      setImageFile(null);
      void qc.invalidateQueries({ queryKey: ["admin-plugins"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const remove = useMutation({
    mutationFn: ({ id, force }: { id: string; force: boolean }) => deleteAdminPlugin(id, force),
    onSuccess: () => {
      toast.success(t("admin.plugins.deleted"));
      void qc.invalidateQueries({ queryKey: ["admin-plugins"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  async function onDelete(p: AdminPlugin) {
    if (!confirm(t("admin.plugins.delete_confirm", { name: p.name }))) return;
    try {
      await remove.mutateAsync({ id: p.id, force: false });
    } catch (e) {
      const msg = e instanceof Error ? e.message : "";
      if (!msg.includes("force=1")) return;
      if (!confirm(`${msg}\n\n${t("admin.maps.delete_force")}`)) return;
      remove.mutate({ id: p.id, force: true });
    }
  }

  function startEdit(p: AdminPlugin) {
    setArchiveFile(null);
    setImageFile(null);
    setForm({
      id: p.id,
      slug: p.slug,
      name: p.name,
      category: p.category,
      version: p.version,
      description: p.description,
      install_path: p.install_path,
      all_games: p.all_games,
      supported_games: p.supported_games,
      file_actions: p.file_actions ?? [],
      uninstall_actions: p.uninstall_actions ?? [],
      restart_required: p.restart_required,
      active: p.active,
    });
  }

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!form) return;
    if (!form.slug.trim() || !form.name.trim()) {
      toast.error(t("admin.maps.need_name_slug"));
      return;
    }
    if (!form.all_games && form.supported_games.length === 0) {
      toast.error(t("admin.plugins.need_games"));
      return;
    }
    save.mutate(form);
  }

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("admin.plugins.title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.plugins.subtitle")}
          </p>
        </div>
        <div className="flex gap-2">
          <Input
            className="w-56"
            placeholder={t("admin.maps.search_placeholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <Button onClick={() => { setForm({ ...EMPTY }); setArchiveFile(null); setImageFile(null); }}>
            + {t("common.create")}
          </Button>
        </div>
      </div>

      {form && (
        <form onSubmit={submit} className="mb-6 space-y-4 rounded-lg border bg-card p-4">
          <div className="grid gap-3 md:grid-cols-3">
            <div>
              <Label>{t("common.name")}</Label>
              <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required />
            </div>
            <div>
              <Label>Slug</Label>
              <Input
                value={form.slug}
                onChange={(e) => setForm({ ...form, slug: e.target.value })}
                disabled={!!form.id}
                required
              />
            </div>
            <div>
              <Label>{t("admin.maps.category")}</Label>
              <Input
                value={form.category}
                placeholder={t("admin.plugins.category_placeholder")}
                onChange={(e) => setForm({ ...form, category: e.target.value })}
              />
            </div>
          </div>

          <div className="grid gap-3 md:grid-cols-2">
            <div>
              <Label>{t("admin.maps.version")}</Label>
              <Input value={form.version} onChange={(e) => setForm({ ...form, version: e.target.value })} />
            </div>
            <div>
              <Label>{t("admin.plugins.install_path")}</Label>
              <Input
                value={form.install_path}
                placeholder="addons/amxmodx"
                onChange={(e) => setForm({ ...form, install_path: e.target.value })}
              />
              <p className="mt-1 text-[11px] text-muted-foreground">
                {t("admin.plugins.install_path_hint")}
              </p>
            </div>
          </div>

          <div>
            <Label>{t("common.description")}</Label>
            <Textarea
              rows={2}
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
            />
          </div>

          <div className="grid gap-3 md:grid-cols-2">
            <div>
              <Label>{t("admin.plugins.archive")}</Label>
              <Input
                type="file"
                accept=".zip,.tar,.tar.gz,.tgz"
                onChange={(e) => setArchiveFile(e.target.files?.[0] ?? null)}
              />
              <p className="mt-1 text-[11px] text-muted-foreground">
                {t("admin.plugins.archive_hint")}
              </p>
            </div>
            <div>
              <Label>{t("admin.plugins.preview")}</Label>
              <Input
                type="file"
                accept="image/*"
                onChange={(e) => setImageFile(e.target.files?.[0] ?? null)}
              />
              <p className="mt-1 text-[11px] text-muted-foreground">
                {t("admin.plugins.preview_hint")}
              </p>
            </div>
          </div>

          <div className="rounded-lg border bg-background p-3">
            <label className="flex items-center gap-2 text-sm font-medium">
              <Switch
                checked={form.all_games}
                onCheckedChange={(v) => setForm({ ...form, all_games: v })}
              />
              {t("admin.plugins.all_games")}
            </label>
            {!form.all_games && (
              <div className="mt-3 flex flex-wrap gap-2">
                {games.map((g) => {
                  const picked = form.supported_games.includes(g.slug);
                  return (
                    <button
                      key={g.id}
                      type="button"
                      onClick={() =>
                        setForm({
                          ...form,
                          supported_games: picked
                            ? form.supported_games.filter((s) => s !== g.slug)
                            : [...form.supported_games, g.slug],
                        })
                      }
                      className={`rounded-full border px-3 py-1 text-xs transition-colors ${
                        picked ? "border-primary bg-primary/10 text-primary" : "text-muted-foreground"
                      }`}
                    >
                      {g.name}
                    </button>
                  );
                })}
              </div>
            )}
          </div>

          <PluginActionsEditor
            title={t("admin.plugins.install_actions")}
            hint={t("admin.plugins.install_actions_hint")}
            actions={form.file_actions}
            onChange={(v) => setForm({ ...form, file_actions: v })}
          />
          <PluginActionsEditor
            title={t("admin.plugins.uninstall_actions")}
            hint={t("admin.plugins.uninstall_actions_hint")}
            actions={form.uninstall_actions}
            onChange={(v) => setForm({ ...form, uninstall_actions: v })}
          />

          <div className="flex flex-wrap items-center gap-6">
            <label className="flex items-center gap-2 text-sm">
              <Switch
                checked={form.restart_required}
                onCheckedChange={(v) => setForm({ ...form, restart_required: v })}
              />
              {t("admin.maps.restart_required")}
            </label>
            <label className="flex items-center gap-2 text-sm">
              <Switch checked={form.active} onCheckedChange={(v) => setForm({ ...form, active: v })} />
              {t("admin.plugins.available")}
            </label>
          </div>

          <div className="flex gap-2">
            <Button type="submit" disabled={save.isPending}>
              {save.isPending
                ? t("common.saving")
                : form.id
                  ? t("common.save")
                  : t("common.create")}
            </Button>
            <Button type="button" variant="ghost" onClick={() => setForm(null)}>
              {t("common.cancel")}
            </Button>
          </div>
        </form>
      )}

      {isLoading ? (
        <div className="space-y-2">
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
        </div>
      ) : plugins.length === 0 ? (
        <div className="rounded-lg border bg-card p-8 text-center text-sm text-muted-foreground">
          {t("admin.plugins.empty")}
        </div>
      ) : (
        <div className="space-y-5">
          {grouped.map(([category, items]) => (
            <div key={category}>
              <div className="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
                {category === NO_CATEGORY
                  ? t("admin.maps.no_category")
                  : category}
              </div>
              <div className="overflow-hidden rounded-lg border bg-card">
                {items.map((p) => (
                  <div key={p.id} className="flex items-center gap-4 border-b p-3 last:border-0">
                    {p.image_url ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={p.image_url} alt="" className="size-10 rounded object-cover" />
                    ) : (
                      <div className="size-10 rounded bg-muted" />
                    )}
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-medium">{p.name}</span>
                        <span className="font-mono text-xs text-muted-foreground">{p.slug}</span>
                        {!p.active && (
                          <Badge variant="secondary">
                            {t("admin.plugins.hidden")}
                          </Badge>
                        )}
                        {p.restart_required && (
                          <Badge variant="outline">
                            {t("admin.maps.restart_badge")}
                          </Badge>
                        )}
                      </div>
                      <div className="mt-0.5 text-xs text-muted-foreground">
                        {p.install_path || t("admin.plugins.no_unpack")} ·{" "}
                        {p.version || t("admin.maps.no_version")} ·{" "}
                        {p.has_archive
                          ? formatSize(p.archive_size, t)
                          : t("admin.maps.no_archive")}{" "}
                        ·{" "}
                        {p.all_games
                          ? t("admin.plugins.all_games_short")
                          : p.supported_games.join(", ") ||
                            t("admin.plugins.no_games")}{" "}
                        · {t("admin.plugins.actions_count", {
                          count: p.file_actions.length,
                        })}
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <Button size="sm" variant="outline" onClick={() => startEdit(p)}>
                        {t("common.edit")}
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        disabled={remove.isPending}
                        onClick={() => void onDelete(p)}
                      >
                        {t("common.delete")}
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </PageShell>
  );
}
