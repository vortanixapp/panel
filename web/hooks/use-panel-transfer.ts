"use client";

import { useQuery } from "@tanstack/react-query";

import { fetchPanelTransfer } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

import { pollMs } from "@/lib/public-settings";
export function usePanelTransfer() {
  return useQuery({
    queryKey: queryKeys.panelTransfer,
    queryFn: fetchPanelTransfer,
    refetchInterval: (query) => (query.state.data?.active ? pollMs(2000) : pollMs(30_000)),
  });
}
