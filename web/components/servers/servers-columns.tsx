"use client";

import Link from "next/link";
import { type ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/data-table";
import { StatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import type { Server } from "@/lib/api";
import type { PanelBasePath } from "@/lib/panel-paths";
import { serverPath } from "@/lib/panel-paths";
import { getServerStatus } from "@/lib/server-status";
import { t } from "@/lib/i18n";

type ServerActions = {
  basePath?: PanelBasePath;
  onPower: (id: string, action: string) => void;
  onDelete: (id: string) => void;
  onReinstall?: (id: string) => void;
  onToggleBlock?: (id: string, blocked: boolean) => void;
  onMigrate?: (id: string, name: string) => void;
  onAssignIP?: (id: string, name: string, nodeId: string) => void;
  powerPending?: boolean;
  canManage?: boolean;
  showAdminOps?: boolean;
};

export function getServersColumns({
  basePath = "",
  onPower,
  onDelete,
  onReinstall,
  onToggleBlock,
  onMigrate,
  onAssignIP,
  powerPending,
  canManage = true,
  showAdminOps = false,
}: ServerActions): ColumnDef<Server>[] {
  return [
    {
      accessorKey: "name",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("common.name")} />
      ),
      cell: ({ row }) => (
        <Link
          href={serverPath(basePath, row.original.id)}
          className="font-medium text-primary hover:underline"
        >
          {row.getValue("name")}
        </Link>
      ),
    },
    {
      accessorKey: "game_id",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("common.game")} />
      ),
      cell: ({ row }) => (
        <span className="text-muted-foreground">{row.getValue("game_id")}</span>
      ),
    },
    {
      id: "owner",
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={t("servers.admin.col_owner")}
        />
      ),
      cell: ({ row }) => {
        const r: any = row.original;
        const owner = r.user_email || r.owner_email || r.user || r.user_id || "—";
        return <span className="text-xs text-muted-foreground">{String(owner)}</span>;
      },
    },
    {
      accessorKey: "status",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("common.status")} />
      ),
      cell: ({ row }) => {
        const st = getServerStatus(row.original as any);
        return <StatusBadge status={st.label} />;
      },
      filterFn: (row, id, value: string[]) =>
        value.includes(row.getValue(id)),
    },
    {
      id: "actions",
      header: () => <span className="sr-only">{t("common.actions")}</span>,
      cell: ({ row }) =>
        canManage ? (
          <div className="flex justify-end gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={powerPending}
              onClick={() => onPower(row.original.id, "start")}
            >
              Start
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={powerPending}
              onClick={() => onPower(row.original.id, "stop")}
            >
              Stop
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => onDelete(row.original.id)}
            >
              {t("common.delete")}
            </Button>
            {showAdminOps && onReinstall && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => onReinstall(row.original.id)}
              >
                Reinstall
              </Button>
            )}
            {showAdminOps && onMigrate && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => onMigrate(row.original.id, row.original.name)}
              >
                {t("servers.admin.migrate")}
              </Button>
            )}
            {showAdminOps && onAssignIP && (
              <Button
                variant="outline"
                size="sm"
                onClick={() =>
                  onAssignIP(
                    row.original.id,
                    row.original.name,
                    String((row.original as any).node_id ?? "")
                  )
                }
              >
                IP
              </Button>
            )}
            {showAdminOps && onToggleBlock && (
              <Button
                variant="outline"
                size="sm"
                onClick={() =>
                  onToggleBlock(
                    row.original.id,
                    !Boolean((row.original as any).is_blocked)
                  )
                }
              >
                {Boolean((row.original as any).is_blocked) ? "Unblock" : "Block"}
              </Button>
            )}
            {showAdminOps && (
              <Button
                variant="outline"
                size="sm"
                disabled={powerPending}
                onClick={() => onPower(row.original.id, "kill")}
              >
                Force stop
              </Button>
            )}
          </div>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        ),
      enableSorting: false,
      enableHiding: false,
    },
  ];
}
