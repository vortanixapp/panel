"use client";

import { Btn, Notice } from "@/components/vx/panel-ui";
import type { SaveSettingsResponse } from "@/lib/game-settings/types";
import { t } from "@/lib/i18n";

/**
 * Что делать после сохранения.
 *
 * Почти любой игровой конфиг читается один раз при старте, поэтому сохранение
 * само по себе ничего не меняет на живом сервере. Раньше об этом не говорилось
 * вовсе: панель показывала «Настройки сохранены», клиент шёл проверять и не
 * находил изменений.
 *
 * Перезапускаем не сами: на сервере могут быть игроки, и решение чем-то жертвовать
 * принимает владелец, а не панель.
 */
export function SettingsRestartPrompt({
  result,
  canRestart,
  restarting,
  onRestart,
  onDismiss,
}: {
  result: SaveSettingsResponse;
  canRestart: boolean;
  restarting: boolean;
  onRestart: () => void;
  onDismiss: () => void;
}) {
  if (!result.restart_required) return null;

  const running = result.server_running;

  return (
    <Notice tone="warn" className="flex flex-wrap items-center justify-between gap-3">
      <span className="leading-[1.5]">
        {running
          ? t("servers.settings.restart_running")
          : t("servers.settings.restart_stopped")}
        {result.restart_fields.length > 0 && (
          <span className="text-[var(--vx-muted)]">
            {t("servers.settings.restart_fields", {
              fields: result.restart_fields.join(", "),
            })}
          </span>
        )}
      </span>
      <span className="flex shrink-0 items-center gap-2">
        {running && canRestart && (
          <Btn size="sm" tone="primary" disabled={restarting} onClick={onRestart}>
            {restarting
              ? t("servers.settings.restarting_now")
              : t("servers.settings.restart_now")}
          </Btn>
        )}
        <Btn size="sm" tone="ghost" onClick={onDismiss}>
          {t("common.hide")}
        </Btn>
      </span>
    </Notice>
  );
}
