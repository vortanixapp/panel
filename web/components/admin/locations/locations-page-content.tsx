"use client";

import Link from "next/link";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Eye,
  MapPin,
  Pause,
  Pencil,
  Play,
  Plus,
  Settings2,
  Trash2,
  Wrench,
} from "lucide-react";
import { toast } from "sonner";
import { useT } from "@/hooks/use-translations";
import { PageShell } from "@/components/layout/page-shell";
import { MaintenanceDialog } from "@/components/admin/locations/maintenance-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import {
  deleteAdminLocation,
  fetchAdminLocations,
  toggleAdminLocation,
  type AdminLocationListItem,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

const COLUMNS =
  "grid-cols-[minmax(220px,1.3fr)_minmax(220px,1.3fr)_150px_140px_210px]";

export function LocationsPageContent() {
  const t = useT();
  const queryClient = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminLocations,
    queryFn: fetchAdminLocations,
  });
  const locations = data?.locations ?? [];
  const [maintenanceFor, setMaintenanceFor] =
    useState<AdminLocationListItem | null>(null);

  const toggleMut = useMutation({
    mutationFn: toggleAdminLocation,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.adminLocations });
      toast.success(t("admin.locations.status_updated"));
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const deleteMut = useMutation({
    mutationFn: deleteAdminLocation,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.adminLocations });
      toast.success(t("admin.locations.deleted"));
    },
    onError: (e: Error) => toast.error(e.message),
  });

  async function onDelete(id: string, name: string) {
    if (!confirm(t("admin.locations.delete_confirm", { name }))) return;
    await deleteMut.mutateAsync(id);
  }

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-5">
        <div className="flex flex-wrap items-end justify-between gap-6">
          <div className="space-y-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-tight">
              {t("common.locations")}
            </h1>
          </div>
          <Button asChild className="h-[38px] text-[13px]">
            <Link href="/admin/locations/create">
              <Plus className="size-4" />
              {t("admin.locations.add")}
            </Link>
          </Button>
        </div>

        <div className="overflow-hidden rounded-2xl border bg-card">
          {isLoading ? (
            <div className="space-y-3 p-6">
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-full" />
            </div>
          ) : locations.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-center">
              <MapPin className="mb-3 size-10 text-muted-foreground" />
              <p className="text-sm text-muted-foreground">
                {t("admin.locations.empty")}
              </p>
              <Button className="mt-4 h-[38px] text-[13px]" asChild>
                <Link href="/admin/locations/create">
                  {t("admin.locations.create_first")}
                </Link>
              </Button>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <div className="flex min-w-[980px] flex-col">
                <div
                  className={cn(
                    "grid gap-3 border-b bg-muted/40 px-5 py-3 font-mono text-[11px] tracking-wider text-muted-foreground uppercase",
                    COLUMNS
                  )}
                >
                  <div>{t("common.location")}</div>
                  <div>IP / SSH</div>
                  <div>Agent</div>
                  <div>{t("common.servers")}</div>
                  <div className="text-right">{t("common.actions")}</div>
                </div>

                {locations.map((loc) => {
                  const active = loc.is_active !== false;
                  return (
                    <div
                      key={loc.id}
                      className={cn(
                        "grid items-center gap-3 border-b px-5 py-4 transition-colors hover:bg-muted/30",
                        COLUMNS
                      )}
                    >
                      <div className="min-w-0 space-y-1.5">
                        <div className="truncate text-[15px] font-medium">
                          {loc.name}
                        </div>
                        <div className="flex flex-wrap items-center gap-2.5">
                          <span className="font-mono text-xs text-muted-foreground">
                            {loc.code || loc.id.slice(0, 8)}
                          </span>
                          {loc.country && (
                            <span className="text-xs text-muted-foreground">
                              {loc.country}
                            </span>
                          )}
                          {loc.maintenance_mode && (
                            <Badge variant="destructive">
                              {t("admin.locations.maintenance")}
                            </Badge>
                          )}
                          <span
                            className={cn(
                              "rounded-md border px-2 py-0.5 text-[11px]",
                              active
                                ? "border-emerald-500/40 text-emerald-500"
                                : "text-muted-foreground"
                            )}
                          >
                            {active
                              ? t("admin.locations.active")
                              : t("admin.locations.inactive")}
                          </span>
                        </div>
                      </div>

                      <div className="min-w-0 space-y-1.5">
                        <div className="truncate font-mono text-[13px]">
                          {loc.ip_address || loc.fqdn || loc.ssh_host || "—"}
                        </div>
                        <div
                          className={cn(
                            "truncate font-mono text-xs",
                            loc.ssh_configured
                              ? "text-emerald-500"
                              : "text-amber-500"
                          )}
                        >
                          {loc.ssh_configured
                            ? `SSH: ${loc.ssh_user}@${loc.ssh_host}`
                            : t("admin.locations.ssh_not_configured")}
                        </div>
                      </div>

                      <div>
                        <span
                          className={cn(
                            "inline-flex items-center gap-2 text-xs",
                            loc.is_online ? "text-emerald-500" : "text-rose-500"
                          )}
                        >
                          <span
                            className={cn(
                              "size-1.5 rounded-full",
                              loc.is_online ? "bg-emerald-500" : "bg-rose-500"
                            )}
                          />
                          {loc.is_online
                            ? t("admin.infra.online")
                            : t("admin.infra.offline")}
                        </span>
                      </div>

                      <div className="space-y-1">
                        <div className="font-mono text-[13px]">
                          {loc.servers_count}
                        </div>
                        <div className="text-xs text-muted-foreground">
                          {t("admin.locations.tariffs_count", {
                            count: loc.tariffs_count,
                          })}
                        </div>
                      </div>

                      <div className="flex items-center justify-end gap-1.5">
                        <Button
                          variant="outline"
                          size="icon"
                          className="size-7 rounded-md"
                          asChild
                        >
                          <Link
                            href={`/admin/locations/${loc.id}`}
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
                            href={`/admin/locations/${loc.id}/setup`}
                            title={t("admin.locations.ssh_setup")}
                          >
                            <Settings2 className="size-3.5" />
                          </Link>
                        </Button>
                        <Button
                          variant="outline"
                          size="icon"
                          className="size-7 rounded-md"
                          asChild
                        >
                          <Link
                            href={`/admin/locations/${loc.id}/edit`}
                            title={t("common.edit")}
                          >
                            <Pencil className="size-3.5" />
                          </Link>
                        </Button>
                        <Button
                          variant="outline"
                          size="icon"
                          className={cn(
                            "size-7 rounded-md",
                            loc.maintenance_mode && "border-amber-500 text-amber-600"
                          )}
                          title={
                            loc.maintenance_mode
                              ? t("admin.locations.maintenance_active")
                              : t("admin.locations.maintenance")
                          }
                          onClick={() => setMaintenanceFor(loc)}
                        >
                          <Wrench className="size-3.5" />
                        </Button>
                        <Button
                          variant="outline"
                          size="icon"
                          className="size-7 rounded-md"
                          title={active ? t("common.disable") : t("common.enable")}
                          disabled={toggleMut.isPending}
                          onClick={() => toggleMut.mutate(loc.id)}
                        >
                          {active ? (
                            <Pause className="size-3.5" />
                          ) : (
                            <Play className="size-3.5" />
                          )}
                        </Button>
                        <Button
                          variant="outline"
                          size="icon"
                          className="size-7 rounded-md text-destructive hover:text-destructive"
                          title={t("common.delete")}
                          disabled={deleteMut.isPending}
                          onClick={() => onDelete(loc.id, loc.name)}
                        >
                          <Trash2 className="size-3.5" />
                        </Button>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}

        {maintenanceFor ? (
          <MaintenanceDialog
            location={maintenanceFor}
            open
            onOpenChange={(open) => {
              if (!open) setMaintenanceFor(null);
            }}
          />
        ) : null}
        </div>
      </div>
    </PageShell>
  );
}
