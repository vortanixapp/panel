"use client";

import { Fragment, useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import {
  createAdminGroup,
  deleteAdminGroup,
  fetchAdminGroups,
  renameAdminGroup,
  updateAdminGroups,
  type AdminGroup,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";

const CATEGORY_LABEL_KEYS: Record<string, string> = {
  dashboard: "admin.groups.cat.dashboard",
  billing: "admin.groups.cat.billing",
  users: "admin.groups.cat.users",
  servers: "admin.groups.cat.servers",
  support: "admin.groups.cat.support",
  kb: "admin.groups.cat.kb",
  notifications: "admin.groups.cat.notifications",
  bug_report: "admin.groups.cat.bug_report",
  locations: "admin.groups.cat.locations",
  mysql: "admin.groups.cat.mysql",
  games: "admin.groups.cat.games",
  tariffs: "admin.groups.cat.tariffs",
  daemons: "admin.groups.cat.daemons",
  plugins: "admin.groups.cat.plugins",
  maps: "admin.groups.cat.maps",
  news: "admin.groups.cat.news",
  promotions: "admin.groups.cat.promotions",
  bonuses: "admin.groups.cat.bonuses",
  mailings: "admin.groups.cat.mailings",
  settings: "admin.groups.cat.settings",
  payment_providers: "admin.groups.cat.payment_providers",
  language: "admin.groups.cat.language",
  logs: "admin.groups.cat.logs",
  jobs: "admin.groups.cat.jobs",
  updates: "admin.groups.cat.updates",
  groups: "admin.groups.cat.groups",
  hosting: "admin.groups.cat.hosting",
  whmcs: "admin.groups.cat.whmcs",
};

type Matrix = Record<string, Record<string, boolean>>;
type DialogMode = "create" | "rename" | "delete" | null;

function groupPermissionsByCategory(keys: string[]) {
  const categories: Record<string, string[]> = {};
  for (const key of keys) {
    const cat = key.split(".")[1] ?? "other";
    if (!categories[cat]) categories[cat] = [];
    categories[cat].push(key);
  }
  return Object.entries(categories);
}

function countChanges(a: Record<string, boolean> | undefined, b: Record<string, boolean> | undefined, keys: string[]) {
  return keys.filter((key) => !!a?.[key] !== !!b?.[key]).length;
}

export function GroupsPageContent() {
  const t = useT();
  const queryClient = useQueryClient();
  const [matrix, setMatrix] = useState<Matrix>({});
  const [baseline, setBaseline] = useState<Matrix>({});
  const baselineRef = useRef<Matrix>({});
  const [selected, setSelected] = useState("");
  const [search, setSearch] = useState("");
  const [dialog, setDialog] = useState<DialogMode>(null);
  const [form, setForm] = useState({ name: "", description: "", copyFrom: "" });

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminGroups,
    queryFn: fetchAdminGroups,
  });

  const groups = useMemo<AdminGroup[]>(() => data?.items ?? [], [data?.items]);
  const keys = useMemo(() => data?.keys ?? [], [data?.keys]);
  const labels = useMemo(() => data?.permissionLabels ?? {}, [data?.permissionLabels]);

  useEffect(() => {
    if (!data?.items) return;
    const fresh: Matrix = {};
    for (const group of data.items) fresh[group.key] = { ...group.permissions };
    setMatrix((prev) => {
      const merged: Matrix = {};
      for (const key of Object.keys(fresh)) {
        const edited =
          prev[key] && countChanges(prev[key], baselineRef.current[key], data.keys) > 0;
        merged[key] = edited ? prev[key] : fresh[key];
      }
      return merged;
    });
    baselineRef.current = fresh;
    setBaseline(fresh);
  }, [data]);

  const current = groups.find((g) => g.key === selected) ?? groups[0];
  const currentPerms = current ? (matrix[current.key] ?? {}) : {};

  const dirtyKeys = useMemo(
    () => groups.filter((g) => countChanges(matrix[g.key], baseline[g.key], keys) > 0).map((g) => g.key),
    [groups, matrix, baseline, keys]
  );
  const changed = useMemo(
    () => groups.reduce((sum, g) => sum + countChanges(matrix[g.key], baseline[g.key], keys), 0),
    [groups, matrix, baseline, keys]
  );

  const groupName = (group: AdminGroup) =>
    group.system ? t(`admin.groups.name_${group.key}`) : group.name;
  const groupDescription = (group: AdminGroup) =>
    group.system
      ? t(`admin.groups.desc_${group.key}`)
      : group.description || t("admin.groups.no_description");
  const permLabel = (perm: string) => {
    const key = `admin.groups.perm.${perm}`;
    const text = t(key);
    return text === key ? (labels[perm] ?? perm) : text;
  };

  const categories = useMemo(() => {
    const q = search.trim().toLowerCase();
    const visible = q
      ? keys.filter((key) => {
          const translated = t(`admin.groups.perm.${key}`);
          return (
            key.toLowerCase().includes(q) ||
            translated.toLowerCase().includes(q) ||
            (labels[key] ?? "").toLowerCase().includes(q)
          );
        })
      : keys;
    return groupPermissionsByCategory(visible);
  }, [keys, labels, search, t]);

  const invalidate = () => queryClient.invalidateQueries({ queryKey: queryKeys.adminGroups });

  const saveMutation = useMutation({
    mutationFn: (payload: Record<string, Record<string, number>>) => updateAdminGroups(payload),
    onSuccess: () => {
      toast.success(t("admin.groups.saved"));
      baselineRef.current = matrix;
      setBaseline(matrix);
      void invalidate();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const createMutation = useMutation({
    mutationFn: createAdminGroup,
    onSuccess: (res) => {
      toast.success(t("admin.groups.created"));
      setSelected(res.group.key);
      setDialog(null);
      void invalidate();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const renameMutation = useMutation({
    mutationFn: (payload: { key: string; name: string; description: string }) =>
      renameAdminGroup(payload.key, { name: payload.name, description: payload.description }),
    onSuccess: () => {
      toast.success(t("admin.groups.renamed"));
      setDialog(null);
      void invalidate();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: deleteAdminGroup,
    onSuccess: () => {
      toast.success(t("admin.groups.deleted"));
      setSelected("");
      setDialog(null);
      void invalidate();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  function setPermissions(groupKey: string, perms: string[], value: boolean) {
    setMatrix((prev) => {
      const next = { ...(prev[groupKey] ?? {}) };
      for (const perm of perms) next[perm] = value;
      return { ...prev, [groupKey]: next };
    });
  }

  function save() {
    const payload: Record<string, Record<string, number>> = {};
    for (const key of dirtyKeys) {
      payload[key] = {};
      for (const [perm, value] of Object.entries(matrix[key] ?? {})) {
        if (value) payload[key][perm] = 1;
      }
    }
    if (Object.keys(payload).length === 0) return;
    saveMutation.mutate(payload);
  }

  function openCreate() {
    setForm({ name: "", description: "", copyFrom: "" });
    setDialog("create");
  }

  function openRename() {
    if (!current) return;
    setForm({ name: current.name, description: current.description, copyFrom: "" });
    setDialog("rename");
  }

  function submitGroupForm(e: React.FormEvent) {
    e.preventDefault();
    if (dialog === "create") {
      createMutation.mutate({
        name: form.name.trim(),
        description: form.description.trim(),
        copy_from: form.copyFrom || undefined,
      });
    } else if (dialog === "rename" && current) {
      renameMutation.mutate({
        key: current.key,
        name: form.name.trim(),
        description: form.description.trim(),
      });
    }
  }

  if (isLoading && !data) {
    return (
      <PageShell variant="admin">
        <div className="w-full space-y-5">
          <Skeleton className="h-16 w-full rounded-2xl" />
          <Skeleton className="h-96 w-full rounded-2xl" />
        </div>
      </PageShell>
    );
  }

  const formPending = createMutation.isPending || renameMutation.isPending;

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-5">
        <div className="flex flex-wrap items-end justify-between gap-6">
          <div className="space-y-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-tight">
              {t("admin.groups.title")}
            </h1>
            <p className="text-sm text-muted-foreground">{t("admin.groups.subtitle")}</p>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <span
              className={cn(
                "font-mono text-xs",
                changed ? "text-amber-500" : "text-muted-foreground"
              )}
            >
              {changed
                ? t("admin.groups.changed", { count: changed })
                : t("admin.groups.changed_none")}
            </span>
            <Button variant="outline" className="h-[38px] text-[13px]" onClick={openCreate}>
              {t("admin.groups.create")}
            </Button>
            <Button
              className="h-[38px] text-[13px]"
              onClick={save}
              disabled={saveMutation.isPending || changed === 0}
            >
              {saveMutation.isPending ? t("common.saving") : t("admin.groups.save")}
            </Button>
          </div>
        </div>

        <div className="grid items-start gap-4 lg:grid-cols-[300px_minmax(0,1fr)]">
          <aside className="flex flex-col gap-2">
            {groups.map((group) => {
              const enabled = keys.filter((key) => matrix[group.key]?.[key]).length;
              const active = current?.key === group.key;
              return (
                <button
                  key={group.key}
                  type="button"
                  onClick={() => setSelected(group.key)}
                  className={cn(
                    "flex flex-col gap-1.5 rounded-xl border bg-card px-4 py-3 text-left transition-colors",
                    active ? "border-primary ring-1 ring-primary/40" : "hover:border-foreground/20"
                  )}
                >
                  <span className="flex items-center justify-between gap-2">
                    <span className="truncate text-[13px] font-medium">{groupName(group)}</span>
                    {dirtyKeys.includes(group.key) && (
                      <span className="size-1.5 shrink-0 rounded-full bg-amber-500" />
                    )}
                  </span>
                  <span className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span>
                      {group.system
                        ? t("admin.groups.badge_system")
                        : t("admin.groups.badge_custom")}
                    </span>
                    <span>·</span>
                    <span>{t("admin.groups.members", { count: group.members })}</span>
                    <span className="ml-auto font-mono">
                      {enabled}/{keys.length}
                    </span>
                  </span>
                </button>
              );
            })}
            <p className="px-1 pt-1 text-xs leading-relaxed text-muted-foreground">
              {t("admin.groups.list_hint")}
            </p>
          </aside>

          {current && (
            <section className="overflow-hidden rounded-2xl border bg-card">
              <div className="flex flex-wrap items-start justify-between gap-4 border-b px-5 py-4">
                <div className="min-w-0 space-y-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <h2 className="text-[15px] font-semibold">{groupName(current)}</h2>
                    <span className="rounded-md border px-2 py-0.5 text-[11px] text-muted-foreground">
                      {current.system
                        ? t("admin.groups.badge_system")
                        : t("admin.groups.badge_custom")}
                    </span>
                  </div>
                  <p className="text-xs text-muted-foreground">{groupDescription(current)}</p>
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPermissions(current.key, keys, true)}
                  >
                    {t("admin.groups.enable_all")}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPermissions(current.key, keys, false)}
                  >
                    {t("admin.groups.disable_all")}
                  </Button>
                  {!current.system && (
                    <>
                      <Button variant="outline" size="sm" onClick={openRename}>
                        {t("admin.groups.rename")}
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        className="text-destructive"
                        onClick={() => setDialog("delete")}
                      >
                        {t("admin.groups.delete")}
                      </Button>
                    </>
                  )}
                </div>
              </div>

              {current.key === "admin" && (
                <div className="border-b bg-muted/20 px-5 py-2.5 text-xs text-muted-foreground">
                  {t("admin.groups.admin_note")}
                </div>
              )}

              <div className="border-b px-5 py-3">
                <Input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder={t("admin.groups.filter_placeholder")}
                  className="h-9 max-w-[360px] rounded-lg text-[13px] md:text-[13px]"
                />
              </div>

              {categories.map(([cat, perms]) => {
                const on = perms.filter((perm) => currentPerms[perm]).length;
                return (
                  <Fragment key={cat}>
                    <div className="flex items-center justify-between gap-3 bg-muted/30 px-5 py-2.5">
                      <span className="font-mono text-[10px] tracking-[0.1em] text-muted-foreground uppercase">
                        {CATEGORY_LABEL_KEYS[cat] ? t(CATEGORY_LABEL_KEYS[cat]) : cat} · {on}/
                        {perms.length}
                      </span>
                      <Switch
                        checked={on === perms.length}
                        onCheckedChange={(value) => setPermissions(current.key, perms, value)}
                        aria-label={CATEGORY_LABEL_KEYS[cat] ? t(CATEGORY_LABEL_KEYS[cat]) : cat}
                      />
                    </div>
                    {perms.map((perm) => (
                      <div
                        key={perm}
                        className="flex items-center justify-between gap-4 border-b px-5 py-3 transition-colors hover:bg-muted/30"
                      >
                        <div className="min-w-0 space-y-1">
                          <div className="text-[13px]">{permLabel(perm)}</div>
                          <div className="font-mono text-[11px] text-muted-foreground/70">
                            {perm}
                          </div>
                        </div>
                        <Switch
                          checked={!!currentPerms[perm]}
                          onCheckedChange={(value) => setPermissions(current.key, [perm], value)}
                          aria-label={`${permLabel(perm)} — ${groupName(current)}`}
                        />
                      </div>
                    ))}
                  </Fragment>
                );
              })}

              {categories.length === 0 && (
                <div className="px-5 py-10 text-center text-[13px] text-muted-foreground">
                  {t("common.not_found")}
                </div>
              )}
            </section>
          )}
        </div>
      </div>

      <Dialog open={dialog !== null} onOpenChange={(open) => !open && setDialog(null)}>
        <DialogContent className="sm:max-w-md">
          {dialog === "delete" && current ? (
            <>
              <DialogHeader>
                <DialogTitle>
                  {t("admin.groups.delete_title", { name: groupName(current) })}
                </DialogTitle>
                <DialogDescription>
                  {current.members > 0
                    ? t("admin.groups.delete_blocked", { count: current.members })
                    : t("admin.groups.delete_confirm")}
                </DialogDescription>
              </DialogHeader>
              <DialogFooter>
                <Button variant="outline" onClick={() => setDialog(null)}>
                  {t("common.cancel")}
                </Button>
                <Button
                  variant="destructive"
                  disabled={current.members > 0 || deleteMutation.isPending}
                  onClick={() => deleteMutation.mutate(current.key)}
                >
                  {t("admin.groups.delete")}
                </Button>
              </DialogFooter>
            </>
          ) : (
            <form onSubmit={submitGroupForm} className="space-y-4">
              <DialogHeader>
                <DialogTitle>
                  {dialog === "create"
                    ? t("admin.groups.create_title")
                    : t("admin.groups.rename_title")}
                </DialogTitle>
                <DialogDescription>
                  {dialog === "create"
                    ? t("admin.groups.create_hint")
                    : t("admin.groups.rename_hint")}
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-1.5">
                <Label className="text-xs font-normal text-muted-foreground">
                  {t("admin.groups.field_name")}
                </Label>
                <Input
                  value={form.name}
                  maxLength={64}
                  required
                  autoFocus
                  onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))}
                  className="h-9 text-[13px] md:text-[13px]"
                />
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs font-normal text-muted-foreground">
                  {t("admin.groups.field_description")}
                </Label>
                <Textarea
                  value={form.description}
                  maxLength={255}
                  rows={2}
                  onChange={(e) => setForm((p) => ({ ...p, description: e.target.value }))}
                  className="text-[13px] md:text-[13px]"
                />
              </div>
              {dialog === "create" && (
                <div className="space-y-1.5">
                  <Label className="text-xs font-normal text-muted-foreground">
                    {t("admin.groups.field_copy_from")}
                  </Label>
                  <select
                    value={form.copyFrom}
                    onChange={(e) => setForm((p) => ({ ...p, copyFrom: e.target.value }))}
                    className="h-9 w-full rounded-lg border border-input bg-transparent px-2.5 text-[13px] outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:bg-input/30"
                  >
                    <option value="">{t("admin.groups.copy_none")}</option>
                    {groups.map((group) => (
                      <option key={group.key} value={group.key}>
                        {groupName(group)}
                      </option>
                    ))}
                  </select>
                </div>
              )}
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setDialog(null)}>
                  {t("common.cancel")}
                </Button>
                <Button type="submit" disabled={formPending || !form.name.trim()}>
                  {dialog === "create" ? t("admin.groups.create") : t("common.save_changes")}
                </Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>
    </PageShell>
  );
}
