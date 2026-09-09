"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  TariffForm,
  adminTariffToFormState,
  defaultTariffFormState,
  tariffFormToPayload,
  type TariffFormState,
} from "@/components/admin/tariffs/tariff-form";
import { fetchAdminTariffEdit, updateAdminTariff } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";

export function TariffEditContent({ id }: { id: string }) {
  const t = useT();
  const queryClient = useQueryClient();
  const [form, setForm] = useState(defaultTariffFormState);
  const [baseline, setBaseline] = useState<TariffFormState>(
    defaultTariffFormState
  );

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminTariffEdit(id),
    queryFn: () => fetchAdminTariffEdit(id),
  });

  useEffect(() => {
    if (data?.tariff) {
      const next = adminTariffToFormState(data.tariff);
      setForm(next);
      setBaseline(next);
    }
  }, [data]);

  const dirty = useMemo(
    () => JSON.stringify(form) !== JSON.stringify(baseline),
    [form, baseline]
  );

  const saveMut = useMutation({
    mutationFn: (payload: Record<string, unknown>) =>
      updateAdminTariff(id, payload),
    onSuccess: async () => {
      setBaseline(form);
      await queryClient.invalidateQueries({ queryKey: queryKeys.adminTariffs() });
      await queryClient.invalidateQueries({ queryKey: queryKeys.adminTariff(id) });
      toast.success(t("common.saved"));
    },
    onError: (e: Error) => toast.error(e.message),
  });

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    await saveMut.mutateAsync(tariffFormToPayload(form));
  }

  if (isLoading && !data) {
    return (
      <PageShell variant="admin">
        <Skeleton className="mb-6 h-10 w-64" />
        <Skeleton className="h-96 w-full" />
      </PageShell>
    );
  }

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-5">
        <div className="flex flex-wrap items-end justify-between gap-6">
          <div className="space-y-1.5">
            <div className="flex items-center gap-2.5 text-xs text-muted-foreground">
              <Link href="/admin/tariffs" className="hover:text-foreground">
                {t("admin.tariffs.title")}
              </Link>
              <span>/</span>
              <span className="text-foreground">
                {form.name || t("common.tariff")}
              </span>
            </div>
            <h1 className="text-[26px] leading-none font-bold tracking-tight">
              {t("admin.tariffs.edit_title", { name: form.name })}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t("admin.tariffs.edit_subtitle")}
            </p>
          </div>
          <div className="flex items-center gap-2.5">
            <Button variant="outline" asChild className="h-[38px] text-[13px]">
              <Link href={`/admin/tariffs/${id}`}>
                {t("admin.locations.view")}
              </Link>
            </Button>
            <Button variant="outline" asChild className="h-[38px] text-[13px]">
              <Link href="/admin/tariffs">← {t("admin.tariffs.to_list")}</Link>
            </Button>
          </div>
        </div>

        <TariffForm
          form={form}
          setForm={setForm}
          locations={data?.locations ?? []}
          games={data?.games ?? []}
          saving={saveMut.isPending}
          submitLabel={t("common.save_changes")}
          onSubmit={onSubmit}
          dirty={dirty}
          onCancel={() => setForm(baseline)}
        />
      </div>
    </PageShell>
  );
}
