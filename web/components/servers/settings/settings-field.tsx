"use client";

import { useState } from "react";

import {
  Toggle,
  VX_FAINT,
  VX_INPUT,
  VX_MONO_LABEL,
  VX_SELECT,
  VX_TEXTAREA,
} from "@/components/vx/panel-ui";
import { isSecretPlaceholder, type SettingField } from "@/lib/game-settings/types";
import { t } from "@/lib/i18n";
import { cn } from "@/lib/utils";

export function SettingsFieldControl({
  field,
  value,
  dirty,
  disabled,
  onChange,
}: {
  field: SettingField;
  value: string;
  dirty: boolean;
  disabled?: boolean;
  onChange: (next: string) => void;
}) {
  const locked = disabled || field.read_only;
  const secretHidden = field.secret && isSecretPlaceholder(value);

  return (
    <label className="grid gap-1.5">
      <span className={cn("flex flex-wrap items-center gap-2 text-[11.5px]", "text-[var(--vx-muted)]")}>
        <span className={dirty ? "text-[var(--vx-fg)]" : undefined}>{field.label}</span>
        {!field.applies_live && !field.read_only && (
          <span className={cn(VX_MONO_LABEL, "text-[9.5px] tracking-[0.06em]")}>
            {t("servers.settings.field_restart")}
          </span>
        )}
        {dirty && (
          <span
            className="h-1 w-1 rounded-full bg-[var(--vx-warn)]"
            aria-label={t("servers.settings.field_changed")}
          />
        )}
      </span>

      <Control
        field={field}
        value={value}
        locked={!!locked}
        secretHidden={!!secretHidden}
        onChange={onChange}
      />

      {(field.hint || field.read_only) && (
        <span className={cn("text-[11px] leading-[1.45]", VX_FAINT)}>
          {field.read_only && !field.hint ? readOnlyReason(field) : field.hint}
        </span>
      )}
    </label>
  );
}

function readOnlyReason(field: SettingField): string {
  if (field.slots) return t("servers.settings.readonly_slots");
  return t("servers.settings.readonly_panel");
}

function Control({
  field,
  value,
  locked,
  secretHidden,
  onChange,
}: {
  field: SettingField;
  value: string;
  locked: boolean;
  secretHidden: boolean;
  onChange: (next: string) => void;
}) {
  const [revealed, setRevealed] = useState(false);

  if (field.kind === "bool") {
    return (
      <span className="flex h-[34px] items-center">
        <Toggle
          checked={value === "true" || value === "1"}
          disabled={locked}
          label={field.label}
          onChange={(next) => onChange(next ? "true" : "false")}
        />
      </span>
    );
  }

  if (field.kind === "enum" && field.options?.length) {
    return (
      <select
        className={cn(VX_SELECT, "w-full")}
        value={value}
        disabled={locked}
        onChange={(e) => onChange(e.target.value)}
      >
        {value !== "" && !field.options.some((o) => o.value === value) && (
          <option value={value}>
            {t("servers.settings.option_custom", { value })}
          </option>
        )}
        {field.options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    );
  }

  if (field.kind === "text") {
    return (
      <textarea
        className={cn(VX_TEXTAREA, "resize-y")}
        rows={3}
        spellCheck={false}
        value={value}
        disabled={locked}
        onChange={(e) => onChange(e.target.value)}
      />
    );
  }

  if (field.kind === "int" || field.kind === "float") {
    return (
      <input
        className={VX_INPUT}
        type="number"
        inputMode={field.kind === "int" ? "numeric" : "decimal"}
        step={field.kind === "int" ? 1 : "any"}
        min={field.min}
        max={field.max}
        value={value}
        disabled={locked}
        onChange={(e) => onChange(e.target.value)}
      />
    );
  }

  if (field.secret) {
    return (
      <span className="relative flex">
        <input
          className={cn(VX_INPUT, "pr-[68px]")}
          type={revealed ? "text" : "password"}
          autoComplete="new-password"
          placeholder={secretHidden ? t("servers.settings.secret_keep") : ""}
          value={secretHidden ? "" : value}
          disabled={locked}
          onChange={(e) => onChange(e.target.value)}
        />
        <button
          type="button"
          disabled={locked}
          onClick={() => setRevealed((v) => !v)}
          className="absolute top-1/2 right-2 -translate-y-1/2 text-[11px] text-[var(--vx-muted)] transition-colors hover:text-[var(--vx-fg)] disabled:opacity-50"
        >
          {revealed
            ? t("servers.settings.hide")
            : t("servers.settings.show")}
        </button>
      </span>
    );
  }

  return (
    <input
      className={VX_INPUT}
      type="text"
      maxLength={field.max_len || undefined}
      value={value}
      disabled={locked}
      onChange={(e) => onChange(e.target.value)}
    />
  );
}
