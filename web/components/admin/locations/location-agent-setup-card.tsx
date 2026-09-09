"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  fetchNodeInstall,
  regenerateAdminLocationAgentToken,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";

type Props = {
  locationId: string;
  agentToken?: string;
  relayUrl?: string;
  description?: string;
  compact?: boolean;
};

const MASKED_TOKEN = "agt_••••••••••••••••••••••••••••••••";

async function copyText(t: TranslateFn, text: string, label: string) {
  try {
    await navigator.clipboard.writeText(text);
    toast.success(t("admin.agent_setup.copied", { label }));
  } catch {
    toast.error(t("common.copy_failed"));
  }
}

export function LocationAgentSetupCard({
  locationId,
  agentToken,
  relayUrl,
  description,
  compact = false,
}: Props) {
  const t = useT();
  const [install, setInstall] = useState<{
    agent_token: string;
    relay_url: string;
    env_file: string;
    script: string;
  } | null>(null);
  const [revealed, setRevealed] = useState(false);

  const loadMut = useMutation({
    mutationFn: () => fetchNodeInstall(locationId),
    onSuccess: (res) => {
      setInstall({
        agent_token: res.agent_token ?? "",
        relay_url: res.relay_url ?? "",
        env_file: res.env_file ?? "",
        script: res.script ?? "",
      });
      setRevealed(true);
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const regenMut = useMutation({
    mutationFn: () => regenerateAdminLocationAgentToken(locationId),
    onSuccess: (res) => {
      toast.success(t("admin.agent_setup.token_regenerated"));
      void loadMut.mutateAsync();
      setInstall((prev) =>
        prev
          ? {
              ...prev,
              agent_token: res.agent_token,
              env_file: prev.env_file.replace(
                /AGENT_TOKEN=.*/,
                `AGENT_TOKEN=${res.agent_token}`
              ),
            }
          : null
      );
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const token = install?.agent_token || agentToken || "";
  const relay = install?.relay_url || relayUrl || "";
  const envPreview =
    install?.env_file ||
    (token
      ? `RELAY_URL=${relay}\nAGENT_TOKEN=${token}\nNODE_ID=${locationId}\n`
      : "");

  const busy = loadMut.isPending || regenMut.isPending;
  const showSecrets = revealed && !!token;

  return (
    <section className="flex flex-col gap-5 rounded-2xl border bg-card px-5 py-5 sm:px-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <div className="text-[15px] leading-none font-semibold">
            {t("admin.agent_setup.title")}
          </div>
          <div className="text-xs text-muted-foreground">
            {description ?? t("admin.agent_setup.description")}
          </div>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            className="h-8 text-xs"
            disabled={busy}
            onClick={() => {
              if (revealed) {
                setRevealed(false);
                return;
              }
              loadMut.mutate();
            }}
          >
            {loadMut.isPending
              ? t("common.loading")
              : revealed
                ? t("admin.agent_setup.hide")
                : t("admin.agent_setup.show")}
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="h-8 text-xs"
            disabled={busy}
            onClick={() => {
              if (!confirm(t("admin.agent_setup.regen_confirm"))) return;
              regenMut.mutate();
            }}
          >
            {regenMut.isPending
              ? t("admin.agent_setup.generating")
              : t("admin.agent_setup.new_token")}
          </Button>
        </div>
      </div>

      <div className="grid gap-3.5 sm:grid-cols-2">
        <ValueBox label="NODE_ID" value={locationId}>
          <CopyButton onClick={() => copyText(t, locationId, "NODE_ID")} />
        </ValueBox>
        <ValueBox
          label="RELAY_URL"
          value={
            revealed && relay ? relay : t("admin.agent_setup.press_show")
          }
          muted={!revealed || !relay}
        />
        {!compact && (
          <ValueBox
            className="sm:col-span-2"
            label="AGENT_TOKEN"
            value={showSecrets ? token : MASKED_TOKEN}
            muted={!showSecrets}
          >
            {showSecrets && (
              <CopyButton onClick={() => copyText(t, token, "AGENT_TOKEN")} />
            )}
          </ValueBox>
        )}
      </div>

      {!compact && showSecrets && envPreview && (
        <div className="flex flex-col gap-2.5 border-t pt-4.5">
          <div className="flex items-center justify-between gap-3.5">
            <span className="text-xs text-muted-foreground">
              {t("admin.agent_setup.env_before")}{" "}
              <span className="font-mono">deploy/agent/.env</span>{" "}
              {t("admin.agent_setup.env_after")}
            </span>
            <Button
              variant="outline"
              size="sm"
              className="h-[30px] rounded-md text-xs"
              onClick={() => copyText(t, envPreview, ".env")}
            >
              {t("admin.location.copy_env")}
            </Button>
          </div>
          <pre className="max-h-40 overflow-auto rounded-xl border bg-muted/30 px-4 py-4 font-mono text-xs leading-relaxed text-muted-foreground">
            {envPreview}
          </pre>
          <span className="text-xs text-muted-foreground">
            {t("admin.agent_setup.on_node")}{" "}
            <span className="font-mono text-foreground">
              cd ~/vortanix/deploy/agent && nano .env && sh agent-up.sh
            </span>
          </span>
        </div>
      )}
    </section>
  );
}

function ValueBox({
  label,
  value,
  muted,
  className,
  children,
}: {
  label: string;
  value: string;
  muted?: boolean;
  className?: string;
  children?: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "flex flex-col gap-2 rounded-xl border bg-muted/30 px-4 py-4",
        className
      )}
    >
      <span className="font-mono text-[11px] tracking-wide text-muted-foreground">
        {label}
      </span>
      <span
        className={cn(
          "font-mono text-xs break-all",
          muted && "text-muted-foreground/60"
        )}
      >
        {value}
      </span>
      {children}
    </div>
  );
}

function CopyButton({ onClick }: { onClick: () => void }) {
  const t = useT();
  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      className="h-[26px] w-fit rounded-md px-2.5 font-mono text-[11px]"
      onClick={onClick}
    >
      {t("admin.agent_setup.copy")}
    </Button>
  );
}
