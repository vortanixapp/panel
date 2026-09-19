"use client";

import { useEffect } from "react";
import { useAccountQuery } from "@/hooks/use-account";
import { restoreDisplayTimeZone, setDisplayTimeZone } from "@/lib/timezone";

if (typeof window !== "undefined") {
  restoreDisplayTimeZone();
}

export function AccountSync() {
  const { data } = useAccountQuery();
  const zone = data?.user.timezone;

  useEffect(() => {
    if (zone !== undefined) setDisplayTimeZone(zone);
  }, [zone]);

  return null;
}
