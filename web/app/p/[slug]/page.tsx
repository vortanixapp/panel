import type { Metadata } from "next";
import { cookies } from "next/headers";
import { notFound, redirect } from "next/navigation";
import { CustomPageView } from "@/components/site/custom-page-view";
import { USER_COOKIE } from "@/lib/accounts";
import { loadRequestI18n } from "@/lib/i18n-server";
import { loadServerSite, viewerFromCookie } from "@/lib/site-server";
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
  const [page, { i18n }] = await Promise.all([findPage(slug), loadRequestI18n()]);
  if (!page) return {};
  const title = localized(page.title, i18n.locale, i18n.default_locale) || slug;
  const description = localized(page.description, i18n.locale, i18n.default_locale);
  return description ? { title, description } : { title };
}

export default async function CustomPage({ params, searchParams }: Props) {
  const { slug } = await params;
  const preview = (await searchParams)["vx-edit"] !== undefined;
  const page = await findPage(slug);
  if (!preview) {
    if (!page) notFound();
    const { loggedIn } = viewerFromCookie((await cookies()).get(USER_COOKIE)?.value);
    const audience = page.layout === "panel" ? "users" : (page.audience ?? "");
    if (audience === "users" && !loggedIn) redirect("/login");
    if (audience === "guests" && loggedIn) redirect("/dashboard");
  }
  return <CustomPageView slug={slug} initial={page} preview={preview} />;
}
