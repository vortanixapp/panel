"use client";

import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Btn, EmptyState, Notice, SubTabs, VX_TEXTAREA } from "@/components/vx/panel-ui";
import { VxInlineLoader } from "@/components/vx/loader";
import { fetchServerSettingsFile, saveServerSettingsFile } from "@/lib/api";
import type { SettingsFile } from "@/lib/game-settings/types";
import { t } from "@/lib/i18n";
import { cn } from "@/lib/utils";

/**
 * Правка конфигурационного файла как текста.
 *
 * Раньше здесь был `<pre>` только на чтение: увидеть конфиг было можно, а
 * поправить редкий параметр, которого нет в форме, — нет. Приходилось идти в
 * SFTP и искать файл руками.
 *
 * Правим только те файлы, которые объявил профиль игры. Произвольный путь — это
 * вкладка SFTP со своим отдельным правом; смешивать их значило бы завести второй,
 * более слабый способ добраться до любого файла.
 */
export function SettingsRawTab({
  serverId,
  files,
  canEdit,
}: {
  serverId: string;
  files: SettingsFile[];
  canEdit: boolean;
}) {
  const editable = files.filter((f) => !f.error || f.exists);
  const [activeId, setActiveId] = useState(editable[0]?.id ?? "");

  useEffect(() => {
    if (!editable.some((f) => f.id === activeId) && editable[0]) {
      setActiveId(editable[0].id);
    }
  }, [editable, activeId]);

  if (files.length === 0) {
    return (
      <EmptyState>{t("servers.settings.raw_none")}</EmptyState>
    );
  }

  const active = files.find((f) => f.id === activeId) ?? files[0];

  return (
    <div className="flex flex-col gap-3.5">
      <SubTabs
        items={files.map((f) => ({ id: f.id, title: f.title || f.path }))}
        active={active.id}
        onSelect={setActiveId}
      />
      <FileEditor key={active.id} serverId={serverId} file={active} canEdit={canEdit} />
    </div>
  );
}

function FileEditor({
  serverId,
  file,
  canEdit,
}: {
  serverId: string;
  file: SettingsFile;
  canEdit: boolean;
}) {
  const queryClient = useQueryClient();
  const [text, setText] = useState("");
  const [base, setBase] = useState("");
  const [dirty, setDirty] = useState(false);
  const [conflict, setConflict] = useState(false);

  const query = useQuery({
    queryKey: ["server-settings-file", serverId, file.id],
    queryFn: () => fetchServerSettingsFile(serverId, file.id),
    enabled: !!serverId && !!file.id,
  });

  useEffect(() => {
    if (query.data) {
      setText(query.data.content);
      setBase(query.data.sha256);
      setDirty(false);
      setConflict(false);
    }
  }, [query.data]);

  const save = useMutation({
    mutationFn: () => saveServerSettingsFile(serverId, file.id, text, base),
    onSuccess: (res) => {
      setBase(res.sha256);
      setDirty(false);
      setConflict(false);
      toast.success(
        res.restart_required && res.server_running
          ? t("servers.settings.file_saved_restart")
          : t("servers.settings.file_saved")
      );
      void queryClient.invalidateQueries({ queryKey: ["server-settings", serverId] });
    },
    onError: (err) => {
      const message = err instanceof Error ? err.message : t("common.save_failed");
      // Файл на ноде изменился, пока вкладка была открыта: игра переписывает
      // конфиг при старте. Затирать чужие правки молча нельзя — текст в поле
      // сохраняем, чтобы пользователю было что перенести.
      if (/изменил|conflict|409|stale/i.test(message)) {
        setConflict(true);
        return;
      }
      toast.error(message);
    },
  });

  if (query.isLoading) return <VxInlineLoader />;

  if (file.truncated) {
    return (
      <EmptyState>{t("servers.settings.file_too_big")}</EmptyState>
    );
  }

  if (query.isError) {
    return (
      <Notice>
        {t("servers.settings.file_read_failed", { path: file.path })}
      </Notice>
    );
  }

  const missing = query.data?.exists === false;

  return (
    <div className="flex flex-col gap-3">
      {missing && (
        <Notice tone="warn">
          {t("servers.settings.file_missing", { path: file.path })}
        </Notice>
      )}

      {conflict && (
        <Notice>{t("servers.settings.file_conflict")}</Notice>
      )}

      <textarea
        value={text}
        spellCheck={false}
        disabled={!canEdit}
        onChange={(e) => {
          setText(e.target.value);
          setDirty(true);
        }}
        rows={22}
        className={cn(VX_TEXTAREA, "resize-y")}
      />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <span className="font-mono text-[11px] text-[var(--vx-faint)]">{file.path}</span>
        <span className="flex items-center gap-2">
          {dirty && (
            <span className="text-[11.5px] text-[var(--vx-warn)]">
              {t("servers.settings.unsaved")}
            </span>
          )}
          <Btn
            size="sm"
            tone="primary"
            disabled={!canEdit || !dirty || save.isPending}
            onClick={() => save.mutate()}
          >
            {save.isPending
              ? t("common.saving")
              : t("servers.settings.save_file")}
          </Btn>
        </span>
      </div>
    </div>
  );
}
