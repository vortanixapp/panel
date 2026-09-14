"use client";

import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Download, ExternalLink, Loader2, RefreshCw, RotateCw } from "lucide-react";
import { toast } from "sonner";

import { AgentUpdatesTab } from "@/components/admin/updates/agent-updates-tab";
import {
  Segmented,
  SelectField,
  SettingsCard,
  ToggleRow,
} from "@/components/admin/settings/settings-ui";
import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  fetchAdminPanelUpdateStatus,
  fetchAdminUpdates,
  startAdminPanelUpdate,
  updateAdminUpdateSettings,
  type AdminUpdates,
  type PanelAutoUpdate,
  type UpdaterJob,
  type UpdaterStatus,
} from "@/lib/api";
import { localeTag } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";

const UPGRADE_FROM_SOURCE = `git pull
docker compose -f deploy/docker-compose.yml up -d --build`;

const UPGRADE_FROM_IMAGES = `git pull
docker compose -f deploy/docker-compose.yml \\
  -f deploy/docker-compose.images.yml pull
docker compose -f deploy/docker-compose.yml \\
  -f deploy/docker-compose.images.yml up -d`;

const HOURS = Array.from({ length: 24 }, (_, h) => ({
  value: String(h),
  label: `${String(h).padStart(2, "0")}:00`,
}));

