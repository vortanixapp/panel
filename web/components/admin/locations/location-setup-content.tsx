"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { LocationAgentSetupCard } from "@/components/admin/locations/location-agent-setup-card";
import { cn } from "@/lib/utils";
import {
  fetchAdminLocationSetup,
  fetchAdminLocationSetupStatus,
  runAdminLocationSetupStep,
} from "@/lib/api";
import { LOCATION_SETUP_STEPS, setupStatusLabel } from "@/lib/location-setup";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";

const STATUS_CLASSES: Record<string, string> = {
  installed: "border-emerald-500/40 text-emerald-500",
  installing: "border-amber-500/40 text-amber-500",
  failed: "border-rose-500/40 text-rose-500",
  pending: "text-muted-foreground",
};

export function LocationSetupContent() {
  const t = useT();
  const params = useParams();
  const id = String(params?.id ?? "");
  const [running, setRunning] = useState(false);
  const [activeComponent, setActiveComponent] = useState("");
  const [log, setLog] = useState("");
  const [pollTimer, setPollTimer] = useState<number | null>(null);
  const logRef = useRef<HTMLPreElement | null>(null);
  const pollingRef = useRef(false);
  const resumedRef = useRef(false);

  const { data, isLoading, refetch } = useQuery({
    queryKey: queryKeys.adminLocationSetup(id),
    queryFn: () => fetchAdminLocationSetup(id),
    enabled: !!id,
  });

  useEffect(() => {
    return () => {
      if (pollTimer) window.clearInterval(pollTimer);
    };
  }, [pollTimer]);

  useEffect(() => {
    const el = logRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [log]);

  const runMut = useMutation({
    mutationFn: (step: string) => runAdminLocationSetupStep(id, step),
  });

  function pollStatus() {
    if (pollingRef.current) return;
    pollingRef.current = true;
    let stopped = false;
    const stopPolling = () => {
      stopped = true;
      pollingRef.current = false;
      window.clearInterval(timer);
      setPollTimer(null);
      setRunning(false);
      setActiveComponent("");
    };
    const tick = async () => {
      if (stopped) return;
      try {
        const status = await fetchAdminLocationSetupStatus(id);
        if (status.log) setLog(status.log);
        if (status.component) setActiveComponent(status.component);
        if (status.completed) {
          stopPolling();
          void refetch();
          toast.success(t("admin.setup.step_done"));
        }
      } catch (e) {
        stopPolling();
        toast.error(
          e instanceof Error ? e.message : t("admin.setup.status_failed")
        );
      }
    };
    const timer = window.setInterval(tick, 1000);
    setPollTimer(timer);
    void tick();
  }

  useEffect(() => {
    if (resumedRef.current || !data) return;
    const installing = Object.entries(data.statuses ?? {}).find(
      ([, value]) => value === "installing"
    );
    if (!installing) return;
    resumedRef.current = true;
    setRunning(true);
    setActiveComponent(installing[0]);
    pollStatus();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data]);

  async function runStep(component: string, endpoint: string) {
    setRunning(true);
    setActiveComponent(component);
    setLog("");
    try {
      const res = await runMut.mutateAsync(endpoint);
      if (res.message) setLog(res.message);
      pollStatus();
    } catch (e) {
      setRunning(false);
      setActiveComponent("");
      toast.error(
        e instanceof Error ? e.message : t("admin.agent.install_failed")
      );
    }
  }

  if (isLoading) {
    return (
      <PageShell variant="admin">
        <div className="w-full space-y-4">
          <Skeleton className="h-16 w-full rounded-2xl" />
          <Skeleton className="h-64 w-full rounded-2xl" />
        </div>
      </PageShell>
    );
  }

  const location = data?.location as Record<string, unknown> | undefined;
  const statuses = data?.statuses ?? {};
  const checks = data?.checks ?? [];
  const sshReady = checks.find((c) => c.key === "ssh")?.ok ?? false;
  const installedCount = LOCATION_SETUP_STEPS.filter(
    (step) => statuses[step.component] === "installed"
  ).length;
  const activeStep = LOCATION_SETUP_STEPS.find(
    (s) => s.component === activeComponent
  );
  const activeLabel = activeStep ? t(activeStep.labelKey) : activeComponent;

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-4">
        <div className="flex flex-wrap items-end justify-between gap-6">
          <div className="space-y-1.5">
            <div className="flex items-center gap-2.5 text-xs text-muted-foreground">
              <Link href="/admin/locations" className="hover:text-foreground">
                {t("common.locations")}
              </Link>
              <span>/</span>
              <Link
                href={`/admin/locations/${id}`}
                className="hover:text-foreground"
              >
                {String(location?.name ?? t("common.location"))}
              </Link>
            </div>
            <h1 className="text-[26px] leading-none font-bold tracking-tight">
              {t("admin.setup.title", { name: String(location?.name ?? "") })}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t("admin.setup.subtitle")}
            </p>
          </div>
          <Button variant="outline" asChild className="h-[38px] text-[13px]">
            <Link href={`/admin/locations/${id}`}>
              ← {t("admin.location.to_location")}
            </Link>
          </Button>
        </div>

        {!sshReady && (
          <div className="flex flex-wrap items-center gap-3.5 rounded-xl border border-amber-500/40 bg-amber-500/5 px-4 py-3.5">
            <span className="size-1.5 shrink-0 rounded-full bg-amber-500" />
            <p className="flex-1 basis-[320px] text-[13px] leading-relaxed text-amber-600 dark:text-amber-200/90">
              {t("admin.locations.ssh_not_configured")} —{" "}
              <Link
                href={`/admin/locations/${id}/edit`}
                className="underline underline-offset-4"
              >
                {t("admin.setup.fill_ssh")}
              </Link>{" "}
              {t("admin.setup.then_test")}
            </p>
          </div>
        )}

        {checks.length > 0 && (
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {checks.map((c) => (
              <div
                key={c.key}
                className="flex items-center justify-between gap-3.5 rounded-xl border bg-card px-5 py-4"
                title={c.hint}
              >
                <span className="text-[13px]">{c.label}</span>
                <span
                  className={cn(
                    "inline-flex shrink-0 items-center gap-2 text-xs",
                    c.ok ? "text-emerald-500" : "text-rose-500"
                  )}
                >
                  <span
                    className={cn(
                      "size-1.5 rounded-full",
                      c.ok ? "bg-emerald-500" : "bg-rose-500"
                    )}
                  />
                  {c.ok ? "OK" : t("common.no")}
                </span>
              </div>
            ))}
          </div>
        )}

        <LocationAgentSetupCard
          locationId={id}
          compact
          description={t("admin.setup.agent_card_hint")}
        />

        <section className="flex flex-col gap-5 rounded-2xl border bg-card px-5 py-5 sm:px-6">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="space-y-1">
              <div className="text-[15px] leading-none font-semibold">
                {t("admin.setup.steps_title")}
              </div>
              <div className="text-xs text-muted-foreground">
                {t("admin.setup.steps_hint")}
              </div>
            </div>
            <span className="font-mono text-xs text-muted-foreground">
              {t("admin.setup.steps_progress", {
                done: installedCount,
                total: LOCATION_SETUP_STEPS.length,
              })}
            </span>
          </div>

          <div className="grid gap-3.5 sm:grid-cols-2 xl:grid-cols-3">
            {LOCATION_SETUP_STEPS.map((step) => {
              const st = statuses[step.component];
              const busy =
                (running && activeComponent === step.component) ||
                st === "installing";
              const installed = st === "installed";
              const failed = st === "failed";
              return (
                <div
                  key={step.component}
                  className={cn(
                    "flex flex-col gap-3.5 rounded-xl border bg-muted/30 px-5 py-5",
                    step.required && "border-foreground/20"
                  )}
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0 space-y-1.5">
                      <div className="text-sm font-semibold">
                        {t(step.labelKey)}
                        {step.required && (
                          <span className="ml-1 text-muted-foreground">*</span>
                        )}
                      </div>
                      <div className="text-xs leading-relaxed text-muted-foreground">
                        {t(step.descriptionKey)}
                      </div>
                    </div>
                    <span
                      className={cn(
                        "shrink-0 rounded-md border px-2 py-0.5 text-[11px]",
                        STATUS_CLASSES[st ?? "pending"] ??
                          STATUS_CLASSES.pending
                      )}
                    >
                      {setupStatusLabel(t, st)}
                    </span>
                  </div>
                  {busy ? (
                    <Button
                      variant="outline"
                      className="h-[34px] w-full border-amber-500/40 text-[13px] text-amber-500"
                      onClick={() =>
                        logRef.current?.scrollIntoView({ behavior: "smooth" })
                      }
                    >
                      {t("admin.setup.show_log")}
                    </Button>
                  ) : (
                    <Button
                      variant={installed ? "outline" : "default"}
                      className="h-[34px] w-full text-[13px]"
                      disabled={running || installed || !sshReady}
                      onClick={() => runStep(step.component, step.endpoint)}
                    >
                      {installed
                        ? t("common.done")
                        : failed
                          ? t("common.retry")
                          : t("common.start")}
                    </Button>
                  )}
                </div>
              );
            })}
          </div>
        </section>

        <section className="flex flex-col gap-3.5 rounded-2xl border bg-card px-5 py-5 sm:px-6">
          <div className="flex flex-wrap items-center justify-between gap-3.5">
            <span className="text-[15px] font-semibold">
              {t("admin.setup.log_title")}
            </span>
            <span className="font-mono text-xs text-muted-foreground">
              {running || activeComponent
                ? t("admin.setup.log_active", { step: activeLabel })
                : t("admin.setup.log_idle")}
            </span>
          </div>
          <pre
            ref={logRef}
            className="max-h-[240px] overflow-auto rounded-xl border bg-muted/30 px-4 py-4 font-mono text-xs leading-relaxed whitespace-pre-wrap text-muted-foreground"
          >
            {log || t("admin.setup.log_empty")}
          </pre>
        </section>
      </div>
    </PageShell>
  );
}
