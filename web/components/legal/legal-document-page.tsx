"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { LandingPublicLayout } from "@/components/landing/landing-public-layout";
import { LegalText } from "@/components/legal/legal-text";
import { fetchPublicLegalDocument } from "@/lib/api";
import { localeTag } from "@/lib/i18n";
import { useT } from "@/hooks/use-translations";

export function LegalDocumentPage() {
  const t = useT();
  const params = useParams();
  const kind = String(params?.kind ?? "");
  const { data, isLoading, isError } = useQuery({
    queryKey: ["legal-document", kind],
    queryFn: () => fetchPublicLegalDocument(kind),
    enabled: kind !== "",
    retry: false,
  });

  return (
    <LandingPublicLayout>
      <article className="mx-auto w-full max-w-[860px] px-4 py-12 sm:px-6 sm:py-16">
        {isLoading ? (
          <div className="flex items-center justify-center py-24 text-muted-foreground">
            <Loader2 className="mr-2 size-5 animate-spin" />
            {t("common.loading")}
          </div>
        ) : isError || !data ? (
          <div className="py-16 text-center">
            <h1 className="text-2xl font-bold">{t("legal.not_published")}</h1>
            <Link href="/" className="mt-4 inline-block text-primary hover:underline">
              {t("legal.to_home")}
            </Link>
          </div>
        ) : (
          <>
            <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">{data.title}</h1>
            <p className="mt-3 text-sm text-muted-foreground">
              {t("legal.revision", {
                version: data.version,
                date: new Date(data.published_at).toLocaleDateString(localeTag()),
              })}
            </p>
            <LegalText body={data.body ?? ""} className="mt-8" />
          </>
        )}
      </article>
    </LandingPublicLayout>
  );
}
