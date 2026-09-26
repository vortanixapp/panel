"use client";

import { useQuery } from "@tanstack/react-query";

import { fetchPanelTransfer } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export function usePanelTransfer() {
  return useQuery({
    queryKey: queryKeys.panelTransfer,
    queryFn: fetchPanelTransfer,
    refetchInterval: (query) => (query.state.data?.active ? 2000 : 30_000),
  });
}
