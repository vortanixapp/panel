import { Badge } from "@/components/ui/badge";
import { t } from "@/lib/i18n";
import { cn } from "@/lib/utils";

export function SupportTicketStatusBadge({
  status,
  className,
}: {
  status: string;
  className?: string;
}) {
  const isClosed = status === "closed";
  return (
    <Badge
      variant={isClosed ? "secondary" : "outline"}
      className={cn(
        "gap-1 font-medium",
        !isClosed &&
          "border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
        className
      )}
    >
      <i
        className={cn(
          "text-xs",
          isClosed ? "ri-checkbox-circle-line" : "ri-chat-3-line"
        )}
      />
      {isClosed ? t("support.status.closed") : t("support.status.open")}
    </Badge>
  );
}
