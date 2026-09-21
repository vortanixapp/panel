"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { RefreshCw, Search } from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { ConfirmDialog } from "@/components/servers/confirm-dialog";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Btn,
  EmptyState,
  Notice,
  Toggle,
  VX_CARD,
  VX_FAINT,
  VX_INPUT,
  VX_MONO_LABEL,
  VX_MUTED,
} from "@/components/vx/panel-ui";
import { AgentsTable } from "@/components/admin/agents/agents-table";
import {
  apiErrorCode,
  fetchAgents,
  setAgentsAutoEnabled,
  setAgentsAutoUpdate,
  updateAgents,
  type AgentRow,
} from "@/lib/api";
import {
  AGENT_FILTERS,
  agentErrorText,
  matchesFilter,
  matchesQuery,
  type AgentFilter,
} from "@/lib/agents";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

function isFilter(v: string | null): v is AgentFilter {
  return v != null && (AGENT_FILTERS as readonly string[]).includes(v);
}

function SummaryCell({
  label,
  value,
  tone,
  onClick,
  active,
}: {
  label: string;
  value: number;
  tone?: "warn" | "danger";
  onClick: () => void;
  active: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "relative isolate bg-[var(--vx-card)] px-[18px] py-4 text-left transition-colors hover:bg-[var(--vx-elevated)]",
        active && "bg-[var(--vx-elevated)]"
      )}
    >
      <div className={VX_MONO_LABEL}>{label}</div>
      <div
        className={cn(
          "mt-2 font-mono text-[24px] font-medium tracking-[-0.02em] tabular-nums",
          value > 0 && tone === "warn" && "text-[var(--vx-warn)]",
          value > 0 && tone === "danger" && "text-[var(--vx-danger)]"
        )}
      >
        {value}
      </div>
    </button>
  );
}

function reportError(err: unknown, fallback: string) {
  toast.error(agentErrorText(apiErrorCode(err), err instanceof Error ? err.message : fallback));
}

