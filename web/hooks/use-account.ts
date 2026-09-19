"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchAccount, type AccountUser } from "@/lib/api";

export const ACCOUNT_KEY = ["account"] as const;

export function useAccountQuery() {
  return useQuery({
    queryKey: ACCOUNT_KEY,
    queryFn: fetchAccount,
    staleTime: 60_000,
  });
}

export function useSetAccount() {
  const qc = useQueryClient();
  return (user: AccountUser) => qc.setQueryData(ACCOUNT_KEY, { user });
}
