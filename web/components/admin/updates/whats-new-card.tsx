"use client";

import { useState } from "react";
import { ChevronDown, ExternalLink } from "lucide-react";
import { SettingsCard } from "@/components/admin/settings/settings-ui";
import { Badge } from "@/components/ui/badge";
import type { AdminUpdates, UpdateRelease } from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";
import { notesTitle, parseNotes, ReleaseNotes } from "./release-notes";
import { formatDay } from "./update-utils";

type Item = { release: UpdateRelease; latest: boolean; installed: boolean };

function collectItems(data: AdminUpdates): { items: Item[]; pending: boolean } {
  const releases = data.releases ?? [];
  if (data.update_available && releases.length > 0) {
    return {
      pending: true,
      items: releases.map((release, index) => ({ release, latest: index === 0, installed: false })),
    };
  }
  if (data.installed_release) {
    return {
      pending: false,
      items: [{ release: data.installed_release, latest: !data.update_available, installed: true }],
    };
  }
  if (data.latest_version && data.notes) {
    return {
      pending: data.update_available,
      items: [
        {
          release: {
            version: data.latest_version,
            notes: data.notes,
            url: data.release_url ?? "",
            published_at: data.published_at ?? "",
            prerelease: data.prerelease === true,
          },
          latest: true,
          installed: !data.update_available,
        },
      ],
    };
  }
  return { pending: false, items: [] };
}

export function WhatsNewCard({ data }: { data: AdminUpdates }) {
  const t = useT();
  const { items, pending } = collectItems(data);

  const description = pending
    ? t("admin.updates.news.pending", { version: data.current_version || "—" })
    : items.length > 0
      ? t("admin.updates.news.installed", { version: items[0].release.version })
      : undefined;

  return (
    <SettingsCard title={t("admin.updates.news.title")} description={description}>
      {items.length === 0 ? (
        <p className="text-[13px] text-muted-foreground">{t("admin.updates.news.empty")}</p>
      ) : (
        <ol>
          {items.map((item, index) => (
            <ReleaseItem
              key={item.release.version}
              item={item}
              defaultOpen={index === 0}
              last={index === items.length - 1}
            />
          ))}
        </ol>
      )}
      {data.releases_limited && (
        <p className="mt-5 text-[12px] text-muted-foreground">
          {t("admin.updates.news.limited", { count: items.length })}
        </p>
      )}
    </SettingsCard>
  );
}

function ReleaseItem({ item, defaultOpen, last }: { item: Item; defaultOpen: boolean; last: boolean }) {
  const t = useT();
  const [open, setOpen] = useState(defaultOpen);
  const { release, latest, installed } = item;
  const title = notesTitle(release.notes);
  const blocks = parseNotes(release.notes);
  const hasBody = blocks.length > (title ? 1 : 0);

  return (
    <li className={cn("relative flex gap-4", !last && "pb-7")}>
      {!last && <span className="absolute top-5 bottom-0 left-[7px] w-px bg-border" />}
      <span
        className={cn(
          "relative mt-1 size-[15px] shrink-0 rounded-full border-2 bg-card",
          latest ? "border-primary" : installed ? "border-emerald-500" : "border-muted-foreground/40"
        )}
      />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-x-2.5 gap-y-1">
          <span className="font-mono text-[14px] font-semibold">{release.version}</span>
          {latest && !installed && <Badge>{t("admin.updates.news.latest")}</Badge>}
          {installed && (
            <Badge variant="secondary" className="text-emerald-600 dark:text-emerald-400">
              {t("admin.updates.news.installed_badge")}
            </Badge>
          )}
          {release.published_at && (
            <span className="text-[12.5px] text-muted-foreground">{formatDay(release.published_at)}</span>
          )}
          {release.url && (
            <a
              href={release.url}
              target="_blank"
              rel="noreferrer noopener"
              aria-label={t("admin.updates.hero.github")}
              className="text-muted-foreground hover:text-foreground"
            >
              <ExternalLink className="size-3.5" />
            </a>
          )}
        </div>
        {title ? (
          <p className="mt-1.5 text-[14px] leading-snug font-medium">{title}</p>
        ) : (
          !hasBody && <p className="mt-1.5 text-[13px] text-muted-foreground">{t("admin.updates.news.no_notes")}</p>
        )}
        {hasBody && open && <ReleaseNotes text={release.notes} skipTitle={Boolean(title)} className="mt-2.5" />}
        {hasBody && (
          <button
            type="button"
            onClick={() => setOpen((v) => !v)}
            aria-expanded={open}
            className="mt-2 inline-flex items-center gap-1 text-[12.5px] text-muted-foreground hover:text-foreground"
          >
            {open ? t("admin.updates.news.less") : t("admin.updates.news.more")}
            <ChevronDown className={cn("size-3.5 transition-transform", open && "rotate-180")} />
          </button>
        )}
      </div>
    </li>
  );
}
