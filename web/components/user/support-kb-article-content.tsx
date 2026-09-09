"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, BookOpen, Eye, MessageSquarePlus } from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchKBArticle, fetchKBArticles } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { btnGhost, btnPrimary } from "@/components/user/panel-parts";

export function SupportKBArticleContent() {
  const t = useT();
  const { slug } = useParams<{ slug: string }>();

  const { data: article, isLoading, isError } = useQuery({
    queryKey: queryKeys.kbArticle(slug ?? ""),
    queryFn: () => fetchKBArticle(slug!),
    enabled: !!slug,
  });

  const { data: related } = useQuery({
    queryKey: queryKeys.kbArticles(article?.category, ""),
    queryFn: () => fetchKBArticles({ category: article?.category }),
    enabled: !!article?.category,
  });
  const others = (related?.articles ?? [])
    .filter((a) => a.slug !== slug)
    .slice(0, 4);

  if (isLoading) {
    return (
      <PageShell variant="user">
        <div className="flex flex-col gap-[22px]">
          <Skeleton className="h-12 w-80 rounded-xl" />
          <Skeleton className="h-96 rounded-2xl" />
        </div>
      </PageShell>
    );
  }

  if (isError || !article) {
    return (
      <PageShell variant="user">
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <span className="flex h-14 w-14 items-center justify-center rounded-2xl bg-muted">
            <BookOpen className="h-6 w-6 text-muted-foreground" />
          </span>
          <p className="text-sm text-muted-foreground">
            {t("support.kb.article.not_found")}
          </p>
          <Link href="/support/kb" className={cn(btnGhost, "h-[34px]")}>
            {t("support.kb.title")}
          </Link>
        </div>
      </PageShell>
    );
  }

  return (
    <PageShell variant="user">
      <div className="flex w-full flex-col gap-[22px]">
        <div className="flex flex-wrap items-center gap-[14px]">
          <Link
            href="/support/kb"
            className="flex h-[34px] w-[34px] flex-none items-center justify-center rounded-[9px] border border-input text-foreground transition-colors hover:bg-accent"
            aria-label={t("support.aria.back_to_kb")}
          >
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div className="min-w-0">
            <h1 className="text-[26px] leading-tight font-bold tracking-[-0.02em]">
              {article.title}
            </h1>
            <p className="mt-1.5 flex items-center gap-2 text-[12.5px] text-muted-foreground">
              <Eye className="h-3.5 w-3.5" />
              {t("support.kb.views", { n: article.views })}
            </p>
          </div>
        </div>

        <div className="grid grid-cols-1 items-start gap-5 lg:grid-cols-[minmax(0,1fr)_300px]">
          <article className="rounded-2xl border border-border bg-card px-[22px] py-5">
            {article.excerpt && (
              <p className="mb-4 text-sm leading-relaxed text-muted-foreground">
                {article.excerpt}
              </p>
            )}
            <div className="text-sm leading-relaxed whitespace-pre-wrap">
              {article.body || t("support.kb.article.empty_body")}
            </div>
          </article>

          <div className="flex flex-col gap-[14px] lg:sticky lg:top-[88px]">
            {others.length > 0 && (
              <div className="flex flex-col gap-[14px] rounded-2xl border border-border bg-card px-[22px] py-5">
                <span className="text-[15px] font-semibold">
                  {t("support.kb.article.nearby")}
                </span>
                <div className="flex flex-col gap-2">
                  {others.map((a) => (
                    <Link
                      key={a.id}
                      href={`/support/kb/${a.slug}`}
                      className="text-[13px] text-muted-foreground transition-colors hover:text-foreground"
                    >
                      {a.title}
                    </Link>
                  ))}
                </div>
              </div>
            )}

            <div className="flex flex-col gap-[14px] rounded-2xl border border-border bg-card px-[22px] py-5">
              <div className="flex flex-col gap-1">
                <span className="text-[15px] font-semibold">
                  {t("support.kb.article.not_helpful")}
                </span>
                <span className="text-[12.5px] text-muted-foreground">
                  {t("support.kb.article.not_helpful_text")}
                </span>
              </div>
              <Link href="/support/create" className={cn(btnPrimary, "h-[38px]")}>
                <MessageSquarePlus className="h-[15px] w-[15px]" />
                {t("support.kb.write_support")}
              </Link>
            </div>
          </div>
        </div>
      </div>
    </PageShell>
  );
}
