"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import { confirmAction, promptAction } from "@/components/action-dialog";
import { Btn, EmptyState, VX_INPUT, VX_MUTED, btnClass } from "@/components/vx/panel-ui";
import {
  addProjectMember,
  assignProjectServers,
  deleteProject,
  fetchProject,
  fetchProjectAvailableServers,
  removeProjectMember,
  removeProjectServers,
  updateProject,
  type ProjectMember,
  type ProjectServer,
} from "@/lib/api";
import { formatAmount } from "@/lib/format";
import { t } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

const CARD = "rounded-[14px] border border-[var(--vx-border)] bg-[var(--vx-card)]";
const DIM = "text-[var(--vx-faint)]";

type Tab = "servers" | "members";

const PERMISSION_PRESET: { key: string; labelKey: string; keys: string[] }[] = [
  { key: "view", labelKey: "projects.perm_view", keys: ["can_view_main", "can_view_metrics", "can_view_logs"] },
  { key: "power", labelKey: "projects.perm_power", keys: ["can_start", "can_stop", "can_restart"] },
  {
    key: "console",
    labelKey: "projects.perm_console",
    keys: ["can_view_console", "can_console_command"],
  },
  { key: "files", labelKey: "projects.perm_files", keys: ["can_view_ftp", "can_files"] },
  {
    key: "settings",
    labelKey: "projects.perm_settings",
    keys: ["can_view_settings", "can_settings_edit"],
  },
];

function money(value: number, currency = "RUB"): string {
  const symbol = currency === "RUB" || !currency ? "₽" : currency;
  return `${formatAmount(value, 0)} ${symbol}`;
}

function statusTone(status: string): string {
  if (status === "running") return "text-[var(--vx-ok)]";
  if (status === "error" || status === "failed") return "text-[var(--vx-danger)]";
  return "text-[var(--vx-muted)]";
}

