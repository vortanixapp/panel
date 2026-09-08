"use client";

import { useMemo } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, ArrowRight, Clock, List, Newspaper } from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import { hasHtmlMarkup, sanitizeNewsHtml } from "@/components/user/news/news-utils";
import {
  NewsTag,
  extractHeadings,
  formatNewsFull,
  readingTime,
} from "@/components/user/news/news-parts";
import { fetchNews, fetchNewsArticle } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { btnGhost } from "@/components/user/panel-parts";

export function NewsArticlePageContent() {
  const t = useT();
  const { slug } = useParams<{ slug: string }>();

  const { data: item, isLoading, isError } = useQuery({
    queryKey: queryKeys.newsItem(slug ?? ""),
    queryFn: () => fetchNewsArticle(slug),
    enabled: !!slug,
  });

  // Соседние новости: дочитав одну, чаще всего идут к следующей.
  const { data: feed } = useQuery({ queryKey: queryKeys.newsList, queryFn: fetchNews });
  const related = (feed?.news ?? []).filter((n) => n.slug !== slug).slice(0, 3);

  const raw = String(item?.body || item?.excerpt || "").trim();
  const isHtml = hasHtmlMarkup(raw);

  // Порядок обязателен: сначала очистка, потом расстановка якорей. Наоборот
  // очистка вырезала бы наши id — их нет в списке разрешённых атрибутов, и
  // оглавление вело бы в никуда.
  const { html, toc } = useMemo(() => {
    if (!isHtml) return { html: "", toc: [] as { id: string; title: string }[] };
    return extractHeadings(sanitizeNewsHtml(raw));
  }, [raw, isHtml]);

  if (isLoading) {
    return (
      <PageShell variant="user">
        <div className="flex flex-col gap-[22px]">
          <Skeleton className="h-14 w-96 rounded-xl" />
          <Skeleton className="h-96 rounded-2xl" />
        </div>
      </PageShell>
    );
  }

  if (isError || !item) {
    return (
      <PageShell variant="user">
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <span className="flex h-14 w-14 items-center justify-center rounded-2xl bg-muted">
            <Newspaper className="h-6 w-6 text-muted-foreground" />
          </span>
          <p className="text-sm text-muted-foreground">{t("news.article.not_found")}</p>
          <Link href="/news" className={cn(btnGhost, "h-[34px]")}>
            {t("news.all")}
          </Link>
        </div>
      </PageShell>
    );
  }

  return (
    <PageShell variant="user">
      <div className="flex w-full flex-col gap-[22px]">
        <div className="flex flex-wrap items-start gap-[14px]">
          <Link
            href="/news"
            className="mt-1 flex h-[34px] w-[34px] flex-none items-center justify-center rounded-[9px] border border-input text-foreground transition-colors hover:bg-accent"
            aria-label={t("news.article.back_aria")}
          >
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <NewsTag tag={item.tag} />
              <span className="text-[12.5px] text-muted-foreground">
                {formatNewsFull(item.published_at)}
              </span>
              <span className="flex items-center gap-1.5 text-[12.5px] text-muted-foreground/70">
                <Clock className="h-3.5 w-3.5" />
                {readingTime(item.body, item.excerpt)}
              </span>
            </div>
            <h1 className="mt-2 text-[26px] leading-tight font-bold tracking-[-0.02em]">
              {item.title}
            </h1>
            {item.excerpt && (
              <p className="mt-2 max-w-3xl text-[13.5px] leading-relaxed text-muted-foreground">
                {item.excerpt}
              </p>
            )}
          </div>
        </div>

        <div className="grid grid-cols-1 items-start gap-5 lg:grid-cols-[minmax(0,1fr)_260px]">
          <article className="flex flex-col gap-4">
            {item.image && (
              <img
                src={item.image}
                alt=""
                className="w-full rounded-2xl border border-border bg-muted object-cover"
              />
            )}
            <div className="rounded-2xl border border-border bg-card px-[22px] py-6">
              {isHtml ? (
                <div
                  className="vx-article"
                  dangerouslySetInnerHTML={{ __html: html }}
                />
              ) : (
                <div className="text-[15px] leading-7 whitespace-pre-line">{raw}</div>
              )}
            </div>
          </article>

          <div className="flex flex-col gap-[14px] lg:sticky lg:top-[88px]">
            {/* Оглавление только когда в тексте есть заголовки: панель с одним
                пунктом занимает место и ничем не помогает. */}
            {toc.length > 1 && (
              <nav className="flex flex-col gap-2.5 rounded-2xl border border-border bg-card px-5 py-4">
                <span className="flex items-center gap-2 text-[12.5px] text-muted-foreground">
                  <List className="h-3.5 w-3.5" />
                  {t("news.article.toc")}
                </span>
                {toc.map((h) => (
                  <a
                    key={h.id}
                    href={`#${h.id}`}
                    className="text-[13px] text-muted-foreground transition-colors hover:text-foreground"
                  >
                    {h.title}
                  </a>
                ))}
              </nav>
            )}

            {related.length > 0 && (
              <div className="flex flex-col gap-3 rounded-2xl border border-border bg-card px-5 py-4">
                <span className="text-[12.5px] text-muted-foreground">
                  {t("news.article.more")}
                </span>
                {related.map((n) => (
                  <Link key={n.slug} href={`/news/${n.slug}`} className="group flex flex-col gap-1">
                    <NewsTag tag={n.tag} className="self-start" />
                    <span className="text-[13px] leading-snug transition-colors group-hover:text-foreground">
                      {n.title}
                    </span>
                  </Link>
                ))}
                <Link
                  href="/news"
                  className="mt-1 inline-flex items-center gap-1.5 text-[12.5px] text-muted-foreground transition-colors hover:text-foreground"
                >
                  {t("news.all")}
                  <ArrowRight className="h-3.5 w-3.5" />
                </Link>
              </div>
            )}
          </div>
        </div>
      </div>
    </PageShell>
  );
}
