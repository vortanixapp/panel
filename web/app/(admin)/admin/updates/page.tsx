"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PageShell } from "@/components/layout/page-shell";
import {
  applyAdminUpdates,
  fetchAdminUpdates,
  setAdminUpdatesAuto,
  updateDaemon,
  type AdminAgentUpdateEvent,
  type AdminUpdateEvent,
  type AdminUpdateRelease,
  type AdminUpdatesDaemon,
  type AdminUpdatesInfo,
  type UpdateComponent,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

// Три обновляемых части системы. Панель обновляет служба обновления, агентов —
// воркер по SSH, а саму службу — она сама. Пути разные, поэтому и вкладки
// разные: раньше всё это было свалено на один экран, и по журналу нельзя было
// понять, что именно обновлялось.
// Подписи храним ключами: список читается на уровне модуля, и готовый текст
// застыл бы на языке, который стоял в момент загрузки страницы.
const TABS: { key: UpdateComponent; labelKey: string; hintKey: string }[] = [
  {
    key: "panel-ui",
    labelKey: "admin.updates.tab_panel",
    hintKey: "admin.updates.tab_panel_hint",
  },
  {
    key: "agent",
    labelKey: "admin.updates.tab_agent",
    hintKey: "admin.updates.tab_agent_hint",
  },
  {
    key: "updater",
    labelKey: "admin.updates.tab_updater",
    hintKey: "admin.updates.tab_updater_hint",
  },
];

function formatDate(raw?: string | null): string {
  if (!raw) return "—";
  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) return "—";
  return date.toLocaleString(localeTag(), { dateStyle: "medium", timeStyle: "short" });
}

const STAGE_LABEL_KEYS: Record<string, string> = {
  started: "admin.updates.stage_started",
  pulling: "admin.updates.stage_pulling",
  healthy: "admin.updates.stage_healthy",
  done: "admin.updates.stage_done",
  failed: "admin.updates.stage_failed",
};

const JOB_STATUS_LABEL_KEYS: Record<string, string> = {
  pending: "admin.updates.job_pending",
  running: "admin.updates.job_running",
  completed: "admin.updates.job_completed",
  failed: "admin.updates.job_failed",
};

