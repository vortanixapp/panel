import { statusVariant, type StatusVariant } from "@/lib/status";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

const variantClass: Record<StatusVariant, string> = {
  success:
    "border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
  warning:
    "border-transparent bg-amber-500/15 text-amber-700 dark:text-amber-400",
  destructive: "",
  secondary: "",
};

const badgeVariant: Record<
  StatusVariant,
  "default" | "destructive" | "outline" | "secondary"
> = {
  success: "outline",
  warning: "outline",
  destructive: "destructive",
  secondary: "secondary",
};

export function StatusBadge({
  status,
  className,
}: {
  status: string;
  className?: string;
}) {
  const variant = statusVariant(status);
  return (
    <Badge
      variant={badgeVariant[variant]}
      className={cn("capitalize", variantClass[variant], className)}
    >
      {status}
    </Badge>
  );
}
