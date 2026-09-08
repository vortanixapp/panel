"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Btn,
  EmptyState,
  Field,
  Notice,
  Panel,
  VX_FAINT,
  VX_INPUT,
  VX_MONO_LABEL,
  Toggle,
} from "@/components/vx/panel-ui";
import { Skeleton } from "@/components/ui/skeleton";
import {
  addServerFriend,
  fetchServerFriends,
  removeServerFriend,
  updateServerFriendPermissions,
  type ServerViewerPermissions,
} from "@/lib/api";
import { isServerExpired, serverProvisioning } from "@/lib/server-lifecycle";
import { cn } from "@/lib/utils";
import { useMe, useServerDetail } from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";

type ServerFriend = {
  id: string;
  user_id: string;
  email: string;
  user_email?: string;
  user_name?: string;
  user_last_name?: string;
} & ServerViewerPermissions;

// Подписи храним ключами: списки читаются на уровне модуля, и готовый текст
// застыл бы на языке, который стоял в момент загрузки страницы.
// Права на вкладки подписаны теми же строками, что и сами вкладки.
const TAB_PERMS: { key: keyof ServerViewerPermissions; labelKey: string }[] = [
  { key: "can_view_console", labelKey: "server.tab.console" },
  { key: "can_view_logs", labelKey: "server.tab.logs" },
  { key: "can_view_metrics", labelKey: "server.tab.metrics" },
  { key: "can_view_ftp", labelKey: "server.tab.ftp" },
  { key: "can_view_mysql", labelKey: "server.tab.mysql" },
  { key: "can_view_cron", labelKey: "server.tab.cron" },
  { key: "can_view_firewall", labelKey: "server.tab.firewall" },
  { key: "can_view_ports", labelKey: "server.tab.ports" },
  { key: "can_view_settings", labelKey: "server.tab.settings" },
];

const ACTION_PERMS: { key: keyof ServerViewerPermissions; labelKey: string }[] = [
  { key: "can_start", labelKey: "servers.friends.action.start" },
  { key: "can_stop", labelKey: "servers.friends.action.stop" },
  { key: "can_restart", labelKey: "servers.friends.action.restart" },
  { key: "can_reinstall", labelKey: "servers.friends.action.reinstall" },
  { key: "can_console_command", labelKey: "servers.friends.action.console_command" },
  { key: "can_files", labelKey: "servers.friends.action.files" },
  { key: "can_cron_manage", labelKey: "server.tab.cron" },
  { key: "can_firewall_manage", labelKey: "server.tab.firewall" },
  { key: "can_ports_manage", labelKey: "server.tab.ports" },
  { key: "can_settings_edit", labelKey: "server.tab.settings" },
];

