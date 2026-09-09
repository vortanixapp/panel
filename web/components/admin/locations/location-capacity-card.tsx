"use client";

import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { useT } from "@/hooks/use-translations";
import type { TranslateFn } from "@/lib/i18n";
import { fetchNodeCapacity, updateNodeCapacity } from "@/lib/api";

function fmtMB(t: TranslateFn, mb: number): string {
  if (!mb) return "0";
  if (Math.abs(mb) >= 1024)
    return `${(mb / 1024).toFixed(1)} ${t("admin.infra.unit_gb")}`;
  return `${Math.round(mb)} ${t("admin.infra.unit_mb")}`;
}

export function LocationCapacityCard({ locationId }: { locationId: string }) {
  const t = useT();
  const qc = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: ["node-capacity", locationId],
    queryFn: () => fetchNodeCapacity(locationId),
    enabled: !!locationId,
  });

  const [maxServers, setMaxServers] = useState("");
  const [overcommit, setOvercommit] = useState("1");
  const [reserved, setReserved] = useState("1024");

  useEffect(() => {
    if (!data) return;
    setMaxServers(data.max_servers === null ? "" : String(data.max_servers));
    setOvercommit(String(data.ram_overcommit ?? 1));
    setReserved(String(data.reserved_ram_mb ?? 1024));
  }, [data]);

  const saveMut = useMutation({
    mutationFn: () =>
      updateNodeCapacity(locationId, {
        max_servers: maxServers.trim() === "" ? null : Number(maxServers),
        ram_overcommit: Number(overcommit) || 1,
        reserved_ram_mb: Number(reserved) || 0,
      }),
    onSuccess: () => {
      toast.success(t("admin.capacity.saved"));
      void qc.invalidateQueries({ queryKey: ["node-capacity", locationId] });
    },
    onError: (e: Error) => toast.error(e.message || t("common.save_failed")),
  });

  if (isLoading || !data) {
    return <Skeleton className="h-[220px] w-full rounded-2xl" />;
  }

  const overloaded = data.free_ram_mb < 0;

  return (
    <section className="flex flex-col gap-4 rounded-2xl border bg-card px-5 py-5 sm:px-6">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-base font-semibold">{t("admin.capacity.title")}</h2>
        {overloaded ? (
          <Badge variant="destructive">
            {t("admin.capacity.oversubscribed")}
          </Badge>
        ) : (
          <Badge variant="secondary">
            {t("admin.capacity.free", { value: fmtMB(t, data.free_ram_mb) })}
          </Badge>
        )}
      </div>

      {!data.metrics_known ? (
        <p className="text-sm text-muted-foreground">
          {t("admin.capacity.no_metrics")}
        </p>
      ) : (
        <div className="grid gap-2 text-[13px] sm:grid-cols-2">
          <Row label={t("admin.capacity.servers_on_node")}>
            {data.servers}
            {data.max_servers !== null
              ? ` ${t("common.of")} ${data.max_servers}`
              : ""}
          </Row>
          <Row label={t("admin.capacity.ram_on_node")}>
            {fmtMB(t, data.node_ram_mb)}
          </Row>
          <Row label={t("admin.capacity.allocated_ram")}>
            {fmtMB(t, data.allocated_ram_mb)}
          </Row>
          <Row label={t("admin.capacity.free_for_new")}>
            {fmtMB(t, data.free_ram_mb)}
          </Row>
          <Row label={t("admin.capacity.free_disk")}>
            {fmtMB(t, data.free_disk_mb)}
          </Row>
          <Row label={t("admin.capacity.allocated_cpu")}>
            {data.allocated_cpu.toFixed(1)}
          </Row>
        </div>
      )}

      <DiskQuotaRow quota={data.disk_quota} />

      {data.games.length > 0 && (
        <div>
          <div className="mb-2 text-[13px] font-medium">
            {t("admin.capacity.games_title")}
          </div>
          <div className="flex flex-wrap gap-2">
            {data.games.slice(0, 12).map((g) => (
              <span
                key={g.slug}
                className="rounded-md border px-2 py-1 text-xs"
                title={t("admin.capacity.game_hint", {
                  ram: g.ram_per_server_mb,
                  by: g.limited_by,
                })}
              >
                {g.name}: <b>{g.fits}</b>
              </span>
            ))}
          </div>
        </div>
      )}

      <div className="flex flex-wrap items-end gap-3 border-t pt-4">
        <div className="space-y-1">
          <Label className="text-xs">{t("admin.capacity.max_servers")}</Label>
          <Input
            value={maxServers}
            onChange={(e) => setMaxServers(e.target.value)}
            placeholder={t("admin.capacity.no_limit")}
            className="w-32"
          />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">{t("admin.capacity.overcommit")}</Label>
          <Input
            type="number"
            step="0.1"
            min="0.1"
            max="4"
            value={overcommit}
            onChange={(e) => setOvercommit(e.target.value)}
            className="w-28"
          />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">{t("admin.capacity.reserved")}</Label>
          <Input
            type="number"
            min="0"
            value={reserved}
            onChange={(e) => setReserved(e.target.value)}
            className="w-32"
          />
        </div>
        <Button onClick={() => saveMut.mutate()} disabled={saveMut.isPending}>
          {t("common.save")}
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">
        {t("admin.capacity.hint")}
      </p>
    </section>
  );
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-baseline justify-between gap-4 border-b py-2 last:border-0">
      <span className="text-muted-foreground">{label}</span>
      <span className="text-right font-mono">{children}</span>
    </div>
  );
}

function DiskQuotaRow({ quota }: { quota: boolean | null }) {
  const t = useT();
  if (quota === null) {
    return (
      <div className="flex flex-wrap items-center gap-2 rounded-xl border border-dashed px-3 py-2.5 text-[13px]">
        <span className="font-medium">{t("admin.quota.title")}</span>
        <Badge variant="secondary">{t("admin.quota.unknown")}</Badge>
        <span className="text-muted-foreground">
          {t("admin.quota.unknown_hint")}
        </span>
      </div>
    );
  }

  return (
    <div className="flex flex-wrap items-center gap-2 rounded-xl border px-3 py-2.5 text-[13px]">
      <span className="font-medium">{t("admin.quota.title")}</span>
      {quota ? (
        <>
          <Badge variant="secondary">{t("admin.quota.on")}</Badge>
          <span className="text-muted-foreground">
            {t("admin.quota.on_hint")}
          </span>
        </>
      ) : (
        <>
          <Badge variant="destructive">{t("admin.quota.off")}</Badge>
          <span className="text-muted-foreground">
            {t("admin.quota.off_hint")}
          </span>
        </>
      )}
    </div>
  );
}
