"use client";

import { useQuery } from "@tanstack/react-query";
import { fetchTenantStatus, TENANT_SLUG } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export function useTenantStatus() {
  return useQuery({
    queryKey: queryKeys.tenantStatus,
    queryFn: () => fetchTenantStatus(TENANT_SLUG),
    staleTime: 60_000,
  });
}
