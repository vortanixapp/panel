import type { TranslateFn } from "@/lib/i18n";
import { LANDING_SECTIONS } from "@/lib/site/defaults";
import type { BlockProps, LText, SiteBlock } from "@/lib/site/types";

export type FieldSpec =
  | { name: string; kind: "text" | "long" | "html"; label: string }
  | { name: string; kind: "url" | "image" | "video" | "icon"; label: string }
  | { name: string; kind: "bool"; label: string }
  | { name: string; kind: "enum"; label: string; options: { value: string; label: string }[] }
  | { name: string; kind: "int"; label: string; min: number; max: number }
  | { name: string; kind: "list"; label: string; max: number; item: FieldSpec[]; itemLabel: string; titleField: string };

export type BlockGroup = "builtin" | "text" | "media" | "marketing" | "service";

export type BlockMeta = {
  type: string;
  group: BlockGroup;
  titleKey: string;
  descriptionKey: string;
  icon: string;
  fields: FieldSpec[];
  site: boolean;
  panel: boolean;
  defaults?: (tr: TranslateFn, locale: string) => BlockProps;
};

const ALIGN: FieldSpec = {
  name: "align",
  kind: "enum",
  label: "template.field.align",
  options: [
    { value: "left", label: "template.option.left" },
    { value: "center", label: "template.option.center" },
  ],
};

function lt(locale: string, text: string): LText {
  return { [locale]: text };
}

const BUILTIN_ICONS: Record<string, string> = {
  hero: "rocket",
  games: "gamepad-2",
  steps: "list-checks",
  hardware: "cpu",
  panel: "terminal",
  locations: "map-pin",
  pricing: "tag",
  faq: "circle-help",
  cta: "megaphone",
};

const BUILTIN: BlockMeta[] = LANDING_SECTIONS.map((name) => ({
  type: `landing.${name}`,
  group: "builtin",
  titleKey: `template.block.landing.${name}`,
  descriptionKey: `template.block.landing.${name}_hint`,
  icon: BUILTIN_ICONS[name] ?? "layout-grid",
  fields: [],
  site: true,
  panel: false,
}));

