"use client";

import { useRouter } from "next/navigation";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { clearAuth } from "@/lib/api";
import { useT } from "@/hooks/use-translations";

interface SignOutDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SignOutDialog({ open, onOpenChange }: SignOutDialogProps) {
  const t = useT();
  const router = useRouter();

  const handleSignOut = () => {
    clearAuth();
    onOpenChange(false);
    router.replace("/login");
  };

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title={t("layout.user.sign_out")}
      desc={t("layout.sign_out.desc")}
      confirmText={t("layout.user.sign_out")}
      destructive
      handleConfirm={handleSignOut}
      className="sm:max-w-sm"
    />
  );
}
