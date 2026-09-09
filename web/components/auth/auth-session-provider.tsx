"use client";

import { useEffect } from "react";
import {
  adoptActiveSession,
  getRefreshToken,
  startAuthRefreshLoop,
  stopAuthRefreshLoop,
} from "@/lib/api";

export function AuthSessionProvider({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    adoptActiveSession();
    if (getRefreshToken()) {
      startAuthRefreshLoop();
    }
    return () => stopAuthRefreshLoop();
  }, []);

  return children;
}
