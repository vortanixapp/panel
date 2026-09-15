"use client";

import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import { Panel, VX_MUTED, btnClass } from "@/components/vx/panel-ui";
import { RentServerPageContent } from "@/components/user/rent-server-page-content";
import { useBrand } from "@/context/brand-provider";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";

export function RentServerGate() {
  const t = useT();
  const { whmcs, ready } = useBrand();

  if (!ready) {
    return (
      <PageShell>
        <Skeleton className="h-[240px] w-full rounded-[14px]" />
      </PageShell>
    );
  }

  if (!whmcs?.ordersOnly) return <RentServerPageContent />;

  return (
    <PageShell>
      <div className="font-panel mx-auto flex w-full max-w-xl flex-col gap-[18px]">
        <Panel
          title={t("billing.whmcs.orders_title")}
          bodyClassName="flex flex-col gap-3.5 p-[18px]"
        >
          <p className={cn("text-[12.5px] leading-[1.6]", VX_MUTED)}>
            {t("billing.whmcs.orders_desc")}
          </p>
          <a href={whmcs.orderUrl} className={cn(btnClass("primary"), "h-9 w-full")}>
            {t("billing.whmcs.order_button")}
          </a>
          {whmcs.clientAreaUrl ? (
            <a href={whmcs.clientAreaUrl} className={cn(btnClass(), "h-9 w-full")}>
              {t("billing.whmcs.client_area")}
            </a>
          ) : null}
        </Panel>
      </div>
    </PageShell>
  );
}
