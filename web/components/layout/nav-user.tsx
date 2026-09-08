"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ChevronsUpDown, LogOut } from "lucide-react";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar";
import { useQueryClient } from "@tanstack/react-query";
import {
  AccountSwitchItems,
  ActiveAccountCheck,
} from "@/components/layout/account-switch-items";
import { accountInitials, listAccounts } from "@/lib/accounts";
import {
  forgetEveryAccount,
  goToAccount,
  nextAccountAfterLogout,
  prepareSwitch,
} from "@/lib/account-switch";
import { activeAccountId, logout as apiLogout } from "@/lib/api";
import { roleLabel } from "@/lib/rbac";
import { useT } from "@/hooks/use-translations";

type NavUserProps = {
  email: string;
  role?: string;
};

export function NavUser({ email, role = "user" }: NavUserProps) {
  const t = useT();
  const router = useRouter();
  const queryClient = useQueryClient();
  const { isMobile } = useSidebar();

  // Сколько аккаунтов сохранено — от этого зависит только строка «выйти из
  // всех». Сам список рисует AccountSwitchItems.
  const [hasOthers, setHasOthers] = useState(false);

  useEffect(() => {
    const id = activeAccountId();
    setHasOthers(listAccounts().some((a) => a.id !== id));
  }, []);

  async function logout() {
    // Кэш запросов чистим обязательно: он переживает выход, и следующий
    // вошедший видел бы в нём данные предыдущего, пока не придут свои.
    const next = nextAccountAfterLogout(activeAccountId());
    try {
      await apiLogout();
    } finally {
      queryClient.clear();
      // Из этого аккаунта вышли, но соседний остался — уводим в него, а не на
      // форму входа: пароля для этого не нужно, и спрашивать его незачем.
      if (next) {
        const outcome = await prepareSwitch(next.id);
        if (outcome.ok) {
          goToAccount(outcome.path);
          return;
        }
      }
      router.replace("/login");
    }
  }

  async function logoutEverywhere() {
    try {
      await apiLogout();
    } finally {
      forgetEveryAccount();
      queryClient.clear();
      router.replace("/login");
    }
  }

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
            >
              <Avatar className="h-8 w-8 rounded-lg">
                <AvatarFallback className="rounded-lg">
                  {accountInitials(email)}
                </AvatarFallback>
              </Avatar>
              <div className="grid flex-1 text-start text-sm leading-tight">
                <span className="truncate font-semibold">{email}</span>
                <span className="truncate text-xs text-muted-foreground">
                  {roleLabel(role)}
                </span>
              </div>
              <ChevronsUpDown className="ms-auto size-4" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-[--radix-dropdown-menu-trigger-width] min-w-64 rounded-lg"
            side={isMobile ? "bottom" : "right"}
            align="end"
            sideOffset={4}
          >
            <DropdownMenuLabel className="p-0 font-normal">
              <div className="flex items-center gap-2 px-1 py-1.5 text-start text-sm">
                <Avatar className="h-8 w-8 rounded-lg">
                  <AvatarFallback className="rounded-lg">
                    {accountInitials(email)}
                  </AvatarFallback>
                </Avatar>
                <div className="grid min-w-0 flex-1 text-start text-sm leading-tight">
                  <span className="truncate font-semibold">{email}</span>
                  <span className="truncate text-xs text-muted-foreground">
                    {roleLabel(role)}
                  </span>
                </div>
                <ActiveAccountCheck on />
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />

            <AccountSwitchItems />

            <DropdownMenuItem asChild>
              <Link href="/settings">{t("common.settings")}</Link>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive" onClick={logout}>
              <LogOut />
              {t("layout.user.sign_out")}
            </DropdownMenuItem>
            {hasOthers && (
              <DropdownMenuItem variant="destructive" onClick={logoutEverywhere}>
                {t("layout.user.sign_out_all")}
              </DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
