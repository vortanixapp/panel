"use client";

import { useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  ArrowLeft,
  Copy,
  Database,
  ExternalLink,
  Eye,
  EyeOff,
  Globe,
  LayoutDashboard,
  Mail,
  RefreshCw,
  Settings,
  type LucideIcon,
} from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import {
  fetchHostingAccount,
  createHostingDomain,
  createHostingDatabase,
  createHostingEmail,
  renewHostingAccount,
  changeHostingPassword,
} from "@/lib/api";
import {
  formatHostingDate,
  formatHostingDateTime,
  getHostingDisplayDomain,
  getHostingStatusLabel,
  isHostingExpired,
  planDiskLabel,
} from "@/lib/hosting-status";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import {
  StatCell,
  StatStrip,
  StatusPill,
  UsageBar,
  btnGhost,
  btnPrimary,
  daysLeft,
  fieldClass,
  pluralDays,
  usagePct,
} from "@/components/user/panel-parts";

type HTab = "overview" | "domains" | "databases" | "emails" | "settings";

const TABS: { id: HTab; Icon: LucideIcon; labelKey: string }[] = [
  { id: "overview", Icon: LayoutDashboard, labelKey: "common.overview" },
  { id: "domains", Icon: Globe, labelKey: "billing.hosting.detail.tab_domains" },
  { id: "databases", Icon: Database, labelKey: "billing.hosting.detail.tab_databases" },
  { id: "emails", Icon: Mail, labelKey: "billing.hosting.detail.tab_emails" },
  { id: "settings", Icon: Settings, labelKey: "common.settings" },
];

function Card({
  title,
  hint,
  children,
}: {
  title: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-[14px] rounded-2xl border border-border bg-card px-[22px] py-5">
      <div className="flex flex-col gap-1">
        <span className="text-[15px] font-semibold">{title}</span>
        {hint && <span className="text-[12.5px] text-muted-foreground">{hint}</span>}
      </div>
      {children}
    </div>
  );
}

