"use client";

import { useRef, useState } from "react";
import { Download, FileCheck2, Upload } from "lucide-react";
import { toast } from "sonner";

import {
  Btn,
  Field,
  InfoRow,
  Notice,
  Panel,
  VX_CODE,
  VX_FAINT,
  VX_INPUT,
  VX_INSET,
  VX_MUTED,
} from "@/components/vx/panel-ui";
import { useT } from "@/hooks/use-translations";
import {
  downloadPanelTransferArchive,
  importPanelTransferArchive,
  inspectPanelTransferArchive,
  type PanelTransferArchiveInfo,
} from "@/lib/api";
import { cn } from "@/lib/utils";
import { formatBytes, SectionLabel } from "./transfer-ui";

export function ArchiveExportCard({ disabled }: { disabled?: boolean }) {
  const t = useT();
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const short = password.length > 0 && password.length < 12;

  const run = async () => {
    setBusy(true);
    try {
      await downloadPanelTransferArchive(password);
      toast.success(t("admin.panel_transfer.export.done"));
      setPassword("");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t("admin.panel_transfer.export.failed"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Panel
      title={
        <span className="inline-flex items-center gap-2">
          <Download className="size-3.5" />
          {t("admin.panel_transfer.export.title")}
        </span>
      }
    >
      <div className="grid gap-3.5">
        <p className={cn("text-[12.5px] leading-[1.55]", VX_FAINT)}>
          {t("admin.panel_transfer.export.hint")}
        </p>
        <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end">
          <Field label={t("admin.panel_transfer.export.password")}>
            <input
              className={VX_INPUT}
              type="password"
              autoComplete="new-password"
              value={password}
              disabled={disabled || busy}
              onChange={(e) => setPassword(e.target.value)}
            />
          </Field>
          <Btn tone="primary" disabled={disabled || busy || password.length < 12} onClick={() => void run()}>
            {busy ? t("admin.panel_transfer.export.busy") : t("admin.panel_transfer.export.action")}
          </Btn>
        </div>
        {short && (
          <div className={cn("text-[11.5px]", VX_MUTED)}>{t("admin.panel_transfer.export.too_short")}</div>
        )}
        <Notice tone="warn">{t("admin.panel_transfer.export.warning")}</Notice>
      </div>
    </Panel>
  );
}

export function ArchiveImportCard({ onDone }: { onDone: () => void }) {
  const t = useT();
  const fileRef = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [password, setPassword] = useState("");
  const [info, setInfo] = useState<PanelTransferArchiveInfo | null>(null);
  const [lines, setLines] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);

  const inspect = async () => {
    if (!file) return;
    setBusy(true);
    try {
      setInfo(await inspectPanelTransferArchive(file, password));
    } catch (e) {
      setInfo(null);
      toast.error(e instanceof Error ? e.message : t("admin.panel_transfer.import.inspect_failed"));
    } finally {
      setBusy(false);
    }
  };

  const run = async () => {
    if (!file) return;
    setBusy(true);
    setLines([]);
    try {
      await importPanelTransferArchive(file, password, (line) => setLines((prev) => [...prev, line]));
      toast.success(t("admin.panel_transfer.import.done"));
      onDone();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t("admin.panel_transfer.import.failed"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Panel
      title={
        <span className="inline-flex items-center gap-2">
          <Upload className="size-3.5" />
          {t("admin.panel_transfer.import.title")}
        </span>
      }
    >
      <div className="grid gap-3.5">
        <p className={cn("text-[12.5px] leading-[1.55]", VX_FAINT)}>
          {t("admin.panel_transfer.import.hint")}
        </p>

        <button
          type="button"
          disabled={busy}
          onClick={() => fileRef.current?.click()}
          className={cn(
            "flex items-center gap-3 rounded-[11px] border-dashed px-3.5 py-3 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-55",
            VX_INSET,
            "hover:border-[var(--vx-border-hover)]"
          )}
        >
          <span className={cn("inline-flex size-[30px] shrink-0 items-center justify-center rounded-[9px] bg-[var(--vx-card)]", VX_MUTED)}>
            <FileCheck2 className="size-4" />
          </span>
          <span className="grid gap-0.5">
            <span className="text-[12.5px] text-[var(--vx-fg)]">
              {file ? file.name : t("admin.panel_transfer.import.pick")}
            </span>
            <span className={cn("text-[11.5px]", VX_FAINT)}>
              {file ? formatBytes(file.size) : t("admin.panel_transfer.import.pick_hint")}
            </span>
          </span>
        </button>
        <input
          ref={fileRef}
          type="file"
          accept=".vxt"
          className="hidden"
          onChange={(e) => {
            setFile(e.target.files?.[0] ?? null);
            setInfo(null);
          }}
        />

        <Field label={t("admin.panel_transfer.import.password")}>
          <input
            className={VX_INPUT}
            type="password"
            autoComplete="new-password"
            value={password}
            disabled={busy}
            onChange={(e) => {
              setPassword(e.target.value);
              setInfo(null);
            }}
          />
        </Field>

        {info && (
          <div className="grid gap-0">
            <InfoRow k={t("admin.panel_transfer.import.archive_version")} v={info.manifest.panel_version} />
            <InfoRow k={t("admin.panel_transfer.import.archive_created")} v={info.manifest.created_at} />
            <InfoRow k={t("admin.panel_transfer.import.archive_db")} v={formatBytes(info.manifest.db_bytes)} />
            <InfoRow
              k={t("admin.panel_transfer.import.archive_uploads")}
              v={formatBytes(info.manifest.uploads_bytes)}
            />
            {info.manifest.source_address && (
              <InfoRow k={t("admin.panel_transfer.import.archive_source")} v={info.manifest.source_address} />
            )}
          </div>
        )}
        {info?.incompatible && <Notice>{info.incompatible}</Notice>}
        {info && !info.incompatible && <Notice tone="warn">{t("admin.panel_transfer.import.warning")}</Notice>}

        <div className="flex flex-wrap gap-2">
          <Btn disabled={!file || password.length < 1 || busy} onClick={() => void inspect()}>
            {t("admin.panel_transfer.import.inspect")}
          </Btn>
          <Btn tone="danger" disabled={!file || !info || !!info.incompatible || busy} onClick={() => void run()}>
            {busy ? t("admin.panel_transfer.import.busy") : t("admin.panel_transfer.import.action")}
          </Btn>
        </div>

        {lines.length > 0 && (
          <div className="grid gap-2">
            <SectionLabel>{t("admin.panel_transfer.progress.log")}</SectionLabel>
            <pre
              className={cn(
                "max-h-[240px] overflow-auto rounded-[10px] px-3 py-2.5 font-mono text-[11.5px] leading-[1.6] whitespace-pre-wrap",
                VX_CODE,
                VX_MUTED
              )}
            >
              {lines.join("\n")}
            </pre>
          </div>
        )}
      </div>
    </Panel>
  );
}
