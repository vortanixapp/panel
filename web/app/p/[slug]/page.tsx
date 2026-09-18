import type { Metadata } from "next";
import { cookies } from "next/headers";
import { notFound } from "next/navigation";
import { CustomPageView } from "@/components/site/custom-page-view";
import { LOCALE_COOKIE_NAME } from "@/lib/i18n";
import { loadServerI18n } from "@/lib/i18n-server";
import { loadServerSite } from "@/lib/site-server";
import { localized } from "@/lib/site/text";

type Props = {
  params: Promise<{ slug: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
};

async function findPage(slug: string) {
  const find = (site: Awaited<ReturnType<typeof loadServerSite>>) =>
    site.custom_pages?.find((page) => page.slug === slug && !page.hidden);
  return find(await loadServerSite()) ?? find(await loadServerSite(true));
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { slug } = await params;
  const store = await cookies();
  const [page, i18n] = await Promise.all([
    findPage(slug),
    loadServerI18n(store.get(LOCALE_COOKIE_NAME)?.value),
  ]);
  if (!page) return {};
  const title = localized(page.title, i18n.locale, i18n.default_locale) || slug;
  const description = localized(page.description, i18n.locale, i18n.default_locale);
  return description ? { title, description } : { title };
}

export default async function CustomPage({ params, searchParams }: Props) {
  const { slug } = await params;
  const query = await searchParams;
  if (query["vx-edit"] === undefined && !(await findPage(slug))) notFound();
  return <CustomPageView slug={slug} />;
}