export function HostingDetailPageContent() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const [tab, setTab] = useState<HTab>("overview");
  const [showPassword, setShowPassword] = useState(false);

  const [newDomain, setNewDomain] = useState("");
  const [newDbName, setNewDbName] = useState("");
  const [newDbUser, setNewDbUser] = useState("");
  const [newEmail, setNewEmail] = useState("");
  const [newEmailPass, setNewEmailPass] = useState("");
  const [renewPeriod, setRenewPeriod] = useState(30);
  const [newPanelPass, setNewPanelPass] = useState("");

  const qc = useQueryClient();
  const accountKey = queryKeys.hostingAccount(id ?? "");
  const refresh = () => void qc.invalidateQueries({ queryKey: accountKey });

  const { data: account, isLoading, isError } = useQuery({
    queryKey: accountKey,
    queryFn: () => fetchHostingAccount(id!),
    enabled: !!id,
  });

  const addDomainMut = useMutation({
    mutationFn: (domain: string) => createHostingDomain(id!, domain),
    onSuccess: () => {
      toast.success(t("billing.hosting.detail.domain_added"));
      setNewDomain("");
      refresh();
    },
    onError: (e: Error) =>
      toast.error(e.message || t("billing.hosting.detail.domain_add_failed")),
  });

  const addDbMut = useMutation({
    mutationFn: (p: { name: string; user?: string }) =>
      createHostingDatabase(id!, p.name, p.user),
    onSuccess: () => {
      toast.success(t("billing.hosting.detail.db_added"));
      setNewDbName("");
      setNewDbUser("");
      refresh();
    },
    onError: (e: Error) =>
      toast.error(e.message || t("billing.hosting.detail.db_add_failed")),
  });

  const addEmailMut = useMutation({
    mutationFn: (p: { address: string; password: string }) =>
      createHostingEmail(id!, p.address, p.password),
    onSuccess: () => {
      toast.success(t("billing.hosting.detail.email_added"));
      setNewEmail("");
      setNewEmailPass("");
      refresh();
    },
    onError: (e: Error) =>
      toast.error(e.message || t("billing.hosting.detail.email_add_failed")),
  });

  const renewMut = useMutation({
    mutationFn: (period: number) => renewHostingAccount(id!, period),
    onSuccess: () => {
      toast.success(t("billing.hosting.detail.renewed"));
      refresh();
    },
    onError: (e: Error) =>
      toast.error(e.message || t("billing.hosting.detail.renew_failed")),
  });

  const changePassMut = useMutation({
    mutationFn: (password: string) => changeHostingPassword(id!, password),
    onSuccess: () => {
      toast.success(t("billing.hosting.detail.password_changed"));
      setNewPanelPass("");
      refresh();
    },
    onError: (e: Error) =>
      toast.error(e.message || t("billing.hosting.detail.password_change_failed")),
  });

  const copyPassword = async () => {
    if (!account?.panel_password) {
      toast.error(t("billing.hosting.detail.password_unavailable"));
      return;
    }
    try {
      await navigator.clipboard.writeText(account.panel_password);
      toast.success(t("common.copied"));
    } catch {
      toast.error(t("billing.hosting.detail.copy_password_failed"));
    }
  };

  if (isLoading) {
    return (
      <PageShell variant="user">
        <div className="flex flex-col gap-[22px]">
          <Skeleton className="h-12 w-80 rounded-xl" />
          <Skeleton className="h-[74px] rounded-[14px]" />
          <Skeleton className="h-11 rounded-xl" />
          <Skeleton className="h-72 rounded-2xl" />
        </div>
      </PageShell>
    );
  }

  if (isError || !account) {
    return (
      <PageShell variant="user">
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <span className="flex h-14 w-14 items-center justify-center rounded-2xl bg-muted">
            <Globe className="h-6 w-6 text-muted-foreground" />
          </span>
          <p className="text-sm text-muted-foreground">
            {t("billing.hosting.detail.not_found")}
          </p>
          <Link href="/hosting/my" className={cn(btnGhost, "h-[34px]")}>
            {t("billing.hosting.my.title")}
          </Link>
        </div>
      </PageShell>
    );
  }

  const plan = account.hosting_plan;
  const domain = getHostingDisplayDomain(account);
  const statusLabel = getHostingStatusLabel(account.status, account.status_label);
  const expired = isHostingExpired(account.expires_at);
  const left = daysLeft(account.expires_at);

  const sites = account.domains?.length ?? 0;
  const dbs = account.databases?.length ?? 0;
  const emails = account.emails?.length ?? 0;
  const limit = (n: number | null | undefined) => (n == null || n <= 0 ? "∞" : String(n));

  const lists = {
    domains: {
      title: t("billing.hosting.detail.tab_domains"),
      hint: t("billing.hosting.detail.domains_hint"),
      placeholder: "example.com",
      Icon: Globe,
      rows: (account.domains ?? []).map((d) => ({
        key: String(d.id),
        name: d.name,
        meta: [
          d.is_primary ? t("billing.hosting.detail.primary") : null,
          d.status !== "active" ? d.status : null,
        ]
          .filter(Boolean)
          .join(" · "),
      })),
    },
    databases: {
      title: t("billing.hosting.detail.tab_databases"),
      hint: t("billing.hosting.detail.databases_hint"),
      placeholder: "db_name",
      Icon: Database,
      rows: (account.databases ?? []).map((d) => ({
        key: String(d.id),
        name: d.name,
        meta: [d.user, d.status !== "active" ? d.status : null].filter(Boolean).join(" · "),
      })),
    },
    emails: {
      title: t("billing.hosting.detail.emails_title"),
      hint: t("billing.hosting.detail.emails_hint"),
      placeholder: "user@domain",
      Icon: Mail,
      rows: (account.emails ?? []).map((e) => ({
        key: String(e.id),
        name: e.address,
        meta: e.status !== "active" ? (e.status ?? "") : "",
      })),
    },
  } as const;

  const credRows = [
    { label: t("billing.hosting.rent.domain"), value: domain },
    { label: t("billing.hosting.detail.login"), value: account.username },
    { label: t("billing.hosting.detail.ip"), value: account.ip_address || "—" },
    {
      label: t("billing.hosting.detail.panel"),
      value: account.hosting_server?.panel_type_label || "—",
    },
    { label: t("common.tariff"), value: plan?.name || "—" },
    {
      label: t("common.created_at"),
      value: account.created_at ? formatHostingDateTime(account.created_at) : "—",
    },
  ];

  return (
    <PageShell variant="user">
      <div className="flex w-full flex-col gap-[22px]">
        <div className="flex flex-wrap items-center gap-[14px]">
          <Link
            href="/hosting/my"
            className="flex h-[34px] w-[34px] flex-none items-center justify-center rounded-[9px] border border-input text-foreground transition-colors hover:bg-accent"
            aria-label={t("billing.hosting.back_aria")}
          >
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2.5">
              <h1 className="text-[26px] leading-none font-bold tracking-[-0.02em]">
                {domain}
              </h1>
              <StatusPill status={account.status} label={statusLabel} />
            </div>
            <p className="mt-1.5 text-[13.5px] text-muted-foreground">
              {[plan?.name, account.hosting_server?.panel_type_label]
                .filter(Boolean)
                .join(" · ") || "—"}
            </p>
          </div>
          <div className="ms-auto flex items-center gap-2">
            <button
              type="button"
              onClick={() => setTab("settings")}
              className={cn(btnGhost, "h-[38px] text-[13px]")}
            >
              <RefreshCw className="h-3.5 w-3.5" />
              {t("billing.hosting.detail.renew")}
            </button>
            {account.panel_login_url && (
              <a
                href={account.panel_login_url}
                target="_blank"
                rel="noopener noreferrer"
                className={cn(btnPrimary, "h-[38px] px-[18px]")}
              >
                <ExternalLink className="h-3.5 w-3.5" />
                {t("billing.hosting.detail.control_panel")}
              </a>
            )}
          </div>
        </div>

        <StatStrip>
          <StatCell label={t("billing.hosting.disk")} value={planDiskLabel(plan)} mono />
          <StatCell
            label={t("billing.hosting.detail.ip")}
            value={account.ip_address || "—"}
            mono
          />
          <StatCell label={t("billing.hosting.detail.login")} value={account.username} mono />
          <StatCell
            label={t("billing.hosting.expires")}
            value={formatHostingDate(account.expires_at)}
            tone={expired || (left != null && left <= 7) ? "warn" : undefined}
            mono
          />
        </StatStrip>

        <div className="no-scrollbar flex gap-1 overflow-x-auto rounded-xl border border-border bg-card p-1">
          {TABS.map(({ id: tid, Icon, labelKey }) => (
            <button
              key={tid}
              type="button"
              onClick={() => setTab(tid)}
              aria-pressed={tab === tid}
              className={cn(
                "flex h-9 flex-none items-center gap-[7px] rounded-[9px] px-3.5 text-[13px] whitespace-nowrap transition-colors",
                tab === tid
                  ? "bg-accent font-medium text-accent-foreground"
                  : "text-muted-foreground hover:text-foreground"
              )}
            >
              <Icon className="h-4 w-4" />
              {t(labelKey)}
            </button>
          ))}
        </div>

        {tab === "overview" && (
          <div className="grid grid-cols-[repeat(auto-fit,minmax(340px,1fr))] gap-4">
            <Card title={t("billing.hosting.detail.credentials")}>
              <div className="flex flex-col">
                {credRows.map((r) => (
                  <div
                    key={r.label}
                    className="flex items-center justify-between gap-3 border-b border-border/60 py-[11px] text-[13px]"
                  >
                    <span className="text-muted-foreground">{r.label}</span>
                    <span className="truncate font-mono">{r.value}</span>
                  </div>
                ))}
                <div className="flex items-center justify-between gap-3 py-[11px] text-[13px]">
                  <span className="text-muted-foreground">{t("common.password")}</span>
                  <span className="flex items-center gap-2">
                    <span className="font-mono">
                      {showPassword ? account.panel_password || "—" : "••••••••••"}
                    </span>
                    <button
                      type="button"
                      onClick={() => setShowPassword(!showPassword)}
                      aria-label={
                        showPassword
                          ? t("billing.hosting.detail.hide_password")
                          : t("billing.hosting.detail.show_password")
                      }
                      className="flex h-7 w-7 items-center justify-center rounded-lg border border-input text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                    >
                      {showPassword ? (
                        <EyeOff className="h-3.5 w-3.5" />
                      ) : (
                        <Eye className="h-3.5 w-3.5" />
                      )}
                    </button>
                    <button
                      type="button"
                      onClick={copyPassword}
                      aria-label={t("billing.hosting.detail.copy_password")}
                      className="flex h-7 w-7 items-center justify-center rounded-lg border border-input text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                    >
                      <Copy className="h-3.5 w-3.5" />
                    </button>
                  </span>
                </div>
              </div>
            </Card>

            <Card title={t("billing.hosting.detail.usage_title")}>
              <div className="flex flex-col gap-[14px]">
                <UsageBar
                  label={t("billing.hosting.disk")}
                  value={planDiskLabel(plan)}
                  pct={null}
                />
                <UsageBar
                  label={t("billing.hosting.detail.sites")}
                  value={`${sites} / ${limit(plan?.max_domains)}`}
                  pct={usagePct(sites, plan?.max_domains)}
                />
                <UsageBar
                  label={t("billing.hosting.detail.tab_databases")}
                  value={`${dbs} / ${limit(plan?.max_databases)}`}
                  pct={usagePct(dbs, plan?.max_databases)}
                />
                <UsageBar
                  label={t("billing.hosting.detail.tab_emails")}
                  value={`${emails} / ${limit(plan?.max_email_accounts)}`}
                  pct={usagePct(emails, plan?.max_email_accounts)}
                />
                <UsageBar
                  label={t("billing.hosting.detail.traffic")}
                  value={
                    (plan?.bandwidth_gb ?? 0) > 0
                      ? `${plan?.bandwidth_gb} GB`
                      : t("billing.hosting.detail.unlimited")
                  }
                  pct={null}
                />
              </div>
              <div className="flex flex-wrap gap-1.5 pt-1.5">
                {[
                  plan?.has_ssl && "SSL",
                  plan?.has_ssh && "SSH",
                  plan?.has_cron && "Cron",
                  plan?.has_backup && t("billing.hosting.tag_backup"),
                ]
                  .filter(Boolean)
                  .map((tag) => (
                    <span
                      key={String(tag)}
                      className="rounded-[7px] bg-muted px-[9px] py-[3px] text-[11.5px] text-[var(--vx-ink-dim)]"
                    >
                      {tag}
                    </span>
                  ))}
              </div>
            </Card>
          </div>
        )}

        {(tab === "domains" || tab === "databases" || tab === "emails") && (
          <Card title={lists[tab].title} hint={lists[tab].hint}>
            <div className="flex flex-wrap gap-2">
              {tab === "domains" && (
                <input
                  type="text"
                  value={newDomain}
                  onChange={(e) => setNewDomain(e.target.value)}
                  placeholder={lists.domains.placeholder}
                  className={cn(fieldClass, "min-w-[220px] flex-1")}
                />
              )}
              {tab === "databases" && (
                <>
                  <input
                    type="text"
                    value={newDbName}
                    onChange={(e) => setNewDbName(e.target.value)}
                    placeholder={lists.databases.placeholder}
                    className={cn(fieldClass, "min-w-[220px] flex-1")}
                  />
                  <input
                    type="text"
                    value={newDbUser}
                    onChange={(e) => setNewDbUser(e.target.value)}
                    placeholder={t("billing.hosting.detail.db_user_placeholder")}
                    className={cn(fieldClass, "min-w-[180px] flex-1")}
                  />
                </>
              )}
              {tab === "emails" && (
                <>
                  <input
                    type="text"
                    value={newEmail}
                    onChange={(e) => setNewEmail(e.target.value)}
                    placeholder={lists.emails.placeholder}
                    className={cn(fieldClass, "min-w-[220px] flex-1")}
                  />
                  <input
                    type="password"
                    value={newEmailPass}
                    onChange={(e) => setNewEmailPass(e.target.value)}
                    placeholder={t("billing.hosting.detail.email_password_placeholder")}
                    className={cn(fieldClass, "min-w-[180px] flex-1")}
                  />
                </>
              )}
              <button
                type="button"
                disabled={
                  tab === "domains"
                    ? !newDomain.trim() || addDomainMut.isPending
                    : tab === "databases"
                      ? !newDbName.trim() || addDbMut.isPending
                      : !newEmail.trim() || !newEmailPass || addEmailMut.isPending
                }
                onClick={() => {
                  if (tab === "domains") addDomainMut.mutate(newDomain.trim());
                  else if (tab === "databases")
                    addDbMut.mutate({
                      name: newDbName.trim(),
                      user: newDbUser.trim() || undefined,
                    });
                  else addEmailMut.mutate({ address: newEmail.trim(), password: newEmailPass });
                }}
                className={cn(btnPrimary, "h-[38px] px-[18px]")}
              >
                {t("common.add")}
              </button>
            </div>

            <div className="flex flex-col gap-2">
              {lists[tab].rows.length === 0 ? (
                <p className="py-2 text-[13px] text-muted-foreground">
                  {t("billing.hosting.detail.list_empty")}
                </p>
              ) : (
                lists[tab].rows.map((r) => {
                  const RowIcon = lists[tab].Icon;
                  return (
                    <div
                      key={r.key}
                      className="flex items-center gap-3 rounded-xl border border-border bg-background px-4 py-[13px]"
                    >
                      <RowIcon className="h-4 w-4 flex-none text-muted-foreground" />
                      <span className="min-w-0 flex-1 truncate font-mono text-[13.5px]">
                        {r.name}
                      </span>
                      {r.meta && (
                        <span className="text-[11.5px] text-muted-foreground">{r.meta}</span>
                      )}
                    </div>
                  );
                })
              )}
            </div>
          </Card>
        )}

        {tab === "settings" && (
          <div className="grid grid-cols-[repeat(auto-fit,minmax(340px,1fr))] gap-4">
            <Card
              title={t("billing.hosting.detail.renew_title")}
              hint={
                left != null
                  ? t("billing.hosting.detail.renew_hint", {
                      days: pluralDays(left),
                      date: formatHostingDate(account.expires_at),
                    })
                  : t("billing.hosting.detail.renew_hint_none")
              }
            >
              <div className="flex flex-wrap gap-2">
                <select
                  value={renewPeriod}
                  onChange={(e) => setRenewPeriod(Number(e.target.value))}
                  className={cn(fieldClass, "min-w-[160px] flex-1 px-2.5")}
                >
                  {[30, 60, 90, 180, 365].map((p) => (
                    <option key={p} value={p}>
                      {t("billing.hosting.days", { days: p })}
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  onClick={() => renewMut.mutate(renewPeriod)}
                  disabled={renewMut.isPending}
                  className={cn(btnPrimary, "h-[38px] px-[18px]")}
                >
                  {renewMut.isPending
                    ? t("billing.hosting.detail.renewing")
                    : t("billing.hosting.detail.renew")}
                </button>
              </div>
            </Card>

            <Card
              title={t("billing.hosting.detail.panel_password")}
              hint={t("billing.hosting.detail.panel_password_hint")}
            >
              <div className="flex flex-wrap gap-2">
                <input
                  type="password"
                  value={newPanelPass}
                  onChange={(e) => setNewPanelPass(e.target.value)}
                  placeholder={t("billing.hosting.detail.new_password")}
                  className={cn(fieldClass, "min-w-[200px] flex-1")}
                />
                <button
                  type="button"
                  onClick={() => {
                    const p = newPanelPass.trim();
                    if (p.length < 8) {
                      toast.error(t("billing.hosting.detail.password_too_short"));
                      return;
                    }
                    changePassMut.mutate(p);
                  }}
                  disabled={!newPanelPass.trim() || changePassMut.isPending}
                  className={cn(btnGhost, "h-[38px] text-[13px]")}
                >
                  {changePassMut.isPending
                    ? t("billing.hosting.detail.changing")
                    : t("billing.hosting.detail.change")}
                </button>
              </div>
            </Card>
          </div>
        )}
      </div>
    </PageShell>
  );
}
