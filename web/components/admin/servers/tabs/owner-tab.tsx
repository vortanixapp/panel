"use client";

import Link from "next/link";
import { useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Btn, EmptyState, InfoRow, Panel, VX_INPUT } from "@/components/vx/panel-ui";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/servers/confirm-dialog";
import {
  fetchAdminServerCard,
  fetchServerFriends,
  setAdminServerOwner,
  type ServerViewerPermissions,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { ownerStatusLabel } from "@/components/admin/servers/labels";

type AdminServerFriend = {
  user_id: string;
  email: string;
} & ServerViewerPermissions;

function grantedCount(friend: AdminServerFriend): number {
  return Object.entries(friend).filter(
    ([key, value]) => key.startsWith("can_") && value === true
  ).length;
}

export function AdminServerOwnerTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();

  const [email, setEmail] = useState("");
  const [confirmOpen, setConfirmOpen] = useState(false);

  const cardQuery = useQuery({
    queryKey: queryKeys.adminServerCard(id),
    queryFn: () => fetchAdminServerCard(id),
    enabled: !!id,
  });
  const friendsQuery = useQuery({
    queryKey: ["server-friends", id],
    queryFn: () => fetchServerFriends(id),
    enabled: !!id,
  });

  const transferMutation = useMutation({
    mutationFn: () => setAdminServerOwner(id, { email: email.trim() }),
    onSuccess: (res) => {
      toast.success(t("servers.admin.owner.transferred", { email: res.email }));
      setEmail("");
      setConfirmOpen(false);
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminServerCard(id) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverDetail(id) });
      void queryClient.invalidateQueries({ queryKey: ["server-friends", id] });
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminServers });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  if (cardQuery.isLoading || !cardQuery.data) {
    return <Skeleton className="h-[360px] w-full rounded-[14px]" />;
  }
  const card = cardQuery.data;
  const friends = (friendsQuery.data?.friends ?? []) as AdminServerFriend[];

  return (
    <div className="grid gap-[18px] lg:grid-cols-2">
      <Panel title={t("servers.admin.card.section_owner")}>
        {card.owner ? (
          <>
            <InfoRow
              k={t("common.email")}
              v={
                <Link
                  href={`/admin/users/${card.owner.id}`}
                  className="text-primary hover:underline"
                >
                  {card.owner.email}
                </Link>
              }
            />
            <InfoRow k={t("common.status")} v={ownerStatusLabel(card.owner.status)} />
            <InfoRow
              k={t("servers.admin.owner.registered")}
              v={
                card.owner.created_at
                  ? new Date(card.owner.created_at).toLocaleDateString(localeTag())
                  : "—"
              }
            />
            <InfoRow
              k={t("servers.admin.owner.balance")}
              v={
                card.owner.balances.length === 0
                  ? "—"
                  : card.owner.balances
                      .map((b) => `${b.balance.toFixed(2)} ${b.currency}`)
                      .join(" · ")
              }
            />
            <InfoRow
              k={t("servers.admin.owner.servers")}
              v={String(card.owner.server_count)}
            />
          </>
        ) : (
          <EmptyState>{t("servers.admin.owner.none")}</EmptyState>
        )}
      </Panel>

      <Panel title={t("servers.admin.owner.transfer_title")}>
        <p className="mb-3 text-[12.5px] text-[var(--vx-muted)]">
          {t("servers.admin.owner.transfer_hint")}
        </p>
        <form
          className="flex flex-wrap items-center gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            if (!email.trim()) return;
            setConfirmOpen(true);
          }}
        >
          <input
            className={cn(VX_INPUT, "h-[34px] min-w-[240px] flex-1")}
            type="email"
            placeholder="client@example.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
          <Btn type="submit" tone="primary" disabled={!email.trim()}>
            {t("servers.admin.owner.transfer")}
          </Btn>
        </form>
      </Panel>

      <div className="lg:col-span-2">
        <Panel title={t("servers.admin.owner.friends_title")}>
          {friends.length === 0 ? (
            <EmptyState>{t("servers.admin.owner.friends_empty")}</EmptyState>
          ) : (
            <div className="flex flex-col">
              {friends.map((friend) => (
                <InfoRow
                  key={friend.user_id}
                  k={friend.email}
                  v={
                    <span className="text-[var(--vx-muted)]">
                      {t("servers.admin.owner.friend_perms", {
                        count: grantedCount(friend),
                      })}
                    </span>
                  }
                />
              ))}
            </div>
          )}
        </Panel>
      </div>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title={t("servers.admin.owner.transfer_title")}
        description={t("servers.admin.owner.transfer_confirm", {
          from: card.owner?.email ?? "—",
          to: email.trim(),
        })}
        confirmLabel={t("servers.admin.owner.transfer")}
        tone="primary"
        pending={transferMutation.isPending}
        onConfirm={() => transferMutation.mutate()}
      />
    </div>
  );
}
