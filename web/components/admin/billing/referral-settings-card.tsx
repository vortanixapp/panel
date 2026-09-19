"use client";

import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { FieldGrid, SettingsCard, TextField, ToggleRow } from "@/components/admin/settings/settings-ui";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useT } from "@/hooks/use-translations";
import { fetchAdminSettings, updateAdminSettings } from "@/lib/api";

const SETTINGS_KEY = ["admin-settings-referral"];

type Form = { enabled: boolean; percent: string; months: string; min: string };

function fromValues(values: Record<string, string> | undefined): Form {
  const read = (key: string) => String(values?.[key] ?? "").trim();
  return {
    enabled: ["1", "true", "yes", "on"].includes(read("referral.enabled").toLowerCase()),
    percent: read("referral.percent") || "10",
    months: read("referral.months") || "12",
    min: read("referral.min_payment") || "0",
  };
}

function toNumber(value: string) {
  const text = value.replace(",", ".").trim();
  return text === "" ? Number.NaN : Number(text);
}

export function ReferralSettingsCard() {
  const t = useT();
  const qc = useQueryClient();
  const query = useQuery({ queryKey: SETTINGS_KEY, queryFn: fetchAdminSettings });
  const initialKey = JSON.stringify(fromValues(query.data?.values));
  const [form, setForm] = useState<Form>(() => JSON.parse(initialKey) as Form);

  useEffect(() => {
    setForm(JSON.parse(initialKey) as Form);
  }, [initialKey]);

  const percent = toNumber(form.percent);
  const months = toNumber(form.months);
  const min = toNumber(form.min);
  const invalid = {
    percent: !(percent >= 1 && percent <= 50),
    months: !(Number.isInteger(months) && months >= 0 && months <= 120),
    min: !(min >= 0),
  };
  const valid = !invalid.percent && !invalid.months && !invalid.min;
  const dirty = JSON.stringify(form) !== initialKey;

  const save = useMutation({
    mutationFn: () =>
      updateAdminSettings({
        "referral.enabled": form.enabled ? "1" : "0",
        "referral.percent": String(percent),
        "referral.months": String(months),
        "referral.min_payment": String(min),
      }),
    onSuccess: () => {
      toast.success(t("admin.referral.saved"));
      void qc.invalidateQueries({ queryKey: SETTINGS_KEY });
    },
    onError: (e: Error) => toast.error(e.message || t("common.save_failed")),
  });

  return (
    <SettingsCard
      title={t("admin.referral.title")}
      description={t("admin.referral.description")}
      action={
        <Button
          type="button"
          className="h-8 px-4 text-[13px]"
          disabled={!dirty || !valid || save.isPending || query.isLoading}
          onClick={() => save.mutate()}
        >
          {save.isPending ? t("common.saving") : t("common.save")}
        </Button>
      }
    >
      {query.isLoading ? (
        <Skeleton className="h-28 w-full" />
      ) : (
        <div className="space-y-5">
          <ToggleRow
            label={t("admin.referral.enabled")}
            hint={t("admin.referral.enabled_hint")}
            checked={form.enabled}
            onCheckedChange={(enabled) => setForm({ ...form, enabled })}
          />
          <FieldGrid cols={3}>
            <TextField
              mono
              label={t("admin.referral.percent")}
              value={form.percent}
              onChange={(percent) => setForm({ ...form, percent })}
              placeholder="10"
              accent={invalid.percent}
              hint={t("admin.referral.percent_hint")}
            />
            <TextField
              mono
              label={t("admin.referral.months")}
              value={form.months}
              onChange={(months) => setForm({ ...form, months })}
              placeholder="12"
              accent={invalid.months}
              hint={t("admin.referral.months_hint")}
            />
            <TextField
              mono
              label={t("admin.referral.min_payment")}
              value={form.min}
              onChange={(min) => setForm({ ...form, min })}
              placeholder="0"
              accent={invalid.min}
              hint={t("admin.referral.min_payment_hint")}
            />
          </FieldGrid>
          <p className="text-[12.5px] text-muted-foreground">{t("admin.referral.rules")}</p>
        </div>
      )}
    </SettingsCard>
  );
}
