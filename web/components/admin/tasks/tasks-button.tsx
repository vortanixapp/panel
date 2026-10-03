"use client";

import { useState } from "react";
import { ListChecks, Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useAdminTaskFeed } from "@/hooks/use-admin-tasks";
import { useMe } from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";
import { isAdminRole } from "@/lib/rbac";
import { cn } from "@/lib/utils";

import { AdminTasksDialog } from "./tasks-dialog";

export function AdminTasksButton() {
  const t = useT();
  const [open, setOpen] = useState(false);
  const { data: me } = useMe();
  const isAdmin = isAdminRole(me?.role);
  const feed = useAdminTaskFeed("active", isAdmin, open);

  if (!isAdmin || (feed.isError && !feed.data)) return null;

  const counts = feed.data?.counts;
  const active = counts?.active ?? 0;
  const running = (counts?.running ?? 0) > 0;
  const failed = (counts?.failed_24h ?? 0) > 0;

  return (
    <>
      <Button
        variant="ghost"
        size="icon"
        className={cn("relative", open && "bg-accent text-accent-foreground")}
        aria-label={t("admin.tasks.aria")}
        title={t("admin.tasks.title")}
        onClick={() => setOpen(true)}
      >
        {running ? <Loader2 className="h-5 w-5 animate-spin" /> : <ListChecks className="h-5 w-5" />}
        {active > 0 ? (
          <span className="absolute -right-0.5 -top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-medium text-primary-foreground">
            {active > 99 ? "99+" : active}
          </span>
        ) : failed ? (
          <span className="absolute right-1 top-1 size-2 rounded-full bg-destructive" />
        ) : null}
      </Button>
      {open ? <AdminTasksDialog open={open} onOpenChange={setOpen} /> : null}
    </>
  );
}
