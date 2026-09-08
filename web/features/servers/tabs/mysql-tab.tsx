"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Btn,
  EmptyState,
  Field,
  Panel,
  VX_FAINT,
  VX_INPUT,
  VX_INSET,
  VX_MONO_LABEL,
  VX_MUTED,
  VX_ROW_LINE,
  VX_SELECT,
} from "@/components/vx/panel-ui";
import { VxInlineLoader } from "@/components/vx/loader";
import {
  createServerMysqlDatabase,
  createServerMysqlUser,
  deleteServerMysql,
  deleteServerMysqlDatabase,
  deleteServerMysqlUser,
  fetchServerMysql,
  migrateServerMysql,
  resetServerMysqlPassword,
  resetServerMysqlUserPassword,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useServerDetail } from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";

type MysqlExtras = {
  mysql_instances?: Array<Record<string, unknown>>;
  catalog?: {
    databases?: string[];
    users?: Array<{ username: string; host: string }>;
  };
};

export function ServerMysqlTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const { data: server } = useServerDetail(id);

  const { data: mysql, isLoading } = useQuery({
    queryKey: queryKeys.serverMysql(id),
    queryFn: () => fetchServerMysql(id),
    enabled: !!id,
  });

  const [targetInstance, setTargetInstance] = useState("");
  const [newDatabase, setNewDatabase] = useState("");
  const [newUser, setNewUser] = useState("");
  const [newUserPassword, setNewUserPassword] = useState("");
  const [grantDatabase, setGrantDatabase] = useState("");
  const [resetUser, setResetUser] = useState("");
  const [resetPassword, setResetPassword] = useState("");

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.serverMysql(id) });
  };
  const onError = (err: unknown) =>
    toast.error(err instanceof Error ? err.message : t("common.error"));

  const resetMutation = useMutation({
    mutationFn: () => resetServerMysqlPassword(id),
    onSuccess: () => {
      toast.success(t("servers.mysql.updated"));
      invalidate();
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverDetail(id) });
    },
    onError,
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteServerMysql(id),
    onSuccess: () => {
      toast.success(t("servers.mysql.deleted"));
      invalidate();
    },
    onError,
  });

  const migrateMutation = useMutation({
    mutationFn: () => migrateServerMysql(id, targetInstance),
    onSuccess: () => {
      toast.success(t("servers.mysql.migration_started"));
      invalidate();
    },
    onError,
  });

  const createDatabaseMutation = useMutation({
    mutationFn: () => createServerMysqlDatabase(id, newDatabase.trim()),
    onSuccess: () => {
      toast.success(t("servers.mysql.db_created"));
      setNewDatabase("");
      invalidate();
    },
    onError,
  });

  const deleteDatabaseMutation = useMutation({
    mutationFn: (database: string) => deleteServerMysqlDatabase(id, database),
    onSuccess: () => {
      toast.success(t("servers.mysql.db_deleted"));
      invalidate();
    },
    onError,
  });

  const createUserMutation = useMutation({
    mutationFn: () =>
      createServerMysqlUser(id, newUser.trim(), newUserPassword, grantDatabase.trim() || undefined),
    onSuccess: () => {
      toast.success(t("servers.mysql.user_created"));
      setNewUser("");
      setNewUserPassword("");
      invalidate();
    },
    onError,
  });

  const deleteUserMutation = useMutation({
    mutationFn: (username: string) => deleteServerMysqlUser(id, username),
    onSuccess: () => {
      toast.success(t("servers.mysql.user_deleted"));
      invalidate();
    },
    onError,
  });

  const resetUserPasswordMutation = useMutation({
    mutationFn: () => resetServerMysqlUserPassword(id, resetUser.trim(), resetPassword),
    onSuccess: () => {
      toast.success(t("servers.mysql.user_password_updated"));
      setResetPassword("");
      invalidate();
    },
    onError,
  });

  const host = String(mysql?.host ?? server?.mysql_host ?? "");
  const hasMysql = host !== "" && !!mysql?.database;
  const extras = (mysql ?? {}) as MysqlExtras;
  const instances = extras.mysql_instances ?? [];
  const catalogDatabases = extras.catalog?.databases ?? [];
  const catalogUsers = extras.catalog?.users ?? [];

  async function copyValue(label: string, value: string) {
    if (!value) return;
    try {
      await navigator.clipboard.writeText(value);
      toast.success(t("servers.mysql.copied", { label }));
    } catch {
      toast.error(t("common.copy_failed"));
    }
  }

  if (isLoading) {
    return <Panel title="MySQL"><VxInlineLoader /></Panel>;
  }

  if (!hasMysql) {
    return (
      <Panel title="MySQL">
        <p className={cn("text-[12.5px]", VX_MUTED)}>
          {t("servers.mysql.not_configured")}
        </p>
        <Btn
          tone="primary"
          className="mt-3.5"
          onClick={() => resetMutation.mutate()}
          disabled={resetMutation.isPending}
        >
          {resetMutation.isPending
            ? t("common.creating")
            : t("servers.mysql.create_db_and_user")}
        </Btn>
      </Panel>
    );
  }

  const accessRows: [string, string][] = [
    [t("servers.mysql.row_host"), String(mysql?.host ?? "")],
    [t("servers.mysql.row_port"), String(mysql?.port ?? 3306)],
    [t("servers.mysql.row_database"), String(mysql?.database ?? "")],
    [t("common.user"), String(mysql?.username ?? "")],
    [
      t("common.password"),
      String(mysql?.password ?? mysql?.mysql_password_decrypted ?? ""),
    ],
  ];

  return (
    <div className="grid grid-cols-1 items-start gap-[18px] lg:grid-cols-2">
      <Panel title={t("servers.mysql.access_title")} flush>
        <div className="px-[18px] pt-1.5 pb-4">
          {accessRows.map(([label, value]) => (
            <div
              key={label}
              className={cn(
                "flex items-center justify-between gap-3 py-2.5 text-[12.5px] last:border-b-0",
                VX_ROW_LINE
              )}
            >
              <span className={VX_MUTED}>{label}</span>
              <span className="flex min-w-0 items-center gap-2">
                <span className="truncate font-mono text-[12px]">{value || "—"}</span>
                <button
                  type="button"
                  onClick={() => copyValue(label, value)}
                  className={cn(
                    "h-6 shrink-0 rounded-[6px] border border-[var(--vx-border-2)] px-2 text-[11px] transition-colors hover:text-[var(--vx-fg)]",
                    VX_FAINT
                  )}
                >
                  {t("servers.mysql.copy_short")}
                </button>
              </span>
            </div>
          ))}

          <div className="mt-3.5 flex flex-wrap gap-2">
            <Btn
              size="sm"
              className="h-8"
              onClick={() => resetMutation.mutate()}
              disabled={resetMutation.isPending}
            >
              {t("servers.mysql.change_password")}
            </Btn>
            <Btn
              size="sm"
              className="h-8 border-[rgba(224,122,122,0.3)] bg-transparent text-[var(--vx-danger)]"
              onClick={() => {
                if (!confirm(t("servers.mysql.delete_confirm"))) return;
                deleteMutation.mutate();
              }}
              disabled={deleteMutation.isPending}
            >
              {t("servers.mysql.delete")}
            </Btn>
          </div>

          {instances.length > 0 && (
            <div className="mt-4 border-t border-[var(--vx-border)] pt-3.5">
              <div className={VX_MONO_LABEL}>{t("servers.mysql.migration")}</div>
              <div className="mt-2.5 flex flex-wrap gap-2">
                <select
                  className={cn(VX_SELECT, "min-w-[180px] flex-1")}
                  value={targetInstance}
                  onChange={(e) => setTargetInstance(e.target.value)}
                >
                  <option value="">{t("servers.mysql.migration_target")}</option>
                  {instances.map((instance, idx) => {
                    const key = String(instance.key ?? instance.id ?? `inst-${idx}`);
                    return (
                      <option key={key} value={key}>
                        {String(instance.name ?? instance.key ?? key)}
                      </option>
                    );
                  })}
                </select>
                <Btn
                  onClick={() => migrateMutation.mutate()}
                  disabled={migrateMutation.isPending || !targetInstance}
                >
                  {migrateMutation.isPending
                    ? t("servers.mysql.migrating")
                    : t("servers.mysql.migrate")}
                </Btn>
              </div>
            </div>
          )}
        </div>
      </Panel>

      <Panel
        title={t("servers.mysql.dbs_and_users")}
        bodyClassName="flex flex-col gap-3.5 p-[18px]"
      >
        <div className="flex gap-2">
          <input
            className={VX_INPUT}
            value={newDatabase}
            onChange={(e) => setNewDatabase(e.target.value)}
            placeholder="example_db"
          />
          <Btn
            tone="primary"
            onClick={() => createDatabaseMutation.mutate()}
            disabled={createDatabaseMutation.isPending || !newDatabase.trim()}
          >
            {t("servers.mysql.create_db")}
          </Btn>
        </div>

        <div className="flex flex-col gap-1.5">
          {catalogDatabases.length === 0 ? (
            <p className={cn("text-[11.5px]", VX_FAINT)}>
              {t("servers.mysql.catalog_dbs_unavailable")}
            </p>
          ) : (
            catalogDatabases.map((db) => (
              <div
                key={db}
                className={cn(
                  "flex items-center justify-between gap-3 rounded-[9px] px-3 py-2.5 font-mono text-[12px]",
                  VX_INSET
                )}
              >
                {db}
                <button
                  type="button"
                  onClick={() => {
                    if (
                      !confirm(t("servers.mysql.db_delete_confirm", { name: db }))
                    )
                      return;
                    deleteDatabaseMutation.mutate(db);
                  }}
                  disabled={deleteDatabaseMutation.isPending}
                  className="font-sans text-[11.5px] text-[var(--vx-danger)] transition-opacity hover:opacity-80 disabled:opacity-40"
                >
                  {t("common.delete")}
                </button>
              </div>
            ))
          )}
        </div>

        <div className="border-t border-[var(--vx-border)] pt-3.5">
          <div className={VX_MONO_LABEL}>{t("servers.mysql.new_user")}</div>
          <div className="mt-2.5 grid gap-2 sm:grid-cols-3">
            <input
              className={VX_INPUT}
              value={newUser}
              onChange={(e) => setNewUser(e.target.value)}
              placeholder="username"
            />
            <input
              className={VX_INPUT}
              value={newUserPassword}
              onChange={(e) => setNewUserPassword(e.target.value)}
              placeholder="password"
            />
            <input
              className={VX_INPUT}
              value={grantDatabase}
              onChange={(e) => setGrantDatabase(e.target.value)}
              placeholder={t("servers.mysql.grant_db_placeholder")}
            />
          </div>
          <Btn
            className="mt-2.5"
            onClick={() => createUserMutation.mutate()}
            disabled={createUserMutation.isPending || !newUser.trim() || !newUserPassword}
          >
            {t("servers.mysql.create_user")}
          </Btn>
        </div>

        <div className="flex flex-col gap-1.5">
          {catalogUsers.length === 0 ? (
            <p className={cn("text-[11.5px]", VX_FAINT)}>
              {t("servers.mysql.catalog_users_unavailable")}
            </p>
          ) : (
            catalogUsers.map((user) => (
              <div
                key={`${user.username}@${user.host}`}
                className={cn(
                  "flex items-center justify-between gap-3 rounded-[9px] px-3 py-2.5 font-mono text-[12px]",
                  VX_INSET
                )}
              >
                {user.username}@{user.host}
                <button
                  type="button"
                  onClick={() => {
                    if (
                      !confirm(
                        t("servers.mysql.user_delete_confirm", {
                          name: user.username,
                        })
                      )
                    )
                      return;
                    deleteUserMutation.mutate(user.username);
                  }}
                  disabled={deleteUserMutation.isPending}
                  className="font-sans text-[11.5px] text-[var(--vx-danger)] transition-opacity hover:opacity-80 disabled:opacity-40"
                >
                  {t("common.delete")}
                </button>
              </div>
            ))
          )}
        </div>

        <div className="border-t border-[var(--vx-border)] pt-3.5">
          <div className={VX_MONO_LABEL}>
            {t("servers.mysql.reset_user_password")}
          </div>
          <div className="mt-2.5 grid gap-2 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]">
            <Field label={t("common.user")}>
              <input
                className={VX_INPUT}
                value={resetUser}
                onChange={(e) => setResetUser(e.target.value)}
                placeholder="username"
              />
            </Field>
            <Field label={t("servers.mysql.new_password")}>
              <input
                className={VX_INPUT}
                value={resetPassword}
                onChange={(e) => setResetPassword(e.target.value)}
                placeholder="new password"
              />
            </Field>
            <Btn
              className="self-end"
              onClick={() => resetUserPasswordMutation.mutate()}
              disabled={resetUserPasswordMutation.isPending || !resetUser.trim() || !resetPassword}
            >
              {t("common.refresh")}
            </Btn>
          </div>
        </div>
      </Panel>
    </div>
  );
}
