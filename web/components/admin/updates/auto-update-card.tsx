"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Segmented, SelectField, SettingsCard } from "@/components/admin/settings/settings-ui";
import { Switch } from "@/components/ui/switch";
import {
  updateAdminUpdateSettings,
  type AdminAgentUpdates,
  type AdminUpdates,
  type PanelAutoUpdate,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";

const HOURS = Array.from({ length: 24 }, (_, h) => ({
  value: String(h),
  label: `${String(h).padStart(2, "0")}:00`,
}));

type Patch = Partial<{
  auto_enabled: boolean;
  agents_auto_enabled: boolean;
  window_start: number;
  window_end: number;
}>;

function Row({
  title,
  hint,
  checked,
  disabled,
  onChange,
  children,
}: {
  title: string;
  hint: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (value: boolean) => void;
  children?: React.ReactNode;
}) {
  return (
    <div className="py-4 first:pt-0 last:pb-0">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="text-[14px] font-medium">{title}</div>
          <p className="mt-1 text-[12.5px] leading-relaxed text-muted-foreground">{hint}</p>
        </div>
        <Switch checked={checked} disabled={disabled} onCheckedChange={onChange} aria-label={title} className="mt-0.5" />
      </div>
      {children}
    </div>
  );
}

export function AutoUpdateCard({
  auto,
  agentsAuto,
  updaterAvailable,
}: {
  auto: PanelAutoUpdate;
  agentsAuto: boolean | null;
  updaterAvailable: boolean;
}) {
  const t = useT();
  const qc = useQueryClient();
  const scheduled = auto.window_start >= 0 && auto.window_end >= 0;

  const save = useMutation({
    mutationFn: (patch: Patch) => updateAdminUpdateSettings(patch),
    onMutate: (patch) => {
      if (patch.agents_auto_enabled !== undefined) {
        qc.setQueryData<AdminAgentUpdates>(queryKeys.adminAgentUpdates, (prev) =>
          prev ? { ...prev, auto_enabled: patch.agents_auto_enabled === true } : prev
        );
      }
    },
    onSuccess: (next) => {
      qc.setQueryData<AdminUpdates>(queryKeys.adminUpdates, (prev) => (prev ? { ...prev, auto: next } : prev));
      toast.success(t("admin.updates.auto.saved"));
    },
    onError: (e: Error) => {
      toast.error(t("admin.updates.action_failed", { error: e.message }));
      void qc.invalidateQueries({ queryKey: queryKeys.adminAgentUpdates });
    },
  });

  return (
    <SettingsCard title={t("admin.updates.auto.title")} description={t("admin.updates.auto.description")}>
      <div className="divide-y">
        <Row
          title={t("admin.updates.auto.panel")}
          hint={t("admin.updates.auto.panel_hint")}
          checked={auto.enabled}
          disabled={save.isPending}
          onChange={(value) => save.mutate({ auto_enabled: value })}
        >
          {!updaterAvailable && (
            <p className="mt-2 text-[12px] text-amber-600 dark:text-amber-500">{t("admin.updates.auto.needs_updater")}</p>
          )}
          {auto.skipped_version && (
            <p className="mt-2 text-[12px] text-rose-600 dark:text-rose-400">
              {t("admin.updates.auto.skipped", { version: auto.skipped_version })}
            </p>
          )}
          <div className={cn("mt-3.5 space-y-3", !auto.enabled && "opacity-60")}>
            <div className="text-[12px] text-muted-foreground">{t("admin.updates.auto.window")}</div>
            <Segmented
              size="sm"
              className="w-full"
              items={[
                { id: "any", label: t("admin.updates.auto.anytime") },
                { id: "hours", label: t("admin.updates.auto.hours") },
              ]}
              value={scheduled ? "hours" : "any"}
              onChange={(next) =>
                save.mutate(next === "any" ? { window_start: -1, window_end: -1 } : { window_start: 3, window_end: 6 })
              }
            />
            {scheduled && (
              <>
                <div className="grid grid-cols-2 gap-3">
                  <SelectField
                    label={t("admin.updates.auto.from")}
                    value={String(auto.window_start)}
                    options={HOURS}
                    onChange={(v) => save.mutate({ window_start: Number(v) })}
                  />
                  <SelectField
                    label={t("admin.updates.auto.to")}
                    value={String(auto.window_end)}
                    options={HOURS}
                    onChange={(v) => save.mutate({ window_end: Number(v) })}
                  />
                </div>
                <p className="text-[12px] text-muted-foreground">{t("admin.updates.auto.window_hint")}</p>
              </>
            )}
          </div>
        </Row>
        <Row
          title={t("admin.updates.auto.agents")}
          hint={t("admin.updates.auto.agents_hint")}
          checked={agentsAuto === true}
          disabled={agentsAuto === null || save.isPending}
          onChange={(value) => save.mutate({ agents_auto_enabled: value })}
        />
      </div>
    </SettingsCard>
  );
}
