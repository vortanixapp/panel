"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Copy, Eye, Pencil, Plus, Tag, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import {
  deleteAdminTariff,
  duplicateAdminTariff,
  fetchAdminTariffCreateForm,
  fetchAdminTariffs,
  type AdminTariff,
  type AdminTariffsFilters,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";

const COLUMNS =
  "grid-cols-[minmax(220px,1.6fr)_110px_minmax(160px,1.2fr)_110px_minmax(160px,1.1fr)_90px_120px_132px]";

function getParams(tariff: AdminTariff, t: TranslateFn) {
  if ((tariff.billing_type || "resources") === "slots") {
    return t("admin.tariffs.slots_range", {
      min: tariff.min_slots,
      max: tariff.max_slots,
    });
  }
  return `${tariff.cpu_cores} CPU / ${tariff.ram_gb} GB / ${tariff.disk_gb} GB`;
}

function pageWindow(current: number, last: number): number[] {
  const pages = new Set<number>([1, last]);
  for (let p = current - 1; p <= current + 1; p += 1) {
    if (p >= 1 && p <= last) pages.add(p);
  }
  return [...pages].sort((a, b) => a - b);
}

export function TariffsPageContent() {
  const t = useT();
  const [page, setPage] = useState(1);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);
  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");
  const [locationId, setLocationId] = useState("");
  const [gameId, setGameId] = useState("");
  const [billingType, setBillingType] = useState("");
  const [available, setAvailable] = useState("");
  const queryClient = useQueryClient();

  useEffect(() => {
    const timer = setTimeout(() => {
      setSearch(searchInput);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchInput]);

  const filters: AdminTariffsFilters = useMemo(
    () => ({
      search,
      location_id: locationId,
      game_id: gameId,
      billing_type: billingType,
      available,
    }),
    [search, locationId, gameId, billingType, available]
  );

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminTariffs(page, filters as Record<string, string>),
    queryFn: () => fetchAdminTariffs(page, filters),
    placeholderData: (prev) => prev,
  });

  const { data: meta } = useQuery({
    queryKey: queryKeys.adminTariffCreate,
    queryFn: fetchAdminTariffCreateForm,
    staleTime: 5 * 60 * 1000,
  });

  const tariffs = data?.tariffs?.data ?? [];
  const lastPage = data?.tariffs?.last_page ?? 1;
  const perPage = data?.tariffs?.per_page ?? 10;
  const total = data?.tariffs?.total ?? tariffs.length;
  const from = total === 0 ? 0 : (page - 1) * perPage + 1;
  const to = total === 0 ? 0 : from + tariffs.length - 1;

  const applyFilter = (apply: () => void) => {
    setPage(1);
    apply();
  };

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: queryKeys.adminTariffs() });

  const deleteMut = useMutation({
    mutationFn: deleteAdminTariff,
    onSuccess: async () => {
      await invalidate();
      toast.success(t("admin.tariffs.deleted"));
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const duplicateMut = useMutation({
    mutationFn: duplicateAdminTariff,
    onSuccess: async () => {
      await invalidate();
      toast.success(t("admin.tariffs.duplicated"));
    },
    onError: (e: Error) => toast.error(e.message),
  });

  async function onDelete(id: string, name: string) {
    if (!confirm(t("admin.tariffs.delete_confirm", { name }))) return;
    setActionLoading(id);
    try {
      await deleteMut.mutateAsync(id);
    } finally {
      setActionLoading(null);
    }
  }

  async function onDuplicate(id: string) {
    setActionLoading(id);
    try {
      await duplicateMut.mutateAsync(id);
    } finally {
      setActionLoading(null);
    }
  }

  async function onExport() {
    setExporting(true);
    try {
      const all = await fetchAdminTariffs(1, { ...filters, per_page: 500 });
      const rows = all.tariffs?.data ?? [];
      const header = [
        t("common.name"),
        t("common.location"),
        t("common.game"),
        t("common.type"),
        t("admin.tariffs.col_params"),
        t("admin.tariffs.col_position"),
        t("common.status"),
      ];
      const csv = [header, ...rows.map((row) => toCsvRow(row, t))]
        .map((row) =>
          row.map((cell) => `"${String(cell).replace(/"/g, '""')}"`).join(";")
        )
        .join("\r\n");
      const blob = new Blob([`﻿${csv}`], {
        type: "text/csv;charset=utf-8",
      });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = "tariffs.csv";
      link.click();
      URL.revokeObjectURL(url);
      toast.success(t("admin.tariffs.exported", { count: rows.length }));
    } catch (e) {
      toast.error(
        e instanceof Error ? e.message : t("admin.tariffs.export_failed")
      );
    } finally {
      setExporting(false);
    }
  }

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-5">
        <div className="flex flex-wrap items-end justify-between gap-6">
          <div className="space-y-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-tight">
              {t("admin.tariffs.title")}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t("admin.tariffs.subtitle")}
            </p>
          </div>
          <div className="flex items-center gap-2.5">
            <Button
              type="button"
              variant="outline"
              className="h-[38px] text-[13px]"
              onClick={onExport}
              disabled={exporting}
            >
              {exporting
                ? t("admin.tariffs.exporting")
                : t("admin.tariffs.export_csv")}
            </Button>
            <Button asChild className="h-[38px] text-[13px]">
              <Link href="/admin/tariffs/create">
                <Plus className="size-4" />
                {t("admin.tariffs.create")}
              </Link>
            </Button>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2.5">
          <Input
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            placeholder={t("admin.tariffs.search_placeholder")}
            className="h-9 max-w-[340px] flex-1 basis-[280px] rounded-lg text-[13px] md:text-[13px]"
          />
          <FilterSelect
            value={locationId}
            onChange={(v) => applyFilter(() => setLocationId(v))}
            placeholder={t("admin.load_chart.all_locations")}
            options={(meta?.locations ?? []).map((l) => ({
              value: l.id,
              label: l.name,
            }))}
          />
          <FilterSelect
            value={gameId}
            onChange={(v) => applyFilter(() => setGameId(v))}
            placeholder={t("admin.tariffs.all_games")}
            options={(meta?.games ?? []).map((g) => ({
              value: g.id,
              label: g.name,
            }))}
          />
          <FilterSelect
            value={billingType}
            onChange={(v) => applyFilter(() => setBillingType(v))}
            placeholder={t("admin.tariffs.any_type")}
            options={[
              { value: "resources", label: t("admin.tariffs.type.resources") },
              { value: "slots", label: t("admin.tariffs.type.slots") },
            ]}
          />
          <FilterSelect
            value={available}
            onChange={(v) => applyFilter(() => setAvailable(v))}
            placeholder={t("admin.tariffs.any_status")}
            options={[
              { value: "1", label: t("admin.tariffs.available") },
              { value: "0", label: t("admin.tariffs.unavailable") },
            ]}
          />
          <span className="ms-auto font-mono text-xs text-muted-foreground">
            {t("admin.tariffs.counter", { total, page, last: lastPage })}
          </span>
        </div>

        <div className="overflow-hidden rounded-2xl border bg-card">
          {isLoading ? (
            <div className="space-y-3 p-6">
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-full" />
            </div>
          ) : tariffs.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-center">
              <Tag className="mb-3 size-10 text-muted-foreground" />
              <p className="text-sm text-muted-foreground">
                {t("admin.tariffs.empty")}
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <div className="flex min-w-[1180px] flex-col">
                <div
                  className={cn(
                    "grid gap-3 border-b bg-muted/40 px-5 py-3.5 text-xs text-muted-foreground",
                    COLUMNS
                  )}
                >
                  <div>{t("common.name")}</div>
                  <div>{t("common.location")}</div>
                  <div>{t("common.game")}</div>
                  <div>{t("common.type")}</div>
                  <div>{t("admin.tariffs.col_params")}</div>
                  <div>{t("admin.tariffs.col_position")}</div>
                  <div>{t("common.status")}</div>
                  <div className="text-right">{t("common.actions")}</div>
                </div>

                {tariffs.map((tariff) => (
                  <div
                    key={tariff.id}
                    className={cn(
                      "grid items-center gap-3 border-b px-5 py-3.5 transition-colors hover:bg-muted/30",
                      COLUMNS
                    )}
                  >
                    <div className="truncate text-sm font-medium">
                      {tariff.name}
                    </div>
                    <div className="text-[13px] text-muted-foreground">
                      {tariff.location?.name || "—"}
                    </div>
                    <div className="truncate text-[13px]">
                      {tariff.game?.name || "—"}
                    </div>
                    <div>
                      <span className="inline-block rounded-md border px-2 py-0.5 text-[11px]">
                        {(tariff.billing_type || "resources") === "slots"
                          ? t("admin.tariffs.type.slots")
                          : t("admin.tariffs.type.resources")}
                      </span>
                    </div>
                    <div className="font-mono text-xs text-muted-foreground">
                      {getParams(tariff, t)}
                    </div>
                    <div className="font-mono text-[13px] text-muted-foreground">
                      {tariff.position}
                    </div>
                    <div>
                      <span
                        className={cn(
                          "inline-flex items-center gap-2 text-xs",
                          tariff.is_available
                            ? "text-emerald-500"
                            : "text-muted-foreground"
                        )}
                      >
                        <span
                          className={cn(
                            "size-1.5 rounded-full",
                            tariff.is_available
                              ? "bg-emerald-500"
                              : "bg-muted-foreground"
                          )}
                        />
                        {tariff.is_available
                          ? t("admin.tariffs.available")
                          : t("admin.tariffs.unavailable")}
                      </span>
                    </div>
                    <div className="flex items-center justify-end gap-1.5">
                      <Button
                        variant="outline"
                        size="icon"
                        className="size-7 rounded-md"
                        asChild
                      >
                        <Link
                          href={`/admin/tariffs/${tariff.id}`}
                          title={t("admin.locations.view")}
                        >
                          <Eye className="size-3.5" />
                        </Link>
                      </Button>
                      <Button
                        variant="outline"
                        size="icon"
                        className="size-7 rounded-md"
                        asChild
                      >
                        <Link
                          href={`/admin/tariffs/${tariff.id}/edit`}
                          title={t("common.edit")}
                        >
                          <Pencil className="size-3.5" />
                        </Link>
                      </Button>
                      <Button
                        variant="outline"
                        size="icon"
                        className="size-7 rounded-md"
                        title={t("common.copy")}
                        disabled={actionLoading === tariff.id}
                        onClick={() => onDuplicate(tariff.id)}
                      >
                        <Copy className="size-3.5" />
                      </Button>
                      <Button
                        variant="outline"
                        size="icon"
                        className="size-7 rounded-md text-destructive hover:text-destructive"
                        title={t("common.delete")}
                        disabled={actionLoading === tariff.id}
                        onClick={() => onDelete(tariff.id, tariff.name)}
                      >
                        <Trash2 className="size-3.5" />
                      </Button>
                    </div>
                  </div>
                ))}

                <div className="flex items-center justify-between gap-4 bg-muted/40 px-5 py-3.5">
                  <span className="text-xs text-muted-foreground">
                    {t("admin.tariffs.range", { from, to, total })}
                  </span>
                  <div className="flex items-center gap-1.5">
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-[30px] rounded-md text-xs"
                      disabled={page <= 1}
                      onClick={() => setPage((p) => Math.max(1, p - 1))}
                    >
                      {t("common.back")}
                    </Button>
                    {pageWindow(page, lastPage).map((p, idx, arr) => (
                      <span key={p} className="flex items-center gap-1.5">
                        {idx > 0 && arr[idx - 1] !== p - 1 && (
                          <span className="px-0.5 font-mono text-xs text-muted-foreground">
                            …
                          </span>
                        )}
                        <Button
                          variant={p === page ? "default" : "outline"}
                          size="sm"
                          className="h-[30px] min-w-[30px] rounded-md px-2 font-mono text-xs"
                          onClick={() => setPage(p)}
                        >
                          {p}
                        </Button>
                      </span>
                    ))}
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-[30px] rounded-md text-xs"
                      disabled={page >= lastPage}
                      onClick={() => setPage((p) => Math.min(lastPage, p + 1))}
                    >
                      {t("common.next")}
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </PageShell>
  );
}

function toCsvRow(tariff: AdminTariff, t: TranslateFn) {
  return [
    tariff.name,
    tariff.location?.name || "—",
    tariff.game?.name || "—",
    (tariff.billing_type || "resources") === "slots"
      ? t("admin.tariffs.type.slots")
      : t("admin.tariffs.type.resources"),
    getParams(tariff, t),
    tariff.position,
    tariff.is_available
      ? t("admin.tariffs.available")
      : t("admin.tariffs.unavailable"),
  ];
}

function FilterSelect({
  value,
  onChange,
  placeholder,
  options,
}: {
  value: string;
  onChange: (next: string) => void;
  placeholder: string;
  options: { value: string; label: string }[];
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="h-9 rounded-lg border border-input bg-transparent px-2.5 text-[13px] outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:bg-input/30"
    >
      <option value="">{placeholder}</option>
      {options.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
    </select>
  );
}
