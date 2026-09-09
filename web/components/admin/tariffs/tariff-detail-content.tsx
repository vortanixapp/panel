"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Copy, Pencil, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  deleteAdminTariff,
  duplicateAdminTariff,
  fetchAdminTariff,
  type AdminTariff,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

function mysqlLabel(engine?: string | null) {
  switch (engine) {
    case "mysql57":
      return "MySQL 5.7";
    case "mariadb":
      return "MariaDB";
    default:
      return "MySQL 8.0";
  }
}

function formatDate(value?: string) {
  if (!value) return "—";
  return new Date(value).toLocaleString(localeTag(), {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function TariffDetailContent({ id }: { id: string }) {
  const t = useT();
  const router = useRouter();
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminTariff(id),
    queryFn: () => fetchAdminTariff(id),
  });

  const tariff = data?.tariff;

  const actionMut = useMutation({
    mutationFn: async (action: "duplicate" | "delete") => {
      if (action === "duplicate") return duplicateAdminTariff(id);
      return deleteAdminTariff(id);
    },
    onSuccess: async (res, action) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.adminTariffs() });
      if (action === "duplicate" && res && "id" in res) {
        toast.success(t("admin.tariffs.copy_created"));
        router.push(`/admin/tariffs/${res.id}/edit`);
        return;
      }
      toast.success(t("admin.tariffs.deleted"));
      router.push("/admin/tariffs");
    },
    onError: (e: Error) => toast.error(e.message),
  });

  if (isLoading && !tariff) {
    return (
      <PageShell variant="admin">
        <Skeleton className="mb-6 h-10 w-64" />
        <Skeleton className="h-96 w-full" />
      </PageShell>
    );
  }

  if (!tariff) {
    return (
      <PageShell variant="admin">
        <p className="py-20 text-center text-muted-foreground">
          {t("admin.tariffs.not_found")}
        </p>
      </PageShell>
    );
  }

  const isSlots = tariff.billing_type === "slots";

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">{tariff.name}</h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.tariffs.detail_subtitle")}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button asChild>
            <Link href={`/admin/tariffs/${id}/edit`}>
              <Pencil className="mr-2 h-4 w-4" />
              {t("common.edit")}
            </Link>
          </Button>
          <Button
            variant="outline"
            disabled={actionMut.isPending}
            onClick={() => {
              if (confirm(t("admin.tariffs.duplicate_confirm"))) {
                actionMut.mutate("duplicate");
              }
            }}
          >
            <Copy className="mr-2 h-4 w-4" />
            {t("common.copy")}
          </Button>
          <Button
            variant="destructive"
            disabled={actionMut.isPending}
            onClick={() => {
              if (confirm(t("admin.tariffs.delete_this_confirm"))) {
                actionMut.mutate("delete");
              }
            }}
          >
            <Trash2 className="mr-2 h-4 w-4" />
            {t("common.delete")}
          </Button>
          <Button variant="ghost" asChild>
            <Link href="/admin/tariffs">← {t("admin.tariffs.to_list")}</Link>
          </Button>
        </div>
      </div>

      <TariffDetailView tariff={tariff} isSlots={isSlots} />
    </PageShell>
  );
}

