import type { AdminTask } from "@/lib/api";
import { localeTag, type TranslateFn } from "@/lib/i18n";

export function taskStatusLabel(task: Pick<AdminTask, "status" | "stuck">, t: TranslateFn): string {
  if (task.stuck) return t("admin.tasks.status.stuck");
  return t(`admin.tasks.status.${task.status}`);
}

export function taskStatusVariant(
  task: Pick<AdminTask, "status" | "stuck">
): "default" | "secondary" | "destructive" | "outline" {
  if (task.stuck || task.status === "failed") return "destructive";
  if (task.status === "running") return "default";
  if (task.status === "done") return "secondary";
  return "outline";
}

export function formatTaskDuration(seconds: number, t: TranslateFn): string {
  const total = Math.max(0, Math.round(seconds));
  if (total < 60) return `${total} ${t("admin.jobs.unit_seconds")}`;
  const minutes = Math.floor(total / 60);
  if (minutes < 60) {
    const rest = total % 60;
    return rest > 0
      ? `${minutes} ${t("admin.jobs.unit_minutes")} ${rest} ${t("admin.jobs.unit_seconds")}`
      : `${minutes} ${t("admin.jobs.unit_minutes")}`;
  }
  const hours = Math.floor(minutes / 60);
  return `${hours} ${t("admin.jobs.unit_hours")} ${minutes % 60} ${t("admin.jobs.unit_minutes")}`;
}

export function formatTaskTime(iso: string | undefined): string {
  if (!iso) return "—";
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString(localeTag());
}

export function stringifyJson(value: unknown): string {
  if (value === undefined || value === null) return "";
  try {
    const text = JSON.stringify(value, null, 2);
    return text === "{}" ? "" : text;
  } catch {
    return "";
  }
}
