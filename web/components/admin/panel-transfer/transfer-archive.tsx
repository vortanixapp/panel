"use client";

import { useRef, useState } from "react";
import { toast } from "sonner";

import { Btn, Field, InfoRow, Notice, Panel, VX_CODE, VX_FAINT, VX_INPUT } from "@/components/vx/panel-ui";
import { useT } from "@/hooks/use-translations";
import {
  downloadPanelTransferArchive,
  importPanelTransferArchive,
  inspectPanelTransferArchive,
  type PanelTransferArchiveInfo,
} from "@/lib/api";
import { cn } from "@/lib/utils";
import { formatBytes } from "./transfer-progress";

export function ArchiveExportCard({ disabled }: { disabled?: boolean }) {
  const t = useT();
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);

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
    <Panel title={t("admin.panel_transfer.export.title")}>
      <div className="grid gap-3">
        <p className={cn("text-[12.5px]", VX_FAINT)}>{t("admin.panel_transfer.export.hint")}</p>
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
        <Notice tone="warn">{t("admin.panel_transfer.export.warning")}</Notice>
        <div>
          <Btn tone="primary" disabled={disabled || busy || password.length < 12} onClick={() => void run()}>
            {busy ? t("admin.panel_transfer.export.busy") : t("admin.panel_transfer.export.action")}
          </Btn>
        </div>
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
    <Panel title={t("admin.panel_transfer.import.title")}>
      <div className="grid gap-3">
        <p className={cn("text-[12.5px]", VX_FAINT)}>{t("admin.panel_transfer.import.hint")}</p>
        <Field label={t("admin.panel_transfer.import.file")}>
          <input
            ref={fileRef}
            type="file"
            accept=".vxt"
            className="text-[12px]"
            disabled={busy}
            onChange={(e) => {
              setFile(e.target.files?.[0] ?? null);
              setInfo(null);
            }}
          />
        </Field>
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
          <div className="grid gap-1">
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
          <Btn
            tone="danger"
            disabled={!file || !info || !!info.incompatible || busy}
            onClick={() => void run()}
          >
            {busy ? t("admin.panel_transfer.import.busy") : t("admin.panel_transfer.import.action")}
          </Btn>
        </div>

        {lines.length > 0 && (
          <pre
            className={cn(
              "max-h-[260px] overflow-auto rounded-[10px] px-3 py-2.5 font-mono text-[11.5px] leading-[1.6] whitespace-pre-wrap",
              VX_CODE
            )}
          >
            {lines.join("\n")}
          </pre>
        )}
      </div>
    </Panel>
  );
}
