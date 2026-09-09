"use client";

import { useEffect, useRef } from "react";
import { usePathname } from "next/navigation";
import LoadingBar, { type LoadingBarRef } from "react-top-loading-bar";

export function NavigationProgress() {
  const ref = useRef<LoadingBarRef>(null);
  const pathname = usePathname();
  const prevPath = useRef(pathname);

  useEffect(() => {
    if (prevPath.current !== pathname) {
      ref.current?.complete();
      prevPath.current = pathname;
    }
  }, [pathname]);

  useEffect(() => {
    ref.current?.continuousStart();
    const timer = setTimeout(() => ref.current?.complete(), 300);
    return () => clearTimeout(timer);
  }, [pathname]);

  return (
    <LoadingBar
      color="var(--muted-foreground)"
      ref={ref}
      shadow
      height={2}
    />
  );
}
