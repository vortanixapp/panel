"use client";

import { useEffect, useRef, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";

import { AddressFields, addressOf, type ProfileAddress } from "@/components/accounting/address-fields";
import { useSetAccount } from "@/hooks/use-account";
import { useT } from "@/hooks/use-translations";
import { updateAccount, type AccountUser } from "@/lib/api";

import { errorText, FormActions, SettingsSection, useUnsavedGuard } from "./ui";

export function AddressCard({ user }: { user: AccountUser }) {
  const t = useT();
  const setAccount = useSetAccount();
  const [form, setForm] = useState<ProfileAddress>(() => addressOf(user));
  const [base, setBase] = useState<ProfileAddress>(() => addressOf(user));
  const baseRef = useRef(base);
  baseRef.current = base;
  const dirty = JSON.stringify(form) !== JSON.stringify(base);
  const userKey = JSON.stringify(addressOf(user));
  useUnsavedGuard(dirty);

  useEffect(() => {
    const next = JSON.parse(userKey) as ProfileAddress;
    setForm((current) => (JSON.stringify(current) === JSON.stringify(baseRef.current) ? next : current));
    setBase(next);
  }, [userKey]);

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
      toast.success(t("profile.saved"));
    },
    onError: (err) => toast.error(errorText(err, t("common.save_failed"))),
  });

  return (
    <SettingsSection title={t("profile.address_title")} description={t("profile.address_hint")}>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (dirty) save.mutate();
        }}
      >
        <AddressFields
          value={form}
          onChange={setForm}
          disabled={save.isPending}
          countryInvalid={!user.profile_complete && !form.country}
        />
        <FormActions dirty={dirty} pending={save.isPending} onReset={() => setForm(base)} />
      </form>
    </SettingsSection>
  );
}
