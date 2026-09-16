"use client";

import Link from "next/link";
import { type ColumnDef } from "@tanstack/react-table";
import {
  MoreHorizontal,
  Play,
  Square,
  RotateCw,
  Zap,
  Lock,
  LockOpen,
  ArrowRightLeft,
  Network,
  RefreshCw,
  Trash2,
  ExternalLink,
} from "lucide-react";
import { type DataTableFeatures } from "@/components/data-table/features";
import { DataTableColumnHeader } from "@/components/data-table";
import { StatusBadge } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { AdminServerListItem } from "@/lib/api";
import { serverPath } from "@/lib/panel-paths";
import { getServerStatus } from "@/lib/server-status";
import { localeTag, t } from "@/lib/i18n";
import { cn } from "@/lib/utils";

export type AdminServerRowActions = {
  onPower: (server: AdminServerListItem, action: string) => void;
  onBlock: (server: AdminServerListItem) => void;
  onUnblock: (server: AdminServerListItem) => void;
  onMigrate: (server: AdminServerListItem) => void;
  onAssignIP: (server: AdminServerListItem) => void;
  onReinstall: (server: AdminServerListItem) => void;
  onDelete: (server: AdminServerListItem) => void;
  canWrite: boolean;
};

function daysLeft(expiresAt: string | null): number | null {
  if (!expiresAt) return null;
  const ms = new Date(expiresAt).getTime() - Date.now();
  return Math.ceil(ms / 86_400_000);
}

function LoadBar({ percent }: { percent?: number }) {
  if (percent === undefined || percent === null) {
    return <span className="text-muted-foreground">—</span>;
  }
  const value = Math.max(0, Math.min(100, percent));
  const tone =
    value >= 90
      ? "bg-destructive"
      : value >= 70
        ? "bg-amber-500"
        : "bg-primary";
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-12 overflow-hidden rounded-full bg-muted">
        <div className={cn("h-full rounded-full", tone)} style={{ width: `${value}%` }} />
      </div>
      <span className="tabular-nums text-xs text-muted-foreground">{Math.round(value)}%</span>
    </div>
  );
}

