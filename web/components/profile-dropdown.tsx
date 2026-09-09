"use client";

import Link from "next/link";
import useDialogState from "@/hooks/use-dialog-state";
import { useMe } from "@/hooks/use-queries";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { roleLabel } from "@/lib/rbac";
import { SignOutDialog } from "@/components/sign-out-dialog";
import {
  AccountSwitchItems,
  ActiveAccountCheck,
} from "@/components/layout/account-switch-items";
import { accountInitials } from "@/lib/accounts";
import { useBrand } from "@/context/brand-provider";
import { useT } from "@/hooks/use-translations";

export function ProfileDropdown() {
  const t = useT();
  const [open, setOpen] = useDialogState();
  const { data: me } = useMe();
  const email = me?.email ?? "";
  const { userMenuVariant } = useBrand();
  const expanded = userMenuVariant === "screenshot";

  return (
    <>
      <DropdownMenu modal={false}>
        <DropdownMenuTrigger asChild>
          {expanded ? (
            <Button
              variant="ghost"
              className="relative h-9 gap-2 rounded-full px-1.5 pr-3"
            >
              <Avatar className="h-7 w-7">
                <AvatarFallback>{accountInitials(email)}</AvatarFallback>
              </Avatar>
              <span className="flex min-w-0 flex-col items-start leading-tight">
                <span className="max-w-[160px] truncate text-xs font-medium">
                  {email}
                </span>
                <span className="text-[10px] text-muted-foreground">
                  {roleLabel(me?.role ?? "user")}
                </span>
              </span>
            </Button>
          ) : (
            <Button variant="ghost" className="relative h-8 w-8 rounded-full">
              <Avatar className="h-8 w-8">
                <AvatarFallback>{accountInitials(email)}</AvatarFallback>
              </Avatar>
            </Button>
          )}
        </DropdownMenuTrigger>
        <DropdownMenuContent className="w-56" align="end" forceMount>
          <DropdownMenuLabel className="font-normal">
            <div className="flex items-center gap-2">
              <div className="flex min-w-0 flex-col gap-1.5">
                <p className="truncate text-sm font-medium leading-none">{email}</p>
                <p className="text-xs leading-none text-muted-foreground">
                  {roleLabel(me?.role ?? "user")}
                </p>
              </div>
              <ActiveAccountCheck on />
            </div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />

          <AccountSwitchItems />
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem asChild>
              <Link href="/settings">{t("layout.user.profile")}</Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <Link href="/settings/account">{t("layout.user.account")}</Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <Link href="/settings?tab=appearance">
                {t("layout.user.appearance")}
              </Link>
            </DropdownMenuItem>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            className="text-destructive focus:text-destructive"
            onClick={() => setOpen(true)}
          >
            {t("layout.user.sign_out")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <SignOutDialog open={!!open} onOpenChange={setOpen} />
    </>
  );
}
