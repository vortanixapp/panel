"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Btn,
  EmptyState,
  Field,
  Notice,
  Panel,
  Toggle,
  VX_FAINT,
  VX_INPUT,
  VX_INPUT_MONO,
  VX_MUTED,
  VX_ROW_LINE,
  VX_SELECT,
  VX_TBL_TD,
  VX_TBL_TH,
  VX_TBL_WRAP,
} from "@/components/vx/panel-ui";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import {
  cancelWipeRun,
  createWipePlan,
  deleteWipePlan,
  fetchServerWipes,
  runServerWipe,
  updateWipePlan,
  type WipeInfo,
  type WipePlan,
  type WipePlanInput,
  type WipeRun,
} from "@/lib/api";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { confirmAction } from "@/components/action-dialog";

const SCHEDULE_TYPES = ["weekly", "monthly", "cron", "once"] as const;
const COUNTDOWNS = [0, 5, 15, 30, 60];

type PlanForm = {
  id: string | null;
  name: string;
  kind: string;
  schedule_type: (typeof SCHEDULE_TYPES)[number];
  weekday: number;
  nth: number;
  time: string;
  cron: string;
  run_at: string;
  timezone: string;
  backup_before: boolean;
  new_seed: boolean;
  notify_owner: boolean;
  announce_minutes: string;
  announce_text: string;
};

function weekdayName(day: number) {
  return new Date(2024, 0, 7 + day).toLocaleDateString(localeTag(), { weekday: "long" });
}

function parseMinutes(raw: string): number[] {
  return raw
    .split(/[\s,;]+/)
    .map((v) => Number.parseInt(v, 10))
    .filter((v) => Number.isFinite(v) && v > 0);
}

