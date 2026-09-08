"use client";

import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Btn, Notice, VX_TEXTAREA } from "@/components/vx/panel-ui";
import { saveServerStartupParams } from "@/lib/api";
import { t } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";

/**
 * Аргументы командной строки сервера.
 *
 * Для части игр это единственный способ что-то настроить: у Valheim здесь имя
 * мира, у Unreal-игр — имя сессии и число мест.
 *
 * Поле долго было декоративным: значение сохранялось в базу, но до контейнера не
 * доходило — образы читают файл, который никто не писал. Теперь доходит.
 */
export function SettingsStartupTab({
  serverId,
  initial,
  canEdit,
  placeholder,
  note,
}: {
  serverId: string;
  initial: string;
  canEdit: boolean;
  placeholder?: string;
  note?: string;
}) {
  const queryClient = useQueryClient();
  const [value, setValue] = useState(initial);

  useEffect(() => setValue(initial), [initial, serverId]);

  // Строка, начинающаяся с пути, по логике образа заменяет команду запуска
  // целиком, а не дописывается к ней. Сервер после такого не поднимается, и
  // причина выглядит как что угодно, кроме опечатки в поле. Бэкенд это тоже
  // отклоняет — здесь предупреждаем раньше, чем пользователь нажмёт «Сохранить».
  const looksLikeCommand = /^\s*\.?\//.test(value);

  const save = useMutation({
    mutationFn: () => saveServerStartupParams(serverId, value),
    onSuccess: () => {
      toast.success(t("servers.settings.startup_saved"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverDetail(serverId) });
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("common.save_failed")),
  });

  return (
    <div className="flex flex-col gap-3">
      {note && <p className="m-0 text-[12px] leading-[1.5] text-[var(--vx-faint)]">{note}</p>}

      <textarea
        value={value}
        onChange={(e) => setValue(e.target.value)}
        rows={4}
        spellCheck={false}
        disabled={!canEdit}
        className={cn(VX_TEXTAREA, "resize-y")}
        placeholder={placeholder ?? t("servers.settings.startup_placeholder")}
      />

      {looksLikeCommand && (
        <Notice tone="warn">
          {t("servers.settings.startup_command_warning")}
        </Notice>
      )}

      <div className="flex items-center justify-between gap-3">
        <span className="text-[11.5px] text-[var(--vx-faint)]">
          {t("servers.settings.startup_quote_hint")}
        </span>
        <Btn
          size="sm"
          disabled={!canEdit || looksLikeCommand || save.isPending || value === initial}
          onClick={() => save.mutate()}
        >
          {save.isPending ? t("common.saving") : t("common.save")}
        </Btn>
      </div>
    </div>
  );
}
