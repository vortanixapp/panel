"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { PageShell } from "@/components/layout/page-shell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchFeatures } from "@/lib/api";
import { useT } from "@/hooks/use-translations";

export default function FeaturesPage() {
  const t = useT();
  const { data, isLoading } = useQuery({
    queryKey: ["features"],
    queryFn: async () => (await fetchFeatures()).features,
  });

  return (
    <PageShell variant="user">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("landing.catalog.features_title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("landing.catalog.features_subtitle")}
          </p>
        </div>
        <Link href="/games" className="text-sm text-primary hover:underline">
          {t("landing.catalog.games_link")}
        </Link>
      </div>

      {isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {(data ?? []).map((feature) => (
            <Card key={feature.title}>
              <CardHeader>
                <CardTitle className="text-base">{feature.title}</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">{feature.description}</p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </PageShell>
  );
}
