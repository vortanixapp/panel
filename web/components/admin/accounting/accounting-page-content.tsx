"use client";

import { AccountingBankStatement } from "@/components/admin/accounting/accounting-bank-statement";
import { AccountingClients } from "@/components/admin/accounting/accounting-clients";
import { AccountingOffsets } from "@/components/admin/accounting/accounting-offsets";
import { AccountingRefunds } from "@/components/admin/accounting/accounting-refunds";
import { AccountingReports } from "@/components/admin/accounting/accounting-reports";
import { AccountingRequisitesForm } from "@/components/admin/accounting/accounting-requisites";
import { AccountingTaxes } from "@/components/admin/accounting/accounting-taxes";
import { PageShell } from "@/components/layout/page-shell";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { usePublicSettings } from "@/context/brand-provider";
import { useT } from "@/hooks/use-translations";

export function AccountingPageContent() {
  const t = useT();
  const { accountingRegions } = usePublicSettings();
  const cis = accountingRegions.includes("cis");
  return (
    <PageShell variant="admin">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">{t("admin.accounting.title")}</h1>
        <p className="text-sm text-muted-foreground">{t("admin.accounting.subtitle")}</p>
      </div>
      <Tabs defaultValue="reports">
        <TabsList className="mb-4 flex h-auto flex-wrap">
          <TabsTrigger value="reports">{t("admin.accounting.tab_reports")}</TabsTrigger>
          <TabsTrigger value="clients">{t("admin.accounting.tab_clients")}</TabsTrigger>
          <TabsTrigger value="refunds">{t("admin.accounting.tab_refunds")}</TabsTrigger>
          {cis && <TabsTrigger value="offsets">{t("admin.accounting.tab_offsets")}</TabsTrigger>}
          {cis && <TabsTrigger value="statement">{t("admin.accounting.tab_statement")}</TabsTrigger>}
          <TabsTrigger value="taxes">{t("admin.accounting.tab_taxes")}</TabsTrigger>
          <TabsTrigger value="requisites">{t("admin.accounting.tab_requisites")}</TabsTrigger>
        </TabsList>
        <TabsContent value="reports">
          <AccountingReports />
        </TabsContent>
        <TabsContent value="clients">
          <AccountingClients />
        </TabsContent>
        <TabsContent value="refunds">
          <AccountingRefunds />
        </TabsContent>
        {cis && (
          <TabsContent value="offsets">
            <AccountingOffsets />
          </TabsContent>
        )}
        {cis && (
          <TabsContent value="statement">
            <AccountingBankStatement />
          </TabsContent>
        )}
        <TabsContent value="taxes">
          <AccountingTaxes />
        </TabsContent>
        <TabsContent value="requisites">
          <AccountingRequisitesForm />
        </TabsContent>
      </Tabs>
    </PageShell>
  );
}
