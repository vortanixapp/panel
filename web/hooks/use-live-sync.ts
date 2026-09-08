"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useLiveDashboard } from "@/hooks/useLiveDashboard";
import { patchInstallProgress, patchNodeStatus, patchServerStatus } from "@/hooks/use-queries";
import { queryKeys } from "@/lib/query-keys";

export function useLiveSync() {
  const qc = useQueryClient();

  useLiveDashboard((ev) => {
    if (ev.type === "node.status" && ev.node_id && ev.status) {
      patchNodeStatus(qc, ev.node_id, ev.status);
    }
    if (ev.type === "server.status" && ev.server_id && ev.status) {
      patchServerStatus(qc, ev.server_id, ev.status);
    }
    if (ev.type === "server.metrics" && ev.server_id) {
      qc.invalidateQueries({ queryKey: queryKeys.metrics(ev.server_id) });
    }
    // Прогресс установки приходил и молча выбрасывался: в событии не было
    // процента, а показывать голое название этапа нечем. Теперь шкала движется
    // по событию, а не рывками раз в две секунды по опросу.
    if (ev.type === "server.install_progress" && ev.server_id && ev.percent != null) {
      patchInstallProgress(qc, ev.server_id, ev.percent, ev.status ?? "", ev.message ?? "");
    }
  });
}