function formatDate(value: string | undefined): string {
  if (!value || value.startsWith("0001-")) return "—";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toLocaleString(localeTag(), {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function UpdatesPageContent() {
  const t = useT();
  const [tab, setTab] = useState("panel");

  return (
    <PageShell variant="admin">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">
          {t("admin.updates.title")}
        </h1>
        <p className="text-sm text-muted-foreground">
          {t("admin.updates.subtitle")}
        </p>
      </div>

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="panel" className="px-4">
            {t("admin.updates.tab.panel")}
          </TabsTrigger>
          <TabsTrigger value="agents" className="px-4">
            {t("admin.updates.tab.agents")}
          </TabsTrigger>
        </TabsList>
        <TabsContent value="panel" className="mt-3">
          <PanelUpdates />
        </TabsContent>
        <TabsContent value="agents" className="mt-3">
          <AgentUpdatesTab />
        </TabsContent>
      </Tabs>
    </PageShell>
  );
}

function PanelUpdates() {
  const t = useT();
  const queryClient = useQueryClient();
  const [watching, setWatching] = useState(false);
  const pageVersion = useRef<string | null>(null);

  const updates = useQuery({
    queryKey: queryKeys.adminUpdates,
    queryFn: () => fetchAdminUpdates(),
    staleTime: 5 * 60_000,
  });
  const data = updates.data;
  if (data && pageVersion.current === null) {
    pageVersion.current = data.current_version;
  }

  const jobRunning = data?.updater?.job?.state === "running";
  const status = useQuery({
    queryKey: queryKeys.adminPanelUpdateStatus,
    queryFn: fetchAdminPanelUpdateStatus,
    enabled: watching || jobRunning,
    refetchInterval: 2500,
    retry: false,
  });

  const statusJobState = status.data?.updater?.job?.state;
  useEffect(() => {
    if (statusJobState && statusJobState !== "running") {
      setWatching(false);
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminUpdates });
    }
  }, [statusJobState, queryClient]);

  const start = useMutation({
    mutationFn: (version: string) => startAdminPanelUpdate(version),
    onSuccess: (res) => {
      toast.success(t("admin.updates.started"));
      queryClient.setQueryData(queryKeys.adminPanelUpdateStatus, {
        current_version: data?.current_version ?? "",
        updater: { ...(data?.updater as UpdaterStatus), job: res.job },
      });
      setWatching(true);
    },
    onError: (e: Error) =>
      toast.error(t("admin.updates.action_failed", { error: e.message })),
  });

  const saveAuto = useMutation({
    mutationFn: updateAdminUpdateSettings,
    onSuccess: (auto) => {
      queryClient.setQueryData<AdminUpdates>(queryKeys.adminUpdates, (prev) =>
        prev ? { ...prev, auto } : prev
      );
      toast.success(t("admin.updates.auto.saved"));
    },
    onError: (e: Error) =>
      toast.error(t("admin.updates.action_failed", { error: e.message })),
  });

  async function recheck() {
    const fresh = await fetchAdminUpdates(true);
    queryClient.setQueryData(queryKeys.adminUpdates, fresh);
  }

  if (updates.isLoading || !data) {
    return <Skeleton className="h-40 w-full rounded-xl" />;
  }

  const updater = data.updater;
  const job: UpdaterJob | null =
    status.data?.updater?.job ?? updater?.job ?? null;
  const restarting = (watching || jobRunning) && status.isError;
  const busy = start.isPending || job?.state === "running" || restarting;
  const available = data.update_available === true;
  const liveVersion = status.data?.current_version ?? data.current_version;
  const reloadVersion =
    job?.state === "succeeded" &&
    pageVersion.current !== null &&
    liveVersion !== pageVersion.current
      ? liveVersion
      : null;

  function install() {
    const version = data?.latest_version;
    if (!version) return;
    if (!window.confirm(t("admin.updates.install_confirm", { version }))) return;
    start.mutate(version);
  }

  return (
    <div className="grid gap-4">
      <div className="rounded-xl border border-border p-6">
        <div className="flex flex-wrap items-center gap-x-8 gap-y-4">
          <div>
            <div className="text-xs tracking-wide text-muted-foreground uppercase">
              {t("admin.updates.installed")}
            </div>
            <div className="mt-1 font-mono text-lg font-semibold">
              {liveVersion || "—"}
            </div>
          </div>
          <div>
            <div className="text-xs tracking-wide text-muted-foreground uppercase">
              {t("admin.updates.latest")}
            </div>
            <div className="mt-1 font-mono text-lg font-semibold">
              {data.latest_version || "—"}
            </div>
          </div>
          <div className="ms-auto flex flex-wrap items-center gap-3">
            {data.checks_disabled ? (
              <Badge variant="secondary">
                {t("admin.updates.checks_disabled")}
              </Badge>
            ) : data.error ? (
              <Badge variant="destructive">{data.error}</Badge>
            ) : available ? (
              <Badge>{t("admin.updates.available")}</Badge>
            ) : (
              <Badge variant="secondary">{t("admin.updates.up_to_date")}</Badge>
            )}
            {available && updater?.available && (
              <Button onClick={install} disabled={busy}>
                {busy ? <Loader2 className="animate-spin" /> : <Download />}
                {t("admin.updates.install", { version: data.latest_version ?? "" })}
              </Button>
            )}
          </div>
        </div>

        <div className="mt-5 flex flex-wrap items-center gap-x-6 gap-y-2 border-t border-border pt-4 text-xs text-muted-foreground">
          <span>
            {t("admin.updates.checked_at", { when: formatDate(data.checked_at) })}
          </span>
          {data.published_at && (
            <span>
              {t("admin.updates.published_at", {
                when: formatDate(data.published_at),
              })}
            </span>
          )}
          {data.repo && <span className="font-mono">{data.repo}</span>}
          {data.release_url && (
            <a
              href={data.release_url}
              target="_blank"
              rel="noreferrer noopener"
              className="inline-flex items-center gap-1 text-primary hover:underline"
            >
              {t("admin.updates.open_release")}
              <ExternalLink className="size-3" />
            </a>
          )}
          <Button
            variant="outline"
            size="sm"
            className="ms-auto"
            onClick={recheck}
            disabled={updates.isFetching}
          >
            <RefreshCw className={updates.isFetching ? "animate-spin" : ""} />
            {t("admin.updates.recheck")}
          </Button>
        </div>
      </div>

      {job && (
        <JobCard job={job} restarting={restarting} reloadVersion={reloadVersion} />
      )}

      <AutoUpdateCard
        auto={data.auto}
        updaterAvailable={updater?.available === true}
        onSave={(patch) => saveAuto.mutate(patch)}
      />

      {data.notes && (
        <div className="rounded-xl border border-border p-6">
          <h2 className="text-sm font-semibold">{t("admin.updates.notes")}</h2>
          <pre className="mt-3 max-h-[420px] overflow-auto text-[13px] leading-relaxed whitespace-pre-wrap text-muted-foreground">
            {data.notes}
          </pre>
        </div>
      )}

      {!updater?.available && <ManualUpdateCard reason={updater?.reason} />}
    </div>
  );
}

function JobCard({
  job,
  restarting,
  reloadVersion,
}: {
  job: UpdaterJob;
  restarting: boolean;
  reloadVersion: string | null;
}) {
  const t = useT();
  const logRef = useRef<HTMLPreElement>(null);

  useEffect(() => {
    const el = logRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [job.log.length]);

  const running = job.state === "running" || restarting;
  const label = running
    ? t("admin.updates.job.running")
    : job.state === "succeeded"
      ? t("admin.updates.job.succeeded")
      : t("admin.updates.job.failed");

  return (
    <SettingsCard
      title={t("admin.updates.job.title", { version: job.target })}
      description={
        <span className="flex flex-wrap gap-x-4">
          <span>{t("admin.updates.job.started", { when: formatDate(job.started_at) })}</span>
          {job.finished_at && !running && (
            <span>
              {t("admin.updates.job.finished", { when: formatDate(job.finished_at) })}
            </span>
          )}
        </span>
      }
      action={
        <Badge
          variant={
            job.state === "failed" && !running
              ? "destructive"
              : job.state === "succeeded" && !running
                ? "secondary"
                : "default"
          }
        >
          {running && <Loader2 className="size-3 animate-spin" />}
          {label}
        </Badge>
      }
    >
      {restarting && (
        <p className="mb-3 text-sm text-muted-foreground">
          {t("admin.updates.job.restarting")}
        </p>
      )}
      {job.state === "failed" && !running && job.error && (
        <p className="mb-3 rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
          {job.error}
        </p>
      )}
      {reloadVersion && (
        <div className="mb-3 flex flex-wrap items-center justify-between gap-3 rounded-lg border px-3 py-2 text-sm">
          <span>{t("admin.updates.job.reload_hint", { version: reloadVersion })}</span>
          <Button size="sm" onClick={() => window.location.reload()}>
            <RotateCw />
            {t("admin.updates.job.reload")}
          </Button>
        </div>
      )}
      <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
        {t("admin.updates.job.log")}
      </div>
      <pre
        ref={logRef}
        className="mt-2 max-h-[360px] overflow-auto rounded-lg bg-[var(--vx-surface-2)] p-4 font-mono text-[12px] leading-relaxed whitespace-pre-wrap"
      >
        {job.log.length ? job.log.join("\n") : "…"}
      </pre>
    </SettingsCard>
  );
}

function AutoUpdateCard({
  auto,
  updaterAvailable,
  onSave,
}: {
  auto: PanelAutoUpdate;
  updaterAvailable: boolean;
  onSave: (
    patch: Partial<{ auto_enabled: boolean; window_start: number; window_end: number }>
  ) => void;
}) {
  const t = useT();
  const scheduled = auto.window_start >= 0 && auto.window_end >= 0;

  return (
    <SettingsCard
      title={t("admin.updates.auto.title")}
      description={t("admin.updates.auto.description")}
    >
      <div className="grid gap-4">
        <ToggleRow
          label={t("admin.updates.auto.toggle")}
          hint={t("admin.updates.auto.toggle_hint")}
          checked={auto.enabled}
          onCheckedChange={(checked) => onSave({ auto_enabled: checked })}
        />
        {!updaterAvailable && (
          <p className="text-xs text-amber-600 dark:text-amber-500">
            {t("admin.updates.auto.needs_updater")}
          </p>
        )}
        {auto.skipped_version && (
          <p className="text-xs text-destructive">
            {t("admin.updates.auto.skipped", { version: auto.skipped_version })}
          </p>
        )}
        <div className="grid gap-3">
          <div className="text-xs text-muted-foreground">
            {t("admin.updates.auto.window")}
          </div>
          <Segmented
            className="w-fit"
            items={[
              { id: "any", label: t("admin.updates.auto.anytime") },
              { id: "hours", label: t("admin.updates.auto.hours") },
            ]}
            value={scheduled ? "hours" : "any"}
            onChange={(next) =>
              onSave(
                next === "any"
                  ? { window_start: -1, window_end: -1 }
                  : { window_start: 3, window_end: 6 }
              )
            }
          />
          {scheduled && (
            <div className="grid max-w-md grid-cols-2 gap-4">
              <SelectField
                label={t("admin.updates.auto.from")}
                value={String(auto.window_start)}
                options={HOURS}
                onChange={(v) => onSave({ window_start: Number(v) })}
              />
              <SelectField
                label={t("admin.updates.auto.to")}
                value={String(auto.window_end)}
                options={HOURS}
                onChange={(v) => onSave({ window_end: Number(v) })}
              />
            </div>
          )}
          {scheduled && (
            <p className="text-xs text-muted-foreground">
              {t("admin.updates.auto.window_hint")}
            </p>
          )}
        </div>
      </div>
    </SettingsCard>
  );
}

function ManualUpdateCard({ reason }: { reason?: string }) {
  const t = useT();
  return (
    <SettingsCard title={t("admin.updates.unavailable")} description={reason}>
      <p className="text-sm text-muted-foreground">{t("admin.updates.enable_updater")}</p>
      <h2 className="mt-5 text-sm font-semibold">{t("admin.updates.how")}</h2>
      <p className="mt-2 text-sm text-muted-foreground">{t("admin.updates.how_hint")}</p>
      <div className="mt-4 grid gap-4 lg:grid-cols-2">
        <div>
          <div className="text-xs font-medium tracking-wide uppercase">
            {t("admin.updates.from_source")}
          </div>
          <pre className="mt-2 overflow-x-auto rounded-lg bg-[var(--vx-surface-2)] p-4 font-mono text-[12.5px]">
            {UPGRADE_FROM_SOURCE}
          </pre>
        </div>
        <div>
          <div className="text-xs font-medium tracking-wide uppercase">
            {t("admin.updates.from_images")}
          </div>
          <pre className="mt-2 overflow-x-auto rounded-lg bg-[var(--vx-surface-2)] p-4 font-mono text-[12.5px]">
            {UPGRADE_FROM_IMAGES}
          </pre>
        </div>
      </div>
      <p className="mt-4 text-sm text-muted-foreground">{t("admin.updates.backup_first")}</p>
    </SettingsCard>
  );
}