function TariffDetailView({
  tariff,
  isSlots,
}: {
  tariff: AdminTariff;
  isSlots: boolean;
}) {
  const t = useT();
  return (
    <div className="rounded-lg border bg-card p-6">
      <div className="grid gap-8 lg:grid-cols-3">
        <section className="space-y-4">
          <h2 className="text-sm font-semibold uppercase tracking-wide">
            {t("admin.tariffs.section_base")}
          </h2>
          <DetailRow
            label={t("common.location")}
            value={tariff.location?.name || tariff.location_id || "—"}
          />
          <DetailRow
            label={t("common.game")}
            value={tariff.game?.name || tariff.game_id || "—"}
          />
          <DetailRow
            label={t("admin.tariff_form.billing_type")}
            value={
              <Badge variant="secondary">
                {isSlots
                  ? t("admin.tariffs.type.slots")
                  : t("admin.tariffs.type.resources")}
              </Badge>
            }
          />
          <DetailRow label="MySQL" value={mysqlLabel(tariff.mysql_engine)} />
          <DetailRow
            label={t("common.status")}
            value={
              tariff.is_available ? (
                <Badge className="bg-emerald-500/10 text-emerald-600 hover:bg-emerald-500/10">
                  {t("admin.tariffs.available")}
                </Badge>
              ) : (
                <Badge variant="outline">
                  {t("admin.tariffs.unavailable")}
                </Badge>
              )
            }
          />
          <DetailRow
            label={t("admin.tariffs.col_position")}
            value={String(tariff.position)}
          />
          <div>
            <p className="mb-2 text-xs font-medium text-muted-foreground">
              {t("admin.tariffs.rental_periods")}
            </p>
            <div className="flex flex-wrap gap-1">
              {(tariff.rental_periods ?? []).map((p) => (
                <Badge key={p} variant="secondary">
                  {t("admin.tariffs.days_badge", { days: p })}
                </Badge>
              ))}
            </div>
          </div>
        </section>

        <section className="space-y-4">
          <h2 className="text-sm font-semibold uppercase tracking-wide">
            {t("admin.tariffs.section_limits")}
          </h2>
          <div className="grid grid-cols-2 gap-3">
            <StatCard
              label="CPU"
              value={tariff.cpu_cores === 0 ? "∞" : String(tariff.cpu_cores)}
              suffix={t("admin.tariffs.cores")}
            />
            <StatCard
              label="RAM"
              value={String(tariff.ram_gb)}
              suffix={t("admin.infra.unit_gb")}
            />
            <StatCard
              label={t("admin.dashboard.col_disk")}
              value={String(tariff.disk_gb)}
              suffix={t("admin.infra.unit_gb")}
            />
            <StatCard
              label="Anti-DDoS"
              value={tariff.allow_antiddos ? t("common.yes") : t("common.no")}
              suffix={
                tariff.allow_antiddos
                  ? t("admin.tariffs.per_month_price", {
                      price: tariff.antiddos_price ?? 0,
                    })
                  : undefined
              }
            />
          </div>
          {isSlots ? (
            <div className="rounded-md border bg-muted/40 p-3">
              <p className="text-xs text-muted-foreground">
                {t("admin.tariffs.type.slots")}
              </p>
              <p className="font-semibold">
                {tariff.min_slots} — {tariff.max_slots}
              </p>
            </div>
          ) : (
            <div className="grid grid-cols-3 gap-2 text-xs">
              <RangeCard label="CPU" min={tariff.cpu_min} max={tariff.cpu_max} />
              <RangeCard label="RAM" min={tariff.ram_min} max={tariff.ram_max} />
              <RangeCard
                label="Disk"
                min={tariff.disk_min}
                max={tariff.disk_max}
              />
            </div>
          )}
        </section>

        <section className="space-y-4">
          <h2 className="text-sm font-semibold uppercase tracking-wide">
            {t("admin.billing.title")}
          </h2>
          {isSlots ? (
            <PriceCard
              label={t("admin.tariffs.price_per_slot")}
              value={Number(tariff.price_per_slot || 0).toFixed(2)}
            />
          ) : (
            <>
              <PriceCard
                label={t("admin.tariffs.base_price")}
                value={Number(tariff.base_price_monthly || 0).toFixed(2)}
              />
              <div className="grid grid-cols-3 gap-2 text-center text-xs">
                <MiniPrice
                  label="CPU"
                  value={tariff.price_per_cpu_core}
                  suffix={t("admin.tariffs.per_core")}
                />
                <MiniPrice
                  label="RAM"
                  value={tariff.price_per_ram_gb}
                  suffix={t("admin.tariffs.per_gb")}
                />
                <MiniPrice
                  label="Disk"
                  value={tariff.price_per_disk_gb}
                  suffix={t("admin.tariffs.per_gb")}
                />
              </div>
            </>
          )}
          <div className="border-t pt-4 text-xs text-muted-foreground">
            <p>
              {t("common.created_at")}: {formatDate(tariff.created_at)}
            </p>
            <p>
              {t("common.updated_at")}: {formatDate(tariff.updated_at)}
            </p>
          </div>
        </section>
      </div>
    </div>
  );
}

function DetailRow({
  label,
  value,
}: {
  label: string;
  value: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-3 border-b py-2 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium">{value}</span>
    </div>
  );
}

function StatCard({
  label,
  value,
  suffix,
}: {
  label: string;
  value: string;
  suffix?: string;
}) {
  return (
    <div className="rounded-md border bg-muted/40 p-3">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-xl font-bold">{value}</p>
      {suffix && <p className="text-xs text-muted-foreground">{suffix}</p>}
    </div>
  );
}

function RangeCard({
  label,
  min,
  max,
}: {
  label: string;
  min?: number | null;
  max?: number | null;
}) {
  return (
    <div className="rounded-md border bg-muted/30 p-2">
      <p className="font-medium text-muted-foreground">{label}</p>
      <p className="font-semibold">
        {min ?? "—"}-{max ?? "—"}
      </p>
    </div>
  );
}

function PriceCard({ label, value }: { label: string; value: string }) {
  const t = useT();
  return (
    <div className="rounded-md border bg-muted/40 p-4 text-center">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-2xl font-bold">{value}</p>
      <p className="text-xs text-muted-foreground">
        {t("admin.analytics.col_per_month")}
      </p>
    </div>
  );
}

function MiniPrice({
  label,
  value,
  suffix,
}: {
  label: string;
  value?: number;
  suffix: string;
}) {
  return (
    <div className="rounded-md border bg-muted/30 p-2">
      <p className="font-medium text-muted-foreground">{label}</p>
      <p className="font-bold">{value ?? 0}</p>
      <p className="text-muted-foreground">{suffix}</p>
    </div>
  );
}
