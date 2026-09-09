"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { PageShell } from "@/components/layout/page-shell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchGames } from "@/lib/api";
import { useT } from "@/hooks/use-translations";

export default function CatalogGamesPage() {
  const t = useT();
  const { data, isLoading } = useQuery({
    queryKey: ["catalog-games"],
    queryFn: async () => (await fetchGames()).games,
  });

  return (
    <PageShell variant="user">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("landing.catalog.games_title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("landing.catalog.games_subtitle")}
          </p>
        </div>
        <Link href="/features" className="text-sm text-primary hover:underline">
          {t("landing.catalog.features_link")}
        </Link>
      </div>

      {isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {(data ?? []).map((game) => (
            <Card key={game.id}>
              <CardHeader>
                <CardTitle className="text-base">
                  <Link
                    href={`/games/${game.slug}`}
                    className="hover:text-primary hover:underline"
                  >
                    {game.name}
                  </Link>
                </CardTitle>
              </CardHeader>
              {game.description && (
                <CardContent>
                  <p className="text-sm text-muted-foreground">{game.description}</p>
                </CardContent>
              )}
            </Card>
          ))}
        </div>
      )}
    </PageShell>
  );
}
