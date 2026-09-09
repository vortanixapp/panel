"use client";

import { type ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/data-table";
import { StatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import type { Node } from "@/lib/api";
import type { TranslateFn } from "@/lib/i18n";

type NodeActions = {
  onInstall: (id: string) => void;
  onDelete: (id: string) => void;
  installPending?: boolean;
};

export function getNodesColumns(
  actions: NodeActions,
  t: TranslateFn
): ColumnDef<Node>[] {
  return [
    {
      accessorKey: "name",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("common.name")} />
      ),
      cell: ({ row }) => (
        <span className="font-medium">{row.getValue("name")}</span>
      ),
    },
    {
      accessorKey: "fqdn",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("admin.nodes.fqdn")} />
      ),
      cell: ({ row }) => (
        <span className="text-muted-foreground">{row.getValue("fqdn")}</span>
      ),
    },
    {
      accessorKey: "status",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("common.status")} />
      ),
      cell: ({ row }) => <StatusBadge status={row.getValue("status")} />,
      filterFn: (row, id, value: string[]) =>
        value.includes(row.getValue(id)),
    },
    {
      id: "actions",
      header: () => <span className="sr-only">{t("common.actions")}</span>,
      cell: ({ row }) => (
        <div className="flex justify-end gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={actions.installPending}
            onClick={() => actions.onInstall(row.original.id)}
          >
            {t("admin.nodes.install")}
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => actions.onDelete(row.original.id)}
          >
            {t("common.delete")}
          </Button>
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    },
  ];
}
