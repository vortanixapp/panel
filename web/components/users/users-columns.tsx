"use client";

import { type ColumnDef } from "@tanstack/react-table";
import { Trash2 } from "lucide-react";
import { DataTableColumnHeader } from "@/components/data-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { PanelUser } from "@/lib/api";
import { localeTag, type TranslateFn } from "@/lib/i18n";
import { roleLabel } from "@/lib/rbac";

type UserActions = {
  onDelete: (user: PanelUser) => void;
  deletePending?: boolean;
};

function formatDate(value: string) {
  try {
    return new Intl.DateTimeFormat(localeTag(), {
      dateStyle: "medium",
      timeStyle: "short",
    }).format(new Date(value));
  } catch {
    return value;
  }
}

// t передаётся параметром: колонки собираются при рендере таблицы, иначе
// заголовки застыли бы на языке, который стоял при загрузке модуля.
export function getUsersColumns(
  actions: UserActions,
  t: TranslateFn
): ColumnDef<PanelUser>[] {
  return [
    {
      accessorKey: "email",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("common.email")} />
      ),
      cell: ({ row }) => (
        <span className="font-medium">{row.getValue("email")}</span>
      ),
    },
    {
      accessorKey: "role",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("admin.users.role")} />
      ),
      cell: ({ row }) => (
        <Badge variant="secondary">{roleLabel(row.getValue("role"))}</Badge>
      ),
      filterFn: (row, id, value: string[]) => value.includes(row.getValue(id)),
    },
    {
      accessorKey: "status",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("common.status")} />
      ),
      cell: ({ row }) => {
        const status = row.getValue("status") as string;
        return (
          <Badge variant={status === "active" ? "default" : "outline"}>
            {status === "active" ? t("admin.users.status.active") : status}
          </Badge>
        );
      },
      filterFn: (row, id, value: string[]) => value.includes(row.getValue(id)),
    },
    {
      accessorKey: "created_at",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("common.created_at")} />
      ),
      cell: ({ row }) => (
        <span className="text-muted-foreground">
          {formatDate(row.getValue("created_at"))}
        </span>
      ),
    },
    {
      id: "actions",
      header: () => <span className="sr-only">{t("common.actions")}</span>,
      cell: ({ row }) => {
        const user = row.original;
        if (user.role === "owner") return null;
        return (
          <div className="flex justify-end">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => actions.onDelete(user)}
              disabled={actions.deletePending}
              aria-label={t("admin.users.delete_user")}
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        );
      },
      enableSorting: false,
      enableHiding: false,
    },
  ];
}
