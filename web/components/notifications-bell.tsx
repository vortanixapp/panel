"use client";

import { useEffect, useRef, useState } from "react";
import { AnimatePresence, m } from "motion/react";
import { Bell } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { NotificationsPopover } from "@/components/notifications/notifications-popover";
import { useUnreadCount } from "@/hooks/use-notifications";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";

export function NotificationsBell() {
  const t = useT();
  const count = useUnreadCount();
  const [open, setOpen] = useState(false);
  const previous = useRef(count);
  const [rings, setRings] = useState(0);

  useEffect(() => {
    if (count > previous.current) setRings((n) => n + 1);
    previous.current = count;
  }, [count]);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className={cn("relative", open && "bg-accent text-accent-foreground")}
          aria-label={t("layout.notifications_aria")}
        >
          <m.span
            key={rings}
            className="inline-flex origin-top"
            animate={rings ? { rotate: [0, -18, 14, -10, 6, -3, 0] } : undefined}
            transition={{ duration: 0.9, ease: "easeInOut" }}
          >
            <Bell className="h-5 w-5" />
          </m.span>
          <AnimatePresence>
            {count > 0 && (
              <m.span
                key="badge"
                initial={{ scale: 0 }}
                animate={{ scale: 1 }}
                exit={{ scale: 0 }}
                transition={{ type: "spring", stiffness: 520, damping: 24 }}
                className="absolute -right-0.5 -top-0.5 flex h-4 min-w-4 items-center justify-center overflow-hidden rounded-full bg-destructive px-1 text-[10px] font-medium text-destructive-foreground"
              >
                <AnimatePresence mode="popLayout" initial={false}>
                  <m.span
                    key={count}
                    initial={{ y: 8, opacity: 0 }}
                    animate={{ y: 0, opacity: 1 }}
                    exit={{ y: -8, opacity: 0 }}
                    transition={{ duration: 0.25 }}
                  >
                    {count > 99 ? "99+" : count}
                  </m.span>
                </AnimatePresence>
              </m.span>
            )}
          </AnimatePresence>
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="end"
        sideOffset={10}
        collisionPadding={8}
        className="w-[calc(100vw-16px)] overflow-hidden rounded-2xl p-0 shadow-xl sm:w-[400px]"
      >
        <NotificationsPopover onClose={() => setOpen(false)} />
      </PopoverContent>
    </Popover>
  );
}
