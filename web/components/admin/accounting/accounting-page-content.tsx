"use client";

import { AccountingClients } from "@/components/admin/accounting/accounting-clients";
import { AccountingReports } from "@/components/admin/accounting/accounting-reports";
import { AccountingRequisitesForm } from "@/components/admin/accounting/accounting-requisites";
import { PageShell } from "@/components/layout/page-shell";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useT } from "@/hooks/use-translations";

export function AccountingPageContent() {
  const t = useT();
  return (
    <PageShell variant="admin">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">{t("admin.accounting.title")}</h1>
        <p className="text-sm text-muted-foreground">{t("admin.accounting.subtitle")}</p>
      </div>
      <Tabs defaultValue="reports">
        <TabsList className="mb-4">
          <TabsTrigger value="reports">{t("admin.accounting.tab_reports")}</TabsTrigger>
          <TabsTrigger value="clients">{t("admin.accounting.tab_clients")}</TabsTrigger>
          <TabsTrigger value="requisites">{t("admin.accounting.tab_requisites")}</TabsTrigger>
        </TabsList>
        <TabsContent value="reports">
          <AccountingReports />
        </TabsContent>
        <TabsContent value="clients">
          <AccountingClients />
        </TabsContent>
        <TabsContent value="requisites">
          <AccountingRequisitesForm />
        </TabsContent>
      </Tabs>
    </PageShell>
  );
}
