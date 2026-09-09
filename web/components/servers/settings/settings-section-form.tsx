"use client";

import { SettingsFieldControl } from "@/components/servers/settings/settings-field";
import { EmptyState } from "@/components/vx/panel-ui";
import type { SettingsSection } from "@/lib/game-settings/types";
import { t } from "@/lib/i18n";

export function SettingsSectionForm({
  section,
  values,
  dirty,
  disabled,
  onChange,
}: {
  section: SettingsSection;
  values: Record<string, string>;
  dirty: Set<string>;
  disabled?: boolean;
  onChange: (key: string, value: string) => void;
}) {
  if (section.fields.length === 0) {
    return <EmptyState>{t("servers.settings.section_empty")}</EmptyState>;
  }

  return (
    <div className="flex flex-col gap-4">
      {section.hint && (
        <p className="m-0 text-[12px] leading-[1.5] text-[var(--vx-faint)]">{section.hint}</p>
      )}
      <div className="grid gap-x-4 gap-y-3.5 sm:grid-cols-2 xl:grid-cols-3">
        {section.fields.map((field) => (
          <SettingsFieldControl
            key={field.key}
            field={field}
            value={values[field.key] ?? ""}
            dirty={dirty.has(field.key)}
            disabled={disabled}
            onChange={(next) => onChange(field.key, next)}
          />
        ))}
      </div>
    </div>
  );
}
