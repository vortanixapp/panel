"use client";

import { useEffect } from "react";
import {
  adoptSession,
  migrateLegacySession,
  stopAuthRefreshLoop,
} from "@/lib/api";

export function AuthSessionProvider({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    void migrateLegacySession().finally(() => adoptSession());
    return () => stopAuthRefreshLoop();
  }, []);

  return children;
}
