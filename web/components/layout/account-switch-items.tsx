"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { Check, CornerUpLeft, Loader2, Plus } from "lucide-react";
import { toast } from "sonner";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { accountInitials, listAccounts, type StoredAccount } from "@/lib/accounts";
import { goToAccount, prepareSwitch } from "@/lib/account-switch";
import { activeAccountId } from "@/lib/api";
import { roleLabel } from "@/lib/rbac";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

export function AccountSwitchItems() {
  const t = useT();
  const router = useRouter();
  const queryClient = useQueryClient();

  const [accounts, setAccounts] = useState<StoredAccount[]>([]);
  const [currentId, setCurrentId] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  const reload = useCallback(() => {
    setAccounts(listAccounts());
    setCurrentId(activeAccountId());
  }, []);

  useEffect(() => {
    reload();
    const onStorage = (e: StorageEvent) => {
      if (e.key === null || e.key === "vortanix_accounts") reload();
    };
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, [reload]);

  const current = accounts.find((a) => a.id === currentId);
  const others = accounts.filter((a) => a.id !== currentId);

  async function switchTo(account: StoredAccount) {
    if (busy) return;
    setBusy(account.id);
    const outcome = await prepareSwitch(account.id);
    if (outcome.ok) {
      queryClient.clear();
      goToAccount(outcome.path);
      return;
    }
    setBusy(null);
    if (outcome.reason === "needs-sign-in") {
      toast.info(t("layout.account.session_expired", { email: outcome.email }));
      router.push(`/login?add=1&email=${encodeURIComponent(outcome.email)}`);
      return;
    }
    toast.error(outcome.message);
    reload();
  }

  return (
    <>
      {current?.viaEmail && (
        <>
          <DropdownMenuItem
            disabled={!!busy}
            onSelect={(e) => {
              e.preventDefault();
              const via = accounts.find((a) => a.id === current.viaId);
              if (via) void switchTo(via);
              else toast.error(t("layout.account.admin_missing"));
            }}
          >
            {busy === current.viaId ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <CornerUpLeft className="size-4" />
            )}
            <span className="truncate">
              {t("layout.account.return_to", { email: current.viaEmail })}
            </span>
          </DropdownMenuItem>
          <DropdownMenuSeparator />
        </>
      )}

      {others.length > 0 && (
        <>
          <DropdownMenuLabel className="text-xs font-normal text-muted-foreground">
            {t("layout.account.others")}
          </DropdownMenuLabel>
          {others.map((account) => (
            <DropdownMenuItem
              key={account.id}
              disabled={!!busy}
              onSelect={(e) => {
                e.preventDefault();
                void switchTo(account);
              }}
              className="gap-2"
            >
              <Avatar className="size-6">
                <AvatarFallback className="text-[10px]">
                  {accountInitials(account.email)}
                </AvatarFallback>
              </Avatar>
              <div className="grid min-w-0 flex-1 leading-tight">
                <span className="truncate text-[13px]">{account.email}</span>
                <span
                  className={cn(
                    "truncate text-[11px]",
                    account.needsSignIn
                      ? "text-[var(--vx-warn)]"
                      : "text-muted-foreground"
                  )}
                >
                  {account.needsSignIn
                    ? t("layout.account.needs_password")
                    : roleLabel(account.role)}
                </span>
              </div>
              {busy === account.id && (
                <Loader2 className="size-4 shrink-0 animate-spin" />
              )}
            </DropdownMenuItem>
          ))}
          <DropdownMenuSeparator />
        </>
      )}

      <DropdownMenuItem asChild>
        <Link href="/login?add=1">
          <Plus className="size-4" />
          {t("layout.account.add")}
        </Link>
      </DropdownMenuItem>
    </>
  );
}

export function ActiveAccountCheck({ on }: { on: boolean }) {
  return <Check size={14} className={cn("ms-auto", !on && "hidden")} />;
}
