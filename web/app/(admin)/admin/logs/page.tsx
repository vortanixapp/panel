"use client";

import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { type ColumnDef } from "@tanstack/react-table";
import { type DataTableFeatures } from "@/components/data-table/features";
import { DataTable, DataTableColumnHeader } from "@/components/data-table";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { fetchAdminLogsFiltered, subscribeAdminLogs, type AdminLogEvent } from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { type TranslateFn } from "@/lib/i18n";
import { toast } from "sonner";

type LogEntry = { id?: string; action: string; resource: string; created_at: string };

// t передаётся параметром: колонки собираются при рендере, иначе заголовки
// застыли бы на языке, который стоял при загрузке модуля.
function getLogsColumns(t: TranslateFn): ColumnDef<DataTableFeatures, LogEntry>[] {
  return [
    {
      accessorKey: "action",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t("admin.logs.col_action")} />
      ),
    },
    { accessorKey: "resource", header: t("admin.logs.col_resource") },
    { accessorKey: "created_at", header: t("common.date") },
  ];
}

export default function AdminLogsPage() {
  const t = useT();
  const [search, setSearch] = useState("");
  const [action, setAction] = useState("");
  const [limit, setLimit] = useState("200");
  const [live, setLive] = useState(false);
  const [streamRows, setStreamRows] = useState<LogEntry[]>([]);

  const logsQuery = useQuery({
    queryKey: ["admin-logs", search, action, limit],
    queryFn: () =>
      fetchAdminLogsFiltered({
        search: search.trim() || undefined,
        action: action.trim() || undefined,
        limit: Number(limit) || 200,
      }),
  });

  useEffect(() => {
    if (!live) return;
    return subscribeAdminLogs(
      (item: AdminLogEvent) => {
        setStreamRows((prev) => {
          if (item.id && prev.some((row) => row.id === item.id)) return prev;
          return [item, ...prev].slice(0, 200);
        });
      },
      (err) => toast.error(err.message)
    );
  }, [live]);

  const columns = useMemo(() => getLogsColumns(t), [t]);

  const streamedIds = new Set(streamRows.map((r) => r.id).filter(Boolean));
  const mergedRows = [
    ...streamRows,
    ...(logsQuery.data?.logs ?? []).filter(
      (row: LogEntry) => !row.id || !streamedIds.has(row.id)
    ),
  ];

  function downloadCsv() {
    const rows = mergedRows
      .map((r) =>
        [r.created_at, r.action, r.resource]
          .map((v) => `"${String(v ?? "").replaceAll('"', '""')}"`)
          .join(",")
      )
      .join("\n");
    const csv = `created_at,action,resource\n${rows}\n`;
    const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "admin-audit-logs.csv";
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
  }

  return (
    <PageShell variant="admin">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">{t("admin.logs.title")}</h1>
        <p className="text-sm text-muted-foreground">{t("admin.logs.subtitle")}</p>
      </div>

      <div className="mb-4 grid gap-3 rounded-2xl border bg-card p-4 md:grid-cols-4">
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">{t("common.search")}</Label>
          <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="action/resource" />
        </div>
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">Action</Label>
          <Input value={action} onChange={(e) => setAction(e.target.value)} placeholder="admin.update.apply" />
        </div>
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">Limit</Label>
          <Input value={limit} onChange={(e) => setLimit(e.target.value)} placeholder="200" />
        </div>
        <div className="flex items-end gap-2">
          <Button variant="outline" onClick={() => logsQuery.refetch()}>
            {t("common.refresh")}
          </Button>
          <Button variant="outline" onClick={downloadCsv}>
            CSV
          </Button>
          <Button variant={live ? "destructive" : "default"} onClick={() => setLive((v) => !v)}>
            {live ? t("admin.logs.stop_live") : t("admin.logs.start_live")}
          </Button>
        </div>
      </div>

      <DataTable
        columns={columns}
        data={mergedRows}
        searchKey="action"
        searchPlaceholder={t("admin.logs.filter_placeholder")}
        emptyMessage={logsQuery.isLoading ? t("common.loading") : t("admin.logs.empty")}
      />
    </PageShell>
  );
}
