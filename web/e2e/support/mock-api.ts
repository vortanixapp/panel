import type { BrowserContext, Page, Route } from "@playwright/test";
import {
  makeSecondServer,
  makeServer,
  metricPoints,
  spendingDays,
  type MockServer,
} from "./data";

export type Req = {
  method: string;
  path: string;
  query: URLSearchParams;
  body: Record<string, unknown>;
};

export type Reply = { status?: number; json?: unknown };
export type Handler = (req: Req) => Reply | undefined;

export type Call = { method: string; path: string; body: Record<string, unknown> };

export type MockUser = { id: string; email: string; role: string; name: string; password: string };

const ALLOWED_PATHS = /^\/(v1)\//;

function claimsCookie(user: MockUser, expiresInSec: number) {
  const claims = {
    user_id: user.id,
    email: user.email,
    role: user.role,
    exp: Math.floor(Date.now() / 1000) + expiresInSec,
  };
  return Buffer.from(JSON.stringify(claims)).toString("base64url");
}

export class MockApi {
  readonly calls: Call[] = [];
  readonly misses: string[] = [];
  readonly hits = new Map<string, number>();
  readonly user: MockUser = {
    id: "u1",
    email: "pupkin@example.test",
    role: "user",
    name: "Pupkin",
    password: "correct-horse",
  };
  servers: MockServer[] = [makeServer(), makeSecondServer()];
  balance = 500;
  powerFails = false;
  refreshFails = false;
  private delays = new Map<string, number>();
  stream: string | null = null;
  streams: string[] | null = null;
  readonly streamRequests: string[] = [];
  private extra: Handler[] = [];
  private payments = new Map<string, { amount: number; polls: number; credited?: boolean }>();

  constructor(private readonly context: BrowserContext) {}

  use(handler: Handler) {
    this.extra.unshift(handler);
  }

  delay(path: string, ms: number) {
    this.delays.set(path, ms);
  }

  hitCount(path: string) {
    return this.hits.get(path) ?? 0;
  }

  server(id: string) {
    return this.servers.find((s) => s.id === id);
  }

  called(method: string, path: RegExp | string) {
    return this.calls.filter(
      (c) => c.method === method && (typeof path === "string" ? c.path === path : path.test(c.path))
    );
  }

  async signIn(expiresInSec = 3600 * 12) {
    await this.context.addCookies([
      {
        name: "vtx_user",
        value: claimsCookie(this.user, expiresInSec),
        url: this.baseURL(),
      },
    ]);
  }

  private baseURL() {
    return process.env.E2E_BASE_URL ?? `http://localhost:${process.env.E2E_PORT ?? 3100}`;
  }

  async install(page: Page) {
    await page.route(
      (url) => ALLOWED_PATHS.test(url.pathname) && url.origin === new URL(this.baseURL()).origin,
      (route) => void this.handle(route)
    );
  }

  private async handle(route: Route) {
    const request = route.request();
    const url = new URL(request.url());
    if (url.pathname === "/v1/notifications/stream") {
      const attempt = this.streamRequests.length;
      this.streamRequests.push(request.headers()["last-event-id"] ?? "");
      if (this.streams) {
        await route.fulfill({
          status: 200,
          headers: { "content-type": "text/event-stream", "cache-control": "no-store" },
          body: this.streams[Math.min(attempt, this.streams.length - 1)],
        });
        return;
      }
      if (this.stream === null) {
        await route.abort("connectionrefused");
        return;
      }
      await route.fulfill({
        status: 200,
        headers: { "content-type": "text/event-stream", "cache-control": "no-store" },
        body: this.stream,
      });
      return;
    }

    let body: Record<string, unknown> = {};
    const raw = request.postData();
    if (raw) {
      try {
        body = JSON.parse(raw) as Record<string, unknown>;
      } catch {
        body = {};
      }
    }
    const req: Req = { method: request.method(), path: url.pathname, query: url.searchParams, body };
    this.hits.set(req.path, (this.hits.get(req.path) ?? 0) + 1);
    if (req.method !== "GET") this.calls.push({ method: req.method, path: req.path, body });

    for (const handler of this.extra) {
      const reply = handler(req);
      if (reply) return this.send(route, reply);
    }
    const wait = this.delays.get(req.path) ?? 0;
    if (wait > 0) await new Promise((resolve) => setTimeout(resolve, wait));
    if (this.powerFails && req.path.endsWith("/power") && req.method === "POST") {
      return this.send(route, { status: 500, json: { error: "Нода недоступна" } });
    }
    const reply = this.respond(req);
    if (reply) return this.send(route, reply);

    this.misses.push(`${req.method} ${req.path}`);
    return this.send(route, { status: 404, json: { error: "mock: not implemented" } });
  }

