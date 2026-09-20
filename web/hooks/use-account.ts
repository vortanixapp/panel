"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchAccount, type AccountUser } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

type MeSnapshot = {
  user_id: string;
  email: string;
  role: string;
  display_name?: string | null;
  avatar_url?: string | null;
};

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
  return (user: AccountUser) => {
    qc.setQueryData(ACCOUNT_KEY, { user });
    qc.setQueryData(queryKeys.me, (prev: MeSnapshot | undefined) =>
      prev
        ? {
            ...prev,
            email: user.email,
            display_name: user.display_name ?? null,
            avatar_url: user.avatar_url ?? null,
          }
        : prev
    );
  };
}
