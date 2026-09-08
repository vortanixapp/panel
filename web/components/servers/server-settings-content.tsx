"use client";

import { useMemo, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { ConfirmDialog } from "@/components/servers/confirm-dialog";
import { ServerTabShell } from "@/components/servers/server-tab-shell";
import { SettingsRawTab } from "@/components/servers/settings/settings-raw-tab";
import { SettingsRestartPrompt } from "@/components/servers/settings/settings-restart-prompt";
import { SettingsSectionForm } from "@/components/servers/settings/settings-section-form";
import { SettingsStartupTab } from "@/components/servers/settings/settings-startup-tab";
import { Btn, EmptyState, Notice, Panel, SubTabs } from "@/components/vx/panel-ui";
import { VxInlineLoader } from "@/components/vx/loader";
import { useDeleteServer, usePowerServer, useServerDetail } from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";
import { fetchServerSettings, saveServerSettings } from "@/lib/api";
import {
  SECTION_RAW,
  SECTION_STARTUP,
  type SaveSettingsResponse,
} from "@/lib/game-settings/types";
import type { PanelVariant } from "@/lib/panel-paths";
import { serversListPath, variantToBasePath } from "@/lib/panel-paths";
import { cn } from "@/lib/utils";

/**
 * Вкладка «Настройки» игрового сервера.
 *
 * Набор полей приходит с бэкенда вместе со значениями: панель не хранит у себя
 * ни одного ключа игры. Раньше список полей жил здесь отдельным файлом и
 * разошёлся с правилами валидации — у Minecraft бэкенд принимал 28 полей,
 * панель показывала 8, а поле вне правил при сохранении молча выбрасывалось,
 * причём с сообщением об успехе.
 */
export function ServerSettingsContent({ variant = "user" }: { variant?: PanelVariant }) {
  const t = useT();
  const basePath = variantToBasePath(variant);
  const router = useRouter();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const { data: server } = useServerDetail(id);
  const deleteServer = useDeleteServer();
  const power = usePowerServer();

  const [draft, setDraft] = useState<Record<string, string>>({});
  const [dirty, setDirty] = useState<Set<string>>(new Set());
  const [active, setActive] = useState("");
  const [lastSave, setLastSave] = useState<SaveSettingsResponse | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const schema = useQuery({
    queryKey: ["server-settings", id],
    queryFn: () => fetchServerSettings(id),
    enabled: !!id,
  });

  const perms = server?.viewer_permissions;
  const canEdit = perms?.can_settings_edit !== false;

  const values = useMemo(
    () => ({ ...(schema.data?.values ?? {}), ...draft }),
    [schema.data?.values, draft]
  );

  const tabs = useMemo(() => {
    const data = schema.data;
    if (!data?.supported) return [];
    const items = data.sections
      .filter((s) => s.fields.length > 0)
      .map((s) => ({
        id: s.id,
        title: s.title,
        badge: s.fields.filter((f) => dirty.has(f.key)).length,
      }));
    // Эти два раздела панель добавляет сама: профиль игры их не объявляет.
    if (data.files.length > 0)
      items.push({ id: SECTION_RAW, title: t("servers.settings.section_raw"), badge: 0 });
    items.push({
      id: SECTION_STARTUP,
      title: t("servers.settings.section_startup"),
      badge: 0,
    });
    return items;
  }, [schema.data, dirty, t]);

  const activeId = tabs.some((tab) => tab.id === active) ? active : (tabs[0]?.id ?? "");
  const activeSection = schema.data?.sections.find((s) => s.id === activeId);

  const save = useMutation({
    mutationFn: () => {
      // Отправляем ТОЛЬКО изменённые поля. Раньше форма слала все ключи разом, и
      // пустое значение оседало в базе, навсегда закрывая собой то, что реально
      // лежит в конфиге.
      const payload: Record<string, string> = {};
      for (const key of dirty) payload[key] = values[key] ?? "";
      return saveServerSettings(id, payload);
    },
    onSuccess: (res) => {
      setDirty(new Set());
      setDraft({});
      setLastSave(res);
      if (!res.restart_required)
        toast.success(res.message ?? t("servers.settings.saved"));
      void queryClient.invalidateQueries({ queryKey: ["server-settings", id] });
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("common.save_failed")),
  });

  async function onDelete() {
    try {
      await deleteServer.mutateAsync(id);
      toast.success(t("servers.settings.server_deleted"));
      router.push(serversListPath(basePath));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("common.error"));
    }
  }

  return (
    <ServerTabShell variant={variant} activeTab="settings">
      <div className="flex flex-col gap-[18px]">
        {schema.isLoading ? (
          <Panel title={t("server.tab.settings")}>
            <VxInlineLoader />
          </Panel>
        ) : schema.isError ? (
          <Panel title={t("server.tab.settings")}>
            <Notice>{t("servers.settings.load_failed")}</Notice>
          </Panel>
        ) : !schema.data?.supported ? (
          <Panel title={t("server.tab.settings")}>
            <EmptyState>
              {schema.data?.reason ?? t("servers.settings.unsupported")}
            </EmptyState>
          </Panel>
        ) : (
          <>
            {schema.data.warnings?.map((text) => (
              <Notice key={text} tone="warn">
                {text}
              </Notice>
            ))}

            {lastSave && (
              <SettingsRestartPrompt
                result={lastSave}
                canRestart={perms?.can_restart !== false}
                restarting={power.isPending}
                onRestart={() => {
                  power.mutate(
                    { id, action: "restart" },
                    {
                      onSuccess: () => {
                        toast.success(t("servers.settings.restarting"));
                        setLastSave(null);
                      },
                    }
                  );
                }}
                onDismiss={() => setLastSave(null)}
              />
            )}

            <Panel
              title={t("servers.settings.title")}
              aside={
                activeId !== SECTION_RAW && activeId !== SECTION_STARTUP ? (
                  <span className="flex items-center gap-2.5">
                    {dirty.size > 0 && (
                      <span className="text-[11.5px] text-[var(--vx-warn)]">
                        {t("servers.settings.dirty_count", { count: dirty.size })}
                      </span>
                    )}
                    <Btn
                      size="sm"
                      tone="primary"
                      disabled={!canEdit || dirty.size === 0 || save.isPending}
                      onClick={() => save.mutate()}
                    >
                      {save.isPending ? t("common.saving") : t("common.save")}
                    </Btn>
                  </span>
                ) : undefined
              }
            >
              <div className="flex flex-col gap-4">
                <SubTabs items={tabs} active={activeId} onSelect={setActive} />

                {schema.data.note && (
                  <p className="m-0 text-[12px] leading-[1.5] text-[var(--vx-faint)]">
                    {schema.data.note}
                  </p>
                )}

                {activeId === SECTION_RAW ? (
                  <SettingsRawTab serverId={id} files={schema.data.files} canEdit={canEdit} />
                ) : activeId === SECTION_STARTUP ? (
                  <SettingsStartupTab
                    serverId={id}
                    initial={server?.startup_params ?? ""}
                    canEdit={canEdit}
                  />
                ) : activeSection ? (
                  <SettingsSectionForm
                    section={activeSection}
                    values={values}
                    dirty={dirty}
                    disabled={!canEdit}
                    onChange={(key, value) => {
                      setDraft((prev) => ({ ...prev, [key]: value }));
                      setDirty((prev) => {
                        const next = new Set(prev);
                        // Возврат к исходному значению снимает пометку: иначе
                        // счётчик считал бы правки, которых уже нет.
                        if (value === (schema.data?.values?.[key] ?? "")) next.delete(key);
                        else next.add(key);
                        return next;
                      });
                    }}
                  />
                ) : (
                  <EmptyState>{t("servers.settings.no_fields")}</EmptyState>
                )}
              </div>
            </Panel>
          </>
        )}

        <ConfirmDialog
          open={confirmDelete}
          onOpenChange={setConfirmDelete}
          title={t("servers.settings.delete_title")}
          description={t("servers.settings.delete_desc", {
            name: server?.name ?? "",
          })}
          confirmLabel={t("servers.settings.delete_confirm_label")}
          requirePhrase={server?.name}
          pending={deleteServer.isPending}
          onConfirm={() => {
            setConfirmDelete(false);
            void onDelete();
          }}
        />

        <div
          className={cn(
            "flex flex-wrap items-center justify-between gap-4 rounded-[14px] px-5 py-[18px]",
            "border border-[rgba(224,122,122,0.28)] bg-[rgba(224,122,122,0.04)]"
          )}
        >
          <div>
            <div className="text-[12.5px] font-medium text-[var(--vx-danger)]">
              {t("servers.settings.danger_zone")}
            </div>
            <div className="mt-[5px] text-[12px] text-[var(--vx-muted)]">
              {t("servers.settings.danger_hint")}
            </div>
          </div>
          <Btn tone="danger" onClick={() => setConfirmDelete(true)} disabled={deleteServer.isPending}>
            {deleteServer.isPending
              ? t("common.deleting")
              : t("servers.settings.delete_confirm_label")}
          </Btn>
        </div>
      </div>
    </ServerTabShell>
  );
}
