"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import {
  addNodeIPs,
  deleteNodeIP,
  fetchNodeIPs,
  importNodeIPs,
  type NodeIPAddress,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";

// Подписи храним ключами: карта читается на уровне модуля, и готовый текст
// застыл бы на языке, который стоял в момент загрузки страницы.
const STATUS_LABEL_KEYS: Record<string, string> = {
  free: "admin.ips.status.free",
  reserved: "admin.ips.status.reserved",
  assigned: "admin.ips.status.assigned",
};

export function LocationIPsCard({ locationId }: { locationId: string }) {
  const t = useT();
  const qc = useQueryClient();
  const [addresses, setAddresses] = useState("");
  const [label, setLabel] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["node-ips", locationId],
    queryFn: () => fetchNodeIPs(locationId),
    enabled: !!locationId,
  });

  const invalidate = () => qc.invalidateQueries({ queryKey: ["node-ips", locationId] });

  const addMut = useMutation({
    mutationFn: () => addNodeIPs(locationId, addresses, label),
    onSuccess: (res) => {
      toast.success(
        res.skipped > 0
          ? t("admin.ips.added_some", {
              added: res.added,
              skipped: res.skipped,
            })
          : t("admin.ips.added", { count: res.added })
      );
      setAddresses("");
      invalidate();
    },
    onError: (e: Error) => toast.error(e.message || t("admin.ips.add_failed")),
  });

  const importMut = useMutation({
    mutationFn: () => importNodeIPs(locationId),
    onSuccess: (res) => {
      toast.success(
        res.found === 0
          ? t("admin.ips.import_empty")
          : t("admin.ips.imported", { added: res.added, found: res.found })
      );
      invalidate();
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.ips.import_failed")),
  });

  const deleteMut = useMutation({
    mutationFn: (ipId: string) => deleteNodeIP(locationId, ipId),
    onSuccess: () => {
      toast.success(t("admin.ips.deleted"));
      invalidate();
    },
    onError: (e: Error) => toast.error(e.message || t("common.delete_failed")),
  });

  if (isLoading) return <Skeleton className="h-[220px] w-full rounded-2xl" />;

  const list = data?.addresses ?? [];

  return (
    <section className="flex flex-col gap-4 rounded-2xl border bg-card px-5 py-5 sm:px-6">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-base font-semibold">{t("admin.ips.title")}</h2>
        <Badge variant="secondary">
          {t("admin.ips.free_of", {
            free: data?.free ?? 0,
            total: list.length,
          })}
        </Badge>
      </div>

      <p className="text-sm text-muted-foreground">{t("admin.ips.hint")}</p>

      {list.length === 0 ? (
        <div className="rounded-lg border p-6 text-center text-sm text-muted-foreground">
          {t("admin.ips.empty")}
        </div>
      ) : (
        <div className="divide-y rounded-lg border">
          {list.map((ip: NodeIPAddress) => (
            <div
              key={ip.id}
              className="flex flex-wrap items-center justify-between gap-3 px-4 py-2.5"
            >
              <div className="min-w-0">
                <span className="font-mono text-sm">{ip.address}</span>
                {ip.label ? (
                  <span className="ml-2 text-xs text-muted-foreground">{ip.label}</span>
                ) : null}
              </div>
              <div className="flex items-center gap-2">
                {ip.status === "assigned" ? (
                  <Badge>
                    {ip.server_name || t("admin.ips.status.assigned")}
                  </Badge>
                ) : (
                  <Badge variant="outline">
                    {STATUS_LABEL_KEYS[ip.status]
                      ? t(STATUS_LABEL_KEYS[ip.status])
                      : ip.status}
                  </Badge>
                )}
                {ip.status !== "assigned" ? (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      if (
                        !confirm(
                          t("admin.ips.delete_confirm", { address: ip.address })
                        )
                      )
                        return;
                      deleteMut.mutate(ip.id);
                    }}
                    disabled={deleteMut.isPending}
                  >
                    {t("common.delete")}
                  </Button>
                ) : null}
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="space-y-2 border-t pt-4">
        <Label className="text-xs">{t("admin.ips.new_addresses")}</Label>
        <Textarea
          value={addresses}
          onChange={(e) => setAddresses(e.target.value)}
          placeholder={"203.0.113.10\n203.0.113.11"}
          rows={3}
        />
        <div className="flex flex-wrap items-end gap-2">
          <div className="flex-1 space-y-1">
            <Label className="text-xs">{t("admin.ips.label")}</Label>
            <Input
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              placeholder={t("admin.ips.label_placeholder")}
            />
          </div>
          <Button
            onClick={() => addMut.mutate()}
            disabled={addMut.isPending || !addresses.trim()}
          >
            {t("common.add")}
          </Button>
          <Button
            variant="outline"
            onClick={() => importMut.mutate()}
            disabled={importMut.isPending}
          >
            {t("admin.ips.import")}
          </Button>
        </div>
        <p className="text-xs text-muted-foreground">
          {t("admin.ips.paste_hint")}
        </p>
      </div>
    </section>
  );
}
