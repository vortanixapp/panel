"use client";

import Link from "next/link";
import { useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Activity,
  ArrowLeft,
  ArrowRight,
  BookOpen,
  Bug,
  CircleCheck,
  CopyCheck,
  Cpu,
  ExternalLink,
  FileText,
  ImagePlus,
  LifeBuoy,
  List,
  Save,
  Send,
  X,
} from "lucide-react";

import { PageShell } from "@/components/layout/page-shell";
import { Switch } from "@/components/ui/switch";
import {
  createAdminBugReport,
  fetchAdminBugReports,
  fetchAdminLogsFiltered,
  fetchNodes,
  fetchPanelVersion,
  uploadSupportAttachment,
  type AdminBugReport,
} from "@/lib/api";
import { useMe } from "@/hooks/use-queries";
import { queryKeys } from "@/lib/query-keys";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { btnGhost, btnPrimary, fieldClass } from "@/components/user/panel-parts";
import { SupportStatusPill } from "@/components/user/support/support-parts";
import { useT } from "@/hooks/use-translations";

type Severity = "low" | "medium" | "high" | "critical";

const COMPONENTS = [
  "panel-ui",
  "core-api",
  "agent",
  "agent-relay",
  "console-gateway",
  "metrics-ingest",
  "worker",
  "status-page",
];

// Подписи храним ключами: список читается на уровне модуля, и готовый текст
// застыл бы на языке, который стоял в момент загрузки страницы.
const SEVERITIES: {
  key: Severity;
  labelKey: string;
  hintKey: string;
  dot: string;
}[] = [
  {
    key: "low",
    labelKey: "admin.bug.severity_low",
    hintKey: "admin.bug.severity_low_hint",
    dot: "var(--vx-ink-faint)",
  },
  {
    key: "medium",
    labelKey: "admin.bug.severity_medium",
    hintKey: "admin.bug.severity_medium_hint",
    dot: "var(--vx-warn)",
  },
  {
    key: "high",
    labelKey: "admin.bug.severity_high",
    hintKey: "admin.bug.severity_high_hint",
    // Между «предупреждением» и «ошибкой»: своего токена под эту ступень нет,
    // а брать чужой значило бы сравнять высокую критичность с критической.
    dot: "color-mix(in srgb, var(--vx-danger) 65%, var(--vx-warn))",
  },
  {
    key: "critical",
    labelKey: "admin.bug.severity_critical",
    hintKey: "admin.bug.severity_critical_hint",
    dot: "var(--vx-danger)",
  },
];

const SHOT_SLOTS = ["admin.bug.shot_1", "admin.bug.shot_2", "admin.bug.shot_3"];
const SHOT_MAX_BYTES = 5 << 20;
const SHOT_TYPES = "image/png,image/jpeg,image/webp";

/** Окно журнала, которое прикладывается к отчёту. Значение попадает и в подпись
 *  переключателя, поэтому живёт одной константой. */
const LOG_WINDOW_MIN = 15;

const DRAFT_KEY = "vx.admin.bug-report.draft";

const NODE_ALL = "__all__";

type Shot = { file: File; url: string };

type Draft = {
  title?: string;
  desc?: string;
  severity?: Severity;
  component?: string;
  node?: string;
};

/** Понятное имя браузера вместо User-Agent целиком: разработчику нужны
 *  движок и версия, а не строка на двести символов. Порядок правил важен —
 *  Edge и Opera представляются ещё и Chrome. */
const UA_RULES: { re: RegExp; name: string }[] = [
  { re: /Firefox\/([\d.]+)/, name: "Firefox" },
  { re: /Edg\/([\d.]+)/, name: "Edge" },
  { re: /OPR\/([\d.]+)/, name: "Opera" },
  { re: /Chrome\/([\d.]+)/, name: "Chrome" },
  { re: /Version\/([\d.]+).+Safari/, name: "Safari" },
];

const OS_RULES: { re: RegExp; name: string }[] = [
  { re: /Windows NT 10\.0/, name: "Windows 10/11" },
  { re: /Windows NT ([\d.]+)/, name: "Windows" },
  { re: /Android ([\d.]+)/, name: "Android" },
  { re: /iPhone OS ([\d_]+)/, name: "iOS" },
  { re: /Mac OS X ([\d_]+)/, name: "macOS" },
  { re: /Linux/, name: "Linux" },
];

