"use client";

import Link from "next/link";
import { useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { ArrowLeft, Bug, CircleCheck, ExternalLink, FileText, ImagePlus, Loader2, Send, X } from "lucide-react";

import { PageShell } from "@/components/layout/page-shell";
import { btnGhost, btnPrimary, fieldClass } from "@/components/user/panel-parts";
import { useT } from "@/hooks/use-translations";
import {
  fetchBugReportInfo,
  fetchNodes,
  fetchPanelVersion,
  prepareBugReport,
  type BugReportPrepared,
  type BugReportSeverity,
} from "@/lib/api";
import { localeTag } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { PreviewDialog } from "./bug-report-preview";
import { EnvironmentCard, SimilarCard, UrgentCard, type EnvRow } from "./bug-report-side";
import {
  COMPONENTS,
  NODE_ALL,
  SEVERITIES,
  browserLabel,
  clearDraft,
  nowLabel,
  readDraft,
  screenLabel,
  versionTag,
  writeDraft,
} from "./bug-report-utils";

export function BugReportPageContent() {
  const t = useT();

  const [title, setTitle] = useState("");
  const [desc, setDesc] = useState("");
  const [severity, setSeverity] = useState<BugReportSeverity>("medium");
  const [component, setComponent] = useState("panel-ui");
  const [node, setNode] = useState(NODE_ALL);
  const [attachLogs, setAttachLogs] = useState(true);
  const [report, setReport] = useState<BugReportPrepared | null>(null);
  const [openedUrl, setOpenedUrl] = useState("");
  const [query, setQuery] = useState("");
  const [ua, setUa] = useState("");
  const [screen, setScreen] = useState("");
  const [clock, setClock] = useState("");
  const restored = useRef(false);
  const titleRef = useRef<HTMLInputElement>(null);
  const descRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    setUa(navigator.userAgent);
    setScreen(screenLabel());
    setClock(nowLabel());
    const onResize = () => setScreen(screenLabel());
    const timer = window.setInterval(() => setClock(nowLabel()), 30_000);
    window.addEventListener("resize", onResize);
    return () => {
      window.clearInterval(timer);
      window.removeEventListener("resize", onResize);
    };
  }, []);

  useEffect(() => {
    if (restored.current) return;
    restored.current = true;
    const draft = readDraft();
    if (!draft) return;
    if (draft.title) setTitle(draft.title);
    if (draft.desc) setDesc(draft.desc);
    if (draft.severity && SEVERITIES.some((s) => s.key === draft.severity)) setSeverity(draft.severity);
    if (draft.component && COMPONENTS.includes(draft.component)) setComponent(draft.component);
    if (draft.node) setNode(draft.node);
    if (draft.title || draft.desc) toast.info(t("admin.bug.draft_restored"));
  }, [t]);

  useEffect(() => {
    if (!restored.current) return;
    const timer = window.setTimeout(() => writeDraft({ title, desc, severity, component, node }), 400);
    return () => window.clearTimeout(timer);
  }, [title, desc, severity, component, node]);

  useEffect(() => {
    const timer = window.setTimeout(() => setQuery(title.trim()), 700);
    return () => window.clearTimeout(timer);
  }, [title]);

  const info = useQuery({
    queryKey: ["admin-bug-report-info"],
    queryFn: fetchBugReportInfo,
    staleTime: 60_000,
  });
  const panelBuild = useQuery({
    queryKey: ["panel-version"],
    queryFn: fetchPanelVersion,
    staleTime: 300_000,
  });
  const nodes = useQuery({
    queryKey: queryKeys.adminNodes,
    queryFn: fetchNodes,
    staleTime: 60_000,
  });

  const nodeList = nodes.data ?? [];
  const nodeKnown = node === NODE_ALL || nodeList.some((n) => n.id === node);
  const uiVersion = versionTag(panelBuild.data ?? "");
  const serverVersion = info.data?.version ?? "";

  const envRows: EnvRow[] = useMemo(() => {
    const rows: EnvRow[] = [{ label: t("admin.bug.env_version"), value: serverVersion }];
    if (uiVersion && serverVersion && uiVersion !== serverVersion) {
      rows.push({ label: t("admin.bug.env_ui"), value: uiVersion });
    }
    const nodesInfo = info.data?.nodes;
    rows.push(
      { label: t("admin.bug.env_database"), value: info.data?.database ?? "" },
      {
        label: t("admin.bug.env_nodes"),
        value: nodesInfo ? t("admin.bug.env_nodes_value", { total: nodesInfo.total, online: nodesInfo.online }) : "",
      }
    );
    if (nodesInfo && nodesInfo.agents.length > 0) {
      rows.push({
        label: t("admin.bug.env_agents"),
        value: nodesInfo.agents.map((a) => (nodesInfo.agents.length > 1 ? `${a.version} ×${a.count}` : a.version)).join(" · "),
      });
    }
    rows.push(
      { label: t("admin.bug.env_browser"), value: browserLabel(ua) },
      { label: t("admin.bug.env_screen"), value: screen },
      { label: t("admin.bug.env_time"), value: clock }
    );
    return rows;
  }, [t, serverVersion, uiVersion, info.data, ua, screen, clock]);

  const prepare = useMutation({
    mutationFn: () =>
      prepareBugReport({
        title: title.trim(),
        description: desc.trim(),
        severity,
        component,
        node_id: node === NODE_ALL || !nodeKnown ? "" : node,
        attach_logs: attachLogs,
        client: {
          ui_version: uiVersion,
          browser: browserLabel(navigator.userAgent),
          screen: screenLabel(),
          language: localeTag(),
          time: nowLabel(),
        },
      }),
    onSuccess: (res) => setReport(res),
    onError: (err) => toast.error(err instanceof Error && err.message ? err.message : t("admin.bug.prepare_failed")),
  });

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!title.trim()) {
      toast.error(t("admin.bug.title_required"));
      titleRef.current?.focus();
      return;
    }
    if (!desc.trim()) {
      toast.error(t("admin.bug.desc_required"));
      descRef.current?.focus();
      return;
    }
    prepare.mutate();
  }

  function resetForm() {
    setTitle("");
    setDesc("");
    setSeverity("medium");
    setComponent("panel-ui");
    setNode(NODE_ALL);
    clearDraft();
  }

  function onOpened(prepared: BugReportPrepared) {
    setOpenedUrl(prepared.url);
    setReport(null);
    resetForm();
  }

  const hasContent = title.trim() !== "" || desc.trim() !== "";

  return (
    <PageShell variant="admin">
      <div className="flex flex-col gap-[22px]">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div className="flex min-w-0 flex-col gap-1.5">
            <span className="flex items-center gap-2 text-xs text-muted-foreground">
              <Bug className="h-3.5 w-3.5" />
              {t("admin.bug.breadcrumb")}
            </span>
            <h1 className="text-[26px] leading-none font-bold tracking-[-0.02em]">{t("admin.bug.title")}</h1>
            <p className="max-w-3xl text-[13.5px] text-muted-foreground">{t("admin.bug.subtitle")}</p>
          </div>
          <div className="flex items-center gap-2">
            {info.data?.my_issues_url && (
              <a
                href={info.data.my_issues_url}
                target="_blank"
                rel="noopener noreferrer"
                title={t("admin.bug.my_reports_hint")}
                className={cn(btnGhost, "h-[38px]")}
              >
                <i className="ri-github-fill text-[16px] leading-none" />
                {t("admin.bug.my_reports")}
                <ExternalLink className="h-3.5 w-3.5 text-muted-foreground" />
              </a>
            )}
            <Link href="/admin" className={cn(btnGhost, "h-[38px]")}>
              <ArrowLeft className="h-4 w-4" />
              {t("common.back")}
            </Link>
          </div>
        </div>

        {openedUrl && (
          <div
            className="flex flex-wrap items-start gap-3 rounded-[14px] border p-4"
            style={{
              borderColor: "color-mix(in srgb, var(--vx-ok) 30%, transparent)",
              background: "var(--vx-ok-tint)",
            }}
          >
            <CircleCheck className="mt-0.5 h-4 w-4 flex-none" style={{ color: "var(--vx-ok)" }} />
            <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-[3px]">
              <span className="text-[13.5px] font-semibold">{t("admin.bug.opened_title")}</span>
              <span className="text-[12.5px] text-muted-foreground">{t("admin.bug.opened_hint")}</span>
            </div>
            <div className="flex flex-none items-center gap-2">
              <a
                href={openedUrl}
                target="_blank"
                rel="noopener noreferrer"
                className={cn(btnGhost, "h-[30px] bg-background px-3 text-[12.5px]")}
              >
                {t("admin.bug.open_again")}
                <ExternalLink className="h-3.5 w-3.5" />
              </a>
              <button
                type="button"
                onClick={() => setOpenedUrl("")}
                aria-label={t("admin.bug.dismiss")}
                className="grid h-[30px] w-[30px] place-items-center rounded-[9px] text-muted-foreground transition-colors hover:bg-background hover:text-foreground"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          </div>
        )}

        <div className="grid grid-cols-1 items-start gap-5 xl:grid-cols-[minmax(0,1fr)_340px]">
          <form
            onSubmit={onSubmit}
            noValidate
            className="flex min-w-0 flex-col gap-[18px] rounded-2xl border border-border bg-card p-4 sm:p-5"
          >
            <div className="flex flex-col gap-[7px]">
              <label htmlFor="bug-title" className="text-[13px] font-medium">
                {t("admin.bug.title_label")}
              </label>
              <input
                id="bug-title"
                ref={titleRef}
                value={title}
                maxLength={200}
                onChange={(e) => setTitle(e.target.value)}
                placeholder={t("admin.bug.title_placeholder")}
                className={cn(fieldClass, "h-10")}
              />
              <span className="text-[11.5px] text-muted-foreground/80">{t("admin.bug.title_hint")}</span>
            </div>

            <div className="flex flex-col gap-[7px]">
              <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                <label htmlFor="bug-desc" className="text-[13px] font-medium">
                  {t("admin.bug.desc_label")}
                </label>
                <span className="text-[11.5px] text-muted-foreground/80">{t("admin.bug.desc_hint")}</span>
                <button
                  type="button"
                  onClick={() => {
                    setDesc(desc.trim() ? `${desc.trimEnd()}\n\n${t("admin.bug.template")}` : t("admin.bug.template"));
                    descRef.current?.focus();
                  }}
                  className={cn(btnGhost, "ms-auto h-7 px-2.5 text-[12px]")}
                >
                  <FileText className="h-3.5 w-3.5" />
                  {t("admin.bug.use_template")}
                </button>
              </div>
              <textarea
                id="bug-desc"
                ref={descRef}
                value={desc}
                maxLength={6000}
                onChange={(e) => setDesc(e.target.value)}
                placeholder={t("admin.bug.desc_placeholder")}
                className={cn(fieldClass, "h-auto min-h-[180px] resize-y py-3 leading-relaxed")}
              />
            </div>

            <div className="flex flex-col gap-[9px]">
              <span className="text-[13px] font-medium">{t("admin.bug.severity")}</span>
              <div className="grid grid-cols-[repeat(auto-fit,minmax(150px,1fr))] gap-2">
                {SEVERITIES.map((item) => {
                  const active = severity === item.key;
                  return (
                    <button
                      key={item.key}
                      type="button"
                      onClick={() => setSeverity(item.key)}
                      aria-pressed={active}
                      className={cn(
                        "flex flex-col items-start gap-[5px] rounded-xl border px-3 py-[11px] text-start transition-colors hover:border-ring",
                        active ? "border-ring bg-accent" : "border-border bg-background"
                      )}
                    >
                      <span
                        className={cn("flex items-center gap-[7px] text-[13px]", active ? "font-semibold" : "font-medium")}
                      >
                        <span className="h-2 w-2 flex-none rounded-full" style={{ background: item.dot }} />
                        {t(`admin.bug.severity_${item.key}`)}
                      </span>
                      <span className="text-[11.5px] leading-snug text-muted-foreground">
                        {t(`admin.bug.severity_${item.key}_hint`)}
                      </span>
                    </button>
                  );
                })}
              </div>
            </div>

            <div className="grid grid-cols-[repeat(auto-fit,minmax(200px,1fr))] gap-3">
              <div className="flex flex-col gap-[7px]">
                <label htmlFor="bug-component" className="text-[13px] font-medium">
                  {t("admin.bug.component")}
                </label>
                <select
                  id="bug-component"
                  value={component}
                  onChange={(e) => setComponent(e.target.value)}
                  className={cn(fieldClass, "h-10")}
                >
                  {COMPONENTS.map((name) => (
                    <option key={name} value={name}>
                      {t(`admin.bug.component.${name}`)}
                    </option>
                  ))}
                </select>
              </div>
              <div className="flex flex-col gap-[7px]">
                <label htmlFor="bug-node" className="text-[13px] font-medium">
                  {t("admin.bug.node")}
                </label>
                <select
                  id="bug-node"
                  value={nodeKnown ? node : NODE_ALL}
                  onChange={(e) => setNode(e.target.value)}
                  className={cn(fieldClass, "h-10")}
                >
                  <option value={NODE_ALL}>{t("admin.bug.node_all")}</option>
                  {nodeList.map((n) => (
                    <option key={n.id} value={n.id}>
                      {n.fqdn ? `${n.name} (${n.fqdn})` : n.name}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            <div className="flex items-start gap-3 rounded-xl border border-dashed border-border bg-background px-3.5 py-3">
              <ImagePlus className="mt-0.5 h-4 w-4 flex-none text-muted-foreground" />
              <div className="flex min-w-0 flex-col gap-0.5">
                <span className="text-[12.5px] font-medium">{t("admin.bug.shots")}</span>
                <span className="text-[11.5px] leading-snug text-muted-foreground">{t("admin.bug.shots_hint")}</span>
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-3.5 border-t border-border pt-3.5">
              <button
                type="submit"
                disabled={prepare.isPending}
                className={cn(btnPrimary, "h-10 px-[18px] text-[13.5px]")}
              >
                {prepare.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
                {prepare.isPending ? t("admin.bug.preparing") : t("admin.bug.submit")}
              </button>
              {hasContent && (
                <button
                  type="button"
                  onClick={() => {
                    resetForm();
                    toast.success(t("admin.bug.cleared"));
                  }}
                  className={cn(btnGhost, "h-10 text-[13px]")}
                >
                  {t("admin.bug.clear")}
                </button>
              )}
              <span className="min-w-[220px] flex-1 text-xs leading-snug text-muted-foreground/80">
                {t("admin.bug.footer")}
              </span>
            </div>
          </form>

          <div className="grid min-w-0 grid-cols-1 items-start gap-4 md:grid-cols-2 xl:grid-cols-1">
            <EnvironmentCard rows={envRows} attachLogs={attachLogs} onAttachLogs={setAttachLogs} />
            <SimilarCard query={query} info={info.data} />
            <UrgentCard />
          </div>
        </div>
      </div>

      <PreviewDialog
        report={report}
        attachLogs={attachLogs}
        onClose={() => setReport(null)}
        onOpened={onOpened}
      />
    </PageShell>
  );
}