export function ServerFriendsTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const { data: server } = useServerDetail(id);
  const { data: me } = useMe();
  const [email, setEmail] = useState("");
  const [friends, setFriends] = useState<ServerFriend[]>([]);

  const { data, isLoading } = useQuery({
    queryKey: ["server-friends", id],
    queryFn: () => fetchServerFriends(id),
    enabled: !!id,
  });

  useEffect(() => {
    setFriends((data?.friends ?? []) as ServerFriend[]);
  }, [data]);

  const addMutation = useMutation({
    mutationFn: () => addServerFriend(id, email),
    onSuccess: () => {
      toast.success(t("servers.friends.added"));
      setEmail("");
      void queryClient.invalidateQueries({ queryKey: ["server-friends", id] });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.friends.add_error")
      ),
  });

  const removeMutation = useMutation({
    mutationFn: (friendId: string) => removeServerFriend(id, friendId),
    onSuccess: () => {
      toast.success(t("servers.friends.revoked"));
      void queryClient.invalidateQueries({ queryKey: ["server-friends", id] });
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("common.delete_failed")),
  });

  const saveMutation = useMutation({
    mutationFn: (friend: ServerFriend) => {
      const permissions: ServerViewerPermissions = {};
      for (const p of [...TAB_PERMS, ...ACTION_PERMS]) {
        permissions[p.key] = !!friend[p.key];
      }
      return updateServerFriendPermissions(id, friend.user_id, permissions);
    },
    onSuccess: () => {
      toast.success(t("servers.friends.perms_saved"));
      void queryClient.invalidateQueries({ queryKey: ["server-friends", id] });
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("common.save_failed")),
  });

  if (!server) return null;

  const expired = isServerExpired(server);
  const busyProvisioning = ["reinstalling", "updating"].includes(serverProvisioning(server));
  const isOwner = !!server.user_id && server.user_id === me?.user_id;

  if (expired) {
    return <Notice>{t("servers.friends.expired")}</Notice>;
  }
  if (busyProvisioning) {
    return <Notice tone="warn">{t("servers.friends.busy")}</Notice>;
  }
  if (!isOwner) {
    return (
      <Panel title={t("server.tab.friends")}>
        <EmptyState>{t("servers.friends.owner_only")}</EmptyState>
      </Panel>
    );
  }
  if (isLoading) return <Skeleton className="h-[320px] w-full rounded-[14px]" />;

  function setPerm(friendId: string, key: keyof ServerViewerPermissions, value: boolean) {
    setFriends((prev) => prev.map((f) => (f.id === friendId ? { ...f, [key]: value } : f)));
  }

  return (
    <div className="flex flex-col gap-[18px]">
      <Panel title={t("servers.friends.add_title")}>
        <div className="flex flex-wrap items-end gap-2">
          <Field
            label={t("servers.friends.email_label")}
            className="min-w-[240px] flex-1"
          >
            <input
              type="email"
              className={VX_INPUT}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="user@example.com"
            />
          </Field>
          <Btn
            tone="primary"
            className="px-4"
            onClick={() => addMutation.mutate()}
            disabled={!email.trim() || addMutation.isPending}
          >
            {t("common.add")}
          </Btn>
        </div>
      </Panel>

      {friends.length === 0 ? (
        <Panel title={t("servers.friends.access_title")}>
          <EmptyState>{t("servers.friends.empty")}</EmptyState>
        </Panel>
      ) : (
        friends.map((friend) => {
          const displayName =
            `${friend.user_name || ""} ${friend.user_last_name || ""}`.trim() ||
            friend.user_email ||
            friend.email;
          return (
            <Panel
              key={friend.id}
              title={
                <span className="flex flex-col">
                  <span className="text-[12.5px] font-medium">{displayName}</span>
                  <span className={cn("mt-[3px] font-mono text-[11.5px] font-normal", VX_FAINT)}>
                    {friend.user_email || friend.email}
                  </span>
                </span>
              }
              aside={
                <div className="flex gap-2">
                  <Btn
                    size="sm"
                    tone="primary"
                    disabled={saveMutation.isPending}
                    onClick={() => saveMutation.mutate(friend)}
                  >
                    {t("common.save")}
                  </Btn>
                  <Btn
                    size="sm"
                    className="border-[rgba(224,122,122,0.3)] bg-transparent text-[var(--vx-danger)]"
                    disabled={removeMutation.isPending}
                    onClick={() => {
                      if (!confirm(t("servers.friends.revoke_confirm"))) return;
                      removeMutation.mutate(friend.id);
                    }}
                  >
                    {t("servers.friends.revoke")}
                  </Btn>
                </div>
              }
              bodyClassName="grid gap-5 p-[18px] lg:grid-cols-2"
            >
              <div>
                <div className={cn(VX_MONO_LABEL, "mb-2.5 tracking-[0.09em]")}>
                  {t("servers.friends.tabs")}
                </div>
                <div className="grid gap-[7px] sm:grid-cols-2">
                  {TAB_PERMS.map((perm) => (
                    <PermRow
                      key={perm.key}
                      label={t(perm.labelKey)}
                      checked={!!friend[perm.key]}
                      onChange={(v) => setPerm(friend.id, perm.key, v)}
                    />
                  ))}
                </div>
              </div>
              <div>
                <div className={cn(VX_MONO_LABEL, "mb-2.5 tracking-[0.09em]")}>
                  {t("common.actions")}
                </div>
                <div className="grid gap-[7px] sm:grid-cols-2">
                  {ACTION_PERMS.map((perm) => (
                    <PermRow
                      key={perm.key}
                      label={t(perm.labelKey)}
                      checked={!!friend[perm.key]}
                      onChange={(v) => setPerm(friend.id, perm.key, v)}
                    />
                  ))}
                </div>
              </div>
            </Panel>
          );
        })
      )}
    </div>
  );
}

function PermRow({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <div className="flex items-center justify-between gap-2.5 rounded-[9px] border border-[var(--vx-border)] bg-[var(--vx-elevated)] px-[11px] py-2 text-[11.5px]">
      <span>{label}</span>
      <Toggle label={label} checked={checked} onChange={onChange} />
    </div>
  );
}
