"use client";

import { useEffect, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";

import { AddressFields, addressOf, EMPTY_ADDRESS, type ProfileAddress } from "@/components/accounting/address-fields";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useAccountQuery, useSetAccount } from "@/hooks/use-account";
import { useT } from "@/hooks/use-translations";
import { PROFILE_INCOMPLETE_EVENT, updateAccount } from "@/lib/api";

const DISMISS_KEY = "vtx_profile_gate_dismissed";

function dismissedThisSession(): boolean {
  try {
    return sessionStorage.getItem(DISMISS_KEY) === "1";
  } catch {
    return false;
  }
}

function rememberDismiss() {
  try {
    sessionStorage.setItem(DISMISS_KEY, "1");
  } catch {
    return;
  }
}

export function ProfileCompletionGate() {
  const t = useT();
  const account = useAccountQuery();
  const setAccount = useSetAccount();
  const user = account.data?.user;
  const incomplete = !!user && user.profile_complete === false;
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState<ProfileAddress>(EMPTY_ADDRESS);

  useEffect(() => {
    if (incomplete && !dismissedThisSession()) setOpen(true);
  }, [incomplete]);

  useEffect(() => {
    const handler = () => setOpen(true);
    window.addEventListener(PROFILE_INCOMPLETE_EVENT, handler);
    return () => window.removeEventListener(PROFILE_INCOMPLETE_EVENT, handler);
  }, []);

  useEffect(() => {
    if (user) setForm(addressOf(user));
  }, [user]);

  const save = useMutation({
    mutationFn: () =>
      updateAccount({
        middle_name: form.middle_name.trim(),
        country: form.country,
        address_line: form.address_line.trim(),
        city: form.city.trim(),
        region: form.region.trim(),
        postal_code: form.postal_code.trim(),
      }),
    onSuccess: (res) => {
      setAccount(res.user);
      setOpen(false);
      toast.success(t("profile.saved"));
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : t("common.save_failed")),
  });

  if (!incomplete) return null;

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) rememberDismiss();
        setOpen(next);
      }}
    >
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{t("profile.gate_title")}</DialogTitle>
          <DialogDescription>{t("profile.gate_text")}</DialogDescription>
        </DialogHeader>
        <AddressFields value={form} onChange={setForm} disabled={save.isPending} countryInvalid={!form.country} />
        <DialogFooter>
          <Button variant="ghost" onClick={() => setOpen(false)} disabled={save.isPending}>
            {t("profile.gate_later")}
          </Button>
          <Button onClick={() => save.mutate()} disabled={!form.country || save.isPending}>
            {t("common.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
