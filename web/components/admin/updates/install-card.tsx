"use client";

import { useState } from "react";
import { Check, ChevronDown, Copy } from "lucide-react";
import { toast } from "sonner";
import { Segmented, SettingsCard } from "@/components/admin/settings/settings-ui";
import type { AdminUpdates } from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";

const FROM_IMAGES = `git pull
docker compose -f deploy/docker-compose.yml \\
  -f deploy/docker-compose.images.yml pull
docker compose -f deploy/docker-compose.yml \\
  -f deploy/docker-compose.images.yml up -d`;

const FROM_SOURCE = `git pull
docker compose -f deploy/docker-compose.yml up -d --build`;

type Method = "images" | "source";

export function InstallCard({
  data,
  manualOpen,
  onManualOpenChange,
}: {
  data: AdminUpdates;
  manualOpen: boolean;
  onManualOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const updater = data.updater;
  const available = updater?.available === true;
  const [method, setMethod] = useState<Method>(updater?.mode === "source" ? "source" : "images");
  const [copied, setCopied] = useState(false);
  const commands = method === "images" ? FROM_IMAGES : FROM_SOURCE;
  const repoUrl = data.repo ? `https://github.com/${data.repo}/releases` : "";

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(commands);
      setCopied(true);
      toast.success(t("common.copied"));
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      toast.error(t("admin.updates.install.copy_failed"));
    }
  };

  const rows: { label: string; value: React.ReactNode }[] = [
    {
      label: t("admin.updates.install.updater"),
      value: (
        <span className="inline-flex items-center gap-2">
          <span className={cn("size-2 rounded-full", available ? "bg-emerald-500" : "bg-amber-500")} />
          {available ? t("admin.updates.install.updater_on") : t("admin.updates.install.updater_off")}
        </span>
      ),
    },
    {
      label: t("admin.updates.install.mode"),
      value: available ? t(`admin.updates.mode.${updater.mode}`) : t("admin.updates.mode.manual"),
    },
  ];
  if (updater?.project) {
    rows.push({ label: t("admin.updates.install.project"), value: <span className="font-mono">{updater.project}</span> });
  }
  if (data.repo) {
    rows.push({
      label: t("admin.updates.install.repo"),
      value: (
        <a href={repoUrl} target="_blank" rel="noreferrer noopener" className="font-mono text-primary hover:underline">
          {data.repo}
        </a>
      ),
    });
  }

  return (
    <div id="manual" className="scroll-mt-24">
      <SettingsCard title={t("admin.updates.install.title")}>
        <dl className="space-y-2.5 text-[13px]">
          {rows.map((row) => (
            <div key={row.label} className="flex items-baseline justify-between gap-4">
              <dt className="shrink-0 text-muted-foreground">{row.label}</dt>
              <dd className="min-w-0 truncate text-end font-medium">{row.value}</dd>
            </div>
          ))}
        </dl>
        {!available && updater?.reason && (
          <p className="mt-3 rounded-lg border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-[12.5px] text-amber-700 dark:text-amber-400">
            {updater.reason}
          </p>
        )}

        <div className="mt-5 border-t pt-4">
          <button
            type="button"
            onClick={() => onManualOpenChange(!manualOpen)}
            aria-expanded={manualOpen}
            className="flex w-full items-center justify-between gap-3 text-start text-[14px] font-medium"
          >
            {t("admin.updates.install.manual")}
            <ChevronDown className={cn("size-4 text-muted-foreground transition-transform", manualOpen && "rotate-180")} />
          </button>
          {manualOpen && (
            <div className="mt-3 space-y-3">
              <p className="text-[12.5px] leading-relaxed text-muted-foreground">
                {available ? t("admin.updates.install.manual_hint") : t("admin.updates.install.enable_updater")}
              </p>
              <Segmented
                size="sm"
                className="w-full"
                items={[
                  { id: "images", label: t("admin.updates.install.from_images") },
                  { id: "source", label: t("admin.updates.install.from_source") },
                ]}
                value={method}
                onChange={setMethod}
              />
              <div className="relative">
                <pre className="overflow-x-auto rounded-xl border bg-[var(--vx-surface-2)] px-4 py-3 pe-12 font-mono text-[12px] leading-relaxed">
                  {commands}
                </pre>
                <button
                  type="button"
                  onClick={() => void copy()}
                  aria-label={t("common.copy")}
                  className="absolute top-2 right-2 grid size-8 place-items-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground"
                >
                  {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
                </button>
              </div>
              <p className="text-[12px] text-amber-600 dark:text-amber-500">{t("admin.updates.install.backup_first")}</p>
            </div>
          )}
        </div>
      </SettingsCard>
    </div>
  );
}