function browserLabel(ua: string): string {
  if (!ua) return "";
  let browser = "";
  for (const rule of UA_RULES) {
    const m = rule.re.exec(ua);
    if (m) {
      browser = `${rule.name} ${(m[1] ?? "").split(".")[0]}`.trim();
      break;
    }
  }
  let os = "";
  for (const rule of OS_RULES) {
    if (rule.re.test(ua)) {
      os = rule.name;
      break;
    }
  }
  if (browser && os) return `${browser} / ${os}`;
  return browser || os || ua.slice(0, 60);
}

function nowLabel(): string {
  const now = new Date();
  const offset = -now.getTimezoneOffset() / 60;
  const sign = offset >= 0 ? "+" : "−";
  const zone = `UTC${sign}${Math.abs(offset)}`;
  return `${now.toLocaleString(localeTag(), {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })} ${zone}`;
}

function tokenize(value: string): string[] {
  return value
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, " ")
    .split(" ")
    .filter((w) => w.length > 3);
}

export function BugReportPageContent() {
  const t = useT();
  const { data: me } = useMe();

  const [title, setTitle] = useState("");
  const [desc, setDesc] = useState("");
  const [severity, setSeverity] = useState<Severity>("medium");
  const [component, setComponent] = useState("panel-ui");
  const [node, setNode] = useState(NODE_ALL);
  const [attachLogs, setAttachLogs] = useState(true);
  const [shots, setShots] = useState<(Shot | null)[]>([null, null, null]);
  // Тот же список ссылкой: обработчик очистки при уходе со страницы иначе
  // видел бы слоты такими, какими они были на первом рендере, и превью
  // выбранных скриншотов оставались бы в памяти вкладки.
  const shotsRef = useRef<(Shot | null)[]>([null, null, null]);
  const [sentTicket, setSentTicket] = useState("");
  // Браузер и время читаются только после монтирования: на сервере navigator
  // недоступен, а разметка с локальным временем разъезжалась бы при гидрации.
  const [ua, setUa] = useState("");
  const [clock, setClock] = useState("");

  useEffect(() => {
    setUa(navigator.userAgent);
    setClock(nowLabel());
    const timer = setInterval(() => setClock(nowLabel()), 30_000);
    return () => clearInterval(timer);
  }, []);

  // Черновик восстанавливаем один раз при входе на страницу: отчёт часто пишут
  // в несколько заходов, между ними успевая уйти в раздел, где ошибка видна.
  useEffect(() => {
    const raw = localStorage.getItem(DRAFT_KEY);
    if (!raw) return;
    let draft: Draft;
    try {
      draft = JSON.parse(raw) as Draft;
    } catch {
      localStorage.removeItem(DRAFT_KEY);
      return;
    }
    if (draft.title) setTitle(draft.title);
    if (draft.desc) setDesc(draft.desc);
    if (draft.severity) setSeverity(draft.severity);
    if (draft.component) setComponent(draft.component);
    if (draft.node) setNode(draft.node);
    if (draft.title || draft.desc) toast.info(t("admin.bug.draft_restored"));
  }, [t]);

  // Ссылки на превью держит браузер, пока их не отозвать. Замену слота убирает
  // setShot, а уход со страницы — этот обработчик.
  useEffect(() => {
    return () => {
      for (const shot of shotsRef.current) if (shot) URL.revokeObjectURL(shot.url);
    };
  }, []);

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
  const reports = useQuery({
    queryKey: ["admin-bug-reports"],
    queryFn: fetchAdminBugReports,
    staleTime: 30_000,
  });

  const nodeLabel = useMemo(() => {
    if (node === NODE_ALL) return t("admin.bug.node_all");
    const found = (nodes.data ?? []).find((n) => n.id === node);
    if (!found) return node;
    return found.fqdn ? `${found.name} (${found.fqdn})` : found.name;
  }, [node, nodes.data, t]);

  const env: { label: string; value: string }[] = useMemo(
    () => [
      { label: t("admin.bug.env_build"), value: panelBuild.data || "—" },
      { label: t("admin.bug.env_browser"), value: browserLabel(ua) || "—" },
      { label: t("admin.bug.env_account"), value: me?.email ?? "—" },
      { label: t("admin.bug.env_time"), value: clock || "—" },
    ],
    [t, panelBuild.data, ua, me?.email, clock]
  );

  const myReports = reports.data?.mine ?? 0;

  // Похожие ищем по совпадению слов заголовка. Пока заголовка нет, показываем
  // просто свежие отчёты: список нужен до того, как автор начал печатать.
  const similar = useMemo(() => {
    const list = reports.data?.reports ?? [];
    const words = tokenize(title);
    if (!words.length) return list.slice(0, 3);
    return list
      .map((report) => {
        const own = new Set(tokenize(report.title));
        return { report, score: words.reduce((acc, w) => acc + (own.has(w) ? 1 : 0), 0) };
      })
      .filter((x) => x.score > 0)
      .sort((a, b) => b.score - a.score)
      .slice(0, 3)
      .map((x) => x.report);
  }, [reports.data, title]);

  function setShot(index: number, file: File | null) {
    const next = [...shotsRef.current];
    const old = next[index];
    if (old) URL.revokeObjectURL(old.url);
    next[index] = file ? { file, url: URL.createObjectURL(file) } : null;
    shotsRef.current = next;
    setShots(next);
  }

  function pickShot(index: number, file: File) {
    if (!file.type.startsWith("image/")) {
      toast.error(t("admin.bug.shot_not_image", { name: file.name }));
      return;
    }
    if (file.size > SHOT_MAX_BYTES) {
      toast.error(t("admin.bug.shot_too_big", { name: file.name }));
      return;
    }
    setShot(index, file);
  }

  /** Журнал панели за последние минуты одним текстовым вложением. null — в окне
   *  нет ни одной записи; молча приложить пустой файл значило бы соврать. */
  async function buildLogsFile(): Promise<File | null> {
    const res = await fetchAdminLogsFiltered({ limit: 200 });
    const since = Date.now() - LOG_WINDOW_MIN * 60_000;
    const rows = (res.logs ?? []).filter((row) => {
      const ts = new Date(row.created_at).getTime();
      return !Number.isNaN(ts) && ts >= since;
    });
    if (!rows.length) return null;
    const text = rows
      .map((row) => `${row.created_at}\t${row.action}\t${row.resource}`)
      .join("\n");
    return new File([text], `panel-${LOG_WINDOW_MIN}min.log`, {
      type: "text/plain",
    });
  }

  /** Те же окружение и нода, что уходят полями в meta обращения, но текстом.
   *  Тред обращения показывает оператору только сообщения, meta он не видит —
   *  иначе первый ответ разработчика всегда один и тот же: «какая версия
   *  панели?». Поля нужны отчётам по компонентам, текст — живому человеку. */
  function composeDescription(): string {
    const lines: string[] = [];
    const body = desc.trim();
    if (body) lines.push(body, "");
    lines.push("—", `${t("admin.bug.node")}: ${nodeLabel}`, `${t("admin.bug.env")}:`);
    for (const row of env) lines.push(`  ${row.label}: ${row.value}`);
    return lines.join("\n");
  }

  const reportMutation = useMutation({
    mutationFn: async () => {
      const created = await createAdminBugReport({
        title: title.trim(),
        description: composeDescription(),
        severity,
        component,
        node: nodeLabel,
        environment: Object.fromEntries(env.map((row) => [row.label, row.value])),
      });

      const files: File[] = shots.filter((s): s is Shot => s !== null).map((s) => s.file);
      let logsEmpty = false;
      if (attachLogs) {
        const logs = await buildLogsFile();
        if (logs) files.push(logs);
        else logsEmpty = true;
      }

      // Вложения идут отдельным запросом: если он не прошёл, обращение уже
      // создано, и терять его номер нельзя — возвращаем вместе с ошибкой.
      let uploadError = "";
      if (files.length) {
        try {
          await uploadSupportAttachment(created.ticket_id, files);
        } catch (err) {
          uploadError = err instanceof Error ? err.message : String(err);
        }
      }
      return {
        ticketId: created.ticket_id,
        uploadError,
        logsEmpty,
        reportNumber: created.report_number,
        deliveryError: created.delivery_error,
      };
    },
    onSuccess: (res) => {
      setSentTicket(res.ticketId);
      // Автору важно знать не номер обращения у себя, а дошёл ли отчёт до того,
      // кто чинит панель: копия в «Поддержке» сама по себе ничего не решает.
      if (res.reportNumber) {
        toast.success(t("admin.bug.delivered", { number: String(res.reportNumber) }));
      } else if (res.deliveryError) {
        toast.warning(t("admin.bug.not_delivered", { reason: res.deliveryError }));
      } else {
        toast.success(t("admin.bug.created", { ticket: res.ticketId.slice(0, 8) }));
      }
      if (res.logsEmpty) toast.info(t("admin.bug.logs_empty"));
      if (res.uploadError) {
        toast.error(t("admin.bug.upload_failed", { message: res.uploadError }));
      }
      localStorage.removeItem(DRAFT_KEY);
      setTitle("");
      setDesc("");
      for (let i = 0; i < shots.length; i += 1) setShot(i, null);
      void reports.refetch();
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("admin.bug.send_failed")),
  });

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!title.trim()) {
      toast.error(t("admin.bug.title_required"));
      return;
    }
    reportMutation.mutate();
  }

  function saveDraft() {
    const draft: Draft = { title, desc, severity, component, node };
    localStorage.setItem(DRAFT_KEY, JSON.stringify(draft));
    toast.success(t("admin.bug.draft_saved"));
  }

  return (
    <PageShell variant="admin">
      <div className="flex flex-col gap-[22px]">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div className="flex flex-col gap-1.5">
            <span className="flex items-center gap-2 text-xs text-muted-foreground">
              <Bug className="h-3.5 w-3.5" />
              {t("admin.bug.breadcrumb")}
            </span>
            <h1 className="text-[26px] leading-none font-bold tracking-[-0.02em]">
              {t("admin.bug.title")}
            </h1>
            <p className="text-[13.5px] text-muted-foreground">{t("admin.bug.subtitle")}</p>
          </div>
          <div className="flex items-center gap-2">
            <Link href="/admin/support" className={cn(btnGhost, "h-[38px]")}>
              <List className="h-4 w-4" />
              {myReports > 0
                ? t("admin.bug.my_reports_n", { count: myReports })
                : t("admin.bug.my_reports")}
            </Link>
            <Link href="/admin" className={cn(btnGhost, "h-[38px]")}>
              <ArrowLeft className="h-4 w-4" />
              {t("common.back")}
            </Link>
          </div>
        </div>

        {sentTicket && (
          <div
            className="flex items-start gap-3 rounded-[14px] border p-4"
            style={{
              borderColor: "color-mix(in srgb, var(--vx-ok) 30%, transparent)",
              background: "var(--vx-ok-tint)",
            }}
          >
            <CircleCheck className="mt-0.5 h-4 w-4 flex-none" style={{ color: "var(--vx-ok)" }} />
            <div className="flex min-w-0 flex-1 flex-col gap-[3px]">
              <span className="text-[13.5px] font-semibold">
                {t("admin.bug.sent_title", { ticket: sentTicket.slice(0, 8) })}
              </span>
              <span className="text-[12.5px] text-muted-foreground">
                {t("admin.bug.sent_hint")}
              </span>
            </div>
            <Link
              href={`/admin/support/${sentTicket}`}
              className={cn(btnGhost, "h-[30px] flex-none px-3 text-[12.5px]")}
            >
              {t("admin.bug.open_thread")}
              <ArrowRight className="h-3.5 w-3.5" />
            </Link>
          </div>
        )}

        <div className="flex flex-wrap items-start gap-5">
          <form
            onSubmit={onSubmit}
            className="flex min-w-0 flex-[1_1_520px] flex-col gap-[18px] rounded-2xl border border-border bg-card p-5"
          >
            <div className="flex flex-col gap-[7px]">
              <label htmlFor="bug-title" className="text-[13px] font-medium">
                {t("admin.bug.title_label")}
              </label>
              <input
                id="bug-title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder={t("admin.bug.title_placeholder")}
                className={cn(fieldClass, "h-10")}
              />
              <span className="text-[11.5px] text-muted-foreground/70">
                {t("admin.bug.title_hint")}
              </span>
            </div>

            <div className="flex flex-col gap-[7px]">
              <div className="flex flex-wrap items-center gap-2">
                <label htmlFor="bug-desc" className="text-[13px] font-medium">
                  {t("admin.bug.desc_label")}
                </label>
                <span className="text-[11.5px] text-muted-foreground/70">
                  {t("admin.bug.desc_hint")}
                </span>
                <button
                  type="button"
                  onClick={() => setDesc(t("admin.bug.template"))}
                  className={cn(btnGhost, "ms-auto h-7 px-2.5 text-[12px]")}
                >
                  <FileText className="h-3.5 w-3.5" />
                  {t("admin.bug.use_template")}
                </button>
              </div>
              <textarea
                id="bug-desc"
                value={desc}
                onChange={(e) => setDesc(e.target.value)}
                placeholder={t("admin.bug.desc_placeholder")}
                className={cn(
                  fieldClass,
                  "h-auto min-h-[180px] resize-y py-3 leading-relaxed"
                )}
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
                        active
                          ? "border-ring bg-accent"
                          : "border-border bg-background"
                      )}
                    >
                      <span
                        className={cn(
                          "flex items-center gap-[7px] text-[13px]",
                          active ? "font-semibold" : "font-medium"
                        )}
                      >
                        <span
                          className="h-2 w-2 flex-none rounded-full"
                          style={{ background: item.dot }}
                        />
                        {t(item.labelKey)}
                      </span>
                      <span className="text-[11.5px] leading-snug text-muted-foreground">
                        {t(item.hintKey)}
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
                      {name}
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
                  value={node}
                  onChange={(e) => setNode(e.target.value)}
                  className={cn(fieldClass, "h-10")}
                >
                  <option value={NODE_ALL}>{t("admin.bug.node_all")}</option>
                  {(nodes.data ?? []).map((n) => (
                    <option key={n.id} value={n.id}>
                      {n.fqdn ? `${n.name} (${n.fqdn})` : n.name}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            <div className="flex flex-col gap-[9px]">
              <div className="flex items-center gap-2">
                <span className="text-[13px] font-medium">{t("admin.bug.shots")}</span>
                <span className="text-[11.5px] text-muted-foreground/70">
                  {t("admin.bug.shots_hint")}
                </span>
              </div>
              <div className="grid grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-2.5">
                {SHOT_SLOTS.map((labelKey, index) => (
                  <ShotSlot
                    key={labelKey}
                    label={t(labelKey)}
                    shot={shots[index]}
                    onPick={(file) => pickShot(index, file)}
                    onClear={() => setShot(index, null)}
                    clearLabel={t("admin.bug.shot_remove")}
                  />
                ))}
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-3.5 border-t border-border pt-3.5">
              <button
                type="submit"
                disabled={reportMutation.isPending}
                className={cn(btnPrimary, "h-10 px-[18px] text-[13.5px]")}
              >
                <Send className="h-4 w-4" />
                {reportMutation.isPending ? t("common.sending") : t("admin.bug.submit")}
              </button>
              <button
                type="button"
                onClick={saveDraft}
                className={cn(btnGhost, "h-10 text-[13px]")}
              >
                <Save className="h-4 w-4" />
                {t("admin.bug.save_draft")}
              </button>
              <span className="text-xs text-muted-foreground/70">
                {t("admin.bug.footer")}
              </span>
            </div>
          </form>

          <div className="flex min-w-[280px] flex-[0_1_340px] flex-col gap-4">
            <section className="flex flex-col gap-3.5 rounded-2xl border border-border bg-card p-[18px]">
              <div className="flex items-center gap-2">
                <Cpu className="h-4 w-4 text-muted-foreground" />
                <span className="text-[13.5px] font-semibold">{t("admin.bug.env")}</span>
                <span className="ms-auto text-[11px] text-muted-foreground/70">
                  {t("admin.bug.env_auto")}
                </span>
              </div>
              <div className="flex flex-col">
                {env.map((row) => (
                  <div
                    key={row.label}
                    className="flex items-center gap-3 border-b border-border/60 py-[7px] last:border-b-0"
                  >
                    <span className="text-[12.5px] text-muted-foreground">{row.label}</span>
                    <span className="ms-auto truncate text-end text-[12.5px] font-medium">
                      {row.value}
                    </span>
                  </div>
                ))}
              </div>
              <label className="flex cursor-pointer items-center gap-2.5">
                <Switch checked={attachLogs} onCheckedChange={setAttachLogs} />
                <span className="text-[12.5px]">
                  {t("admin.bug.attach_logs", { minutes: LOG_WINDOW_MIN })}
                </span>
              </label>
            </section>

            <section className="flex flex-col gap-3 rounded-2xl border border-border bg-card p-[18px]">
              <div className="flex items-center gap-2">
                <CopyCheck className="h-4 w-4 text-muted-foreground" />
                <span className="text-[13.5px] font-semibold">
                  {t("admin.bug.similar")}
                </span>
              </div>
              <p className="text-xs leading-relaxed text-muted-foreground/70">
                {t("admin.bug.similar_hint")}
              </p>
              {reports.isError ? (
                <span className="text-[11.5px]" style={{ color: "var(--vx-danger)" }}>
                  {t("admin.bug.similar_failed", {
                    message:
                      reports.error instanceof Error
                        ? reports.error.message
                        : String(reports.error),
                  })}
                </span>
              ) : similar.length === 0 ? (
                <span className="text-[12px] text-muted-foreground/70">
                  {t("admin.bug.similar_empty")}
                </span>
              ) : (
                <div className="flex flex-col gap-2">
                  {similar.map((report) => (
                    <SimilarRow key={report.id} report={report} />
                  ))}
                </div>
              )}
            </section>

            <section className="flex flex-col gap-3 rounded-2xl border border-border bg-card p-[18px]">
              <div className="flex items-center gap-2">
                <LifeBuoy className="h-4 w-4 text-muted-foreground" />
                <span className="text-[13.5px] font-semibold">{t("admin.bug.urgent")}</span>
              </div>
              <p className="text-[12.5px] leading-relaxed text-muted-foreground">
                {t("admin.bug.urgent_hint")}
              </p>
              <div className="flex flex-col gap-[7px]">
                <Link href="/admin/nodes" className={cn(btnGhost, "h-[34px] justify-start text-[12.5px]")}>
                  <Activity className="h-3.5 w-3.5" />
                  {t("admin.bug.nodes_status")}
                  <ExternalLink className="ms-auto h-3 w-3 text-muted-foreground/70" />
                </Link>
                <Link
                  href="/admin/support/kb"
                  className={cn(btnGhost, "h-[34px] justify-start text-[12.5px]")}
                >
                  <BookOpen className="h-3.5 w-3.5" />
                  {t("admin.bug.kb")}
                  <ExternalLink className="ms-auto h-3 w-3 text-muted-foreground/70" />
                </Link>
              </div>
            </section>
          </div>
        </div>
      </div>
    </PageShell>
  );
}

function SimilarRow({ report }: { report: AdminBugReport }) {
  return (
    <Link
      href={`/admin/support/${report.id}`}
      className="flex flex-col gap-1.5 rounded-xl border border-border bg-background px-3 py-[11px] transition-colors hover:border-ring"
    >
      <span className="text-[12.5px] leading-snug font-medium">{report.title}</span>
      <span className="flex items-center gap-2 text-[11.5px] text-muted-foreground">
        <span className="font-mono">#{report.id.slice(0, 6)}</span>
        {report.component && (
          <>
            <span className="h-[3px] w-[3px] rounded-full bg-muted-foreground/50" />
            <span className="truncate">{report.component}</span>
          </>
        )}
        <SupportStatusPill status={report.status} className="ms-auto flex-none" />
      </span>
    </Link>
  );
}

function ShotSlot({
  label,
  shot,
  onPick,
  onClear,
  clearLabel,
}: {
  label: string;
  shot: Shot | null;
  onPick: (file: File) => void;
  onClear: () => void;
  clearLabel: string;
}) {
  const inputRef = useRef<HTMLInputElement>(null);

  return (
    <div className="relative h-[104px]">
      <button
        type="button"
        onClick={() => inputRef.current?.click()}
        className="flex h-full w-full flex-col items-center justify-center gap-1.5 overflow-hidden rounded-xl border border-dashed border-border bg-background px-2 text-center transition-colors hover:border-ring"
      >
        {shot ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={shot.url} alt={label} className="h-full w-full object-cover" />
        ) : (
          <>
            <ImagePlus className="h-4 w-4 text-muted-foreground" />
            <span className="text-[11.5px] text-muted-foreground">{label}</span>
          </>
        )}
      </button>
      {shot && (
        <button
          type="button"
          onClick={onClear}
          aria-label={clearLabel}
          className="absolute end-1.5 top-1.5 flex h-6 w-6 items-center justify-center rounded-lg border border-border bg-card text-muted-foreground transition-colors hover:text-foreground"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      )}
      <input
        ref={inputRef}
        type="file"
        accept={SHOT_TYPES}
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) onPick(file);
          e.target.value = "";
        }}
      />
    </div>
  );
}