export function AgentsPage() {
  const t = useT();
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const qc = useQueryClient();
  const filterParam = params.get("filter");
  const filter: AgentFilter = isFilter(filterParam) ? filterParam : "all";
  const [query, setQuery] = useState(params.get("q") ?? "");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [bulkOpen, setBulkOpen] = useState(false);

  const agentsQuery = useQuery({
    queryKey: queryKeys.agents,
    queryFn: fetchAgents,
    refetchInterval: (q) => {
      const rows = q.state.data?.agents ?? [];
      const busy = rows.some((r) => r.labels.includes("updating") || r.labels.includes("restarting"));
      return busy ? 3000 : 15000;
    },
  });
  const data = agentsQuery.data;
  const rows = useMemo(() => data?.agents ?? [], [data]);

  function setParam(key: string, value: string) {
    const next = new URLSearchParams(params.toString());
    if (value && !(key === "filter" && value === "all")) next.set(key, value);
    else next.delete(key);
    const qs = next.toString();
    router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false });
  }

  const counts = useMemo(() => {
    const out = {} as Record<AgentFilter, number>;
    for (const f of AGENT_FILTERS) out[f] = rows.filter((r) => matchesFilter(r, f)).length;
    return out;
  }, [rows]);

  const visible = useMemo(
    () => rows.filter((r) => matchesFilter(r, filter) && matchesQuery(r, query)),
    [rows, filter, query]
  );
  const outdated = useMemo(
    () => rows.filter((r) => r.outdated && !r.labels.includes("updating")),
    [rows]
  );

  const invalidate = () => void qc.invalidateQueries({ queryKey: queryKeys.agents });

  const autoAllMutation = useMutation({
    mutationFn: (enabled: boolean) => setAgentsAutoEnabled(enabled),
    onSuccess: (_r, enabled) => {
      toast.success(enabled ? t("admin.agents.auto.enabled_all") : t("admin.agents.auto.disabled_all"));
      invalidate();
    },
    onError: (e) => reportError(e, t("common.error")),
  });

  const updateMutation = useMutation({
    mutationFn: (body: { node_ids?: string[]; outdated?: boolean }) => updateAgents(body),
    onSuccess: (res) => {
      const failed = res.results.filter((r) => !r.ok);
      if (res.started > 0) toast.success(t("admin.agents.update.started", { count: res.started }));
      for (const f of failed.slice(0, 3)) {
        const row = rows.find((r) => r.id === f.id);
        toast.error(`${row?.name ?? f.id}: ${f.error ?? t("common.error")}`);
      }
      setBulkOpen(false);
      setSelected(new Set());
      invalidate();
    },
    onError: (e) => reportError(e, t("common.error")),
  });

  const autoMutation = useMutation({
    mutationFn: ({ ids, value }: { ids: string[]; value: boolean }) => setAgentsAutoUpdate(ids, value),
    onSuccess: (_r, vars) => {
      toast.success(vars.value ? t("admin.agents.auto.enabled_some") : t("admin.agents.auto.disabled_some"));
      invalidate();
    },
    onError: (e) => reportError(e, t("common.error")),
  });

  function toggleSelect(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function toggleAll(ids: string[]) {
    setSelected((prev) => {
      const all = ids.every((id) => prev.has(id));
      return all ? new Set() : new Set(ids);
    });
  }

  const summary = data?.summary;
  const selectedIds = [...selected].filter((id) => rows.some((r) => r.id === id));

  return (
    <PageShell variant="admin">
      <div className="font-panel flex flex-col gap-[18px]">
        <div className={cn("rounded-[16px]", VX_CARD)}>
          <div className="flex flex-wrap items-start justify-between gap-5 px-[22px] py-5">
            <div className="min-w-[240px]">
              <div className={cn("font-mono text-[10.5px] font-medium tracking-[0.1em] uppercase", VX_FAINT)}>
                {t("admin.agents.eyebrow")}
              </div>
              <h1 className="m-0 mt-2 text-[25px] font-medium tracking-[-0.015em]">{t("admin.agents.title")}</h1>
              <div className={cn("mt-[7px] text-[12.5px]", VX_MUTED)}>
                {data
                  ? t("admin.agents.subtitle", { version: data.target_version })
                  : t("admin.agents.subtitle_loading")}
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <label className="inline-flex items-center gap-2.5 text-[12.5px]">
                <Toggle
                  checked={Boolean(data?.auto_enabled)}
                  disabled={!data || autoAllMutation.isPending}
                  onChange={(v) => autoAllMutation.mutate(v)}
                  label={t("admin.agents.auto.global")}
                />
                <span className={VX_MUTED}>{t("admin.agents.auto.global")}</span>
              </label>
              <Btn
                tone="primary"
                disabled={outdated.length === 0 || updateMutation.isPending}
                onClick={() => setBulkOpen(true)}
              >
                <RefreshCw className="h-3.5 w-3.5" />
                {t("admin.agents.update.outdated_button", { count: outdated.length })}
              </Btn>
            </div>
          </div>
        </div>

        {agentsQuery.isLoading ? (
          <Skeleton className="h-[86px] w-full rounded-[14px]" />
        ) : summary ? (
          <div className="grid grid-cols-2 gap-px overflow-hidden rounded-[14px] border border-[var(--vx-border)] bg-[var(--vx-border)] sm:grid-cols-3 lg:grid-cols-6">
            <SummaryCell label={t("admin.agents.summary.total")} value={summary.total} active={filter === "all"} onClick={() => setParam("filter", "all")} />
            <SummaryCell label={t("admin.agents.summary.online")} value={summary.online} active={filter === "online"} onClick={() => setParam("filter", "online")} />
            <SummaryCell label={t("admin.agents.summary.offline")} value={summary.offline} tone="danger" active={filter === "offline"} onClick={() => setParam("filter", "offline")} />
            <SummaryCell label={t("admin.agents.summary.outdated")} value={summary.outdated} tone="warn" active={filter === "outdated"} onClick={() => setParam("filter", "outdated")} />
            <SummaryCell
              label={t("admin.agents.summary.updating")}
              value={summary.updating + summary.update_failed}
              tone={summary.update_failed > 0 ? "danger" : undefined}
              active={filter === "updating" || filter === "update_failed"}
              onClick={() => setParam("filter", summary.update_failed > 0 ? "update_failed" : "updating")}
            />
            <SummaryCell label={t("admin.agents.summary.disk_low")} value={summary.disk_low} tone="danger" active={filter === "disk_low"} onClick={() => setParam("filter", "disk_low")} />
          </div>
        ) : null}

        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-1 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
            {AGENT_FILTERS.map((f) => {
              const on = f === filter;
              return (
                <button
                  key={f}
                  type="button"
                  onClick={() => setParam("filter", f)}
                  className={cn(
                    "inline-flex h-[30px] shrink-0 items-center gap-1.5 rounded-[8px] border px-3 text-[12.5px] transition-colors",
                    on
                      ? "border-[var(--vx-border-strong)] bg-[var(--vx-tint)] font-medium text-[var(--vx-fg-strong)]"
                      : "border-transparent text-[var(--vx-muted)] hover:text-[var(--vx-fg)]"
                  )}
                >
                  {t(`admin.agents.filter.${f}`)}
                  <span className={cn("font-mono text-[11px]", VX_FAINT)}>{counts[f] ?? 0}</span>
                </button>
              );
            })}
          </div>
          <label className="relative w-full sm:w-[260px]">
            <Search className={cn("pointer-events-none absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2", VX_FAINT)} />
            <input
              className={cn(VX_INPUT, "pl-8")}
              placeholder={t("admin.agents.search")}
              value={query}
              onChange={(e) => {
                setQuery(e.target.value);
                setParam("q", e.target.value);
              }}
            />
          </label>
        </div>

        {selectedIds.length > 0 && (
          <div className={cn("flex flex-wrap items-center justify-between gap-3 rounded-[12px] px-4 py-2.5", VX_CARD)}>
            <span className="text-[12.5px]">{t("admin.agents.bulk.selected", { count: selectedIds.length })}</span>
            <div className="flex flex-wrap gap-2">
              <Btn size="sm" disabled={updateMutation.isPending} onClick={() => updateMutation.mutate({ node_ids: selectedIds })}>
                {t("admin.agents.bulk.update")}
              </Btn>
              <Btn size="sm" disabled={autoMutation.isPending} onClick={() => autoMutation.mutate({ ids: selectedIds, value: true })}>
                {t("admin.agents.bulk.auto_on")}
              </Btn>
              <Btn size="sm" disabled={autoMutation.isPending} onClick={() => autoMutation.mutate({ ids: selectedIds, value: false })}>
                {t("admin.agents.bulk.auto_off")}
              </Btn>
              <Btn size="sm" tone="ghost" onClick={() => setSelected(new Set())}>
                {t("common.cancel")}
              </Btn>
            </div>
          </div>
        )}

        {agentsQuery.isError ? (
          <Notice>
            <div className="flex flex-wrap items-center justify-between gap-3">
              <span>{t("admin.agents.load_error")}</span>
              <Btn size="sm" onClick={() => agentsQuery.refetch()}>
                {t("common.retry")}
              </Btn>
            </div>
          </Notice>
        ) : agentsQuery.isLoading ? (
          <Skeleton className="h-[360px] w-full rounded-[14px]" />
        ) : rows.length === 0 ? (
          <div className={cn("rounded-[14px] px-6 py-10 text-center", VX_CARD)}>
            <div className="text-[14px] font-medium">{t("admin.agents.empty.title")}</div>
            <div className={cn("mt-2 text-[12.5px]", VX_MUTED)}>{t("admin.agents.empty.body")}</div>
            <Link href="/admin/locations" className="mt-4 inline-block text-[12.5px] text-primary hover:underline">
              {t("admin.agents.empty.link")}
            </Link>
          </div>
        ) : visible.length === 0 ? (
          <div className={cn("rounded-[14px]", VX_CARD)}>
            <EmptyState>{t("admin.agents.empty.filtered")}</EmptyState>
          </div>
        ) : (
          <AgentsTable
            rows={visible}
            selected={selected}
            onToggle={toggleSelect}
            onToggleAll={() => toggleAll(visible.map((r) => r.id))}
            onUpdate={(row: AgentRow) => updateMutation.mutate({ node_ids: [row.id] })}
            onAuto={(row: AgentRow, value: boolean) => autoMutation.mutate({ ids: [row.id], value })}
            pending={updateMutation.isPending || autoMutation.isPending}
          />
        )}
      </div>

      <ConfirmDialog
        open={bulkOpen}
        onOpenChange={setBulkOpen}
        tone="primary"
        title={t("admin.agents.update.confirm_title", { count: outdated.length })}
        description={
          <div className="grid gap-2">
            <span>{t("admin.agents.update.confirm_body", { version: data?.target_version ?? "" })}</span>
            <ul className="max-h-[200px] overflow-auto rounded-[10px] border border-[var(--vx-border)] px-3 py-2 font-mono text-[12px]">
              {outdated.map((r) => (
                <li key={r.id} className="flex justify-between gap-3 py-0.5">
                  <span className="truncate">{r.name}</span>
                  <span className={VX_FAINT}>{r.version || "—"}</span>
                </li>
              ))}
            </ul>
          </div>
        }
        confirmLabel={t("admin.agents.update.confirm_button")}
        pending={updateMutation.isPending}
        onConfirm={() => updateMutation.mutate({ outdated: true })}
      />
    </PageShell>
  );
}
