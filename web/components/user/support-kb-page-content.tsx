"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeft,
  ArrowRight,
  BookOpen,
  ChevronDown,
  CreditCard,
  Eye,
  Globe,
  HelpCircle,
  KeyRound,
  MessageSquarePlus,
  Search,
  Server,
  type LucideIcon,
} from "lucide-react";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchKBArticles, fetchSupportCreateForm, type KBArticle } from "@/lib/api";
import { t } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { btnGhost, btnPrimary, fieldClass } from "@/components/user/panel-parts";
import { FALLBACK_DEPARTMENTS } from "@/components/user/support/support-parts";

/** Значок раздела. Набор отделов настраивается для каждой панели, поэтому для
 *  незнакомого берём нейтральный: своя иконка у чужого раздела была бы
 *  случайной, а её отсутствие ломало бы ряд карточек. */
const SECTION_ICONS: Record<string, LucideIcon> = {
  technical: Server,
  hosting: Globe,
  billing: CreditCard,
  account: KeyRound,
  abuse: KeyRound,
};

function sectionIcon(id: string): LucideIcon {
  return SECTION_ICONS[id] ?? BookOpen;
}

function plural(n: number): string {
  const mod10 = n % 10;
  const mod100 = n % 100;
  if (mod10 === 1 && mod100 !== 11) return t("support.kb.articles_one", { n });
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) {
    return t("support.kb.articles_few", { n });
  }
  return t("support.kb.articles_many", { n });
}

/** Частый вопрос: заголовок статьи с раскрывающейся выжимкой.
 *
 * Список строится по числу открытий, а не ведётся руками: отсортированный
 * вручную он устаревает молча, и «частым» остаётся то, что таким быть перестало.
 */
function FaqRow({
  article,
  open,
  onToggle,
}: {
  article: KBArticle;
  open: boolean;
  onToggle: () => void;
}) {
  return (
    <div className="border-b border-border/60 last:border-0">
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={open}
        className="flex w-full items-center gap-3 py-3.5 text-start"
      >
        <span className="flex-1 text-[13.5px] font-medium">{article.title}</span>
        <ChevronDown
          className={cn(
            "h-4 w-4 flex-none text-muted-foreground transition-transform",
            open && "rotate-180"
          )}
        />
      </button>
      {open && (
        <div className="pb-4">
          <p className="text-[13px] leading-relaxed text-muted-foreground">
            {article.excerpt || t("support.kb.excerpt_empty")}
          </p>
          <Link
            href={`/support/kb/${article.slug}`}
            className="mt-2.5 inline-flex items-center gap-1.5 text-[12.5px] transition-colors hover:text-foreground"
          >
            {t("support.kb.read_article")}
            <ArrowRight className="h-3.5 w-3.5" />
          </Link>
        </div>
      )}
    </div>
  );
}

