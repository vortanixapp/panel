"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  TariffForm,
  defaultTariffFormState,
  tariffFormToPayload,
} from "@/components/admin/tariffs/tariff-form";
import { createAdminTariff, fetchAdminTariffCreateForm } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";

export function TariffCreateContent() {
  const t = useT();
  const router = useRouter();
  const queryClient = useQueryClient();
  const [form, setForm] = useState(defaultTariffFormState);

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminTariffCreate,
    queryFn: fetchAdminTariffCreateForm,
  });

  const dirty = useMemo(
    () => JSON.stringify(form) !== JSON.stringify(defaultTariffFormState),
    [form]
  );

  const createMut = useMutation({
    mutationFn: createAdminTariff,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.adminTariffs() });
      toast.success(t("admin.tariffs.created"));
      router.push("/admin/tariffs");
    },
    onError: (e: Error) => toast.error(e.message),
  });

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    await createMut.mutateAsync(tariffFormToPayload(form));
  }

  if (isLoading) {
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
                {t("admin.tariffs.new_crumb")}
              </span>
            </div>
            <h1 className="text-[26px] leading-none font-bold tracking-tight">
              {t("admin.tariffs.create_title")}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t("admin.tariffs.create_subtitle")}
            </p>
          </div>
          <Button variant="outline" asChild className="h-[38px] text-[13px]">
            <Link href="/admin/tariffs">← {t("admin.tariffs.to_list")}</Link>
          </Button>
        </div>

        <TariffForm
          form={form}
          setForm={setForm}
          locations={data?.locations ?? []}
          games={data?.games ?? []}
          saving={createMut.isPending}
          submitLabel={t("admin.tariffs.create")}
          onSubmit={onSubmit}
          dirty={dirty}
          onCancel={() => router.push("/admin/tariffs")}
        />
      </div>
    </PageShell>
  );
}
