"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useBrand } from "@/context/brand-provider";
import { useT } from "@/hooks/use-translations";

const STORAGE_KEY = "vx_cookie_notice";

export function CookieBanner() {
  const t = useT();
  const { legal, ready } = useBrand();
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (!ready || legal?.cookie_banner === false) {
      setVisible(false);
      return;
    }
    try {
      setVisible(window.localStorage.getItem(STORAGE_KEY) !== "1");
    } catch {
      setVisible(true);
    }
  }, [ready, legal?.cookie_banner]);

  if (!visible) return null;

  const kinds = new Set((legal?.documents ?? []).map((doc) => doc.kind));
  const href = kinds.has("cookies") ? "/legal/cookies" : kinds.has("privacy") ? "/legal/privacy" : "";

  const accept = () => {
    try {
      window.localStorage.setItem(STORAGE_KEY, "1");
    } catch {}
    setVisible(false);
  };

  return (
    <div className="fixed inset-x-3 bottom-3 z-[60] mx-auto flex max-w-[720px] flex-wrap items-center gap-3 rounded-2xl border border-border bg-card/95 p-4 text-sm shadow-2xl backdrop-blur sm:inset-x-6">
      <p className="min-w-0 flex-1 text-muted-foreground">
        {t("legal.cookie.text")}{" "}
        {href && (
          <Link href={href} className="text-primary hover:underline">
            {t("legal.cookie.more")}
          </Link>
        )}
      </p>
      <button
        type="button"
        onClick={accept}
        className="vx-btn rounded-lg px-4 py-2 text-[13px] font-semibold"
      >
        {t("legal.cookie.accept")}
      </button>
    </div>
  );
}