  private async send(route: Route, reply: Reply) {
    await route.fulfill({
      status: reply.status ?? 200,
      contentType: "application/json",
      body: JSON.stringify(reply.json ?? {}),
    });
  }

  private dashboard() {
    const running = this.servers.filter((s) => s.runtime_status === "running");
    return {
      balance: this.balance,
      balance_currency: "RUB",
      wallets: [],
      total_servers: this.servers.length,
      active_servers: running.length,
      expiring_soon_count: this.servers.filter((s) => Date.parse(s.expires_at) - Date.now() < 3 * 86_400_000).length,
      open_support_tickets_count: 0,
      next_charge_text: "",
      monthly_spend: 210,
      next_renewal: {
        server_id: this.servers[0]?.id,
        name: this.servers[0]?.name,
        expires_at: this.servers[0]?.expires_at,
        auto_renew: true,
        period_days: 30,
        cost: 210,
      },
      spending: { currency: "RUB", debit: 210, credit: 300, days: spendingDays() },
      recent_servers: this.servers,
      recent_transactions: [
        { id: "1", type: "debit", amount: 210, description: "Аренда сервера", created_at: new Date(Date.now() - 86_400_000).toISOString() },
        { id: "2", type: "credit", amount: 250, description: "Пополнение картой", created_at: new Date(Date.now() - 3 * 86_400_000).toISOString() },
      ],
      news: [
        { id: "n1", slug: "new-locations", title: "Новые локации в Нидерландах", excerpt: "", published_at: "2026-10-08T10:00:00Z", image: null },
      ],
    };
  }

  private serverStatus(server: MockServer) {
    return {
      ok: true,
      online: server.runtime_status === "running",
      status: server.status,
      runtime_status: server.runtime_status,
      provisioning_status: server.provisioning_status,
      max_players: server.max_players,
      online_players: server.online_players,
      players_online: server.players_online,
      current_map: server.current_map,
      answered: server.runtime_status === "running",
      uptime: server.uptime,
      disk_used_mb: server.disk_used_mb,
      disk_total_mb: server.disk_total_mb,
    };
  }

  private power(server: MockServer, action: string) {
    if (action === "start" || action === "restart") {
      server.status = "running";
      server.runtime_status = "running";
    } else if (action === "stop" || action === "kill") {
      server.status = "stopped";
      server.runtime_status = "stopped";
      server.online_players = 0;
      server.players_online = [];
    }
  }