const CONTENT: BlockMeta[] = [
  {
    type: "heading",
    group: "text",
    titleKey: "template.block.heading",
    descriptionKey: "template.block.heading_hint",
    icon: "hash",
    site: true,
    panel: true,
    fields: [
      { name: "eyebrow", kind: "text", label: "template.field.eyebrow" },
      { name: "title", kind: "text", label: "template.field.title" },
      { name: "text", kind: "long", label: "template.field.text" },
      ALIGN,
      {
        name: "size",
        kind: "enum",
        label: "template.field.size",
        options: [
          { value: "md", label: "template.option.md" },
          { value: "lg", label: "template.option.lg" },
          { value: "xl", label: "template.option.xl" },
        ],
      },
    ],
    defaults: (tr, locale) => ({
      title: lt(locale, tr("template.defaults.heading_title")),
      text: lt(locale, tr("template.defaults.heading_text")),
      size: "lg",
    }),
  },
  {
    type: "text",
    group: "text",
    titleKey: "template.block.text",
    descriptionKey: "template.block.text_hint",
    icon: "file-text",
    site: true,
    panel: true,
    fields: [
      { name: "html", kind: "html", label: "template.field.content" },
      {
        name: "width",
        kind: "enum",
        label: "template.field.width",
        options: [
          { value: "narrow", label: "template.option.narrow" },
          { value: "wide", label: "template.option.wide" },
        ],
      },
      ALIGN,
    ],
    defaults: (tr, locale) => ({
      html: lt(locale, `<p>${tr("template.defaults.text_body")}</p>`),
    }),
  },
  {
    type: "image",
    group: "media",
    titleKey: "template.block.image",
    descriptionKey: "template.block.image_hint",
    icon: "image",
    site: true,
    panel: true,
    fields: [
      { name: "src", kind: "image", label: "template.field.image" },
      { name: "alt", kind: "text", label: "template.field.alt" },
      { name: "caption", kind: "text", label: "template.field.caption" },
      { name: "url", kind: "url", label: "template.field.link" },
      { name: "new_tab", kind: "bool", label: "template.field.new_tab" },
      {
        name: "width",
        kind: "enum",
        label: "template.field.width",
        options: [
          { value: "narrow", label: "template.option.narrow" },
          { value: "wide", label: "template.option.wide" },
          { value: "full", label: "template.option.full" },
        ],
      },
      { name: "rounded", kind: "bool", label: "template.field.rounded" },
    ],
    defaults: () => ({ width: "wide", rounded: true }),
  },
  {
    type: "media",
    group: "media",
    titleKey: "template.block.media",
    descriptionKey: "template.block.media_hint",
    icon: "layout-grid",
    site: true,
    panel: true,
    fields: [
      { name: "src", kind: "image", label: "template.field.image" },
      { name: "alt", kind: "text", label: "template.field.alt" },
      { name: "eyebrow", kind: "text", label: "template.field.eyebrow" },
      { name: "title", kind: "text", label: "template.field.title" },
      { name: "text", kind: "long", label: "template.field.text" },
      { name: "button", kind: "text", label: "template.field.button" },
      { name: "url", kind: "url", label: "template.field.button_link" },
      {
        name: "side",
        kind: "enum",
        label: "template.field.image_side",
        options: [
          { value: "left", label: "template.option.left" },
          { value: "right", label: "template.option.right" },
        ],
      },
    ],
    defaults: (tr, locale) => ({
      title: lt(locale, tr("template.defaults.media_title")),
      text: lt(locale, tr("template.defaults.media_text")),
      button: lt(locale, tr("template.defaults.more")),
      url: "/register",
    }),
  },
  {
    type: "features",
    group: "marketing",
    titleKey: "template.block.features",
    descriptionKey: "template.block.features_hint",
    icon: "sparkles",
    site: true,
    panel: true,
    fields: [
      { name: "title", kind: "text", label: "template.field.title" },
      { name: "text", kind: "long", label: "template.field.text" },
      { name: "columns", kind: "int", label: "template.field.columns", min: 2, max: 4 },
      {
        name: "items",
        kind: "list",
        label: "template.field.cards",
        itemLabel: "template.field.card",
        titleField: "title",
        max: 24,
        item: [
          { name: "icon", kind: "icon", label: "template.field.icon" },
          { name: "title", kind: "text", label: "template.field.title" },
          { name: "text", kind: "long", label: "template.field.text" },
          { name: "url", kind: "url", label: "template.field.link" },
        ],
      },
    ],
    defaults: (tr, locale) => ({
      title: lt(locale, tr("template.defaults.features_title")),
      columns: 3,
      items: [
        { icon: "cpu", title: lt(locale, tr("template.defaults.feature1_title")), text: lt(locale, tr("template.defaults.feature1_text")) },
        { icon: "shield-check", title: lt(locale, tr("template.defaults.feature2_title")), text: lt(locale, tr("template.defaults.feature2_text")) },
        { icon: "zap", title: lt(locale, tr("template.defaults.feature3_title")), text: lt(locale, tr("template.defaults.feature3_text")) },
      ],
    }),
  },
  {
    type: "cta",
    group: "marketing",
    titleKey: "template.block.cta",
    descriptionKey: "template.block.cta_hint",
    icon: "megaphone",
    site: true,
    panel: true,
    fields: [
      { name: "title", kind: "text", label: "template.field.title" },
      { name: "text", kind: "long", label: "template.field.text" },
      { name: "button", kind: "text", label: "template.field.button" },
      { name: "url", kind: "url", label: "template.field.button_link" },
      { name: "button2", kind: "text", label: "template.field.button2" },
      { name: "url2", kind: "url", label: "template.field.button2_link" },
      {
        name: "tone",
        kind: "enum",
        label: "template.field.tone",
        options: [
          { value: "accent", label: "template.option.accent" },
          { value: "muted", label: "template.option.muted" },
        ],
      },
    ],
    defaults: (tr, locale) => ({
      title: lt(locale, tr("template.defaults.cta_title")),
      text: lt(locale, tr("template.defaults.cta_text")),
      button: lt(locale, tr("template.defaults.cta_button")),
      url: "/register",
    }),
  },
  {
    type: "faq",
    group: "marketing",
    titleKey: "template.block.faq",
    descriptionKey: "template.block.faq_hint",
    icon: "circle-help",
    site: true,
    panel: true,
    fields: [
      { name: "title", kind: "text", label: "template.field.title" },
      { name: "text", kind: "long", label: "template.field.text" },
      {
        name: "items",
        kind: "list",
        label: "template.field.questions",
        itemLabel: "template.field.question",
        titleField: "q",
        max: 40,
        item: [
          { name: "q", kind: "text", label: "template.field.question" },
          { name: "a", kind: "long", label: "template.field.answer" },
        ],
      },
    ],
    defaults: (tr, locale) => ({
      title: lt(locale, tr("template.defaults.faq_title")),
      items: [
        { q: lt(locale, tr("template.defaults.faq1_q")), a: lt(locale, tr("template.defaults.faq1_a")) },
        { q: lt(locale, tr("template.defaults.faq2_q")), a: lt(locale, tr("template.defaults.faq2_a")) },
      ],
    }),
  },
  {
    type: "stats",
    group: "marketing",
    titleKey: "template.block.stats",
    descriptionKey: "template.block.stats_hint",
    icon: "chart-bar",
    site: true,
    panel: true,
    fields: [
      { name: "title", kind: "text", label: "template.field.title" },
      {
        name: "items",
        kind: "list",
        label: "template.field.figures",
        itemLabel: "template.field.figure",
        titleField: "value",
        max: 8,
        item: [
          { name: "value", kind: "text", label: "template.field.value" },
          { name: "label", kind: "text", label: "template.field.label" },
        ],
      },
    ],
    defaults: (tr, locale) => ({
      items: [
        { value: lt(locale, "99.9%"), label: lt(locale, tr("template.defaults.stat1")) },
        { value: lt(locale, "40+"), label: lt(locale, tr("template.defaults.stat2")) },
        { value: lt(locale, "60 s"), label: lt(locale, tr("template.defaults.stat3")) },
        { value: lt(locale, "24/7"), label: lt(locale, tr("template.defaults.stat4")) },
      ],
    }),
  },
  {
    type: "notice",
    group: "service",
    titleKey: "template.block.notice",
    descriptionKey: "template.block.notice_hint",
    icon: "bell",
    site: true,
    panel: true,
    fields: [
      {
        name: "tone",
        kind: "enum",
        label: "template.field.tone",
        options: [
          { value: "info", label: "template.option.info" },
          { value: "success", label: "template.option.success" },
          { value: "warning", label: "template.option.warning" },
          { value: "danger", label: "template.option.danger" },
        ],
      },
      { name: "title", kind: "text", label: "template.field.title" },
      { name: "text", kind: "long", label: "template.field.text" },
      { name: "button", kind: "text", label: "template.field.link_text" },
      { name: "url", kind: "url", label: "template.field.link" },
      { name: "dismissible", kind: "bool", label: "template.field.dismissible" },
    ],
    defaults: (tr, locale) => ({
      tone: "info",
      title: lt(locale, tr("template.defaults.notice_title")),
      text: lt(locale, tr("template.defaults.notice_text")),
    }),
  },
  {
    type: "video",
    group: "media",
    titleKey: "template.block.video",
    descriptionKey: "template.block.video_hint",
    icon: "play",
    site: true,
    panel: true,
    fields: [
      { name: "url", kind: "video", label: "template.field.video" },
      { name: "title", kind: "text", label: "template.field.title" },
      { name: "caption", kind: "text", label: "template.field.caption" },
    ],
  },
  {
    type: "html",
    group: "service",
    titleKey: "template.block.html",
    descriptionKey: "template.block.html_hint",
    icon: "code",
    site: true,
    panel: true,
    fields: [
      { name: "html", kind: "html", label: "template.field.html" },
      { name: "scripts", kind: "bool", label: "template.field.scripts" },
    ],
  },
  {
    type: "spacer",
    group: "service",
    titleKey: "template.block.spacer",
    descriptionKey: "template.block.spacer_hint",
    icon: "sliders-horizontal",
    site: true,
    panel: true,
    fields: [
      {
        name: "size",
        kind: "enum",
        label: "template.field.size",
        options: [
          { value: "sm", label: "template.option.sm" },
          { value: "md", label: "template.option.md" },
          { value: "lg", label: "template.option.lg" },
          { value: "xl", label: "template.option.xl" },
        ],
      },
      { name: "line", kind: "bool", label: "template.field.line" },
    ],
    defaults: () => ({ size: "md" }),
  },
];

