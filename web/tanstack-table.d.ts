import "@tanstack/react-table";
import type { RowData, TableFeatures } from "@tanstack/react-table";

declare module "@tanstack/react-table" {
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  interface ColumnMeta<
    TFeatures extends TableFeatures,
    TData extends RowData,
    TValue,
  > {
    className?: string;
    tdClassName?: string;
    thClassName?: string;
    label?: string;
    mobile?: "lead" | "full" | "actions" | "block" | "hidden";
  }
}
