"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchGame } from "@/lib/api";
import { useT } from "@/hooks/use-translations";

export default function CatalogGamePage() {
  const t = useT();
  const { slug } = useParams<{ slug: string }>();
  const { data, isLoading, isError } = useQuery({
    queryKey: ["catalog-game", slug],
    queryFn: () => fetchGame(slug),
    enabled: !!slug,
  });

  return (
    <PageShell variant="user">
      <div className="mb-2">
        <Link href="/games" className="text-sm text-muted-foreground hover:text-foreground">
          {t("landing.catalog.back_to_games")}
        </Link>
      </div>
      {isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : isError || !data ? (
        <p className="text-sm text-destructive">
          {t("landing.catalog.game_not_found")}
        </p>
      ) : (
        <div>
          <h1 className="mb-4 text-2xl font-bold tracking-tight">{data.name}</h1>
          {data.description && (
            <p className="text-muted-foreground">{data.description}</p>
          )}
        </div>
      )}
    </PageShell>
  );
}
