import type { TranslateFn } from "@/lib/i18n";

// Подписи лендинга собирают функции landingXxx(t) в конце файла: константа
// с готовым текстом застыла бы на языке, который стоял при загрузке страницы.
// Здесь остались только данные без подписей — их переводить нечего.
export const LANDING_DEPLOY_LOG = [
  { t: "00,0", text: "$ vortanix create --game minecraft --ram 8G", color: "text-primary" },
  { t: "02,4", text: "node FRA-2 selected · 4 cores pinned", color: "text-muted-foreground" },
  { t: "08,1", text: "volume nvme-gen4 attached · 80 GB", color: "text-muted-foreground" },
  { t: "14,7", text: "anti-ddos profile applied · L3-L7", color: "text-muted-foreground" },
  { t: "22,9", text: "paper 1.21.4 downloaded · aikar flags set", color: "text-muted-foreground" },
  { t: "36,5", text: "world generated · 4 096 chunks", color: "text-muted-foreground" },
  { t: "40,2", text: "server online · 203.0.113.10:25565", color: "text-primary" },
] as const;

export const LANDING_FEATURED_GAMES = [
  { name: "Minecraft", tag: "MC", from: 199, hue: 122 },
  { name: "CS2", tag: "CS2", from: 349, hue: 32 },
  { name: "Rust", tag: "RST", from: 690, hue: 14 },
  { name: "ARK: SE", tag: "ARK", from: 790, hue: 190 },
  { name: "Valheim", tag: "VH", from: 399, hue: 212 },
  { name: "Palworld", tag: "PAL", from: 590, hue: 268 },
] as const;

export const LANDING_MORE_GAMES = [
  { name: "DayZ", tag: "DZ", from: 690 },
  { name: "Garry's Mod", tag: "GM", from: 249 },
  { name: "Terraria", tag: "TER", from: 149 },
  { name: "FiveM", tag: "5M", from: 890 },
  { name: "Team Fortress 2", tag: "TF2", from: 249 },
  { name: "Project Zomboid", tag: "PZ", from: 349 },
] as const;

export const LANDING_TPS_BARS: readonly number[] = (() => {
  const dips: Record<number, number> = { 18: 0.62, 19: 0.55, 20: 0.6, 21: 0.72 };
  const out: number[] = [];
  for (let i = 0; i < 24; i++) {
    const k = dips[i] ?? 0.86 + ((i * 13) % 11) / 100;
    out.push(Math.round(k * 100));
  }
  return out;
})();

export const LANDING_PANEL_PREVIEW_LOG = [
  { time: "18:02:16", text: '[Server] Done (4,102s)! For help, type "help"', color: "text-primary" },
  { time: "18:07:44", text: "[Auth] Player Nikita_QQ joined the game", color: "text-muted-foreground" },
  { time: "18:09:02", text: "[Warn] Skipped 12 ticks (chunk gen)", color: "text-amber-300" },
  { time: "18:11:38", text: "[Backup] Snapshot 06h complete · 3,1 ГБ", color: "text-muted-foreground/70" },
  { time: "18:14:20", text: "[Auth] mrGrief tried /op — denied", color: "text-red-400" },
  { time: "18:16:03", text: "[Sched] restart-warning broadcast sent", color: "text-muted-foreground/70" },
] as const;

// Ниже — блоки с подписями. Они собираются функцией в момент отрисовки:
// константа на уровне модуля посчиталась бы один раз, и язык лендинга застыл
// бы на том, что стоял при загрузке страницы. Текст живёт в словаре landing.*.

export function landingNav(t: TranslateFn) {
  return [
    { label: t("landing.nav.pricing"), href: "#pricing" },
    { label: t("landing.nav.games"), href: "/games" },
    { label: t("landing.nav.features"), href: "/features" },
    { label: t("landing.nav.faq"), href: "#faq" },
  ];
}

export function landingHeroBadge(t: TranslateFn) {
  return t("landing.hero.badge");
}

