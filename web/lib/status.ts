export type StatusVariant = "success" | "warning" | "destructive" | "secondary";

export function statusVariant(status: string): StatusVariant {
  switch (status) {
    case "online":
    case "running":
      return "success";
    case "starting":
    case "stopping":
      return "warning";
    case "error":
      return "destructive";
    default:
      return "secondary";
  }
}

export function statusClass(status: string): string {
  return `status-${status}`;
}