export default function AdminUpdatesPage() {
  const t = useT();
  const qc = useQueryClient();
  const [tab, setTab] = useState<UpdateComponent>("panel-ui");
  const [message, setMessage] = useState("");

  const { data, isLoading, isError } = useQuery({
    queryKey: ["admin-updates", tab],
    queryFn: () => fetchAdminUpdates(tab),
    refetchInterval: (query) => {
      const info = query.state.data as AdminUpdatesInfo | undefined;
      // Пока обновление доступно или идёт установка, этапы сменяют друг друга
      // за секунды — раз в минуту экран показывал бы уже случившееся.
      const stage = info?.events?.[0]?.stage;
      const running = stage != null && stage !== "done" && stage !== "failed";
      return info?.update_available || running ? 3_000 : 20_000;
    },
  });

  const check = useMutation({
    mutationFn: () => fetchAdminUpdates(tab, true),
    onSuccess: (fresh) => {
      qc.setQueryData(["admin-updates", tab], fresh);
      setMessage(t("admin.updates.checked_just_now"));
    },
  });

  const invalidate = () => qc.invalidateQueries({ queryKey: ["admin-updates"] });

  const apply = useMutation({
    mutationFn: () => applyAdminUpdates(tab),
    onSuccess: () => {
      setMessage(t("admin.updates.started"));
      invalidate();
    },
    onError: (e: Error) => setMessage(e.message || t("admin.updates.start_failed")),
  });

  const setAuto = useMutation({
    mutationFn: (enabled: boolean) => setAdminUpdatesAuto(enabled),
    onSuccess: (res) => {
      setMessage(
        res.auto_update ? t("admin.updates.auto_on") : t("admin.updates.auto_off")
      );
      invalidate();
    },
    onError: (e: Error) =>
      setMessage(e.message || t("admin.updates.auto_change_failed")),
  });

  const updateNode = useMutation({
    mutationFn: (nodeId: string) => updateDaemon(nodeId),
    onSuccess: () => {
      setMessage(t("admin.updates.agent_queued"));
      invalidate();
    },
    onError: (e: Error) => setMessage(e.message || t("admin.updates.agent_failed")),
  });

  const info = data;
  // Параметр назван item, а не t: имя t уже занято функцией перевода.
  const active = TABS.find((item) => item.key === tab)!;

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("admin.updates.title")}
          </h1>
          <p className="text-sm text-muted-foreground">{t(active.hintKey)}</p>
        </div>
        <button
          onClick={() => check.mutate()}
          disabled={check.isPending}
          className="rounded-full border px-4 py-2 text-sm font-medium transition-colors hover:bg-muted/50 disabled:opacity-50"
        >
          {check.isPending ? t("admin.updates.checking") : t("layout.check_now")}
        </button>
      </div>

      <div className="mb-5 flex flex-wrap gap-1 border-b">
        {TABS.map((item) => (
          <button
            key={item.key}
            onClick={() => {
              setTab(item.key);
              setMessage("");
            }}
            className={`-mb-px border-b-2 px-4 py-2.5 text-sm font-medium transition-colors ${
              tab === item.key
                ? "border-foreground text-foreground"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            {t(item.labelKey)}
          </button>
        ))}
      </div>

      {message && (
        <div className="mb-4 rounded-lg border bg-muted/30 px-4 py-2.5 text-sm">{message}</div>
      )}

      {isError ? (
        <div className="rounded-lg border bg-card p-6 text-sm text-muted-foreground">
          {t("admin.updates.load_failed")}
        </div>
      ) : isLoading || !info ? (
        <div className="rounded-lg border bg-card p-6 text-sm text-muted-foreground">
          {t("common.loading")}
        </div>
      ) : (
        <div className="space-y-5">
          {tab === "agent" ? (
            <AgentsPanel
              daemons={info.daemons ?? []}
              target={info.target_version}
              onUpdate={(id) => updateNode.mutate(id)}
              busy={updateNode.isPending}
            />
          ) : (
            <StatePanel
              info={info}
              onApply={() => apply.mutate()}
              applying={apply.isPending}
              onAuto={(enabled) => setAuto.mutate(enabled)}
              autoPending={setAuto.isPending}
            />
          )}

          <ReleasesPanel releases={info.releases ?? []} />

          {tab === "agent" ? (
            <AgentLogPanel events={info.agent_events ?? []} />
          ) : (
            <LogPanel events={info.events ?? []} />
          )}
        </div>
      )}
    </PageShell>
  );
}

function StatePanel({
  info,
  onApply,
  applying,
  onAuto,
  autoPending,
}: {
  info: AdminUpdatesInfo;
  onApply: () => void;
  applying: boolean;
  onAuto: (enabled: boolean) => void;
  autoPending: boolean;
}) {
  const t = useT();
  const auto = Boolean(info.auto_update);
  return (
    <div className="rounded-lg border bg-card p-5">
      <div className="flex flex-wrap gap-x-10 gap-y-4">
        <Metric
          label={t("admin.updates.current_version")}
          value={info.current_version || "—"}
        />
        <Metric
          label={t("admin.updates.available")}
          value={info.target_version || "—"}
          accent={info.update_available}
        />
        <Metric
          label={t("admin.updates.verified")}
          value={formatDate(info.last_verified_at)}
          muted
        />
      </div>

      {!info.update_available && (
        <div className="mt-4 rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-4 py-2.5 text-sm text-emerald-600">
          {t("admin.updates.up_to_date")}
        </div>
      )}

      {info.update_available && (
        <div className="mt-5 flex flex-wrap gap-2 border-t pt-5">
          <button
            onClick={onApply}
            disabled={applying}
            className="rounded-full bg-foreground px-5 py-2 text-sm font-medium text-background transition-opacity hover:opacity-90 disabled:opacity-50"
          >
            {applying ? t("admin.updates.applying") : t("admin.updates.apply")}
          </button>
          <span className="self-center text-sm text-muted-foreground">
            {auto
              ? t("admin.updates.auto_on_note")
              : t("admin.updates.manual_note")}
          </span>
        </div>
      )}

      {/* Переключатель автоматической установки. По умолчанию выключен:
          раскатанная версия раньше приезжала сама, без спроса, и владелец не
          выбирал момент, когда у него сменится рабочая панель. */}
      <div className="mt-5 flex flex-wrap items-center justify-between gap-3 border-t pt-5">
        <div>
          <div className="text-sm font-medium">{t("admin.updates.auto_title")}</div>
          <div className="mt-0.5 text-sm text-muted-foreground">
            {auto
              ? t("admin.updates.auto_on_desc")
              : t("admin.updates.auto_off_desc")}
          </div>
        </div>
        <button
          type="button"
          role="switch"
          aria-checked={auto}
          disabled={autoPending}
          onClick={() => onAuto(!auto)}
          className={`relative h-6 w-11 shrink-0 rounded-full transition-colors disabled:opacity-50 ${
            auto ? "bg-foreground" : "bg-muted-foreground/30"
          }`}
        >
          <span
            className={`absolute top-0.5 h-5 w-5 rounded-full bg-background transition-transform ${
              auto ? "translate-x-[22px]" : "translate-x-0.5"
            }`}
          />
        </button>
      </div>
    </div>
  );
}

function Metric({
  label,
  value,
  accent,
  muted,
}: {
  label: string;
  value: string;
  accent?: boolean;
  muted?: boolean;
}) {
  return (
    <div>
      <div className="text-[11px] font-semibold tracking-wide text-muted-foreground uppercase">
        {label}
      </div>
      <div
        className={`mt-1 font-mono ${muted ? "text-sm text-muted-foreground" : "text-2xl font-bold"} ${
          accent ? "text-amber-600" : ""
        }`}
      >
        {value}
      </div>
    </div>
  );
}

function AgentsPanel({
  daemons,
  target,
  onUpdate,
  busy,
}: {
  daemons: AdminUpdatesDaemon[];
  target: string;
  onUpdate: (id: string) => void;
  busy: boolean;
}) {
  const t = useT();
  const outdated = daemons.filter((d) => d.version && target && d.version !== target).length;

  return (
    <div className="rounded-lg border bg-card">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b px-5 py-4">
        <div className="text-sm font-semibold">{t("layout.nodes")}</div>
        <div className="text-xs text-muted-foreground">
          {daemons.length === 0
            ? t("admin.updates.nodes_none")
            : outdated === 0
              ? t("admin.updates.nodes_all_on", { version: target || "—" })
              : t("admin.updates.nodes_outdated", {
                  count: outdated,
                  total: daemons.length,
                })}
        </div>
      </div>
      {daemons.length === 0 ? (
        <div className="p-6 text-center text-sm text-muted-foreground">
          {t("admin.updates.nodes_empty")}
        </div>
      ) : (
        daemons.map((d) => {
          const stale = !!d.version && !!target && d.version !== target;
          return (
            <div key={d.id} className="flex items-center gap-4 border-b px-5 py-3 last:border-0">
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2 text-sm font-medium">
                  {d.name}
                  <span className="font-mono text-xs text-muted-foreground">
                    {d.version || t("admin.updates.version_unknown")}
                  </span>
                  {stale && (
                    <span className="rounded-full bg-amber-500/10 px-2 py-0.5 text-[11px] text-amber-600">
                      {t("admin.updates.version_available", { version: target })}
                    </span>
                  )}
                </div>
                <div className="mt-0.5 text-xs text-muted-foreground">
                  {d.status === "online" ? t("admin.updates.online") : d.status} ·{" "}
                  {t("admin.updates.last_seen")} {formatDate(d.last_seen_at)}
                </div>
              </div>
              <button
                onClick={() => onUpdate(d.id)}
                disabled={busy || !stale}
                className="rounded-full border px-4 py-1.5 text-sm transition-colors hover:bg-muted/50 disabled:opacity-40"
              >
                {stale
                  ? t("admin.updates.update")
                  : t("admin.updates.node_up_to_date")}
              </button>
            </div>
          );
        })
      )}
    </div>
  );
}

function ReleasesPanel({ releases }: { releases: AdminUpdateRelease[] }) {
  const t = useT();
  return (
    <div className="rounded-lg border bg-card">
      <div className="border-b px-5 py-4 text-sm font-semibold">
        {t("admin.updates.releases_title")}
      </div>
      {releases.length === 0 ? (
        <div className="p-6 text-center text-sm text-muted-foreground">
          {t("admin.updates.releases_empty")}
        </div>
      ) : (
        releases.map((rel) => (
          <div key={rel.version} className="border-b px-5 py-4 last:border-0">
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-mono text-sm font-semibold">{rel.version}</span>
              {rel.is_current && (
                <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] text-emerald-600">
                  {t("admin.updates.installed")}
                </span>
              )}
              {rel.is_target && !rel.is_current && (
                <span className="rounded-full bg-amber-500/10 px-2 py-0.5 text-[11px] text-amber-600">
                  {t("admin.updates.available_badge")}
                </span>
              )}
              {rel.channel !== "stable" && (
                <span className="rounded-full border px-2 py-0.5 text-[11px] text-muted-foreground">
                  {rel.channel}
                </span>
              )}
              <span className="text-xs text-muted-foreground">{formatDate(rel.published_at)}</span>
            </div>
            {rel.notes ? (
              <p className="mt-1.5 text-[13px] leading-relaxed whitespace-pre-line">{rel.notes}</p>
            ) : (
              <p className="mt-1.5 text-[13px] text-muted-foreground">
                {t("admin.updates.notes_empty")}
              </p>
            )}
            {rel.mandatory_after && (
              <p className="mt-1 text-xs text-amber-600">
                {t("admin.updates.mandatory_after", {
                  date: formatDate(rel.mandatory_after),
                })}
              </p>
            )}
          </div>
        ))
      )}
    </div>
  );
}

function LogPanel({ events }: { events: AdminUpdateEvent[] }) {
  const t = useT();
  return (
    <div className="rounded-lg border bg-card">
      <div className="border-b px-5 py-4 text-sm font-semibold">
        {t("admin.updates.log_title")}
      </div>
      {events.length === 0 ? (
        <div className="p-6 text-center text-sm text-muted-foreground">
          {t("admin.updates.log_empty")}
        </div>
      ) : (
        <div className="space-y-3 px-5 py-4">
          {events.map((e, i) => (
            <div key={i} className="flex gap-3">
              <span
                className={`mt-1.5 size-2 shrink-0 rounded-full ${
                  e.stage === "done"
                    ? "bg-emerald-500"
                    : e.stage === "failed"
                      ? "bg-red-500"
                      : "bg-muted-foreground"
                }`}
              />
              <div className="min-w-0">
                <div className="flex flex-wrap items-baseline gap-2 text-sm">
                  <span className="font-medium">
                    {STAGE_LABEL_KEYS[e.stage] ? t(STAGE_LABEL_KEYS[e.stage]) : e.stage}
                  </span>
                  {e.version && (
                    <span className="font-mono text-xs text-muted-foreground">{e.version}</span>
                  )}
                  <span className="text-xs text-muted-foreground">{formatDate(e.created_at)}</span>
                </div>
                {e.message && (
                  <div className="mt-0.5 text-[13px] text-muted-foreground">{e.message}</div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function AgentLogPanel({ events }: { events: AdminAgentUpdateEvent[] }) {
  const t = useT();
  return (
    <div className="rounded-lg border bg-card">
      <div className="border-b px-5 py-4 text-sm font-semibold">
        {t("admin.updates.agent_log_title")}
      </div>
      {events.length === 0 ? (
        <div className="p-6 text-center text-sm text-muted-foreground">
          {t("admin.updates.agent_log_empty")}
        </div>
      ) : (
        <div className="space-y-3 px-5 py-4">
          {events.map((e, i) => (
            <div key={i} className="flex gap-3">
              <span
                className={`mt-1.5 size-2 shrink-0 rounded-full ${
                  e.status === "completed"
                    ? "bg-emerald-500"
                    : e.status === "failed"
                      ? "bg-red-500"
                      : "bg-muted-foreground"
                }`}
              />
              <div className="min-w-0">
                <div className="flex flex-wrap items-baseline gap-2 text-sm">
                  <span className="font-medium">
                    {e.node || t("admin.updates.node_deleted")}
                  </span>
                  <span className="text-xs text-muted-foreground">
                    {JOB_STATUS_LABEL_KEYS[e.status]
                      ? t(JOB_STATUS_LABEL_KEYS[e.status])
                      : e.status}
                  </span>
                  {e.version && (
                    <span className="font-mono text-xs text-muted-foreground">{e.version}</span>
                  )}
                  <span className="text-xs text-muted-foreground">{formatDate(e.created_at)}</span>
                </div>
                {e.error && <div className="mt-0.5 text-[13px] text-red-500">{e.error}</div>}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