export function ProjectDetailContent() {
  useT();
  const params = useParams<{ id: string }>();
  const id = String(params?.id ?? "");
  const router = useRouter();
  const queryClient = useQueryClient();
  const [tab, setTab] = useState<Tab>("servers");
  const [memberEmail, setMemberEmail] = useState("");
  const [memberPerms, setMemberPerms] = useState<string[]>(["view", "power", "console"]);
  const [picker, setPicker] = useState(false);
  const [picked, setPicked] = useState<string[]>([]);

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.project(id),
    queryFn: () => fetchProject(id),
    enabled: !!id,
  });

  const available = useQuery({
    queryKey: ["project-available-servers"],
    queryFn: fetchProjectAvailableServers,
    enabled: picker,
  });

  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.project(id) });
    void queryClient.invalidateQueries({ queryKey: queryKeys.projects });
    void queryClient.invalidateQueries({ queryKey: ["project-available-servers"] });
  };

  const updateMutation = useMutation({
    mutationFn: (payload: { name?: string; comment?: string }) => updateProject(id, payload),
    onSuccess: () => {
      toast.success(t("projects.updated"));
      refresh();
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteProject(id),
    onSuccess: () => {
      toast.success(t("projects.deleted"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.projects });
      router.push("/projects");
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("projects.delete_failed")),
  });

  const assignMutation = useMutation({
    mutationFn: (serverIds: string[]) => assignProjectServers(id, serverIds),
    onSuccess: () => {
      toast.success(t("projects.servers_added"));
      setPicker(false);
      setPicked([]);
      refresh();
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  const removeServerMutation = useMutation({
    mutationFn: (serverId: string) => removeProjectServers(id, [serverId]),
    onSuccess: () => {
      toast.success(t("projects.server_removed"));
      refresh();
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  const addMemberMutation = useMutation({
    mutationFn: () =>
      addProjectMember(id, {
        email: memberEmail.trim(),
        permissions: permissionsFromPreset(memberPerms),
      }),
    onSuccess: () => {
      toast.success(t("projects.member_added"));
      setMemberEmail("");
      refresh();
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("projects.member_add_failed")),
  });

  const removeMemberMutation = useMutation({
    mutationFn: (userId: string) => removeProjectMember(id, userId),
    onSuccess: () => {
      toast.success(t("projects.member_removed"));
      refresh();
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  const servers = data?.servers ?? [];
  const members = data?.members ?? [];
  const monthly = useMemo(
    () => servers.reduce((sum, s) => sum + Number(s.monthly_cost ?? 0), 0),
    [servers]
  );

  async function onRename() {
    const name = await promptAction(t("projects.rename_prompt"), {
      title: t("projects.rename"),
      confirmText: t("common.save"),
      defaultValue: data?.project.name ?? "",
    });
    if (!name || !name.trim()) return;
    updateMutation.mutate({ name: name.trim() });
  }

  async function onComment() {
    const comment = await promptAction(t("projects.comment_prompt"), {
      title: t("projects.edit_comment"),
      confirmText: t("common.save"),
      defaultValue: data?.project.comment ?? "",
    });
    if (comment === null) return;
    updateMutation.mutate({ comment });
  }

  async function onDelete() {
    const ok = await confirmAction(
      t("projects.delete_confirm", { name: data?.project.name ?? "" }),
      { title: t("projects.delete"), confirmText: t("common.delete"), destructive: true }
    );
    if (ok) deleteMutation.mutate();
  }

  async function onRemoveServer(server: ProjectServer) {
    const ok = await confirmAction(
      t("projects.remove_server_confirm", { name: server.name }),
      { title: t("projects.remove_server"), confirmText: t("projects.remove") }
    );
    if (ok) removeServerMutation.mutate(server.id);
  }

  async function onRemoveMember(member: ProjectMember) {
    const ok = await confirmAction(
      t("projects.member_remove_confirm", { email: member.email }),
      { title: t("projects.tab_members"), confirmText: t("projects.remove") }
    );
    if (ok) removeMemberMutation.mutate(member.user_id);
  }

  if (isLoading) {
    return (
      <PageShell variant="user">
        <div className="flex w-full flex-col gap-4">
          <Skeleton className="h-[120px] w-full rounded-[14px]" />
          <Skeleton className="h-[320px] w-full rounded-[14px]" />
        </div>
      </PageShell>
    );
  }

  if (isError || !data) {
    return (
      <PageShell variant="user">
        <div className={cn(CARD, "flex flex-col items-center gap-3 px-5 py-14 text-center")}>
          <i className="ri-error-warning-line text-[28px] text-[var(--vx-ghost)]" />
          <span className="text-[15px] font-semibold">{t("common.not_found")}</span>
          <Link href="/projects" className={btnClass("primary", "sm")}>
            {t("projects.title")}
          </Link>
        </div>
      </PageShell>
    );
  }

  return (
    <PageShell variant="user">
      <div className="font-panel flex w-full flex-col gap-4">
        <div className={cn(CARD, "flex flex-wrap items-center gap-4 p-5")}>
          <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-[12px] bg-[var(--vx-tint)]">
            <i className="ri-folder-3-line text-[22px]" />
          </span>
          <div className="min-w-[220px] flex-1">
            <h1 className="m-0 text-[20px] font-bold">{data.project.name}</h1>
            <p className={cn("mt-1 text-[13px]", VX_MUTED)}>
              {data.project.comment || t("projects.no_comment")}
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Btn onClick={() => void onRename()}>{t("projects.rename")}</Btn>
            <Btn onClick={() => void onComment()}>{t("projects.edit_comment")}</Btn>
            <Btn tone="danger" onClick={() => void onDelete()} disabled={deleteMutation.isPending}>
              {t("common.delete")}
            </Btn>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
          <Stat label={t("projects.tab_servers")} value={String(servers.length)} />
          <Stat label={t("projects.tab_members")} value={String(members.length)} />
          <Stat
            label={t("projects.monthly_label")}
            value={money(monthly, data.project.currency)}
          />
        </div>

        <div className={cn(CARD, "overflow-hidden")}>
          <div className="flex flex-wrap items-center gap-x-1 gap-y-2.5 border-b border-[var(--vx-border)] px-3 py-2.5">
            {(
              [
                { id: "servers" as const, label: t("projects.tab_servers") },
                { id: "members" as const, label: t("projects.tab_members") },
              ] satisfies { id: Tab; label: string }[]
            ).map((item) => (
              <button
                key={item.id}
                type="button"
                onClick={() => setTab(item.id)}
                className={cn(
                  "h-8 rounded-[8px] px-3 text-[13px] font-medium transition-colors",
                  tab === item.id
                    ? "bg-[var(--vx-tint)] text-[var(--vx-fg)]"
                    : cn(VX_MUTED, "hover:text-[var(--vx-fg)]")
                )}
              >
                {item.label}
              </button>
            ))}
            {tab === "servers" && (
              <div className="flex w-full items-center gap-2 sm:ml-auto sm:w-auto">
                <Btn className="flex-1 sm:flex-none" onClick={() => setPicker((v) => !v)}>
                  <i className="ri-add-line" />
                  {t("projects.add_servers")}
                </Btn>
                <Link
                  href="/rent-server"
                  className={cn(btnClass("primary", "sm"), "flex-1 justify-center sm:flex-none")}
                >
                  {t("projects.order_here")}
                </Link>
              </div>
            )}
          </div>

          {tab === "servers" ? (
            <>
              {picker && (
                <div className="border-b border-[var(--vx-border)] bg-[var(--vx-card-2)] p-4">
                  <div className="mb-2.5 text-[13px] font-semibold">
                    {t("projects.add_servers_title")}
                  </div>
                  {available.isLoading ? (
                    <Skeleton className="h-16 w-full rounded-[10px]" />
                  ) : (available.data?.servers ?? []).length === 0 ? (
                    <EmptyState>{t("projects.add_servers_empty")}</EmptyState>
                  ) : (
                    <>
                      <div className="flex flex-wrap gap-2">
                        {(available.data?.servers ?? []).map((server) => {
                          const on = picked.includes(server.id);
                          return (
                            <button
                              key={server.id}
                              type="button"
                              onClick={() =>
                                setPicked((prev) =>
                                  on ? prev.filter((x) => x !== server.id) : [...prev, server.id]
                                )
                              }
                              className={cn(
                                "flex items-center gap-2 rounded-[8px] border px-3 py-2 text-[12.5px] transition-colors",
                                on
                                  ? "border-[var(--vx-fg-strong)] bg-[var(--vx-tint)]"
                                  : "border-[var(--vx-border)] bg-[var(--vx-bg)]"
                              )}
                            >
                              <i className={on ? "ri-checkbox-line" : "ri-checkbox-blank-line"} />
                              <span className="truncate">{server.name}</span>
                              <span className={cn("font-mono text-[11px]", DIM)}>
                                {server.game_name}
                              </span>
                            </button>
                          );
                        })}
                      </div>
                      <Btn
                        tone="primary"
                        className="mt-3"
                        disabled={picked.length === 0 || assignMutation.isPending}
                        onClick={() => assignMutation.mutate(picked)}
                      >
                        {t("projects.add_selected")}
                      </Btn>
                    </>
                  )}
                </div>
              )}

              {servers.length === 0 ? (
                <div className="flex flex-col items-center gap-2.5 px-5 py-12 text-center">
                  <i className="ri-server-line text-[26px] text-[var(--vx-ghost)]" />
                  <span className="text-[14px] font-semibold">
                    {t("projects.no_servers_title")}
                  </span>
                  <span className={cn("max-w-[400px] text-[12.5px]", VX_MUTED)}>
                    {t("projects.no_servers_text")}
                  </span>
                </div>
              ) : (
                servers.map((server) => (
                  <div
                    key={server.id}
                    className="flex flex-wrap items-center gap-3 border-b border-[var(--vx-divider)] px-4 py-3 last:border-b-0"
                  >
                    <span className="flex min-w-[180px] flex-1 flex-col max-sm:basis-full">
                      <span className="truncate text-[13.5px] font-medium">{server.name}</span>
                      <span className={cn("truncate font-mono text-[11px]", DIM)}>
                        {server.game_name} · {server.location || "—"}
                        {server.ip ? ` · ${server.ip}:${server.port}` : ""}
                      </span>
                    </span>
                    <span className={cn("font-mono text-[12px]", statusTone(server.status))}>
                      {server.status}
                    </span>
                    <span className="font-mono text-[12.5px]">
                      {money(server.monthly_cost, data.project.currency)}
                    </span>
                    <Link
                      href={`/servers/${server.id}`}
                      className={cn(btnClass("default", "sm"), "max-sm:ml-auto")}
                    >
                      {t("projects.open_server")}
                    </Link>
                    <Btn
                      size="sm"
                      onClick={() => void onRemoveServer(server)}
                      disabled={removeServerMutation.isPending}
                      aria-label={t("projects.remove_server")}
                    >
                      <i className="ri-close-line" />
                    </Btn>
                  </div>
                ))
              )}
            </>
          ) : (
            <>
              <div className="border-b border-[var(--vx-border)] p-4">
                <div className={cn("mb-2.5 text-[12.5px]", VX_MUTED)}>
                  {t("projects.members_hint")}
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  <input
                    className={cn(VX_INPUT, "h-9 min-w-[220px] flex-1")}
                    value={memberEmail}
                    onChange={(e) => setMemberEmail(e.target.value)}
                    placeholder={t("projects.member_email")}
                  />
                  <Btn
                    tone="primary"
                    disabled={!memberEmail.trim() || addMemberMutation.isPending}
                    onClick={() => addMemberMutation.mutate()}
                  >
                    {t("projects.member_add")}
                  </Btn>
                </div>
                <div className="mt-3 flex flex-wrap items-center gap-2">
                  <span className={cn("text-[12px]", VX_MUTED)}>{t("projects.perms_title")}</span>
                  {PERMISSION_PRESET.map((preset) => {
                    const on = memberPerms.includes(preset.key);
                    return (
                      <button
                        key={preset.key}
                        type="button"
                        onClick={() =>
                          setMemberPerms((prev) =>
                            on ? prev.filter((x) => x !== preset.key) : [...prev, preset.key]
                          )
                        }
                        className={cn(
                          "rounded-full border px-3 py-1 text-[12px] transition-colors",
                          on
                            ? "border-[var(--vx-fg-strong)] bg-[var(--vx-tint)] text-[var(--vx-fg)]"
                            : cn("border-[var(--vx-border)]", VX_MUTED)
                        )}
                      >
                        {t(preset.labelKey)}
                      </button>
                    );
                  })}
                </div>
              </div>

              {members.length === 0 ? (
                <div className="flex flex-col items-center gap-2.5 px-5 py-12 text-center">
                  <i className="ri-user-add-line text-[26px] text-[var(--vx-ghost)]" />
                  <span className="text-[14px] font-semibold">
                    {t("projects.no_members_title")}
                  </span>
                  <span className={cn("max-w-[400px] text-[12.5px]", VX_MUTED)}>
                    {t("projects.no_members_text")}
                  </span>
                </div>
              ) : (
                members.map((member) => (
                  <div
                    key={member.user_id}
                    className="flex flex-wrap items-center gap-3 border-b border-[var(--vx-divider)] px-4 py-3 last:border-b-0"
                  >
                    <span className="min-w-[200px] flex-1 truncate font-mono text-[13px]">
                      {member.email}
                    </span>
                    <span className={cn("flex flex-wrap gap-1.5 text-[11px]", VX_MUTED)}>
                      {PERMISSION_PRESET.filter((preset) =>
                        preset.keys.some((key) => member.permissions?.[key])
                      ).map((preset) => (
                        <span
                          key={preset.key}
                          className="rounded-full bg-[var(--vx-veil)] px-2 py-0.5"
                        >
                          {t(preset.labelKey)}
                        </span>
                      ))}
                    </span>
                    <Btn
                      size="sm"
                      onClick={() => void onRemoveMember(member)}
                      disabled={removeMemberMutation.isPending}
                      aria-label={t("projects.remove")}
                    >
                      <i className="ri-close-line" />
                    </Btn>
                  </div>
                ))
              )}
            </>
          )}
        </div>
      </div>
    </PageShell>
  );
}

function permissionsFromPreset(active: string[]): Record<string, boolean> {
  const out: Record<string, boolean> = {};
  for (const preset of PERMISSION_PRESET) {
    if (!active.includes(preset.key)) continue;
    for (const key of preset.keys) out[key] = true;
  }
  out.can_view_main = true;
  return out;
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className={cn(CARD, "px-4 py-3")}>
      <div className={cn("truncate text-[11.5px]", VX_MUTED)}>{label}</div>
      <div className="mt-1 font-mono text-[18px] font-semibold">{value}</div>
    </div>
  );
}
