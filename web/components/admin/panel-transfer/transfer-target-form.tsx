"use client";

import { Field, VX_INPUT, VX_INPUT_MONO, VX_TEXTAREA } from "@/components/vx/panel-ui";
import { useT } from "@/hooks/use-translations";

export type TargetState = {
  host: string;
  port: string;
  user: string;
  auth: "password" | "key";
  password: string;
  privateKey: string;
};

export const emptyTarget: TargetState = {
  host: "",
  port: "22",
  user: "root",
  auth: "password",
  password: "",
  privateKey: "",
};

export function targetPayload(target: TargetState) {
  return {
    host: target.host.trim(),
    port: Number(target.port) || 22,
    user: target.user.trim(),
    password: target.auth === "password" ? target.password : "",
    private_key: target.auth === "key" ? target.privateKey : "",
  };
}

export function targetReady(target: TargetState) {
  if (!target.host.trim() || !target.user.trim()) return false;
  return target.auth === "password" ? target.password.length > 0 : target.privateKey.trim().length > 0;
}

export function TransferTargetForm({
  target,
  onChange,
  disabled,
}: {
  target: TargetState;
  onChange: (next: TargetState) => void;
  disabled?: boolean;
}) {
  const t = useT();
  const set = (patch: Partial<TargetState>) => onChange({ ...target, ...patch });

  return (
    <div className="grid gap-3">
      <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_110px]">
        <Field label={t("admin.panel_transfer.target.host")}>
          <input
            className={VX_INPUT_MONO}
            value={target.host}
            disabled={disabled}
            placeholder="203.0.113.10"
            onChange={(e) => set({ host: e.target.value })}
          />
        </Field>
        <Field label={t("admin.panel_transfer.target.port")}>
          <input
            className={VX_INPUT_MONO}
            value={target.port}
            disabled={disabled}
            inputMode="numeric"
            onChange={(e) => set({ port: e.target.value.replace(/[^0-9]/g, "") })}
          />
        </Field>
      </div>
      <Field label={t("admin.panel_transfer.target.user")}>
        <input
          className={VX_INPUT_MONO}
          value={target.user}
          disabled={disabled}
          onChange={(e) => set({ user: e.target.value })}
        />
      </Field>
      <div className="flex flex-wrap gap-4 text-[12px]">
        <label className="inline-flex items-center gap-2">
          <input
            type="radio"
            checked={target.auth === "password"}
            disabled={disabled}
            onChange={() => set({ auth: "password" })}
          />
          {t("admin.panel_transfer.target.auth_password")}
        </label>
        <label className="inline-flex items-center gap-2">
          <input
            type="radio"
            checked={target.auth === "key"}
            disabled={disabled}
            onChange={() => set({ auth: "key" })}
          />
          {t("admin.panel_transfer.target.auth_key")}
        </label>
      </div>
      {target.auth === "password" ? (
        <Field label={t("admin.panel_transfer.target.password")}>
          <input
            className={VX_INPUT}
            type="password"
            autoComplete="new-password"
            value={target.password}
            disabled={disabled}
            onChange={(e) => set({ password: e.target.value })}
          />
        </Field>
      ) : (
        <Field label={t("admin.panel_transfer.target.private_key")}>
          <textarea
            className={VX_TEXTAREA}
            rows={5}
            value={target.privateKey}
            disabled={disabled}
            placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
            onChange={(e) => set({ privateKey: e.target.value })}
          />
        </Field>
      )}
    </div>
  );
}