export function SupportKBPageContent() {
  // Помощники модуля зовут t() напрямую, здесь хук нужен ради подписки:
  // без него страница не перерисуется при смене языка.
  useT();
  const [query, setQuery] = useState("");
  const [section, setSection] = useState("");
  const [openFaq, setOpenFaq] = useState<string | null>(null);

  const { data: form } = useQuery({
    queryKey: queryKeys.supportCreateForm,
    queryFn: fetchSupportCreateForm,
  });
  const departments = form?.departments?.length ? form.departments : FALLBACK_DEPARTMENTS;

  // Всю базу тянем одним запросом: разделы показывают счётчики, а считать их
  // на сервере отдельной ручкой ради десятка статей незачем.
  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.kbArticles(undefined, ""),
    queryFn: () => fetchKBArticles(),
  });
  const all = useMemo(() => data?.articles ?? [], [data]);

  const counts = useMemo(() => {
    const by: Record<string, number> = {};
    for (const a of all) by[a.category] = (by[a.category] ?? 0) + 1;
    return by;
  }, [all]);

  const sections = departments.filter((d) => (counts[d.id] ?? 0) > 0);

  const q = query.trim().toLowerCase();
  const found = useMemo(
    () =>
      all.filter(
        (a) =>
          (!section || a.category === section) &&
          (!q ||
            a.title.toLowerCase().includes(q) ||
            (a.excerpt ?? "").toLowerCase().includes(q))
      ),
    [all, section, q]
  );

  // Частые вопросы показываем только на общем экране: при поиске или выбранном
  // разделе человек ищет конкретное, и подборка «вообще популярного» ему мешает.
  const browsing = !q && !section;
  const faq = useMemo(
    () => [...all].sort((a, b) => b.views - a.views).slice(0, 5),
    [all]
  );

  return (
    <PageShell variant="user">
      <div className="flex w-full flex-col gap-[22px]">
        <div className="flex flex-wrap items-center gap-[14px]">
          <Link
            href="/support"
            className="flex h-[34px] w-[34px] flex-none items-center justify-center rounded-[9px] border border-input text-foreground transition-colors hover:bg-accent"
            aria-label={t("support.aria.back_to_tickets")}
          >
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div className="flex flex-col gap-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-[-0.02em]">
              {t("support.kb.title")}
            </h1>
            <p className="text-[13.5px] text-muted-foreground">
              {t("support.kb.subtitle")}
            </p>
          </div>
          <Link
            href="/support/create"
            className={cn(btnPrimary, "ms-auto h-[38px] px-[18px]")}
          >
            <MessageSquarePlus className="h-[15px] w-[15px]" />
            {t("support.list.create")}
          </Link>
        </div>

        <label className="relative block">
          <Search className="pointer-events-none absolute start-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t("support.kb.search_placeholder")}
            className={cn(fieldClass, "h-[46px] ps-9 pe-28 text-sm")}
          />
          {all.length > 0 && (
            <span className="pointer-events-none absolute end-3 top-1/2 -translate-y-1/2 text-[12px] text-muted-foreground/70">
              {plural(all.length)}
            </span>
          )}
        </label>

        {isError && (
          <p className="text-sm text-destructive">{t("support.kb.load_failed")}</p>
        )}

        {isLoading ? (
          <div className="flex flex-col gap-2.5">
            <Skeleton className="h-24 rounded-2xl" />
            <Skeleton className="h-40 rounded-2xl" />
          </div>
        ) : all.length === 0 ? (
          <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed border-input bg-card px-5 py-12 text-center">
            <span className="flex h-[38px] w-[38px] items-center justify-center rounded-[10px] bg-muted text-muted-foreground">
              <BookOpen className="h-4 w-4" />
            </span>
            <div className="text-sm font-medium">{t("support.kb.empty_title")}</div>
            <div className="max-w-md text-[12.5px] text-muted-foreground">
              {t("support.kb.empty_text")}
            </div>
            <Link href="/support/create" className={cn(btnGhost, "mt-1 h-[34px]")}>
              {t("support.kb.write_support")}
            </Link>
          </div>
        ) : (
          <>
            {browsing && sections.length > 0 && (
              <div className="grid grid-cols-[repeat(auto-fit,minmax(230px,1fr))] gap-2.5">
                {sections.map((d) => {
                  const Icon = sectionIcon(d.id);
                  return (
                    <button
                      key={d.id}
                      type="button"
                      onClick={() => setSection(d.id)}
                      className="flex flex-col gap-2 rounded-2xl border border-border bg-card px-5 py-4 text-start transition-colors hover:border-[var(--vx-panel-hover-line)]"
                    >
                      <span className="flex h-[34px] w-[34px] items-center justify-center rounded-[10px] border border-border bg-muted text-foreground">
                        <Icon className="h-4 w-4" />
                      </span>
                      <span className="text-[14px] font-medium">{d.name}</span>
                      <span className="text-[12px] text-muted-foreground">
                        {plural(counts[d.id] ?? 0)}
                      </span>
                    </button>
                  );
                })}
              </div>
            )}

            {!browsing && (
              <div className="flex flex-wrap items-center gap-2">
                <button
                  type="button"
                  onClick={() => {
                    setSection("");
                    setQuery("");
                  }}
                  className={cn(btnGhost, "h-[34px]")}
                >
                  <ArrowLeft className="h-3.5 w-3.5" />
                  {t("support.kb.all_sections")}
                </button>
                <span className="text-[12.5px] text-muted-foreground">
                  {found.length > 0
                    ? t("support.kb.found", { items: plural(found.length) })
                    : t("common.not_found")}
                </span>
              </div>
            )}

            {!browsing && (
              <div className="flex flex-col gap-2.5">
                {found.map((a) => (
                  <Link
                    key={a.id}
                    href={`/support/kb/${a.slug}`}
                    className="flex flex-col gap-2 rounded-2xl border border-border bg-card px-5 py-[18px] transition-colors hover:border-[var(--vx-panel-hover-line)]"
                  >
                    <div className="flex flex-wrap items-center gap-[10px]">
                      <span className="text-[15.5px] font-semibold">{a.title}</span>
                      <span className="ms-auto flex items-center gap-1.5 text-[11.5px] text-muted-foreground/70">
                        <Eye className="h-3.5 w-3.5" />
                        {a.views}
                      </span>
                    </div>
                    {a.excerpt && (
                      <p className="text-[12.5px] leading-relaxed text-muted-foreground">
                        {a.excerpt}
                      </p>
                    )}
                  </Link>
                ))}

                {found.length === 0 && (
                  <div className="rounded-2xl border border-dashed border-input bg-card px-5 py-10 text-center">
                    <p className="text-sm font-medium">{t("common.not_found")}</p>
                    <p className="mt-1 text-[12.5px] text-muted-foreground">
                      {t("support.kb.not_found_text")}
                    </p>
                    <Link href="/support/create" className={cn(btnPrimary, "mt-3 h-[34px]")}>
                      {t("support.kb.write_support")}
                    </Link>
                  </div>
                )}
              </div>
            )}

            {browsing && faq.length > 0 && (
              <div className="flex flex-col gap-1 rounded-2xl border border-border bg-card px-[22px] py-5">
                <div className="flex items-center gap-2 pb-1.5 text-[15px] font-semibold">
                  <HelpCircle className="h-4 w-4 text-muted-foreground" />
                  {t("support.kb.faq_title")}
                </div>
                {faq.map((a) => (
                  <FaqRow
                    key={a.id}
                    article={a}
                    open={openFaq === a.id}
                    onToggle={() => setOpenFaq(openFaq === a.id ? null : a.id)}
                  />
                ))}
                <div className="mt-3 flex flex-wrap items-center gap-3 border-t border-border pt-4">
                  <span className="text-[13px] text-muted-foreground">
                    {t("support.kb.no_answer")}
                  </span>
                  <Link href="/support/create" className={cn(btnGhost, "h-[34px]")}>
                    <MessageSquarePlus className="h-3.5 w-3.5" />
                    {t("support.kb.write_support")}
                  </Link>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </PageShell>
  );
}
