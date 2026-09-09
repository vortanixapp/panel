"use client";

import { Fragment, useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { fetchAdminGroups, updateAdminGroups } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";

const CATEGORY_LABEL_KEYS: Record<string, string> = {
  dashboard: "admin.groups.cat.dashboard",
  billing: "admin.groups.cat.billing",
  users: "admin.groups.cat.users",
  servers: "admin.groups.cat.servers",
  support: "admin.groups.cat.support",
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
  groups: "admin.groups.cat.groups",
  hosting: "admin.groups.cat.hosting",
};

type Matrix = Record<string, Record<string, boolean>>;

function groupPermissionsByCategory(keys: string[]) {
  const categories: Record<string, string[]> = {};
  for (const key of keys) {
    const cat = key.split(".")[1] ?? "other";
    if (!categories[cat]) categories[cat] = [];
    categories[cat].push(key);
  }
  return categories;
}

function countEnabled(matrix: Matrix, role: string, keys: string[]) {
  return keys.filter((key) => matrix[role]?.[key]).length;
}

export function GroupsPageContent() {
  const t = useT();
  const queryClient = useQueryClient();
  const [matrix, setMatrix] = useState<Matrix>({});
  const [baseline, setBaseline] = useState<Matrix>({});
  const [search, setSearch] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminGroups,
    queryFn: fetchAdminGroups,
  });

  useEffect(() => {
    if (data?.matrix) {
      setMatrix(data.matrix);
      setBaseline(data.matrix);
    }
  }, [data]);

  const saveMutation = useMutation({
    mutationFn: (payload: Record<string, Record<string, number>>) =>
      updateAdminGroups(payload),
    onSuccess: (res) => {
      toast.success(res.message ?? t("admin.groups.saved"));
      setBaseline(matrix);
      queryClient.invalidateQueries({ queryKey: queryKeys.adminGroups });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const groups = useMemo(() => data?.groups ?? {}, [data?.groups]);
  const keys = useMemo(() => data?.keys ?? [], [data?.keys]);
  const labels = data?.permissionLabels ?? {};
  const roles = Object.entries(groups);

  const categories = useMemo(() => {
    const q = search.trim().toLowerCase();
    const visible = q
      ? keys.filter(
          (key) =>
            key.toLowerCase().includes(q) ||
            (labels[key] ?? "").toLowerCase().includes(q)
        )
      : keys;
    return Object.entries(groupPermissionsByCategory(visible));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [keys, search, data]);

  const changed = useMemo(() => {
    let count = 0;
    for (const [role] of roles) {
      for (const key of keys) {
        if (!!matrix[role]?.[key] !== !!baseline[role]?.[key]) count += 1;
      }
    }
    return count;
  }, [matrix, baseline, roles, keys]);

  function togglePermission(role: string, perm: string) {
    setMatrix((prev) => ({
      ...prev,
      [role]: {
        ...(prev[role] ?? {}),
        [perm]: !prev[role]?.[perm],
      },
    }));
  }

  async function save() {
    const payload: Record<string, Record<string, number>> = {};
    for (const [role, perms] of Object.entries(matrix)) {
      payload[role] = {};
      for (const [perm, val] of Object.entries(perms)) {
        if (val) payload[role][perm] = 1;
      }
    }
    await saveMutation.mutateAsync(payload);
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

  const gridCols = `minmax(280px,1fr) repeat(${Math.max(roles.length, 1)}, 150px)`;

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-5">
        <div className="flex flex-wrap items-end justify-between gap-6">
          <div className="space-y-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-tight">
              {t("admin.groups.title")}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t("admin.groups.subtitle")}
            </p>
          </div>
          <div className="flex items-center gap-3">
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
            <Button
              className="h-[38px] text-[13px]"
              onClick={save}
              disabled={saveMutation.isPending}
            >
              {saveMutation.isPending
                ? t("common.saving")
                : t("admin.groups.save")}
            </Button>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t("admin.groups.filter_placeholder")}
            className="h-9 max-w-[360px] flex-1 basis-[300px] rounded-lg text-[13px] md:text-[13px]"
          />
          <div className="flex flex-wrap gap-2.5">
            {roles.map(([role, label]) => (
              <div
                key={role}
                className="flex items-baseline gap-2.5 rounded-lg border bg-card px-3.5 py-2"
              >
                <span className="text-xs text-muted-foreground">{label}</span>
                <span className="font-mono text-[13px]">
                  {countEnabled(matrix, role, keys)}
                </span>
              </div>
            ))}
          </div>
        </div>

        <div className="overflow-hidden rounded-2xl border bg-card">
          <div className="overflow-x-auto">
            <div className="flex min-w-[900px] flex-col">
              <div
                className="grid gap-3 border-b bg-muted/40 px-5 py-3 font-mono text-[11px] tracking-wider text-muted-foreground uppercase"
                style={{ gridTemplateColumns: gridCols }}
              >
                <div>{t("admin.groups.col_permission")}</div>
                {roles.map(([role, label]) => (
                  <div key={role} className="text-center">
                    {label}
                  </div>
                ))}
              </div>

              {categories.map(([cat, perms]) => (
                <Fragment key={cat}>
                  <div className="bg-muted/20 px-5 py-2.5 font-mono text-[10px] tracking-[0.1em] text-muted-foreground uppercase">
                    {CATEGORY_LABEL_KEYS[cat] ? t(CATEGORY_LABEL_KEYS[cat]) : cat}
                  </div>
                  {perms.map((perm) => (
                    <div
                      key={perm}
                      className="grid items-center gap-3 border-b px-5 py-3 transition-colors hover:bg-muted/30"
                      style={{ gridTemplateColumns: gridCols }}
                    >
                      <div className="min-w-0 space-y-1">
                        <div className="text-[13px]">
                          {labels[perm] ?? perm}
                        </div>
                        <div className="font-mono text-[11px] text-muted-foreground/70">
                          {perm}
                        </div>
                      </div>
                      {roles.map(([role]) => (
                        <div key={role} className="flex justify-center">
                          <Switch
                            checked={!!matrix[role]?.[perm]}
                            onCheckedChange={() => togglePermission(role, perm)}
                            aria-label={`${labels[perm] ?? perm} — ${groups[role]}`}
                          />
                        </div>
                      ))}
                    </div>
                  ))}
                </Fragment>
              ))}

              {categories.length === 0 && (
                <div className="px-5 py-10 text-center text-[13px] text-muted-foreground">
                  {t("common.not_found")}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </PageShell>
  );
}
