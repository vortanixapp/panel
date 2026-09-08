"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ArrowRight, CheckCheck, Newspaper, Pin, Search } from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchNews, markNewsRead, type NewsItem } from "@/lib/api";
import { t } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { btnGhost, btnPrimary, fieldClass } from "@/components/user/panel-parts";
import {
  NEWS_TAGS,
  NewsDateBlock,
  NewsTag,
  readingTime,
} from "@/components/user/news/news-parts";

/** Закреплённая новость: крупная карточка над лентой. Она же первая в списке
 *  по сортировке сервера, поэтому из ленты её убираем — иначе дублируется. */
function PinnedCard({ item }: { item: NewsItem }) {
  return (
    <Link
      href={`/news/${item.slug}`}
      className="flex flex-col gap-4 rounded-2xl border border-border bg-card px-5 py-[18px] transition-colors hover:border-[var(--vx-panel-hover-line)] sm:flex-row sm:items-center"
    >
      {item.image ? (
        <img
          src={item.image}
          alt=""
          className="h-[150px] w-full flex-none rounded-xl border border-border bg-muted object-cover sm:w-[260px]"
        />
      ) : (
        <span className="flex h-[150px] w-full flex-none items-center justify-center rounded-xl border border-dashed border-input bg-muted/40 text-muted-foreground sm:w-[260px]">
          <Newspaper className="h-6 w-6" />
        </span>
      )}

      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="flex items-center gap-1.5 rounded-full border border-border px-[9px] py-[3px] text-[11.5px] text-muted-foreground">
            <Pin className="h-3 w-3" />
            {t("news.pinned")}
          </span>
          <NewsTag tag={item.tag} />
          <span className="text-[11.5px] text-muted-foreground/70">
            {readingTime(item.body, item.excerpt)}
          </span>
        </div>

        <h2 className="mt-2.5 text-xl leading-tight font-bold tracking-[-0.01em]">
          {item.title}
        </h2>
        {item.excerpt && (
          <p className="mt-2 text-[13.5px] leading-relaxed text-muted-foreground">
            {item.excerpt}
          </p>
        )}
        <span className="mt-3 inline-flex items-center gap-1.5 text-[13px] font-medium">
          {t("news.read")}
          <ArrowRight className="h-3.5 w-3.5" />
        </span>
      </div>
    </Link>
  );
}

function FeedRow({ item }: { item: NewsItem }) {
  return (
    <Link
      href={`/news/${item.slug}`}
      className="group flex items-start gap-4 rounded-2xl border border-border bg-card px-5 py-[18px] transition-colors hover:border-[var(--vx-panel-hover-line)]"
    >
      <NewsDateBlock value={item.published_at} />

      {item.image ? (
        <img
          src={item.image}
          alt=""
          className="hidden h-[68px] w-[104px] flex-none rounded-lg border border-border bg-muted object-cover sm:block"
        />
      ) : (
        <span className="hidden h-[68px] w-[104px] flex-none items-center justify-center rounded-lg border border-dashed border-input bg-muted/40 text-muted-foreground sm:flex">
          <Newspaper className="h-4 w-4" />
        </span>
      )}

      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <NewsTag tag={item.tag} />
          {item.unread && (
            <span className="flex items-center gap-1.5 text-[11.5px] text-[var(--vx-warn)]">
              <span className="h-1.5 w-1.5 rounded-full bg-[var(--vx-warn)]" />
              {t("news.new")}
            </span>
          )}
          <span className="text-[11.5px] text-muted-foreground/70">
            {readingTime(item.body, item.excerpt)}
          </span>
        </div>
        <h3 className="mt-1.5 text-[15.5px] font-semibold">{item.title}</h3>
        {item.excerpt && (
          <p className="mt-1 line-clamp-2 text-[12.5px] leading-relaxed text-muted-foreground">
            {item.excerpt}
          </p>
        )}
      </div>

      <ArrowRight className="mt-1 hidden h-4 w-4 flex-none text-muted-foreground transition-colors group-hover:text-foreground md:block" />
    </Link>
  );
}