export function landingHeroGauges(t: TranslateFn) {
  return [
    {
      label: t("landing.hero.gauge_tps_label"),
      value: t("landing.hero.gauge_tps_value"),
      pct: 99,
    },
    {
      label: t("landing.hero.gauge_ping_label"),
      value: t("landing.hero.gauge_ping_value"),
      pct: 22,
    },
    {
      label: t("landing.hero.gauge_uptime_label"),
      value: t("landing.hero.gauge_uptime_value"),
      pct: 99,
    },
  ];
}

export function landingHeroStats(t: TranslateFn) {
  return [
    {
      value: t("landing.hero.stat_deploy_value"),
      label: t("landing.hero.stat_deploy_label"),
    },
    {
      value: t("landing.hero.stat_uptime_value"),
      label: t("landing.hero.stat_uptime_label"),
    },
    {
      value: t("landing.hero.stat_support_value"),
      label: t("landing.hero.stat_support_label"),
    },
    {
      value: t("landing.hero.stat_ddos_value"),
      label: t("landing.hero.stat_ddos_label"),
    },
  ];
}

export function landingGamesTitle(t: TranslateFn) {
  return t("landing.games.title");
}

export function landingSteps(t: TranslateFn) {
  return [
    {
      n: "01",
      title: t("landing.step1.title"),
      body: t("landing.step1.body"),
      meta: t("landing.step1.meta"),
    },
    {
      n: "02",
      title: t("landing.step2.title"),
      body: t("landing.step2.body"),
      meta: t("landing.step2.meta"),
    },
    {
      n: "03",
      title: t("landing.step3.title"),
      body: t("landing.step3.body"),
      meta: t("landing.step3.meta"),
    },
  ];
}

export function landingBento(t: TranslateFn) {
  return [
    {
      kicker: t("landing.bento.cpu_kicker"),
      title: t("landing.bento.cpu_title"),
      body: t("landing.bento.cpu_body"),
      metric: t("landing.bento.cpu_metric"),
    },
    {
      kicker: t("landing.bento.disk_kicker"),
      title: t("landing.bento.disk_title"),
      body: t("landing.bento.disk_body"),
      metric: t("landing.bento.disk_metric"),
    },
    {
      kicker: t("landing.bento.network_kicker"),
      title: t("landing.bento.network_title"),
      body: t("landing.bento.network_body"),
      metric: t("landing.bento.network_metric"),
    },
    {
      kicker: t("landing.bento.scale_kicker"),
      title: t("landing.bento.scale_title"),
      body: t("landing.bento.scale_body"),
      metric: t("landing.bento.scale_metric"),
    },
  ];
}

export function landingPanelPoints(t: TranslateFn) {
  return [
    t("landing.panel.point_console"),
    t("landing.panel.point_sftp"),
    t("landing.panel.point_roles"),
    t("landing.panel.point_api"),
  ];
}

export function landingPanelMetrics(t: TranslateFn) {
  return [
    {
      label: t("landing.panel.metric_cpu_label"),
      value: t("landing.panel.metric_cpu_value"),
      pct: 38,
    },
    {
      label: t("landing.panel.metric_ram_label"),
      value: t("landing.panel.metric_ram_value"),
      pct: 68,
    },
    {
      label: t("landing.panel.metric_tps_label"),
      value: t("landing.panel.metric_tps_value"),
      pct: 99,
    },
    {
      label: t("landing.panel.metric_players_label"),
      value: t("landing.panel.metric_players_value"),
      pct: 39,
    },
  ];
}

export function landingLocations(t: TranslateFn) {
  const available = t("landing.location.status_available");
  const lowSlots = t("landing.location.status_low_slots");
  const rows: {
    cityKey: string;
    ms: number;
    gbit: number;
    load: string;
    pct: number;
    ok: boolean;
  }[] = [
    { cityKey: "landing.location.moscow", ms: 4, gbit: 10, load: "62%", pct: 62, ok: true },
    { cityKey: "landing.location.spb", ms: 9, gbit: 10, load: "48%", pct: 48, ok: true },
    { cityKey: "landing.location.frankfurt", ms: 28, gbit: 40, load: "71%", pct: 71, ok: true },
    { cityKey: "landing.location.amsterdam", ms: 34, gbit: 40, load: "55%", pct: 55, ok: true },
    { cityKey: "landing.location.warsaw", ms: 22, gbit: 20, load: "89%", pct: 89, ok: false },
    { cityKey: "landing.location.new_york", ms: 96, gbit: 40, load: "41%", pct: 41, ok: true },
    { cityKey: "landing.location.singapore", ms: 148, gbit: 20, load: "33%", pct: 33, ok: true },
  ];
  return rows.map((row) => ({
    city: t(row.cityKey),
    ping: t("landing.location.ping", { ms: row.ms }),
    uplink: t("landing.location.uplink", { gbit: row.gbit }),
    load: row.load,
    pct: row.pct,
    status: row.ok ? available : lowSlots,
    ok: row.ok,
  }));
}

