"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { AnimatePresence, m } from "motion/react";
import { Bell } from "lucide-react";
import { Button } from "@/components/ui/button";
import { fetchNotificationsUnreadCount } from "@/lib/api";
import { useT } from "@/hooks/use-translations";

export function NotificationsBell() {
  const t = useT();
  const { data } = useQuery({
    queryKey: ["notifications-unread"],
    queryFn: fetchNotificationsUnreadCount,
    refetchInterval: 60_000,
    refetchOnWindowFocus: true,
  });
  const count = data?.count ?? 0;
  const previous = useRef(count);
  const [rings, setRings] = useState(0);

  useEffect(() => {
    if (count > previous.current) setRings((n) => n + 1);
    previous.current = count;
  }, [count]);

  return (
    <Button variant="ghost" size="icon" className="relative" asChild>
      <Link href="/notifications" aria-label={t("layout.notifications_aria")}>
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
      </Link>
    </Button>
  );
}
