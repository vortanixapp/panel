"use client";

import { InfoRow, Panel, Tile } from "@/components/vx/panel-ui";
import { AgentMetricChart } from "@/components/admin/agents/agent-metric-chart";
import type { AgentTabProps } from "@/components/admin/agents/agent-shell";
import { diskUsedPct, formatDateTime, formatDuration, formatMB, pct } from "@/lib/agents";
import { useT } from "@/hooks/use-translations";

export function AgentOverviewTab({ id, agent }: AgentTabProps) {
  const t = useT();
  const r = agent.resources;
  const f = agent.facts ?? {};
  const cpu = pct(r.cpu_percent);
  const ram = pct(r.ram_percent);
  const disk = diskUsedPct(agent);
  const serversPct = agent.servers.total ? (agent.servers.running / agent.servers.total) * 100 : 0;
  const yes = t("admin.agents.yes");
  const no = t("admin.agents.no");

  const hostRows: [string, React.ReactNode][] = [
    [t("admin.agents.fact.hostname"), f.hostname || "—"],
    [t("admin.agents.fact.os"), f.os || agent.platform || "—"],
    [t("admin.agents.fact.kernel"), f.kernel || "—"],
    [t("admin.agents.fact.arch"), f.arch || "—"],
    [t("admin.agents.fact.cpu"), f.cpu_model ? `${f.cpu_model}${f.cpus ? ` × ${f.cpus}` : ""}` : f.cpus ?? "—"],
    [t("admin.agents.fact.ram"), f.mem_total_mb ? formatMB(f.mem_total_mb) : "—"],
    [t("admin.agents.fact.host_uptime"), formatDuration(agent.host_uptime_sec ?? f.uptime_sec)],
    [t("admin.agents.fact.data_dir"), f.data_dir || "—"],
    [t("admin.agents.fact.quota"), f.quota == null ? "—" : f.quota ? yes : no],
  ];
  const dockerRows: [string, React.ReactNode][] = [
    [t("admin.agents.fact.docker"), f.docker?.version ? `${f.docker.version}${f.docker.api ? ` (API ${f.docker.api})` : ""}` : f.docker_error || "—"],
    [t("admin.agents.fact.cgroup"), f.docker?.cgroup ? `v${f.docker.cgroup} · ${f.docker.cgroup_driver ?? ""}` : "—"],
    [t("admin.agents.fact.storage"), f.docker?.storage_driver || "—"],
    [t("admin.agents.fact.memory_limit"), f.docker?.memory_limit == null ? "—" : f.docker.memory_limit ? yes : no],
    [t("admin.agents.fact.image"), f.agent?.image || "—"],
    [t("admin.agents.fact.container"), f.agent?.container_name || "—"],
    [t("admin.agents.fact.restart_policy"), f.agent?.restart_policy || "—"],
  ];
  const linkRows: [string, React.ReactNode][] = [
    [t("admin.agents.fact.protocol"), agent.proto ? `v${agent.proto}` : "—"],
    [t("admin.agents.fact.connected_at"), formatDateTime(agent.connection.connected_at)],
    [t("admin.agents.fact.remote_addr"), agent.connection.remote_addr || "—"],
    [t("admin.agents.fact.rtt"), agent.connection.rtt_ms != null ? `${agent.connection.rtt_ms} ms` : "—"],
    [t("admin.agents.fact.last_seen"), formatDateTime(agent.last_seen)],
    [t("admin.agents.fact.disconnected_at"), agent.connection.disconnected_at ? `${formatDateTime(agent.connection.disconnected_at)}${agent.connection.disconnect_reason ? ` · ${agent.connection.disconnect_reason}` : ""}` : "—"],
    [t("admin.agents.fact.boot_id"), agent.boot_id ? agent.boot_id.slice(0, 12) : "—"],
  ];

  return (
    <div className="flex flex-col gap-[18px]">
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Tile label="CPU" value={cpu == null ? "—" : `${Math.round(cpu)}%`} sub={f.cpus ? t("admin.agents.tile.cores", { count: f.cpus }) : "—"} pct={cpu ?? 0} />
        <Tile
          label="RAM"
          value={ram == null ? "—" : `${Math.round(ram)}%`}
          sub={r.ram_total_mb ? `${formatMB(r.ram_used_mb)} / ${formatMB(r.ram_total_mb)}` : "—"}
          pct={ram ?? 0}
        />
        <Tile
          label={t("admin.agents.disk_short")}
          value={disk == null ? "—" : `${Math.round(disk)}%`}
          sub={r.disk_total_mb ? t("admin.agents.tile.disk_free", { free: formatMB(r.disk_free_mb), total: formatMB(r.disk_total_mb) }) : "—"}
          pct={disk ?? 0}
        />
        <Tile
          label={t("admin.agents.col.servers")}
          value={`${agent.servers.running}/${agent.servers.total}`}
          sub={t("admin.agents.tile.servers_running")}
          pct={serversPct}
        />
      </div>

      <AgentMetricChart id={id} />

      <div className="grid gap-[18px] lg:grid-cols-3">
        <Panel title={t("admin.agents.facts.host")}>
          {hostRows.map(([k, v]) => (
            <InfoRow key={k} k={k} v={v} />
          ))}
        </Panel>
        <Panel title={t("admin.agents.facts.docker")}>
          {dockerRows.map(([k, v]) => (
            <InfoRow key={k} k={k} v={v} />
          ))}
        </Panel>
        <Panel title={t("admin.agents.facts.link")}>
          {linkRows.map(([k, v]) => (
            <InfoRow key={k} k={k} v={v} />
          ))}
        </Panel>
      </div>
    </div>
  );
}