export function landingFaq(t: TranslateFn) {
  return [
    { q: t("landing.faq1.q"), a: t("landing.faq1.a") },
    { q: t("landing.faq2.q"), a: t("landing.faq2.a") },
    { q: t("landing.faq3.q"), a: t("landing.faq3.a") },
    { q: t("landing.faq4.q"), a: t("landing.faq4.a") },
    { q: t("landing.faq5.q"), a: t("landing.faq5.a") },
  ];
}

export function landingPricing(t: TranslateFn) {
  return [
    {
      id: "start",
      name: t("landing.pricing.start_name"),
      badge: t("landing.pricing.start_badge"),
      price: "199",
      hourly: "0,42",
      highlighted: false,
      specs: [
        t("landing.pricing.start_spec1"),
        t("landing.pricing.start_spec2"),
        t("landing.pricing.start_spec3"),
        t("landing.pricing.start_spec4"),
        t("landing.pricing.start_spec5"),
      ],
    },
    {
      id: "community",
      name: t("landing.pricing.community_name"),
      badge: t("landing.pricing.community_badge"),
      price: "690",
      hourly: "1,44",
      highlighted: true,
      specs: [
        t("landing.pricing.community_spec1"),
        t("landing.pricing.community_spec2"),
        t("landing.pricing.community_spec3"),
        t("landing.pricing.community_spec4"),
        t("landing.pricing.community_spec5"),
      ],
    },
    {
      id: "project",
      name: t("landing.pricing.project_name"),
      badge: t("landing.pricing.project_badge"),
      price: "1890",
      hourly: "3,94",
      highlighted: false,
      specs: [
        t("landing.pricing.project_spec1"),
        t("landing.pricing.project_spec2"),
        t("landing.pricing.project_spec3"),
        t("landing.pricing.project_spec4"),
        t("landing.pricing.project_spec5"),
      ],
    },
    {
      id: "cluster",
      name: t("landing.pricing.cluster_name"),
      badge: t("landing.pricing.cluster_badge"),
      price: "5900",
      hourly: "12,29",
      highlighted: false,
      specs: [
        t("landing.pricing.cluster_spec1"),
        t("landing.pricing.cluster_spec2"),
        t("landing.pricing.cluster_spec3"),
        t("landing.pricing.cluster_spec4"),
        t("landing.pricing.cluster_spec5"),
      ],
    },
  ];
}

export function landingFooterCols(t: TranslateFn) {
  return [
    {
      title: t("landing.footer.product"),
      links: [
        { label: t("landing.nav.pricing"), href: "#pricing" },
        { label: t("landing.nav.games"), href: "/games" },
        { label: t("landing.nav.features"), href: "/features" },
        { label: t("landing.footer.status"), href: "/status" },
      ],
    },
    {
      title: t("landing.footer.company"),
      links: [
        { label: t("landing.footer.about"), href: "/about" },
        { label: t("landing.footer.blog"), href: "/blog" },
        { label: t("landing.footer.support"), href: "/support" },
      ],
    },
    {
      title: t("landing.footer.resources"),
      links: [
        { label: t("landing.footer.kb"), href: "/kb" },
        { label: t("landing.nav.faq"), href: "#faq" },
        { label: t("landing.footer.api"), href: "/kb" },
      ],
    },
    {
      title: t("landing.footer.account"),
      links: [
        { label: t("landing.footer.sign_in"), href: "/login" },
        { label: t("landing.footer.register"), href: "/register" },
      ],
    },
  ];
}
