export const SERVER_A = "e6fca27e-ed8d-4176-b158-81c92805fec5";
export const SERVER_B = "5b0d9d7e-6c53-4c2b-9d3c-0b6f2f0f7a11";

export type MockServer = {
  id: string;
  name: string;
  ip_address: string;
  port: number;
  status: string;
  runtime_status: string;
  provisioning_status: string;
  created_at: string;
  user_id: string;
  expires_at: string;
  game_id: string;
  game: { name: string; slug: string; image: null };
  location: { name: string; country: string; city: string };
  tariff: Record<string, unknown>;
  limits: Record<string, unknown>;
  uptime: string;
  disk_used_mb: number;
  disk_total_mb: number;
  auto_renew: boolean;
  available_game_versions: { id: string; name: string }[];
  game_version_id: string;
  viewer_permissions: Record<string, unknown>;
  online_players: number;
  max_players: number;
  current_map: string;
  players_online: { name: string }[];
  cpu_percent: number;
  ram_percent: number;
};

function inDays(days: number) {
  return new Date(Date.now() + days * 86_400_000).toISOString();
}

export function makeServer(overrides: Partial<MockServer> = {}): MockServer {
  return {
    id: SERVER_A,
    name: "Survival Craft",
    ip_address: "45.93.200.222",
    port: 25565,
    status: "active",
    runtime_status: "running",
    provisioning_status: "ready",
    created_at: "2026-10-04T20:57:00Z",
    user_id: "u1",
    expires_at: inDays(25),
    game_id: "mc-java",
    game: { name: "Minecraft Java", slug: "mc-java", image: null },
    location: { name: "Россия", country: "RU", city: "Москва" },
    tariff: {
      id: "t1",
      name: "Оптимальный",
      ram_mb: 2048,
      disk_mb: 3072,
      slots: 20,
      billing_type: "resources",
      renewal_periods: [15, 30, 60],
    },
    limits: { memory_mb: 2048, cpu: 2, slots: 20 },
    uptime: "1д 00ч 11м",
    disk_used_mb: 450,
    disk_total_mb: 3072,
    auto_renew: true,
    available_game_versions: [{ id: "v1", name: "1.21.4" }],
    game_version_id: "v1",
    viewer_permissions: {},
    online_players: 3,
    max_players: 20,
    current_map: "world",
    players_online: [{ name: "Steve" }, { name: "Alex" }, { name: "Notch" }],
    cpu_percent: 12,
    ram_percent: 58,
    ...overrides,
  };
}

export function makeSecondServer(): MockServer {
  return makeServer({
    id: SERVER_B,
    name: "Creative Hub",
    port: 25566,
    status: "stopped",
    runtime_status: "stopped",
    online_players: 0,
    players_online: [],
    cpu_percent: 0,
    ram_percent: 0,
    expires_at: inDays(2),
  });
}

export function metricPoints(count = 60) {
  const now = Math.floor(Date.now() / 1000);
  return Array.from({ length: count }, (_, i) => ({
    ts: now - (count - i) * 60,
    cpu_pct: 20 + 15 * Math.sin(i / 5),
    mem_used_mb: 1400 + 80 * Math.sin(i / 7),
    mem_limit_mb: 2048,
  }));
}

export function spendingDays() {
  return Array.from({ length: 30 }, (_, i) => ({
    date: new Date(Date.now() - (29 - i) * 86_400_000).toISOString(),
    debit: i === 20 ? 210 : 0,
    credit: 0,
  }));
}