export function NewsPageContent() {
  // Карточки ленты зовут t() напрямую, здесь хук нужен ради подписки:
  // без него страница не перерисуется при смене языка.
  useT();
  const qc = useQueryClient();
  const [tag, setTag] = useState("all");
  const [query, setQuery] = useState("");

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.newsList,
    queryFn: fetchNews,
  });
  const items = useMemo(() => data?.news ?? [], [data]);

  const readMut = useMutation({
    mutationFn: markNewsRead,
    onSuccess: () => {
      toast.success(t("news.marked"));
      void qc.invalidateQueries({ queryKey: queryKeys.newsList });
    },
    onError: (e: Error) => toast.error(e.message || t("news.mark_failed")),
  });

  const unread = items.filter((n) => n.unread).length;

  // Счётчики считаем по всей ленте, а не по отфильтрованной: иначе, выбрав
  // рубрику, видишь единицу у всех остальных.
  const counts = useMemo(() => {
    const by: Record<string, number> = { all: items.length };
    for (const n of items) {
      const key = n.tag || "";
      by[key] = (by[key] ?? 0) + 1;
    }
    return by;
  }, [items]);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return items.filter(
      (n) =>
        (tag === "all" || n.tag === tag) &&
        (!q ||
          n.title.toLowerCase().includes(q) ||
          (n.excerpt ?? "").toLowerCase().includes(q))
    );
  }, [items, tag, query]);

  // Закреплённую показываем отдельной карточкой, но только когда она прошла
  // фильтр: иначе она висела бы наверху при поиске, которому не отвечает.
  const pinned = visible.find((n) => n.pinned);
  const rest = pinned ? visible.filter((n) => n !== pinned) : visible;

  const filters = [
    { id: "all", label: t("common.all") },
    ...NEWS_TAGS.filter((tag) => (counts[tag.id] ?? 0) > 0),
  ];

  if (isLoading) {
    return (
      <PageShell variant="user">
        <div className="flex flex-col gap-[22px]">
          <Skeleton className="h-12 w-64 rounded-xl" />
          <Skeleton className="h-11 rounded-xl" />
          <Skeleton className="h-44 rounded-2xl" />
          <Skeleton className="h-28 rounded-2xl" />
          <Skeleton className="h-28 rounded-2xl" />
        </div>
      </PageShell>
    );
  }

  return (
    <PageShell variant="user">
      <div className="flex w-full flex-col gap-[22px]">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div className="flex flex-col gap-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-[-0.02em]">
              {t("news.title")}
            </h1>
            <p className="text-[13.5px] text-muted-foreground">
              {t("news.subtitle")}
            </p>
          </div>
          {unread > 0 && (
            <button
              type="button"
              onClick={() => readMut.mutate()}
              disabled={readMut.isPending}
              className={cn(btnGhost, "h-[38px] text-[13px]")}
            >
              <CheckCheck className="h-3.5 w-3.5" />
              {readMut.isPending
                ? t("news.marking")
                : t("news.mark_read", { count: unread })}
            </button>
          )}
        </div>

        {isError && (
          <p className="text-sm text-destructive">{t("news.load_failed")}</p>
        )}

        {items.length > 0 && (
          <div className="flex flex-wrap items-center gap-2">
            <div className="no-scrollbar flex gap-1 overflow-x-auto rounded-xl border border-border bg-card p-1">
              {filters.map((f) => (
                <button
                  key={f.id}
                  type="button"
                  onClick={() => setTag(f.id)}
                  aria-pressed={tag === f.id}
                  className={cn(
                    "flex h-9 flex-none items-center gap-2 rounded-[9px] px-3.5 text-[13px] whitespace-nowrap transition-colors",
                    tag === f.id
                      ? "bg-accent font-medium text-accent-foreground"
                      : "text-muted-foreground hover:text-foreground"
                  )}
                >
                  {f.label}
                  <span className="font-mono text-[11.5px] text-muted-foreground/70">
                    {counts[f.id] ?? 0}
                  </span>
                </button>
              ))}
            </div>

            <label className="relative ms-auto min-w-[220px] flex-1 sm:max-w-[300px] sm:flex-none">
              <Search className="pointer-events-none absolute start-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <input
                type="search"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder={t("news.search_placeholder")}
                className={cn(fieldClass, "ps-9")}
              />
            </label>
          </div>
        )}

        {pinned && <PinnedCard item={pinned} />}

        <div className="flex flex-col gap-2.5">
          {rest.map((item) => (
            <FeedRow key={item.slug} item={item} />
          ))}
        </div>

        {visible.length === 0 && (
          <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed border-input bg-card px-5 py-12 text-center">
            <span className="flex h-[38px] w-[38px] items-center justify-center rounded-[10px] bg-muted text-muted-foreground">
              {items.length === 0 ? (
                <Newspaper className="h-4 w-4" />
              ) : (
                <Search className="h-4 w-4" />
              )}
            </span>
            {items.length === 0 ? (
              <>
                <div className="text-sm font-medium">{t("news.empty_title")}</div>
                <div className="text-[12.5px] text-muted-foreground">
                  {t("news.empty_text")}
                </div>
              </>
            ) : (
              <>
                <div className="text-sm font-medium">{t("common.not_found")}</div>
                <button
                  type="button"
                  onClick={() => {
                    setTag("all");
                    setQuery("");
                  }}
                  className={cn(btnPrimary, "mt-1 h-[34px]")}
                >
                  {t("news.reset_filters")}
                </button>
              </>
            )}
          </div>
        )}
      </div>
    </PageShell>
  );
}
