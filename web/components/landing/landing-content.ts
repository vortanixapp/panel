import type { TranslateFn } from "@/lib/i18n";

export type HeroGame = {
  id: string;
  name: string;
  server: string;
  build: string;
  port: number;
  slots: number;
  ramGb: number;
  price: number;
  image: string;
  log: string[];
};

export const HERO_GAMES: HeroGame[] = [
  {
    id: "minecraft",
    name: "Minecraft",
    server: "survival",
    build: "Paper 1.21.4",
    port: 25565,
    slots: 20,
    ramGb: 4,
    price: 199,
    image: "/landing/games/minecraft.svg",
    log: [
      "Starting minecraft server version 1.21.4",
      "Preparing level \"world\"",
      "Preparing spawn area: 100%",
      "Done (3.84s)! For help, type \"help\"",
    ],
  },
  {
    id: "cs2",
    name: "Counter-Strike 2",
    server: "retake-mirage",
    build: "CS2 · MetaMod",
    port: 27015,
    slots: 12,
    ramGb: 2,
    price: 349,
    image: "/landing/games/cs2.svg",
    log: [
      "Loading map \"de_mirage\"",
      "MetaMod:Source 2.0 loaded 4 plugins",
      "VAC secure mode is activated.",
      "Connection to Steam servers successful.",
    ],
  },
  {
    id: "rust",
    name: "Rust",
    server: "vanilla-weekly",
    build: "Rust · Oxide",
    port: 28015,
    slots: 100,
    ramGb: 12,
    price: 690,
    image: "/landing/games/rust.svg",
    log: [
      "Loading Prefab Bundles",
      "Generating procedural map, size 3500",
      "Oxide loaded 11 plugins",
      "Server startup complete",
    ],
  },
  {
    id: "valheim",
    name: "Valheim",
    server: "midgard",
    build: "Valheim · BepInEx",
    port: 2456,
    slots: 10,
    ramGb: 4,
    price: 399,
    image: "/landing/games/valheim.svg",
    log: [
      "Loading world \"Midgard\"",
      "BepInEx loaded 6 mods",
      "Game server connected",
      "Session \"Midgard\" is ready",
    ],
  },
  {
    id: "palworld",
    name: "Palworld",
    server: "pal-island",
    build: "Palworld",
    port: 8211,
    slots: 32,
    ramGb: 16,
    price: 590,
    image: "/landing/games/palworld.svg",
    log: [
      "Loading save data",
      "World settings applied",
      "REST API listening on 8212",
      "Running Palworld dedicated server",
    ],
  },
];

export const FEATURED_GAMES = [
  { name: "Minecraft", note: "Paper · Forge · Fabric", from: 199, image: "/landing/games/minecraft.svg" },
  { name: "Counter-Strike 2", note: "MetaMod · CounterStrikeSharp", from: 349, image: "/landing/games/cs2.svg" },
  { name: "Rust", note: "Oxide · Carbon", from: 690, image: "/landing/games/rust.svg" },
  { name: "ARK: Survival Evolved", note: "Steam Workshop", from: 790, image: "/landing/games/ark.svg" },
  { name: "Valheim", note: "BepInEx · ValheimPlus", from: 399, image: "/landing/games/valheim.svg" },
  { name: "Palworld", note: "PalGuard", from: 590, image: "/landing/games/palworld.svg" },
] as const;

export const MORE_GAMES = [
  { name: "DayZ", from: 690 },
  { name: "Garry's Mod", from: 249 },
  { name: "Terraria", from: 149 },
  { name: "FiveM", from: 890 },
  { name: "Team Fortress 2", from: 249 },
  { name: "Project Zomboid", from: 349 },
] as const;

export const TPS_SERIES: readonly number[] = [
  19.96, 19.97, 19.95, 19.98, 19.97, 19.96, 19.94, 19.95, 19.93, 19.91, 19.9, 19.88,
  19.9, 19.87, 19.85, 19.86, 19.82, 19.76, 19.52, 19.31, 19.44, 19.63, 19.84, 19.93,
];