export function ServerWipesTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const [planForm, setPlanForm] = useState<PlanForm | null>(null);
  const [runOpen, setRunOpen] = useState(false);
  const [runKind, setRunKind] = useState("");
  const [runCountdown, setRunCountdown] = useState(5);
  const [runBackup, setRunBackup] = useState(true);
  const [runSeed, setRunSeed] = useState(true);

  const { data, isLoading } = useQuery({
    queryKey: ["server-wipes", id],
    queryFn: () => fetchServerWipes(id),
    enabled: !!id,
    refetchInterval: (query) => (query.state.data?.active_run ? 5_000 : 30_000),
  });

  const refresh = () => void queryClient.invalidateQueries({ queryKey: ["server-wipes", id] });
  const fail = (err: unknown) => toast.error(err instanceof Error ? err.message : t("servers.wipes.error"));

  const saveMutation = useMutation({
    mutationFn: async (form: PlanForm) => {
      const body: WipePlanInput = {
        name: form.name,
        kind: form.kind,
        schedule_type: form.schedule_type,
        weekday: form.weekday,
        nth: form.nth,
        time: form.time,
        cron: form.cron,
        run_at: form.schedule_type === "once" ? form.run_at : "",
        timezone: form.timezone,
        backup_before: form.backup_before,
        new_seed: form.new_seed,
        notify_owner: form.notify_owner,
        announce_minutes: parseMinutes(form.announce_minutes),
        announce_text: form.announce_text,
      };
      if (form.id) {
        await updateWipePlan(id, form.id, body);
        return;
      }
      await createWipePlan(id, { ...body, enabled: true });
    },
    onSuccess: () => {
      toast.success(t("servers.wipes.saved"));
      setPlanForm(null);
      refresh();
    },
    onError: fail,
  });

  const patchMutation = useMutation({
    mutationFn: ({ planId, body }: { planId: string; body: WipePlanInput }) => updateWipePlan(id, planId, body),
    onSuccess: refresh,
    onError: fail,
  });

  const deleteMutation = useMutation({
    mutationFn: (planId: string) => deleteWipePlan(id, planId),
    onSuccess: () => {
      toast.success(t("servers.wipes.deleted"));
      refresh();
    },
    onError: fail,
  });

  const runMutation = useMutation({
    mutationFn: () =>
      runServerWipe(id, {
        kind: runKind,
        backup_before: runBackup,
        new_seed: runSeed,
        countdown_minutes: runCountdown,
      }),
    onSuccess: () => {
      toast.success(t("servers.wipes.run_started"));
      setRunOpen(false);
      refresh();
    },
    onError: fail,
  });

  const cancelMutation = useMutation({
    mutationFn: (runId: string) => cancelWipeRun(id, runId),
    onSuccess: () => {
      toast.success(t("servers.wipes.run_cancelled"));
      refresh();
    },
    onError: fail,
  });

  if (isLoading || !data) return <Skeleton className="h-[320px] w-full rounded-[14px]" />;
  if (!data.supported) {
    return (
      <Panel title={t("servers.wipes.title")}>
        <EmptyState>{t("servers.wipes.unsupported")}</EmptyState>
      </Panel>
    );
  }

  const info: WipeInfo = data;
  const kinds = info.kinds ?? [];
  const timeZone = info.timezone || "UTC";
  const plans = info.plans ?? [];
  const runs = info.runs ?? [];
  const active = info.active_run ?? null;
  const kindLabel = (kind: string) => t(`servers.wipes.kind.${kind}`);

  const formatAt = (iso: string | null | undefined, tz = timeZone) =>
    iso
      ? new Date(iso).toLocaleString(localeTag(), {
          timeZone: tz,
          day: "2-digit",
          month: "2-digit",
          hour: "2-digit",
          minute: "2-digit",
        })
      : "—";

  const scheduleText = (p: WipePlan) => {
    switch (p.schedule_type) {
      case "weekly":
        return t("servers.wipes.sched_weekly", { day: weekdayName(p.weekday), time: p.time });
      case "monthly":
        return t("servers.wipes.sched_monthly", {
          nth: p.nth >= 5 ? t("servers.wipes.nth_last") : t("servers.wipes.nth_n", { n: p.nth }),
          day: weekdayName(p.weekday),
          time: p.time,
        });
      case "cron":
        return `cron ${p.cron}`;
      default:
        return t("servers.wipes.sched_once", { time: formatAt(p.run_at, p.timezone) });
    }
  };

  const openCreate = () =>
    setPlanForm({
      id: null,
      name: "",
      kind: kinds[0] ?? "",
      schedule_type: "weekly",
      weekday: 4,
      nth: 1,
      time: "19:00",
      cron: "0 19 * * 4",
      run_at: "",
      timezone: timeZone,
      backup_before: true,
      new_seed: !!info.seed,
      notify_owner: true,
      announce_minutes: (info.default_announce_minutes ?? [60, 30, 10, 5, 1]).join(", "),
      announce_text: "",
    });

  const openEdit = (p: WipePlan) =>
    setPlanForm({
      id: p.id,
      name: p.name,
      kind: p.kind,
      schedule_type: p.schedule_type,
      weekday: p.weekday,
      nth: p.nth,
      time: p.time,
      cron: p.cron || "0 19 * * 4",
      run_at: p.run_at_local ?? "",
      timezone: p.timezone,
      backup_before: p.backup_before,
      new_seed: p.new_seed,
      notify_owner: p.notify_owner,
      announce_minutes: p.announce_minutes.join(", "),
      announce_text: p.announce_text,
    });

  const openRun = () => {
    setRunKind(kinds[0] ?? "");
    setRunCountdown(5);
    setRunBackup(true);
    setRunSeed(!!info.seed);
    setRunOpen(true);
  };

  const set = <K extends keyof PlanForm>(key: K, value: PlanForm[K]) =>
    setPlanForm((prev) => (prev ? { ...prev, [key]: value } : prev));

  const statusText = (status: string) => t(`servers.wipes.status.${status}`);

  return (
    <div className="flex flex-col gap-4">
      {active && (
        <Notice tone="warn">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <span>
              {t("servers.wipes.active", {
                kind: kindLabel(active.kind),
                status: statusText(active.status),
                time: formatAt(active.starts_at),
              })}
            </span>
            {active.status === "announcing" && (
              <Btn size="sm" tone="danger" disabled={cancelMutation.isPending} onClick={() => cancelMutation.mutate(active.id)}>
                {t("servers.wipes.cancel_run")}
              </Btn>
            )}
          </div>
        </Notice>
      )}

      <Panel
        title={t("servers.wipes.plans_title")}
        aside={
          <div className="flex flex-wrap gap-2">
            <Btn size="sm" tone="danger" onClick={openRun} disabled={!!active}>
              {t("servers.wipes.run_now")}
            </Btn>
            <Btn size="sm" tone="primary" onClick={openCreate} disabled={plans.length >= (info.max_plans ?? 10)}>
              {t("common.add")}
            </Btn>
          </div>
        }
        flush
      >
        <p className={cn("px-[18px] pt-3 text-[11.5px] leading-[1.6]", VX_FAINT)}>
          {t("servers.wipes.hint", { tz: timeZone })}
        </p>
        {plans.length === 0 ? (
          <EmptyState>{t("servers.wipes.plans_empty")}</EmptyState>
        ) : (
          <div className="px-[18px] pt-2 pb-2">
            {plans.map((p) => (
              <div key={p.id} className={cn("flex flex-wrap items-center gap-x-4 gap-y-2 py-3 last:border-b-0", VX_ROW_LINE)}>
                <div className="min-w-[220px] flex-1">
                  <div className="text-[13px] font-medium">{p.name || kindLabel(p.kind)}</div>
                  <div className={cn("text-[11.5px]", VX_MUTED)}>
                    {kindLabel(p.kind)} · {scheduleText(p)}
                  </div>
                  <div className={cn("mt-0.5 text-[11px]", VX_FAINT)}>
                    {p.enabled
                      ? t("servers.wipes.next", { time: formatAt(p.next_run_at, p.timezone) }) +
                        (p.skip_next ? ` · ${t("servers.wipes.skip_marked")}` : "")
                      : t("servers.wipes.disabled")}
                    {p.last_status ? ` · ${t("servers.wipes.last", { status: t(`servers.wipes.last_${p.last_status}`) })}` : ""}
                  </div>
                </div>
                <Toggle
                  label={t("servers.wipes.enabled")}
                  checked={p.enabled}
                  disabled={patchMutation.isPending}
                  onChange={(enabled) => patchMutation.mutate({ planId: p.id, body: { enabled } })}
                />
                <div className="flex flex-wrap gap-2">
                  {p.enabled && (
                    <Btn
                      size="sm"
                      disabled={patchMutation.isPending}
                      onClick={() => patchMutation.mutate({ planId: p.id, body: { skip_next: !p.skip_next } })}
                    >
                      {p.skip_next ? t("servers.wipes.unskip") : t("servers.wipes.skip")}
                    </Btn>
                  )}
                  <Btn size="sm" onClick={() => openEdit(p)}>
                    {t("common.edit")}
                  </Btn>
                  <Btn
                    size="sm"
                    tone="danger"
                    disabled={deleteMutation.isPending}
                    onClick={async () => {
                      if (!(await confirmAction(t("servers.wipes.delete_confirm")))) return;
                      deleteMutation.mutate(p.id);
                    }}
                  >
                    {t("common.delete")}
                  </Btn>
                </div>
              </div>
            ))}
          </div>
        )}
      </Panel>

      <Panel title={t("servers.wipes.history_title")} flush>
        <div className={VX_TBL_WRAP}>
          <table className="vx-tbl vx-tbl-flat w-full table-fixed border-collapse text-left">
            <thead>
              <tr>
                <th className={cn(VX_TBL_TH, "w-[130px]")}>{t("servers.wipes.col_time")}</th>
                <th className={VX_TBL_TH}>{t("servers.wipes.col_kind")}</th>
                <th className={cn(VX_TBL_TH, "w-[120px]")}>{t("servers.wipes.col_status")}</th>
                <th className={VX_TBL_TH}>{t("servers.wipes.col_details")}</th>
              </tr>
            </thead>
            <tbody>
              {runs.length === 0 ? (
                <tr>
                  <td colSpan={4}>
                    <EmptyState>{t("servers.wipes.history_empty")}</EmptyState>
                  </td>
                </tr>
              ) : (
                runs.map((r: WipeRun) => (
                  <tr key={r.id} className={cn("text-[12.5px] last:border-b-0", VX_ROW_LINE)}>
                    <td className={cn(VX_TBL_TD, "font-mono text-[12px]")} data-cell="full">
                      {formatAt(r.starts_at)}
                    </td>
                    <td className={VX_TBL_TD} data-label={t("servers.wipes.col_kind")}>
                      {kindLabel(r.kind)} · {t(`servers.wipes.source_${r.source}`)}
                    </td>
                    <td
                      className={cn(
                        VX_TBL_TD,
                        r.status === "failed" && "text-[var(--vx-danger)]",
                        r.status === "completed" && "text-[var(--vx-ok)]"
                      )}
                      data-label={t("servers.wipes.col_status")}
                    >
                      {statusText(r.status)}
                    </td>
                    <td className={cn(VX_TBL_TD, VX_MUTED, "text-[11.5px] break-words")} data-cell="block" data-label={t("servers.wipes.col_details")}>
                      {r.error
                        ? r.error
                        : [
                            r.details?.deleted != null ? t("servers.wipes.detail_deleted", { n: r.details.deleted }) : "",
                            r.details?.seed && !String(r.details.seed).startsWith("skipped")
                              ? t("servers.wipes.detail_seed", { seed: r.details.seed })
                              : "",
                            r.details?.backup && !String(r.details.backup).startsWith("skipped")
                              ? t("servers.wipes.detail_backup", { file: r.details.backup })
                              : "",
                          ]
                            .filter(Boolean)
                            .join(" · ") || "—"}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </Panel>

      <Dialog open={planForm !== null} onOpenChange={(open) => !open && setPlanForm(null)}>
        <DialogContent className="max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{planForm?.id ? t("servers.wipes.edit_title") : t("servers.wipes.new_title")}</DialogTitle>
          </DialogHeader>
          {planForm && (
            <div className="flex flex-col gap-3">
              <Field label={t("servers.wipes.f_name")}>
                <input className={VX_INPUT} value={planForm.name} maxLength={80} onChange={(e) => set("name", e.target.value)} />
              </Field>
              <Field label={t("servers.wipes.f_kind")}>
                <select className={VX_SELECT} value={planForm.kind} onChange={(e) => set("kind", e.target.value)}>
                  {kinds.map((k) => (
                    <option key={k} value={k}>
                      {kindLabel(k)}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label={t("servers.wipes.f_schedule")}>
                <select
                  className={VX_SELECT}
                  value={planForm.schedule_type}
                  onChange={(e) => set("schedule_type", e.target.value as PlanForm["schedule_type"])}
                >
                  {SCHEDULE_TYPES.map((s) => (
                    <option key={s} value={s}>
                      {t(`servers.wipes.type_${s}`)}
                    </option>
                  ))}
                </select>
              </Field>
              {(planForm.schedule_type === "weekly" || planForm.schedule_type === "monthly") && (
                <div className="grid grid-cols-2 gap-3">
                  {planForm.schedule_type === "monthly" && (
                    <Field label={t("servers.wipes.f_nth")}>
                      <select className={VX_SELECT} value={planForm.nth} onChange={(e) => set("nth", Number(e.target.value))}>
                        {[1, 2, 3, 4, 5].map((n) => (
                          <option key={n} value={n}>
                            {n >= 5 ? t("servers.wipes.nth_last") : t("servers.wipes.nth_n", { n })}
                          </option>
                        ))}
                      </select>
                    </Field>
                  )}
                  <Field label={t("servers.wipes.f_weekday")}>
                    <select className={VX_SELECT} value={planForm.weekday} onChange={(e) => set("weekday", Number(e.target.value))}>
                      {[1, 2, 3, 4, 5, 6, 0].map((d) => (
                        <option key={d} value={d}>
                          {weekdayName(d)}
                        </option>
                      ))}
                    </select>
                  </Field>
                  <Field label={t("servers.wipes.f_time")}>
                    <input className={VX_INPUT_MONO} type="time" value={planForm.time} onChange={(e) => set("time", e.target.value)} />
                  </Field>
                </div>
              )}
              {planForm.schedule_type === "cron" && (
                <Field label={t("servers.wipes.f_cron")}>
                  <input className={VX_INPUT_MONO} value={planForm.cron} onChange={(e) => set("cron", e.target.value)} placeholder="0 19 * * 4" />
                </Field>
              )}
              {planForm.schedule_type === "once" && (
                <Field label={t("servers.wipes.f_run_at")}>
                  <input className={VX_INPUT_MONO} type="datetime-local" value={planForm.run_at} onChange={(e) => set("run_at", e.target.value)} />
                </Field>
              )}
              <Field label={t("servers.wipes.f_timezone")}>
                <input className={VX_INPUT_MONO} value={planForm.timezone} onChange={(e) => set("timezone", e.target.value)} placeholder="Europe/Moscow" />
              </Field>
              <Field label={t("servers.wipes.f_announce_minutes")}>
                <input className={VX_INPUT_MONO} value={planForm.announce_minutes} onChange={(e) => set("announce_minutes", e.target.value)} placeholder="60, 30, 10, 5, 1" />
              </Field>
              {info.announce && (
                <Field label={t("servers.wipes.f_announce_text")}>
                  <input
                    className={VX_INPUT}
                    value={planForm.announce_text}
                    maxLength={200}
                    onChange={(e) => set("announce_text", e.target.value)}
                    placeholder={t("servers.wipes.f_announce_placeholder")}
                  />
                </Field>
              )}
              <ToggleRow label={t("servers.wipes.f_backup")} checked={planForm.backup_before} onChange={(v) => set("backup_before", v)} />
              {info.seed && <ToggleRow label={t("servers.wipes.f_seed")} checked={planForm.new_seed} onChange={(v) => set("new_seed", v)} />}
              <ToggleRow label={t("servers.wipes.f_notify")} checked={planForm.notify_owner} onChange={(v) => set("notify_owner", v)} />
            </div>
          )}
          <DialogFooter>
            <Btn onClick={() => setPlanForm(null)}>{t("common.cancel")}</Btn>
            <Btn tone="primary" disabled={saveMutation.isPending || !planForm?.kind} onClick={() => planForm && saveMutation.mutate(planForm)}>
              {saveMutation.isPending ? t("common.saving") : t("common.save")}
            </Btn>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={runOpen} onOpenChange={setRunOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("servers.wipes.run_title")}</DialogTitle>
          </DialogHeader>
          <div className="flex flex-col gap-3">
            <Notice tone="warn">{t("servers.wipes.run_warning")}</Notice>
            <Field label={t("servers.wipes.f_kind")}>
              <select className={VX_SELECT} value={runKind} onChange={(e) => setRunKind(e.target.value)}>
                {kinds.map((k) => (
                  <option key={k} value={k}>
                    {kindLabel(k)}
                  </option>
                ))}
              </select>
            </Field>
            <Field label={t("servers.wipes.f_countdown")}>
              <select className={VX_SELECT} value={runCountdown} onChange={(e) => setRunCountdown(Number(e.target.value))}>
                {COUNTDOWNS.map((m) => (
                  <option key={m} value={m}>
                    {m === 0 ? t("servers.wipes.immediately") : t("servers.wipes.in_minutes", { n: m })}
                  </option>
                ))}
              </select>
            </Field>
            <ToggleRow label={t("servers.wipes.f_backup")} checked={runBackup} onChange={setRunBackup} />
            {info.seed && <ToggleRow label={t("servers.wipes.f_seed")} checked={runSeed} onChange={setRunSeed} />}
          </div>
          <DialogFooter>
            <Btn onClick={() => setRunOpen(false)}>{t("common.cancel")}</Btn>
            <Btn tone="danger" disabled={runMutation.isPending || !runKind} onClick={() => runMutation.mutate()}>
              {runMutation.isPending ? t("common.saving") : t("servers.wipes.run_confirm")}
            </Btn>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function ToggleRow({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <div className="flex items-center justify-between gap-3 text-[12.5px]">
      <span>{label}</span>
      <Toggle label={label} checked={checked} onChange={onChange} />
    </div>
  );
}
