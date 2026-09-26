"use client";

import Link from "next/link";

import { useMe } from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";
import { isOwnerRole } from "@/lib/rbac";

export function FrozenBanner() {
  const t = useT();
  const { data: me } = useMe();
  if (!me?.panel_frozen) return null;

  return (
    <div className="fixed inset-x-0 bottom-0 z-50 flex flex-wrap items-center justify-center gap-2 border-t border-[rgba(232,160,60,0.28)] bg-[rgba(232,160,60,0.12)] px-4 py-2 text-[12.5px] text-[var(--vx-warn)] backdrop-blur">
      <span>{t("layout.frozen.banner")}</span>
      {isOwnerRole(me.role) && (
        <Link href="/admin/panel-transfer" className="underline">
          {t("layout.frozen.link")}
        </Link>
      )}
    </div>
  );
}
