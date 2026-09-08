"use client";

import { localeTag, t } from "@/lib/i18n";
import { cn } from "@/lib/utils";

/**
 * Рубрики новостей и мелочи ленты.
 *
 * Набор рубрик задаёт владелец панели, поэтому неизвестное значение здесь не
 * ошибка: показываем его как есть нейтральным цветом, а не прячем новость и не
 * подменяем чужую рубрику своей.
 */

// Раскладка читается на уровне модуля, поэтому храним ключи: готовые подписи
// застыли бы на языке, который стоял в момент загрузки страницы.
const TAGS: Record<string, { labelKey: string; token: string }> = {
  update: { labelKey: "news.tag.update", token: "--vx-info" },
  promo: { labelKey: "news.tag.promo", token: "--vx-ok" },
  maintenance: { labelKey: "news.tag.maintenance", token: "--vx-warn" },
  platform: { labelKey: "news.tag.platform", token: "--vx-violet" },
};

// label — вычисляемое свойство, а не строка: форму {id, label, token} читает
// и админский список рубрик, а язык может смениться после загрузки.
export const NEWS_TAGS = Object.entries(TAGS).map(([id, meta]) => ({
  id,
  token: meta.token,
  get label() {
    return t(meta.labelKey);
  },
}));

export function newsTagMeta(tag: string | undefined) {
  if (!tag) return { label: t("news.tag.none"), token: "--vx-ink-ghost" };
  const meta = TAGS[tag];
  if (!meta) return { label: tag, token: "--vx-ink-ghost" };
  return { label: t(meta.labelKey), token: meta.token };
}

export function NewsTag({ tag, className }: { tag: string | undefined; className?: string }) {
  const meta = newsTagMeta(tag);
  return (
    <span
      className={cn(
        "rounded-full px-[9px] py-[3px] text-[11.5px] whitespace-nowrap",
        className
      )}
      style={{
        color: `var(${meta.token})`,
        background: `color-mix(in srgb, var(${meta.token}) 12%, transparent)`,
      }}
    >
      {meta.label}
    </span>
  );
}

/** Дата отдельным блоком: в ленте по ней ведут взглядом, и день крупнее месяца
 *  читается быстрее строки «24 августа 2026». */
export function NewsDateBlock({ value }: { value: string | null | undefined }) {
  const d = value ? new Date(value) : null;
  if (!d || Number.isNaN(d.getTime())) {
    return (
      <div className="w-[46px] flex-none text-center">
        <div className="text-[11px] text-muted-foreground">—</div>
      </div>
    );
  }
  return (
    <div className="w-[46px] flex-none text-center">
      <div className="font-mono text-[22px] leading-none font-bold tracking-[-0.02em]">
        {String(d.getDate()).padStart(2, "0")}
      </div>
      <div className="mt-1 text-[11px] text-muted-foreground">
        {d.toLocaleDateString(localeTag(), { month: "long" })}
      </div>
    </div>
  );
}

/** Сколько читать. Считаем по тексту, а не храним: цифра, выставленная руками,
 *  расходится с текстом на первой же правке. 900 знаков в минуту — средняя
 *  скорость чтения по-русски. */
export function readingTime(body: string | null | undefined, excerpt?: string | null): string {
  const text = String(body || excerpt || "").replace(/<[^>]+>/g, " ");
  const minutes = Math.max(1, Math.round(text.trim().length / 900));
  return t("news.reading_time", { minutes });
}

export function formatNewsFull(value: string | null | undefined): string {
  if (!value) return t("news.no_date");
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return t("news.no_date");
  return d.toLocaleString(localeTag(), {
    day: "numeric",
    month: "long",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

/** Заголовки статьи для оглавления.
 *
 * Берём их из самого текста, а не из отдельного поля: оглавление, которое
 * ведётся вручную, расходится со статьёй молча. Идентификаторы проставляем
 * здесь же, чтобы ссылки вели в нужное место.
 */
export function extractHeadings(html: string): {
  html: string;
  toc: { id: string; title: string }[];
} {
  const toc: { id: string; title: string }[] = [];
  let index = 0;

  const withIds = html.replace(
    /<h([23])(\s[^>]*)?>([\s\S]*?)<\/h\1>/gi,
    (_m, level: string, attrs: string | undefined, inner: string) => {
      const title = inner.replace(/<[^>]+>/g, "").trim();
      if (!title) return _m;
      const id = `sec-${++index}`;
      toc.push({ id, title });
      return `<h${level}${attrs ?? ""} id="${id}">${inner}</h${level}>`;
    }
  );

  return { html: withIds, toc };
}
