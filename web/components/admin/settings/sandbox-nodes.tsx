"use client";

import { useQuery } from "@tanstack/react-query";
import { Notice, VX_FAINT } from "@/components/vx/panel-ui";
import {
  SandboxBadge,
  SandboxInstallButton,
  sandboxGaps,
  sandboxState,
} from "@/components/admin/agents/sandbox-install";
import { fetchAgents, type AgentRow } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

type Translate = ReturnType<typeof useT>;

const NAMES_SHOWN = 6;

function names(rows: AgentRow[]) {
  const list = rows.slice(0, NAMES_SHOWN).map((row) => row.name || row.code);
  if (rows.length > NAMES_SHOWN) list.push(`+${rows.length - NAMES_SHOWN}`);
  return list.join(", ");
}

export function sandboxWarnings(t: Translate, nodes: AgentRow[], mode: string, strict: boolean) {
  if (mode === "off") return [];
  const { missing, unknown } = sandboxGaps(nodes);
  const out: string[] = [];
  if (missing.length > 0) {
    out.push(t(strict ? "admin.agents.sandbox.warn_strict" : "admin.agents.sandbox.warn_soft", { nodes: names(missing) }));
  }
  if (unknown.length > 0) {
    out.push(t("admin.agents.sandbox.warn_old", { nodes: names(unknown) }));
  }
  return out;
}

export function SandboxNodes({ mode, strict }: { mode: string; strict: boolean }) {
  const t = useT();
  const query = useQuery({
    queryKey: queryKeys.agents,
    queryFn: fetchAgents,
    refetchInterval: 20_000,
  });
  const nodes = query.data?.agents ?? [];
  const canWrite = (query.data as { can_write?: boolean } | undefined)?.can_write !== false;
  const warnings = sandboxWarnings(t, nodes, mode, strict);

  return (
    <div className="flex flex-col gap-3" data-testid="sandbox-nodes">
      <div className="text-[13px] font-medium">{t("admin.agents.sandbox.nodes_title")}</div>
      {warnings.map((text) => (
        <Notice key={text} tone="warn">
          {text}
        </Notice>
      ))}
      {nodes.length === 0 ? (
        <div className={cn("text-[12.5px]", VX_FAINT)}>{t("admin.agents.sandbox.nodes_empty")}</div>
      ) : (
        <div className="divide-y rounded-[12px] border">
          {nodes.map((node) => (
            <div key={node.id} data-node-id={node.id} className="flex flex-wrap items-center gap-3 px-4 py-2.5">
              <div className="min-w-0 flex-1">
                <div className="truncate text-[13px] font-medium">{node.name || node.code}</div>
                <div className={cn("truncate text-[11.5px]", VX_FAINT)}>{node.host || node.code}</div>
              </div>
              <SandboxBadge node={node} />
              {sandboxState(node) !== "present" && (
                <SandboxInstallButton nodeId={node.id} node={node} canWrite={canWrite} />
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
