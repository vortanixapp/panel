"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { fetchGame } from "@/lib/api";
import { useT } from "@/hooks/use-translations";

export function GameDetailClient({ slug }: { slug: string }) {
  const t = useT();
  const { data: game, isLoading } = useQuery({
    queryKey: ["game", slug],
    queryFn: () => fetchGame(slug),
  });

  if (isLoading) return <div className="container py-12">{t("common.loading")}</div>;
  if (!game)
    return (
      <div className="container py-12">{t("landing.catalog.game_not_found")}</div>
    );

  return (
    <div className="container py-12">
      <Link href="/games" className="text-sm text-muted-foreground hover:underline">
        {t("landing.catalog.back_to_catalog")}
      </Link>
      <h1 className="mt-4 text-3xl font-bold">{game.name}</h1>
      {game.description && (
        <p className="mt-4 max-w-2xl text-muted-foreground">{game.description}</p>
      )}
    </div>
  );
}