  private respond(req: Req): Reply | undefined {
    const { method, path, body } = req;

    if (path === "/v1/tenants/status") return { json: { bootstrapped: true } };
    if (path === "/v1/branding") return { json: {} };
    if (path === "/v1/auth/social/providers") return { json: { ok: true, providers: [] } };
    if (path === "/v1/auth/accounts") return { json: { accounts: [], current_id: this.user.id } };
    if (path === "/v1/notifications/unread-count") return { json: { count: 0 } };
    if (path === "/v1/account/legal") return { json: { pending: [] } };

    if (path === "/v1/auth/login" && method === "POST") {
      if (body.email !== this.user.email || body.password !== this.user.password) {
        return { status: 401, json: { error: "Неверный email или пароль" } };
      }
      void this.signIn();
      return {
        json: {
          access_token: "e2e-access",
          user: { id: this.user.id, email: this.user.email, role: this.user.role },
          start_page: "/dashboard",
        },
      };
    }
    if (path === "/v1/auth/logout") return { json: { status: "ok" } };
    if (path === "/v1/auth/refresh") {
      if (this.refreshFails) return { status: 401, json: { error: "session expired" } };
      return { json: { access_token: "e2e-access" } };
    }

    if (path === "/v1/me") {
      return {
        json: {
          user_id: this.user.id,
          email: this.user.email,
          role: this.user.role,
          tenant_id: "t",
          tenant_slug: "default",
          display_name: this.user.name,
        },
      };
    }
    if (path === "/v1/account") {
      return {
        json: {
          user: {
            id: this.user.id,
            email: this.user.email,
            timezone: "Europe/Moscow",
            locale: "ru",
            first_name: this.user.name,
            email_verified: true,
          },
        },
      };
    }

    if (path === "/v1/dashboard") return { json: this.dashboard() };
    if (path === "/v1/my-servers" || path === "/v1/servers") {
      return {
        json: {
          servers: this.servers,
          total: this.servers.length,
          active_count: this.servers.filter((s) => s.runtime_status === "running").length,
          expiring_soon: 1,
        },
      };
    }
    if (path === "/v1/daily-bonus") {
      return { json: { can_spin: false, streak: 3, prizes_disabled: false, next_spin_at: null, prizes: [] } };
    }
    if (path === "/v1/account/referrals") {
      return {
        json: { enabled: true, code: "abc", percent: 10, months: 0, min_payment: 0, currency: "RUB", stats: { invited: 0, paid: 0, earned: [] }, referrals: [] },
      };
    }
    if (path === "/v1/projects" && method === "GET") return { json: { projects: [], unassigned_servers: 0 } };
    if (path === "/v1/trial" && method === "GET") return { json: { available: false, enabled: false, hours: 0, next_at: null } };
    if (path === "/v1/games") {
      return { json: { games: [{ id: "mc-java", slug: "mc-java", name: "Minecraft Java", description: null, image: null, active: true }, { id: "samp", slug: "samp", name: "SA-MP", description: null, image: null, active: true }] } };
    }
    if (path === "/v1/tariffs/public") return { json: { tariffs: [] } };
    if (path === "/v1/rent-server" && method === "GET") {
      const catalog = {
        games: [
          { id: "mc-java", name: "Minecraft Java", slug: "mc-java", min_price: 150, servers_count: 12 },
          { id: "samp", name: "SA-MP", slug: "samp", min_price: 90, servers_count: 4 },
        ],
        tariffs: [
          { id: "t-start", name: "Старт", game_id: "mc-java", price_monthly: 150, currency: "RUB", slots: 10, ram_mb: 1024, disk_mb: 2048, billing_type: "fixed", payment_mode: "prepaid", rental_periods: [30], base_price_monthly: 150, price_from: 150 },
          { id: "t-opt", name: "Оптимальный", game_id: "mc-java", price_monthly: 250, currency: "RUB", slots: 20, ram_mb: 2048, disk_mb: 3072, billing_type: "fixed", payment_mode: "prepaid", rental_periods: [30], base_price_monthly: 250, price_from: 250 },
        ],
        nodes: [{ id: "node-msk", name: "Москва", country: "RU", code: "msk", is_online: true, servers_count: 5 }],
        game_versions: [{ id: "v1", game_id: "mc-java", game_slug: "mc-java", name: "1.21.4", version: "1.21.4", source_type: "archive" }],
      };
      if (req.query.toString()) {
        return { json: { ...catalog, calculated_cost: 250, base_cost: 250, total_cost: 250, currency: "RUB", payment_mode: "prepaid", monthly_cost: 250, breakdown: [] } };
      }
      return { json: catalog };
    }
    if (path === "/v1/rent-server" && method === "POST") {
      const id = "9f1c2d3e-0000-4000-8000-00000000abcd";
      this.balance -= 250;
      return { json: { id, server_id: id, status: "provisioning", count: 1, servers: [{ id, name: String(body.name), cost: 250 }] } };
    }
    if (path === "/v1/account/identification") {
      return { json: { required: false, identified: false, identified_at: null, method: "", methods: [] } };
    }
    if (path === "/v1/billing/promo-codes") return { json: { promo_codes: [] } };
    if (path === "/v1/billing/refund-requests") {
      return { json: { requests: [], wallets: [{ id: "w1", currency: "RUB", balance: this.balance }] } };
    }
    if (path === "/v1/billing/documents") {
      return {
        json: {
          payer: { user_id: this.user.id, email: this.user.email, person_name: this.user.name, payer_type: "individual" },
          currency: "RUB",
          currencies: ["RUB"],
          months: [],
          company: { name: "", inn: "", tax_system: "", ready: false },
        },
      };
    }
    if (path === "/v1/billing/topup" && method === "GET") {
      const wallet = { id: "w1", currency: "RUB", balance: this.balance, is_default: true };
      return {
        json: {
          wallets: [wallet],
          selected_wallet: wallet,
          payments: [],
          providers: [{ id: "prov-test", code: "testpay", name: "Тестовый шлюз", fee_percent: 0, currency: "RUB" }],
          enabled_providers: ["testpay"],
        },
      };
    }
    if (path === "/v1/billing/topup/create" && method === "POST") {
      const id = `pay-${this.payments.size + 1}`;
      this.registerPayment(id, Number(body.amount));
      return { json: { payment_id: id, status: "pending" } };
    }
    const payment = /^\/v1\/billing\/payments\/([a-z0-9-]+)$/.exec(path);
    if (payment) {
      const status = this.paymentStatus(payment[1]);
      if (!status) return { status: 404, json: { error: "payment not found" } };
      return {
        json: {
          id: payment[1],
          amount: this.payments.get(payment[1])?.amount ?? 0,
          currency: "RUB",
          status,
          provider: "testpay",
          provider_name: "Тестовый шлюз",
        },
      };
    }
    if (path === "/v1/billing") {
      const wallet = { id: "w1", currency: "RUB", balance: this.balance, is_default: true };
      return {
        json: {
          wallets: [wallet],
          selected_wallet: wallet,
          transactions: [
            { id: "1", type: "debit", amount: 210, description: "Аренда сервера", created_at: new Date(Date.now() - 86_400_000).toISOString() },
            { id: "2", type: "credit", amount: 250, description: "Пополнение картой", created_at: new Date(Date.now() - 3 * 86_400_000).toISOString() },
          ],
          credits_total: 250,
          debits_total: 210,
          available_currencies: ["RUB"],
        },
      };
    }

    const match = /^\/v1\/servers\/([0-9a-f-]+)\/([a-z0-9/_-]+)$/.exec(path);
    if (match) {
      const server = this.server(match[1]);
      if (!server) return { status: 404, json: { error: "server not found" } };
      const tail = match[2];
      if (tail === "detail") return { json: server };
      if (tail === "runtime") return { json: { supported: false } };
      if (tail === "auto-start" && method === "POST") {
        (server as unknown as Record<string, unknown>).auto_start_enabled = body.enabled === true;
        return { json: { auto_start: body.enabled === true } };
      }
      if (tail === "status") return { json: this.serverStatus(server) };
      if (tail === "metrics") return { json: { points: metricPoints() } };
      if (tail === "power" && method === "POST") {
        this.power(server, String(body.action));
        return { json: { status: "ok" } };
      }
      if (tail === "logs") {
        const lines = [
          "[20:57:08 INFO]: Starting minecraft server version 1.21.4",
          "[20:57:13 WARN]: Can't keep up! Is the server overloaded?",
          "[20:57:14 INFO]: Done (5.912s)! For help, type \"help\"",
          "[00:08:52 ERROR]: Failed to save player data",
        ];
        return { json: { lines, times: lines.map((_, i) => new Date(Date.now() - (4 - i) * 60_000).toISOString()) } };
      }
      if (tail === "firewall") {
        return {
          json: {
            rules: [{ id: "1", protocol: "tcp", port_from: 25565, enabled: true, action: "deny", source: "203.0.113.0/24" }],
            ports: [{ port: 25565, protocol: "tcp", purpose: "основной порт сервера", primary: true }],
            conn_limit: 0,
          },
        };
      }
      if (tail === "firewall/list") {
        return {
          json: {
            rules: [{ id: "1", protocol: "tcp", port_from: 25565, enabled: true, action: "deny", source: "203.0.113.0/24" }],
            ports: [{ port: 25565, protocol: "tcp", purpose: "основной порт сервера", primary: true }],
            conn_limit: 0,
          },
        };
      }
      if (tail === "renew/preview" && method === "POST") {
        return { json: { period_days: Number(body.period) || 30, price: 250, currency: "RUB", base_cost: 250 } };
      }
      if (tail === "ports/list") {
        return { json: { ports: [{ id: "1", port: 25565, protocol: "tcp", purpose: "Игровой порт", is_primary: true }], primary_port: 25565, can_edit_ports: false } };
      }
      if (tail === "plugins") {
        return {
          json: {
            items: [
              { plugin: { id: "p1", slug: "essentialsx", name: "EssentialsX", version: "2.21", description: "Команды и экономика" }, server_plugin: { installed: true, enabled: true } },
              { plugin: { id: "p3", slug: "worldedit", name: "WorldEdit", version: "7.3", description: "Редактор карты" }, server_plugin: { installed: false, enabled: false } },
            ],
          },
        };
      }
      if (tail === "backups") return { json: { backups: [], entries: [] } };
      if (tail === "backup-schedule") return { json: { enabled: false, frequency: "daily", hour_utc: 3, day_of_week: 1, keep_count: 5 } };
      if (tail === "friends") return { json: { friends: [] } };
      if (tail === "cron") return { json: { jobs: [{ id: "1", schedule: "0 */6 * * *", command: "save-all", enabled: true }] } };
      if (tail === "cron/list") return { json: { jobs: [{ id: "1", schedule: "0 */6 * * *", command: "save-all", enabled: true }], timezone: "UTC" } };
      if (tail === "files/list") return { json: { files: [] } };
      if (tail === "console-ticket") return { json: { ticket: "e2e-ticket", session_id: "e2e-session" } };
      if (tail === "ftp") return { json: { accounts: [{ id: "a1", username: "mc_e6fca27e", password: "s3cret", status: "active" }], host: server.ip_address, port: 2022, limit: 3 } };
      if (tail === "mysql") return { json: {} };
      if (tail === "tariffs") {
        return {
          json: {
            tariffs: [
              { id: "t0", name: "Старт", billing_type: "resources", currency: "RUB", base_price_monthly: 100, cpu_min: 1, cpu_max: 1, ram_min: 1, ram_max: 1, disk_min: 2, disk_max: 2, min_slots: 10, max_slots: 10 },
              { id: "t1", name: "Оптимальный", billing_type: "resources", currency: "RUB", base_price_monthly: 250, cpu_min: 2, cpu_max: 2, ram_min: 2, ram_max: 2, disk_min: 3, disk_max: 3, min_slots: 20, max_slots: 20 },
            ],
          },
        };
      }
    }

    if (/^\/v1\/monitoring\/[0-9a-f-]+\/stats$/.test(path)) {
      const n = 14;
      return {
        json: {
          ok: true,
          days: 7,
          series: {
            labels: Array.from({ length: n }, (_, i) => `0${(i % 9) + 1}.10`),
            online: Array.from({ length: n }, (_, i) => 2 + (i % 4)),
            cpu: Array.from({ length: n }, (_, i) => 20 + (i % 5)),
            ram: Array.from({ length: n }, (_, i) => 60 + (i % 4)),
            disk: Array.from({ length: n }, (_, i) => 10 + i * 0.4),
          },
        },
      };
    }

    return undefined;
  }

  paymentStatus(id: string) {
    const payment = this.payments.get(id);
    if (!payment) return undefined;
    payment.polls += 1;
    if (payment.polls < 2) return "pending";
    if (!payment.credited) {
      payment.credited = true;
      this.balance += payment.amount;
    }
    return "completed";
  }

  registerPayment(id: string, amount: number) {
    this.payments.set(id, { amount, polls: 0 });
  }
}