export const PANEL_LOG = [
  { time: "18:02:16", text: "[Server] Done (4.102s)! For help, type \"help\"", tone: "ok" },
  { time: "18:07:44", text: "[Auth] Nikita_QQ joined the game", tone: "muted" },
  { time: "18:09:02", text: "[Warn] Can't keep up! Skipped 12 ticks", tone: "warn" },
  { time: "18:11:38", text: "[Backup] Snapshot complete · 3.1 GB", tone: "muted" },
  { time: "18:14:20", text: "[Auth] mrGrief tried /op — denied", tone: "danger" },
  { time: "18:16:03", text: "[Sched] Restart warning sent to players", tone: "muted" },
] as const;

export const PANEL_FILES = [
  { name: "world/", size: "2.8 GB", dir: true },
  { name: "plugins/", size: "146 MB", dir: true },
  { name: "logs/", size: "38 MB", dir: true },
  { name: "server.properties", size: "1.4 KB", dir: false },
  { name: "bukkit.yml", size: "3.2 KB", dir: false },
  { name: "ops.json", size: "212 B", dir: false },
] as const;

export function landingSteps(t: TranslateFn) {
  return [1, 2, 3].map((n) => ({
    n: `0${n}`,
    title: t(`landing.steps.s${n}_title`),
    body: t(`landing.steps.s${n}_body`),
    meta: t(`landing.steps.s${n}_meta`),
  }));
}

export function landingSpecs(t: TranslateFn) {
  return ["cpu", "disk", "network", "scale"].map((key) => ({
    key,
    label: t(`landing.hardware.${key}_label`),
    value: t(`landing.hardware.${key}_value`),
    note: t(`landing.hardware.${key}_note`),
  }));
}

export function landingPanelTabs(t: TranslateFn) {
  return [
    { id: "console", label: t("landing.panel.tab_console") },
    { id: "files", label: t("landing.panel.tab_files") },
    { id: "backups", label: t("landing.panel.tab_backups") },
    { id: "schedule", label: t("landing.panel.tab_schedule") },
  ] as const;
}

export function landingPanelPoints(t: TranslateFn) {
  return [1, 2, 3, 4].map((n) => t(`landing.panel.point${n}`));
}

export function landingLocations(t: TranslateFn) {
  const rows = [
    { key: "moscow", ms: 4, gbit: 10, load: 62 },
    { key: "spb", ms: 9, gbit: 10, load: 48 },
    { key: "frankfurt", ms: 28, gbit: 40, load: 71 },
    { key: "amsterdam", ms: 34, gbit: 40, load: 55 },
    { key: "warsaw", ms: 22, gbit: 20, load: 89 },
    { key: "new_york", ms: 96, gbit: 40, load: 41 },
    { key: "singapore", ms: 148, gbit: 20, load: 33 },
  ];
  return rows.map((row) => ({
    ...row,
    city: t(`landing.locations.${row.key}`),
    busy: row.load > 85,
  }));
}

export function landingFaq(t: TranslateFn) {
  return [1, 2, 3, 4, 5].map((n) => ({
    q: t(`landing.faq.q${n}`),
    a: t(`landing.faq.a${n}`),
  }));
}

export function landingPricing(t: TranslateFn) {
  const plans = [
    { id: "start", monthly: 199, hourly: 0.42, highlighted: false },
    { id: "community", monthly: 690, hourly: 1.44, highlighted: true },
    { id: "project", monthly: 1890, hourly: 3.94, highlighted: false },
    { id: "cluster", monthly: 5900, hourly: 12.29, highlighted: false },
  ];
  return plans.map((plan) => ({
    ...plan,
    name: t(`landing.pricing.${plan.id}_name`),
    audience: t(`landing.pricing.${plan.id}_audience`),
    specs: [1, 2, 3, 4, 5].map((n) => t(`landing.pricing.${plan.id}_spec${n}`)),
  }));
}
