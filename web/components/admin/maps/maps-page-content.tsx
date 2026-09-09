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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  createAdminMap,
  deleteAdminMap,
  fetchAdminGames,
  fetchAdminMaps,
  uploadAdminMapArchive,
  type AdminMap,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";

type FormState = {
  id?: string;
  slug: string;
  name: string;
  category: string;
  version: string;
  game_slug: string;
  restart_required: boolean;
  active: boolean;
};

const EMPTY: FormState = {
  slug: "",
  name: "",
  category: "",
  version: "",
  game_slug: "",
  restart_required: false,
  active: true,
};

const ANY_GAME = "__any__";

const NO_CATEGORY = "__no_category__";

export function MapsPageContent() {
  const t = useT();
  const qc = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: ["admin-maps"],
    queryFn: async () => (await fetchAdminMaps()).maps,
  });
  const { data: gamesData } = useQuery({
    queryKey: ["admin-games-for-maps"],
    queryFn: async () => (await fetchAdminGames()).games,
  });
  const maps = useMemo(() => data ?? [], [data]);
  const games = gamesData ?? [];

  const [form, setForm] = useState<FormState | null>(null);
  const [archiveFile, setArchiveFile] = useState<File | null>(null);
  const [search, setSearch] = useState("");

  const grouped = useMemo(() => {
    const q = search.trim().toLowerCase();
    const filtered = q
      ? maps.filter((m) => m.name.toLowerCase().includes(q) || m.slug.toLowerCase().includes(q))
      : maps;
    const map = new Map<string, AdminMap[]>();
    for (const m of filtered) {
      const key = m.category || NO_CATEGORY;
      map.set(key, [...(map.get(key) ?? []), m]);
    }
    return [...map.entries()].sort((a, b) => a[0].localeCompare(b[0], "ru"));
  }, [maps, search]);

  const save = useMutation({
    mutationFn: async (state: FormState) => {
      const res = await createAdminMap({
        id: state.id,
        slug: state.slug,
        name: state.name,
        category: state.category,
        version: state.version,
        game_slug: state.game_slug,
        restart_required: state.restart_required,
        active: state.active,
      });
      const id = state.id ?? res.id;
      if (id && archiveFile) await uploadAdminMapArchive(id, archiveFile);
    },
    onSuccess: () => {
      toast.success(form?.id ? t("admin.maps.saved") : t("admin.maps.created"));
      setForm(null);
      setArchiveFile(null);
      void qc.invalidateQueries({ queryKey: ["admin-maps"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const remove = useMutation({
    mutationFn: ({ id, force }: { id: string; force: boolean }) => deleteAdminMap(id, force),
    onSuccess: () => {
      toast.success(t("admin.maps.deleted"));
      void qc.invalidateQueries({ queryKey: ["admin-maps"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  async function onDelete(m: AdminMap) {
    if (!confirm(t("admin.maps.delete_confirm", { name: m.name }))) return;
    try {
      await remove.mutateAsync({ id: m.id, force: false });
    } catch (e) {
      const msg = e instanceof Error ? e.message : "";
      if (!msg.includes("force=1")) return;
      if (!confirm(`${msg}\n\n${t("admin.maps.delete_force")}`)) return;
      remove.mutate({ id: m.id, force: true });
    }
  }

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!form) return;
    if (!form.slug.trim() || !form.name.trim()) {
      toast.error(t("admin.maps.need_name_slug"));
      return;
    }
    if (!form.id && !archiveFile) {
      toast.error(t("admin.maps.need_archive"));
      return;
    }
    save.mutate(form);
  }

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("admin.maps.title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.maps.subtitle")}
          </p>
        </div>
        <div className="flex gap-2">
          <Input
            className="w-56"
            placeholder={t("admin.maps.search_placeholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <Button onClick={() => { setForm({ ...EMPTY }); setArchiveFile(null); }}>
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
                placeholder="de_dust2"
                onChange={(e) => setForm({ ...form, slug: e.target.value })}
                disabled={!!form.id}
                required
              />
            </div>
            <div>
              <Label>{t("admin.maps.category")}</Label>
              <Input
                value={form.category}
                placeholder={t("admin.maps.category_placeholder")}
                onChange={(e) => setForm({ ...form, category: e.target.value })}
              />
            </div>
          </div>

          <div className="grid gap-3 md:grid-cols-3">
            <div>
              <Label>{t("admin.maps.version")}</Label>
              <Input value={form.version} onChange={(e) => setForm({ ...form, version: e.target.value })} />
            </div>
            <div>
              <Label>{t("common.game")}</Label>
              <Select
                value={form.game_slug || ANY_GAME}
                onValueChange={(v) => setForm({ ...form, game_slug: v === ANY_GAME ? "" : v })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ANY_GAME}>
                    {t("admin.maps.any_game")}
                  </SelectItem>
                  {games.map((g) => (
                    <SelectItem key={g.id} value={g.slug}>
                      {g.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>{t("admin.maps.archive")}</Label>
              <Input type="file" accept=".zip" onChange={(e) => setArchiveFile(e.target.files?.[0] ?? null)} />
              <p className="mt-1 text-[11px] text-muted-foreground">
                {t("admin.maps.archive_hint")}
              </p>
            </div>
          </div>

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
              {t("admin.maps.available")}
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
      ) : maps.length === 0 ? (
        <div className="rounded-lg border bg-card p-8 text-center text-sm text-muted-foreground">
          {t("admin.maps.empty")}
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
                {items.map((m) => (
                  <div key={m.id} className="flex items-center gap-4 border-b p-3 last:border-0">
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-medium">{m.name}</span>
                        <span className="font-mono text-xs text-muted-foreground">{m.slug}</span>
                        {!m.active && (
                          <Badge variant="secondary">
                            {t("admin.maps.hidden")}
                          </Badge>
                        )}
                        {m.restart_required && (
                          <Badge variant="outline">
                            {t("admin.maps.restart_badge")}
                          </Badge>
                        )}
                        {!m.archive_path && (
                          <Badge variant="destructive">
                            {t("admin.maps.no_archive")}
                          </Badge>
                        )}
                      </div>
                      <div className="mt-0.5 text-xs text-muted-foreground">
                        {m.game_slug || t("admin.maps.any_game_short")} ·{" "}
                        {m.version || t("admin.maps.no_version")} ·{" "}
                        {t("admin.maps.files_count", { count: m.file_count })}
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          setArchiveFile(null);
                          setForm({
                            id: m.id,
                            slug: m.slug,
                            name: m.name,
                            category: m.category,
                            version: m.version,
                            game_slug: m.game_slug,
                            restart_required: m.restart_required,
                            active: m.active,
                          });
                        }}
                      >
                        {t("common.edit")}
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        disabled={remove.isPending}
                        onClick={() => void onDelete(m)}
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