export const BLOCK_TYPES: BlockMeta[] = [...BUILTIN, ...CONTENT];

export const BLOCK_GROUPS: BlockGroup[] = ["builtin", "text", "media", "marketing", "service"];

const BY_TYPE = new Map(BLOCK_TYPES.map((meta) => [meta.type, meta]));

export function blockMeta(type: string): BlockMeta | undefined {
  return BY_TYPE.get(type);
}

export function isBuiltinBlock(type: string): boolean {
  return type.startsWith("landing.");
}

export function newId(prefix: string): string {
  const random =
    typeof crypto !== "undefined" && "getRandomValues" in crypto
      ? Array.from(crypto.getRandomValues(new Uint8Array(6)), (b) => b.toString(36).padStart(2, "0")).join("")
      : Math.random().toString(36).slice(2, 12);
  return `${prefix}-${random.slice(0, 10)}`;
}

export function createBlock(type: string, tr: TranslateFn, locale: string): SiteBlock {
  if (isBuiltinBlock(type)) return { id: type.slice("landing.".length), type };
  const meta = blockMeta(type);
  const props = meta?.defaults?.(tr, locale);
  return props ? { id: newId("b"), type, props } : { id: newId("b"), type };
}

export function cloneBlock(block: SiteBlock): SiteBlock {
  return { ...structuredClone(block), id: newId("b") };
}
