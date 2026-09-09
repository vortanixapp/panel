"use client";

import { useQuery } from "@tanstack/react-query";
import { type ColumnDef } from "@tanstack/react-table";
import { type DataTableFeatures } from "@/components/data-table/features";
import { PageShell } from "@/components/layout/page-shell";
import { DataTable } from "@/components/data-table";
import { Skeleton } from "@/components/ui/skeleton";
import { adminFetch } from "@/lib/api";
import { useT } from "@/hooks/use-translations";

type AdminListPageProps<T extends object> = {
  title: string;
  description?: string;
  queryKey?: readonly unknown[];
  queryFn?: () => Promise<T[]>;
  columns?: ColumnDef<DataTableFeatures, T>[];
  searchKey?: string;
  searchPlaceholder?: string;
  emptyMessage?: string;
  path?: string;
  dataKey?: string;
};

function defaultColumns<T extends object>(rows: T[]): ColumnDef<DataTableFeatures, T>[] {
  if (rows.length === 0) {
    return [{ accessorKey: "id", header: "ID" }];
  }
  const keys = Object.keys(rows[0]).slice(0, 6);
  return keys.map((key) => ({
    accessorKey: key,
    header: key,
    cell: ({ row }) => {
      const v = row.getValue(key);
      if (v === null || v === undefined) return "—";
      if (typeof v === "object") return JSON.stringify(v);
      return String(v);
    },
  }));
}

export function AdminListPage<T extends object>(props: AdminListPageProps<T>) {
  const t = useT();
  const {
    title,
    description,
    searchKey,
    searchPlaceholder = t("common.search_placeholder"),
    emptyMessage = t("admin.list.empty"),
    path,
    dataKey,
    queryKey: explicitKey,
    queryFn: explicitFn,
    columns: explicitColumns,
  } = props;

  const legacy = Boolean(path && dataKey);
  const queryKey = explicitKey ?? (legacy ? ["admin-list", path, dataKey] : ["admin-list", title]);
  const queryFn =
    explicitFn ??
    (legacy
      ? async () => {
          const res = await adminFetch<Record<string, T[]>>(path!);
          return res[dataKey!] ?? [];
        }
      : async () => [] as T[]);

  const { data = [], isLoading, isError } = useQuery({
    queryKey,
    queryFn,
    enabled: legacy || Boolean(explicitFn),
  });

  const columns = explicitColumns ?? defaultColumns(data);

  return (
    <PageShell variant="admin">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">{title}</h1>
        {description && (
          <p className="text-sm text-muted-foreground">{description}</p>
        )}
      </div>

      {isError && (
        <p className="mb-4 text-sm text-destructive">
          {t("admin.list.load_failed")}
        </p>
      )}

      {isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <DataTable
          columns={columns}
          data={data}
          searchKey={searchKey}
          searchPlaceholder={searchPlaceholder}
          emptyMessage={emptyMessage}
        />
      )}
    </PageShell>
  );
}
