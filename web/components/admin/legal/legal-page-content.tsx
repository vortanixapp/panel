"use client";

import { LegalConsents } from "@/components/admin/legal/legal-consents";
import { LegalDocuments } from "@/components/admin/legal/legal-documents";
import { LegalSettings } from "@/components/admin/legal/legal-settings";
import { PageShell } from "@/components/layout/page-shell";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useT } from "@/hooks/use-translations";

export function LegalPageContent() {
  const t = useT();
  return (
    <PageShell variant="admin">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">{t("admin.legal.title")}</h1>
        <p className="text-sm text-muted-foreground">{t("admin.legal.subtitle")}</p>
      </div>
      <Tabs defaultValue="documents">
        <TabsList className="mb-4">
          <TabsTrigger value="documents">{t("admin.legal.tab_documents")}</TabsTrigger>
          <TabsTrigger value="consents">{t("admin.legal.tab_consents")}</TabsTrigger>
          <TabsTrigger value="settings">{t("admin.legal.tab_settings")}</TabsTrigger>
        </TabsList>
        <TabsContent value="documents">
          <LegalDocuments />
        </TabsContent>
        <TabsContent value="consents">
          <LegalConsents />
        </TabsContent>
        <TabsContent value="settings">
          <LegalSettings />
        </TabsContent>
      </Tabs>
    </PageShell>
  );
}