export function getAdminServersColumns(
  actions: AdminServerRowActions
): ColumnDef<DataTableFeatures, AdminServerListItem>[] {
  return [
    {
      id: "select",
      header: ({ table }) => (
        <Checkbox
          checked={
            table.getIsAllPageRowsSelected() ||
            (table.getIsSomePageRowsSelected() && "indeterminate")
          }
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label={t("servers.admin.list.select_all")}
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label={t("servers.admin.list.select_row")}
        />
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: "name",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("common.name")} />
      ),
      cell: ({ row }) => {
        const s = row.original;
        return (
          <div className="flex flex-col gap-1">
            <Link
              href={serverPath("/admin", s.id)}
              className="font-medium text-primary hover:underline"
            >
              {s.name}
            </Link>
            <div className="flex flex-wrap gap-1">
              {s.is_blocked && (
                <Badge variant="destructive" className="h-4 px-1 text-[10px]">
                  {t("servers.admin.list.badge_blocked")}
                </Badge>
              )}
              {s.provisioning_status === "failed" && (
                <Badge variant="outline" className="h-4 px-1 text-[10px]">
                  {t("servers.admin.list.badge_failed")}
                </Badge>
              )}
            </div>
          </div>
        );
      },
    },
    {
      id: "game",
      accessorFn: (row) => row.game?.name ?? "",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("common.game")} />
      ),
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.game?.name ?? "—"}
        </span>
      ),
      filterFn: (row, id, value: string[]) => value.includes(row.getValue(id)),
    },
    {
      id: "owner",
      accessorFn: (row) => row.owner_email ?? "",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("servers.admin.col_owner")} />
      ),
      cell: ({ row }) => (
        <span className="text-xs text-muted-foreground">
          {row.original.owner_email || "—"}
        </span>
      ),
    },
    {
      id: "location",
      accessorFn: (row) => row.location?.name ?? "",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("servers.admin.list.col_location")} />
      ),
      cell: ({ row }) => {
        const loc = row.original.location;
        if (!loc) return <span className="text-muted-foreground">—</span>;
        return (
          <div className="flex flex-col">
            <span className="text-sm">{loc.name}</span>
            {loc.city && (
              <span className="text-[11px] text-muted-foreground">{loc.city}</span>
            )}
          </div>
        );
      },
      filterFn: (row, id, value: string[]) => value.includes(row.getValue(id)),
    },
    {
      id: "address",
      accessorFn: (row) => (row.port ? `${row.ip_address}:${row.port}` : row.ip_address),
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("servers.admin.list.col_address")} />
      ),
      cell: ({ row }) => {
        const s = row.original;
        const address = s.port ? `${s.ip_address}:${s.port}` : s.ip_address;
        if (!address) return <span className="text-muted-foreground">—</span>;
        return <span className="font-mono text-xs">{address}</span>;
      },
    },
    {
      id: "tariff",
      accessorFn: (row) => row.tariff?.name ?? "",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("servers.admin.list.col_tariff")} />
      ),
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.tariff?.name ?? "—"}
        </span>
      ),
    },
    {
      id: "expires",
      accessorFn: (row) => row.expires_at ?? "",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("servers.admin.list.col_expires")} />
      ),
      cell: ({ row }) => {
        const expires = row.original.expires_at;
        if (!expires) return <span className="text-muted-foreground">—</span>;
        const left = daysLeft(expires);
        const tone =
          left === null
            ? ""
            : left < 0
              ? "text-destructive"
              : left <= 7
                ? "text-amber-600 dark:text-amber-500"
                : "";
        return (
          <div className="flex flex-col">
            <span className={cn("text-sm tabular-nums", tone)}>
              {new Date(expires).toLocaleDateString(localeTag())}
            </span>
            {left !== null && (
              <span className={cn("text-[11px] text-muted-foreground", tone)}>
                {left < 0
                  ? t("servers.admin.list.expired")
                  : t("servers.admin.list.days_left", { days: left })}
              </span>
            )}
          </div>
        );
      },
    },
    {
      id: "load",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("servers.admin.list.col_load")} />
      ),
      cell: ({ row }) => (
        <div className="flex flex-col gap-1">
          <LoadBar percent={row.original.cpu_percent} />
          <LoadBar percent={row.original.ram_percent} />
        </div>
      ),
      enableSorting: false,
    },
    {
      accessorKey: "status",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("common.status")} />
      ),
      cell: ({ row }) => {
        const st = getServerStatus(row.original);
        return <StatusBadge status={st.label} />;
      },
      filterFn: (row, id, value: string[]) => value.includes(row.getValue(id)),
    },
    {
      id: "actions",
      header: () => <span className="sr-only">{t("common.actions")}</span>,
      cell: ({ row }) => {
        const s = row.original;
        return (
          <div className="flex justify-end">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <MoreHorizontal className="h-4 w-4" />
                  <span className="sr-only">{t("common.actions")}</span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-52">
                <DropdownMenuItem asChild>
                  <Link href={serverPath("/admin", s.id)}>
                    <ExternalLink className="mr-2 h-4 w-4" />
                    {t("servers.admin.actions.open")}
                  </Link>
                </DropdownMenuItem>
                {actions.canWrite && (
                  <>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onClick={() => actions.onPower(s, "start")}>
                      <Play className="mr-2 h-4 w-4" />
                      {t("common.start")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => actions.onPower(s, "stop")}>
                      <Square className="mr-2 h-4 w-4" />
                      {t("servers.shell.power_off")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => actions.onPower(s, "restart")}>
                      <RotateCw className="mr-2 h-4 w-4" />
                      {t("common.restart")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => actions.onPower(s, "kill")}>
                      <Zap className="mr-2 h-4 w-4" />
                      {t("servers.admin.actions.kill")}
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    {s.is_blocked ? (
                      <DropdownMenuItem onClick={() => actions.onUnblock(s)}>
                        <LockOpen className="mr-2 h-4 w-4" />
                        {t("servers.admin.actions.unblock")}
                      </DropdownMenuItem>
                    ) : (
                      <DropdownMenuItem onClick={() => actions.onBlock(s)}>
                        <Lock className="mr-2 h-4 w-4" />
                        {t("servers.admin.actions.block")}
                      </DropdownMenuItem>
                    )}
                    <DropdownMenuItem onClick={() => actions.onMigrate(s)}>
                      <ArrowRightLeft className="mr-2 h-4 w-4" />
                      {t("servers.admin.migrate")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => actions.onAssignIP(s)}>
                      <Network className="mr-2 h-4 w-4" />
                      {t("servers.admin.actions.ip")}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => actions.onReinstall(s)}>
                      <RefreshCw className="mr-2 h-4 w-4" />
                      {t("servers.shell.reinstall")}
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      variant="destructive"
                      onClick={() => actions.onDelete(s)}
                    >
                      <Trash2 className="mr-2 h-4 w-4" />
                      {t("common.delete")}
                    </DropdownMenuItem>
                  </>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        );
      },
      enableSorting: false,
      enableHiding: false,
    },
  ];
}
