import type { AdminSettingsData } from "@/components/admin/settings/types";
import type {
  SaveSettingsResponse,
  ServerSettingsSchema,
  SettingsFileContent,
} from "@/lib/game-settings/types";
import { decodeAccess, forgetAccount, markNeedsSignIn, rememberTokens } from "@/lib/accounts";
import { runtimeConfig } from "@/lib/runtime-config";
// t зовём только внутри функций: на уровне модуля язык заморозился бы тем,
// каким он был при загрузке страницы, и переключение его бы не меняло.
import { t } from "@/lib/i18n";

export const API_URL = runtimeConfig().api_url;
export const CONSOLE_URL = runtimeConfig().console_url;
export const TENANT_SLUG = "default";

export function tenantSlug(): string {
  return TENANT_SLUG;
}

const ACCESS_TOKEN_KEY = "vortanix_access_token";
const REFRESH_TOKEN_KEY = "vortanix_refresh_token";

export function getAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(ACCESS_TOKEN_KEY);
}

export function setTokens(access: string, refresh: string) {
  localStorage.setItem(ACCESS_TOKEN_KEY, access);
  // Пустую строку под видом токена не храним. Часть ответов входа объявляет
  // refresh_token необязательным (Telegram, соцсети, вход под пользователем), и
  // на месте вызова стоит `res.refresh_token ?? ""` — такая запись создавала бы
  // сессию, которую нечем продлить, а `""` при этом ложна, и ветки выхода на
  // /login, проверяющие getRefreshToken(), вели бы себя непредсказуемо.
  if (refresh) {
    localStorage.setItem(REFRESH_TOKEN_KEY, refresh);
  } else {
    localStorage.removeItem(REFRESH_TOKEN_KEY);
  }
  // Единственная точка, где список аккаунтов пополняется. Владелец берётся из
  // самого токена, поэтому продление активной сессии обновляет её запись, а
  // вход в другой аккаунт заводит соседнюю, не трогая первую.
  rememberTokens(access, refresh);
  startAuthRefreshLoop();
}

function decodeJwtExp(token: string): number | null {
  try {
    const payload = token.split(".")[1];
    if (!payload) return null;
    const normalized = payload.replace(/-/g, "+").replace(/_/g, "/");
    const decoded = JSON.parse(atob(normalized)) as { exp?: number };
    return typeof decoded.exp === "number" ? decoded.exp : null;
  } catch {
    return null;
  }
}

function accessTokenExpiresSoon(thresholdMs = 5 * 60 * 1000): boolean {
  const access = getAccessToken();
  if (!access) return true;
  const exp = decodeJwtExp(access);
  if (!exp) return false;
  return exp * 1000 - Date.now() < thresholdMs;
}

export async function ensureValidSession(): Promise<boolean> {
  const refresh = getRefreshToken();
  if (!refresh) {
    return !!getAccessToken();
  }
  if (!getAccessToken() || accessTokenExpiresSoon()) {
    try {
      const refreshed = await refreshTokens();
      setTokens(refreshed.access_token, refreshed.refresh_token);
      return true;
    } catch {
      clearAuth();
      // Результат этой функции почти везде игнорируется: её зовут фоновый цикл
      // обновления и apiFetch перед каждым запросом. Без редиректа протухшая
      // сессия стиралась молча, и страница оставалась висеть на своём лоадере —
      // на /admin/* это давало вечное «Проверяем доступ».
      redirectToLogin();
      return false;
    }
  }
  return true;
}

let refreshLoopTimer: ReturnType<typeof setInterval> | null = null;

export function startAuthRefreshLoop() {
  if (typeof window === "undefined") return;
  if (refreshLoopTimer) clearInterval(refreshLoopTimer);
  if (!getRefreshToken()) return;
  void ensureValidSession();
  refreshLoopTimer = setInterval(() => {
    if (!getRefreshToken()) {
      if (refreshLoopTimer) clearInterval(refreshLoopTimer);
      refreshLoopTimer = null;
      return;
    }
    void ensureValidSession();
  }, 60_000);
}

export function stopAuthRefreshLoop() {
  if (refreshLoopTimer) {
    clearInterval(refreshLoopTimer);
    refreshLoopTimer = null;
  }
}

/**
 * Занести уже открытую сессию в список аккаунтов.
 *
 * Нужна ровно один раз на браузер: у тех, кто вошёл до появления списка, токены
 * лежат в основных ключах, а записи нет — и себя в меню они бы не увидели, пока
 * не выйдут и не войдут заново. `rememberTokens` идемпотентен, повторные вызовы
 * лишь освежают запись.
 */
export function adoptActiveSession() {
  const access = getAccessToken();
  if (!access) return;
  rememberTokens(access, getRefreshToken() ?? "");
}

/** Чей токен лежит в основных ключах прямо сейчас. */
export function activeAccountId(): string | null {
  const token = getAccessToken();
  if (!token) return null;
  return decodeAccess(token)?.user_id ?? null;
}

export function clearAuth() {
  // Сессия оборвалась не по воле человека: истёк срок, её закрыли из настроек
  // или с другого устройства. Запись в списке аккаунтов оставляем с пометкой —
  // аккаунт никуда не делся, ему просто снова нужен пароль. Удалять её здесь
  // нельзя: тогда чужая протухшая сессия молча выносила бы аккаунт из меню.
  const id = activeAccountId();
  if (id) markNeedsSignIn(id);
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
  stopAuthRefreshLoop();
}

/** Выход по кнопке: аккаунт уходит из списка совсем. */
export function clearAuthAndForget() {
  const id = activeAccountId();
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
  stopAuthRefreshLoop();
  if (id) forgetAccount(id);
}

// Уходим на вход перезагрузкой страницы: она стирает кэш запросов из памяти.
// Иначе на экране остались бы данные того, чья сессия только что закончилась.
export function redirectToLogin() {
  if (typeof window === "undefined") return;
  if (window.location.pathname.startsWith("/login")) return;
  window.location.replace("/login");
}

export function getRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

let refreshPromise: Promise<{ access_token: string; refresh_token: string }> | null =
  null;

async function refreshTokens() {
  if (refreshPromise) return refreshPromise;
  const refresh = getRefreshToken();
  if (!refresh) throw new Error("Session expired");
  refreshPromise = apiFetch<{ access_token: string; refresh_token: string }>(
    "/v1/auth/refresh",
    { method: "POST", body: JSON.stringify({ refresh_token: refresh }) },
    "api",
    true
  ).finally(() => {
    refreshPromise = null;
  });
  return refreshPromise;
}

export async function logout() {
  try {
    await apiFetch<{ status: string }>("/v1/auth/logout", { method: "POST" });
  } finally {
    clearAuthAndForget();
  }
}

export async function apiFetch<T>(
  path: string,
  options: RequestInit = {},
  base: "api" = "api",
  retried = false
): Promise<T> {
  const url = API_URL + path;
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };
  if (base === "api" && path !== "/v1/auth/refresh" && getRefreshToken()) {
    await ensureValidSession();
  }
  const token = getAccessToken();
  if (token && base === "api") {
    headers.Authorization = `Bearer ${token}`;
  }
  const res = await fetch(url, { ...options, headers });
  let data: { error?: string; message?: string } = {};
  const text = await res.text();
  if (text) {
    try {
      data = JSON.parse(text) as { error?: string };
    } catch {
      data = {};
    }
  }
  if (res.status === 401 && base === "api" && !retried && getRefreshToken()) {
    try {
      const refreshed = await refreshTokens();
      setTokens(refreshed.access_token, refreshed.refresh_token);
      return apiFetch<T>(path, options, base, true);
    } catch {
      clearAuth();
      redirectToLogin();
      throw new Error(data.error ?? data.message ?? "Session expired");
    }
  }
  if (!res.ok) {
    throw new Error(data.error ?? data.message ?? "Request failed");
  }
  return data as T;
}

export async function fetchSetupStatus() {
  return apiFetch<{
    bootstrapped: boolean;
    suggested_domain: string;
  }>("/v1/setup/status");
}

export async function bootstrapPanel(
  email: string,
  password: string,
  panelName?: string
) {
  return apiFetch<{
    access_token: string;
    refresh_token: string;
    tenant_slug: string;
  }>("/v1/tenants/bootstrap", {
    method: "POST",
    body: JSON.stringify({
      owner_email: email,
      owner_password: password,
      panel_name: panelName,
    }),
  });
}

export async function fetchTenantStatus(tenantSlug: string) {
  return apiFetch<{ bootstrapped: boolean }>(
    `/v1/tenants/status?slug=${encodeURIComponent(tenantSlug)}`
  );
}

export async function register(
  email: string,
  password: string,
  tenantSlug: string,
  profile?: { name?: string; lastName?: string }
) {
  return apiFetch<{
    access_token: string;
    refresh_token: string;
    user: { id: string; email: string; role: string };
  }>("/v1/auth/register", {
    method: "POST",
    body: JSON.stringify({
      email,
      password,
      tenant_slug: tenantSlug,
      ...(profile?.name ? { name: profile.name } : {}),
      ...(profile?.lastName ? { last_name: profile.lastName } : {}),
    }),
  });
}

export async function login(
  email: string,
  password: string,
  tenantSlug: string,
  remember = false
) {
  // Вход двухшаговый: при включённой двухфакторной проверке сервер отвечает
  // 200 без токенов, называя requires_2fa. Тип это обязан отражать — пока он
  // обещал токены всегда, форма считала такой ответ успехом и клала в
  // localStorage строку "undefined".
  return apiFetch<{
    access_token?: string;
    refresh_token?: string;
    requires_2fa?: boolean;
    two_factor_token?: string;
    user?: { id: string; email: string; role: string };
  }>("/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password, tenant_slug: tenantSlug, remember }),
  });
}

export type SocialProvidersInfo = {
  ok?: boolean;
  providers?: string[];
  telegram?: { enabled?: boolean; bot_username?: string };
};

export async function fetchSocialProviders() {
  return apiFetch<SocialProvidersInfo>("/v1/auth/social/providers");
}

export type TelegramAuthUser = Record<string, string | number | undefined>;

function telegramQuery(user: TelegramAuthUser) {
  const q = new URLSearchParams();
  for (const [key, value] of Object.entries(user)) {
    if (value === undefined || value === null || value === "") continue;
    q.set(key, String(value));
  }
  return q;
}

export async function telegramLogin(user: TelegramAuthUser, tenantSlug: string) {
  const q = telegramQuery(user);
  q.set("tenant_slug", tenantSlug);
  return apiFetch<{
    ok?: boolean;
    access_token?: string;
    refresh_token?: string;
    redirect?: string;
    user?: { id: string; email: string; role: string };
  }>(`/v1/auth/telegram/callback?${q.toString()}`);
}

export async function socialRedirect(provider: string, tenantSlug: string) {
  return apiFetch<{ ok?: boolean; url?: string }>(
    `/v1/auth/social/${encodeURIComponent(provider)}/redirect?tenant_slug=${encodeURIComponent(tenantSlug)}`
  );
}

export async function socialExchange(
  provider: string,
  code: string,
  state: string,
  linkMode = false
) {
  const path = linkMode
    ? `/v1/account/social/${encodeURIComponent(provider)}/exchange`
    : `/v1/auth/social/${encodeURIComponent(provider)}/exchange`;
  return apiFetch<{
    ok?: boolean;
    access_token?: string;
    refresh_token?: string;
    redirect?: string;
    user?: { id: string; email: string; role: string };
    message?: string;
  }>(path, {
    method: "POST",
    body: JSON.stringify({ code, state }),
  });
}

export async function resendEmailVerification() {
  return apiFetch<{
    ok?: boolean;
    status?: string;
    message?: string;
    verification_url?: string;
  }>("/v1/email/verification-notification", { method: "POST" });
}

export async function verifyEmailURL(verifyURL: string) {
  const token = getAccessToken();
  const res = await fetch(verifyURL, {
    headers: {
      Accept: "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  });
  const data = await res.json();
  if (!res.ok) {
    throw new Error(data.error ?? data.message ?? "Verification failed");
  }
  return data as { ok?: boolean; message?: string };
}

export async function fetchUsers() {
  const res = await apiFetch<{ users: PanelUser[] | { data: PanelUser[] } }>(
    "/v1/users"
  );
  if (Array.isArray(res.users)) return res.users;
  return res.users?.data ?? [];
}

export type AdminUsersListResponse = {
  users:
    | PanelUser[]
    | {
        data: PanelUser[];
        current_page: number;
        last_page: number;
        per_page: number;
        total: number;
      };
  q?: string;
  role?: string;
  counts?: {
    total: number;
    admins: number;
    support: number;
    users: number;
    active?: number;
    unverified?: number;
    blocked?: number;
  };
};

export async function fetchAdminUsers(params?: {
  page?: number;
  q?: string;
  role?: string;
}) {
  const sp = new URLSearchParams();
  if (params?.q) sp.set("q", params.q);
  if (params?.role) sp.set("role", params.role);
  sp.set("page", String(params?.page ?? 1));
  const qs = sp.toString();
  return apiFetch<AdminUsersListResponse>(`/v1/users?${qs}`);
}

export function normalizeAdminUsersList(res: AdminUsersListResponse): {
  users: PanelUser[];
  currentPage: number;
  lastPage: number;
  counts?: AdminUsersListResponse["counts"];
} {
  const raw = res.users;
  if (Array.isArray(raw)) {
    return { users: raw, currentPage: 1, lastPage: 1, counts: res.counts };
  }
  return {
    users: raw?.data ?? [],
    currentPage: raw?.current_page ?? 1,
    lastPage: raw?.last_page ?? 1,
    counts: res.counts,
  };
}

export type AdminUserWallet = {
  id: string;
  currency: string;
  balance: number;
  is_default: boolean;
};

export type AdminUserServer = {
  id: string;
  name: string;
  ip_address: string;
  port: number;
  game: { name: string } | null;
};

export type AdminUserTransaction = {
  id: string;
  type: string;
  amount: number;
  description: string;
  created_at: string;
  wallet: { currency: string } | null;
};

export type AdminUserSession = {
  id: string;
  ip_address: string;
  user_agent: string;
  last_activity: string;
};

export type AdminUserDetail = {
  user: {
    id: string;
    name: string;
    last_name: string | null;
    public_id: string | null;
    email: string;
    phone: string | null;
    telegram_id: string | null;
    discord_id: string | null;
    vk_id: string | null;
    role: string;
    status: string;
    is_blocked: boolean;
    two_factor_enabled?: boolean;
    email_verified_at: string | null;
    created_at: string;
  };
  servers: AdminUserServer[];
  wallets: AdminUserWallet[];
  defaultWallet: AdminUserWallet | null;
  transactions: AdminUserTransaction[];
  sessions: AdminUserSession[];
};

export async function fetchAdminUser(id: string) {
  return apiFetch<AdminUserDetail>(`/v1/users/${id}`);
}

export async function fetchAdminUserCreateForm() {
  return apiFetch<{ roles: string[]; currencies: string[] }>("/v1/users/create");
}

export async function createPanelUser(body: {
  name?: string;
  email: string;
  password: string;
  password_confirmation?: string;
  role?: "user" | "admin" | "support";
}) {
  return apiFetch<{ ok?: boolean; id: string; email: string; role: string }>(
    "/v1/users",
    {
      method: "POST",
      body: JSON.stringify(body),
    }
  );
}

export async function deletePanelUser(id: string) {
  return apiFetch<{ ok?: boolean; status: string }>(`/v1/users/${id}`, {
    method: "DELETE",
  });
}

export async function updatePanelUser(
  id: string,
  body: Record<string, unknown>
) {
  return apiFetch<{ ok?: boolean; status: string }>(`/v1/users/${id}`, {
    method: "PUT",
    body: JSON.stringify(body),
  });
}

export async function togglePanelUserBlock(id: string) {
  return apiFetch<{ ok?: boolean; message?: string; status?: string }>(
    `/v1/users/${id}/toggle-block`,
    { method: "POST" }
  );
}

export async function verifyPanelUserEmail(id: string) {
  return apiFetch<{ ok?: boolean; status: string; message?: string }>(
    `/v1/users/${id}/verify-email`,
    {
      method: "POST",
    }
  );
}

export async function impersonatePanelUser(id: string) {
  return apiFetch<{
    ok?: boolean;
    token?: string;
    access_token?: string;
    refresh_token?: string;
    impersonator_id?: string;
    message?: string;
  }>(`/v1/users/${id}/impersonate`, { method: "POST" });
}

export async function createAdminUserWallet(id: string, currency: string) {
  return apiFetch<{ ok?: boolean; wallet_id: string; message?: string }>(
    `/v1/users/${id}/wallets/create`,
    {
      method: "POST",
      body: JSON.stringify({ currency }),
    }
  );
}

export type PanelUser = {
  id: string;
  email: string;
  name?: string;
  role: string;
  status: string;
  is_admin?: boolean;
  is_blocked?: boolean;
  email_verified_at?: string | null;
  balance?: number;
  servers_count?: number;
  created_at: string;
};

export async function fetchMe() {
  return apiFetch<{
    user_id: string;
    email: string;
    role: string;
    tenant_id: string;
    tenant_slug: string;
  }>("/v1/me");
}

export async function changePassword(currentPassword: string, newPassword: string) {
  return apiFetch<{ status: string }>("/v1/me/password", {
    method: "PATCH",
    body: JSON.stringify({
      current_password: currentPassword,
      new_password: newPassword,
    }),
  });
}

export type Node = {
  id: string;
  name: string;
  fqdn: string;
  status: string;
  meta: Record<string, unknown>;
  last_seen_at?: string;
  created_at: string;
};

export type Server = {
  id: string;
  node_id: string;
  game_id: string;
  name: string;
  status: string;
  config: Record<string, unknown>;
  limits: Record<string, unknown>;
  created_at: string;
};

export async function fetchNodes() {
  const res = await apiFetch<{
    locations: AdminLocationListItem[];
    nodes?: AdminLocationListItem[];
  }>("/v1/admin/locations");
  const list = res.locations ?? res.nodes ?? [];
  return list.map(locationToNode);
}

function locationToNode(loc: AdminLocationListItem): Node {
  const meta =
    loc.meta && typeof loc.meta === "object"
      ? (loc.meta as Record<string, unknown>)
      : {};
  const online =
    loc.is_online === true ||
    loc.agent_status === "online" ||
    loc.status === "online";
  return {
    id: loc.id,
    name: loc.name,
    fqdn: loc.fqdn ?? loc.code ?? "",
    status: online ? "online" : "offline",
    meta,
    last_seen_at:
      typeof loc.last_seen_at === "string" ? loc.last_seen_at : undefined,
    created_at:
      typeof loc.created_at === "string"
        ? loc.created_at
        : new Date().toISOString(),
  };
}

export async function createNode(name: string, fqdn: string) {
  return createAdminLocation({ name, code: fqdn, fqdn });
}

export async function deleteNode(id: string) {
  return deleteAdminLocation(id);
}

export async function fetchNodeInstall(id: string) {
  return apiFetch<{
    script: string;
    node_id: string;
    location_id?: string;
    agent_token?: string;
    relay_url?: string;
    env_file?: string;
    fqdn?: string;
  }>(`/v1/admin/locations/${id}/install`);
}

export async function regenerateAdminLocationAgentToken(id: string) {
  return apiFetch<{ ok: boolean; node_id: string; agent_token: string }>(
    `/v1/admin/locations/${id}/agent-token/regenerate`,
    { method: "POST" }
  );
}

export async function testAdminLocationSSH(id: string) {
  return apiFetch<{ ok: boolean; output?: string; error?: string }>(
    `/v1/admin/locations/${id}/ssh/test`,
    { method: "POST" }
  );
}

export async function testAdminLocationSSHBody(body: {
  ssh_host: string;
  ssh_user: string;
  ssh_password: string;
  ssh_port?: number;
}) {
  return apiFetch<{ ok: boolean; output?: string; error?: string }>(
    "/v1/admin/locations/ssh/test",
    { method: "POST", body: JSON.stringify(body) }
  );
}

export async function fetchMyServers() {
  return apiFetch<{
    servers: DashboardServer[];
    total: number;
    active_count: number;
    expiring_soon: number;
  }>("/v1/my-servers");
}

export async function fetchServers() {
  const res = await apiFetch<{ servers: Server[] }>("/v1/servers");
  return res.servers;
}

export async function createServer(nodeId: string, name: string, gameId = "test") {
  return apiFetch<Server>("/v1/servers", {
    method: "POST",
    body: JSON.stringify({ node_id: nodeId, name, game_id: gameId }),
  });
}

export async function deleteServer(id: string) {
  return apiFetch<{ status: string }>(`/v1/servers/${id}`, { method: "DELETE" });
}

export async function powerServer(id: string, action: string) {
  return apiFetch<{ status: string }>(`/v1/servers/${id}/power`, {
    method: "POST",
    body: JSON.stringify({ action }),
  });
}

export async function fetchConsoleTicket(serverId: string) {
  return apiFetch<{ ticket: string; session_id: string }>(
    `/v1/servers/${serverId}/console-ticket`,
    { method: "POST" }
  );
}

export type MetricPoint = {
  ts: number;
  cpu_pct: number;
  mem_used_mb: number;
  mem_limit_mb: number;
};

export async function fetchServerMetrics(serverId: string) {
  const res = await apiFetch<{ points: MetricPoint[] }>(`/v1/servers/${serverId}/metrics`);
  return res.points;
}

export function consoleWsUrl(ticket: string) {
  return `${CONSOLE_URL}/v1/console?ticket=${encodeURIComponent(ticket)}`;
}

export async function forgotPassword(email: string, tenantSlug: string) {
  return apiFetch<{ status: string; token?: string }>("/v1/auth/forgot-password", {
    method: "POST",
    body: JSON.stringify({ email, tenant_slug: tenantSlug }),
  });
}

export async function resetPassword(token: string, newPassword: string) {
  return apiFetch<{ status: string }>("/v1/auth/reset-password", {
    method: "POST",
    body: JSON.stringify({ token, new_password: newPassword }),
  });
}

export async function challenge2FA(twoFactorToken: string, code: string) {
  return apiFetch<{
    access_token: string;
    refresh_token: string;
    user: { id: string; email: string; role: string };
  }>("/v1/auth/2fa/challenge", {
    method: "POST",
    body: JSON.stringify({ two_factor_token: twoFactorToken, code }),
  });
}

export async function refreshToken(refresh: string) {
  return apiFetch<{ access_token: string; refresh_token: string }>(
    "/v1/auth/refresh",
    {
      method: "POST",
      body: JSON.stringify({ refresh_token: refresh }),
    }
  );
}

export type AccountUser = {
  id: string;
  email: string;
  role: string;
  two_factor_enabled: boolean;
  email_verified?: boolean;
  display_name?: string | null;
  first_name?: string | null;
  last_name?: string | null;
  phone?: string | null;
  locale?: string;
  avatar_url?: string | null;
  linked_providers?: string[];
};

export type AccountSession = {
  id: string;
  ip_address?: string;
  user_agent?: string;
  last_activity?: string;
  created_at?: string;
  is_current?: boolean;
};

export async function fetchAccount() {
  return apiFetch<{ user: AccountUser }>("/v1/account");
}

export async function updateAccount(data: Record<string, unknown>) {
  return apiFetch<{ status: string }>("/v1/account", {
    method: "PATCH",
    body: JSON.stringify(data),
  });
}

export async function socialLinkRedirect(provider: string, tenantSlug: string) {
  return apiFetch<{ ok?: boolean; url?: string }>(
    `/v1/account/social/${encodeURIComponent(provider)}/redirect?tenant_slug=${encodeURIComponent(tenantSlug)}`
  );
}

export async function telegramLinkAccount(user: TelegramAuthUser) {
  const q = telegramQuery(user);
  return apiFetch<{ ok?: boolean; status?: string; message?: string }>(
    `/v1/account/telegram/link?${q.toString()}`
  );
}

export async function socialUnlink(provider: string) {
  return apiFetch<{ status?: string; ok?: boolean }>(
    `/v1/account/social/${encodeURIComponent(provider)}/unlink`,
    { method: "POST" }
  );
}

export async function generate2FA() {
  return apiFetch<{ secret: string; uri: string }>(
    "/v1/account/2fa/generate",
    { method: "POST" }
  );
}

export async function enable2FA() {
  return apiFetch<{ status: string }>("/v1/account/2fa/enable", {
    method: "POST",
  });
}

// Пароль обязателен: второй фактор снимался одной кнопкой, и тому, кто увёл
// живой сеанс, этого хватало, чтобы убрать защиту.
export async function disable2FA(password: string) {
  return apiFetch<{ status: string }>("/v1/account/2fa/disable", {
    method: "POST",
    body: JSON.stringify({ password }),
  });
}

export async function fetchAccountSessions() {
  return apiFetch<{
    sessions: AccountSession[];
    current_session_id?: string;
  }>("/v1/account/sessions");
}

export async function destroyAccountSession(body: {
  session_id?: string;
  all_others?: boolean;
}) {
  return apiFetch<{ status: string }>("/v1/account/sessions/destroy", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export async function changeAccountEmail(data: {
  email: string;
  current_password: string;
}) {
  return apiFetch<{
    status: string;
    email: string;
    verification_url?: string;
  }>("/v1/account/email/change", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function uploadAccountAvatar(file: File) {
  const form = new FormData();
  form.append("avatar", file);
  const url = API_URL + "/v1/account/avatar";
  const headers: Record<string, string> = {};
  const token = getAccessToken();
  if (token) headers.Authorization = `Bearer ${token}`;
  const res = await fetch(url, { method: "POST", headers, body: form });
  const data = await res.json();
  if (!res.ok) {
    throw new Error(data.error ?? "Upload failed");
  }
  return data as { status: string; avatar_url: string };
}

export async function uploadAccountAvatarUrl(avatarUrl: string) {
  return apiFetch<{ status: string; avatar_url: string }>("/v1/account/avatar", {
    method: "POST",
    body: JSON.stringify({ avatar_url: avatarUrl }),
  });
}

export type ServerViewerPermissions = {
  can_view_console?: boolean;
  can_view_logs?: boolean;
  can_view_metrics?: boolean;
  can_view_ftp?: boolean;
  can_view_mysql?: boolean;
  can_view_cron?: boolean;
  can_view_firewall?: boolean;
  can_view_ports?: boolean;
  can_view_settings?: boolean;
  can_view_friends?: boolean;
  can_start?: boolean;
  can_stop?: boolean;
  can_restart?: boolean;
  can_reinstall?: boolean;
  can_console_command?: boolean;
  can_files?: boolean;
  can_cron_manage?: boolean;
  can_firewall_manage?: boolean;
  can_ports_manage?: boolean;
  can_settings_edit?: boolean;
};

export type DashboardServer = {
  id: string;
  name: string;
  ip_address: string;
  port: number;
  status: string;
  runtime_status: string;
  provisioning_status: string;
  provisioning_error?: string | null;
  created_at?: string | null;
  user_id?: string;
  expires_at: string | null;
  game_id?: string;
  game: { name: string; slug: string; image: string | null } | null;
  auto_renew?: boolean;
  rental_period_days?: number;
  location: {
    name: string;
    country: string;
    city: string;
    maintenance?: {
      enabled: boolean;
      reason: string;
      until: string | null;
    };
  } | null;
  tariff: {
    id?: string;
    name: string;
    price_monthly?: number;
    currency?: string;
    ram_mb?: number;
    disk_mb?: number;
    slots?: number;
    billing_type?: string;
    renewal_periods?: number[];
    base_price_monthly?: number;
    price_per_slot?: number;
    price_per_cpu_core?: number;
    price_per_ram_gb?: number;
    price_per_disk_gb?: number;
    antiddos_price?: number;
  } | null;
  limits?: Record<string, unknown>;
  auto_start_enabled?: boolean;
  startup_params?: string;
  game_version_id?: string;
  steam_updatable?: boolean;
  available_game_versions?: {
    id: string;
    name: string;
    version?: string;
    source_type?: string;
    manual_install?: boolean;
    install_note?: string;
  }[];
  online_players?: number;
  max_players?: number;
  current_map?: string;
  uptime?: string | null;
  players_online?: { name: string; score?: number; ping?: number }[];
  provisioning_progress?: {
    percent?: number | string;
    message?: string;
    stage?: string;
  } | null;
  disk_used_mb?: number;
  disk_total_mb?: number;
  is_blocked?: boolean;
  blocked_reason?: string | null;
  cpu_percent?: number;
  ram_percent?: number;
  mem_used_mb?: number;
  mem_limit_mb?: number;
  metrics_at?: string;
  viewer_permissions?: ServerViewerPermissions;
  mysql_host?: string;
  mysql_port?: number;
  mysql_database?: string;
  mysql_username?: string;
  mysql_password_decrypted?: string;
  mysql_instance_key?: string;
};

export type DashboardTransaction = {
  id: string;
  type: string;
  amount: number;
  description: string;
  created_at: string;
};

export type DashboardNews = {
  id: string;
  slug: string;
  title: string;
  excerpt: string;
  published_at: string | null;
  image: string | null;
};

export type DashboardData = {
  balance: number;
  balance_currency: string;
  total_servers: number;
  active_servers: number;
  expiring_soon_count: number;
  open_support_tickets_count: number;
  next_charge_text: string;
  recent_servers: DashboardServer[];
  recent_transactions: DashboardTransaction[];
  news: DashboardNews[];
};

export async function fetchDashboard() {
  return apiFetch<DashboardData>("/v1/dashboard");
}

export type ActivityEntry = {
  id: number;
  action: string;
  resource: string;
  category: string;
  category_key: string;
  description: string;
  user_email: string;
  ip: string;
  meta?: Record<string, unknown>;
  created_at: string;
};

export type ActivityStats = {
  events_24h: number;
  server_actions: number;
  logins: number;
  errors_7d: number;
};

export type ActivityResponse = {
  activity: ActivityEntry[];
  total: number;
  has_more: boolean;
  stats: ActivityStats;
  range: number;
};

export type ActivityQuery = {
  q?: string;
  category?: string;
  range?: number;
  limit?: number;
  offset?: number;
};

function activityQueryString(params: ActivityQuery = {}) {
  const qs = new URLSearchParams();
  if (params.q) qs.set("q", params.q);
  if (params.category && params.category !== "all") qs.set("category", params.category);
  if (params.range) qs.set("range", String(params.range));
  if (params.limit) qs.set("limit", String(params.limit));
  if (params.offset) qs.set("offset", String(params.offset));
  const s = qs.toString();
  return s ? `?${s}` : "";
}

export async function fetchActivity(params: ActivityQuery = {}) {
  return apiFetch<ActivityResponse>(`/v1/activity${activityQueryString(params)}`);
}

export async function downloadActivityCsv(params: ActivityQuery = {}) {
  if (getRefreshToken()) await ensureValidSession();
  const token = getAccessToken();
  const res = await fetch(`${API_URL}/v1/activity/export.csv${activityQueryString(params)}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) throw new Error(t("errors.activity.export_failed"));

  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `vortanix-activity-${new Date().toISOString().slice(0, 10)}.csv`;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

export type Notification = {
  id: string;
  type: string;
  title: string;
  body: string;
  /** Идентификатор раздела для фильтра на сервере. */
  group: string;
  category: string;
  icon: string;
  tone: "info" | "ok" | "warn" | "bad";
  action: string;
  href: string;
  unread: boolean;
  read_at: string;
  created_at: string;
};

export type NotificationChannels = {
  email: boolean;
  telegram: boolean;
  discord: boolean;
  telegram_chat_id: string;
  discord_webhook: string;
};

export type NotificationGroup = {
  id: string;
  label: string;
  count: number;
  unread: number;
};

export type NotificationsResponse = {
  notifications: Notification[];
  unread: number;
  /** Разделы со счётчиками по всей ленте, а не по выданной странице. */
  groups: NotificationGroup[];
  has_more: boolean;
  channels: NotificationChannels;
  error?: string;
};

export type NotificationsQuery = {
  group?: string;
  unread?: boolean;
  offset?: number;
  limit?: number;
};

// Фильтр и пагинация считаются на сервере. Раньше ручка не принимала ни одного
// параметра и жёстко отдавала последние 200 записей: двести первая была
// недостижима навсегда, а счётчики разделов считались по выданной странице.
export async function fetchNotifications(q: NotificationsQuery = {}) {
  const params = new URLSearchParams();
  if (q.group) params.set("group", q.group);
  if (q.unread) params.set("unread", "1");
  if (q.offset) params.set("offset", String(q.offset));
  if (q.limit) params.set("limit", String(q.limit));
  const suffix = params.toString();
  return apiFetch<NotificationsResponse>("/v1/notifications" + (suffix ? "?" + suffix : ""));
}

export async function deleteNotification(id: string) {
  return apiFetch<{ status: string }>(`/v1/notifications/${id}`, { method: "DELETE" });
}

export async function clearReadNotifications() {
  return apiFetch<{ status: string; deleted: number }>("/v1/notifications/read", {
    method: "DELETE",
  });
}

export async function fetchNotificationsUnreadCount() {
  return apiFetch<{ count: number }>("/v1/notifications/unread-count");
}

export async function markAllNotificationsRead() {
  return apiFetch<{ status: string }>("/v1/notifications/read-all", {
    method: "POST",
  });
}

export async function markNotificationRead(id: string) {
  return apiFetch<{ status: string }>(`/v1/notifications/${id}/read`, {
    method: "POST",
  });
}

export async function fetchNotificationChannels() {
  return apiFetch<{ channels: NotificationChannels }>("/v1/notifications/channels");
}

export async function updateNotificationChannels(payload: Partial<NotificationChannels>) {
  return apiFetch<{ ok: boolean; channels: NotificationChannels }>(
    "/v1/notifications/channels",
    { method: "PUT", body: JSON.stringify(payload) }
  );
}

export type BonusPrize = {
  id: string;
  label: string;
  type: string;
  value: number;
  weight: number;
  color: string;
  icon: string;
  chance: number;
  duration_hours: number;
};

export type BonusStreakDay = {
  day: string;
  weekday: number;
  claimed: boolean;
  today: boolean;
};

export type BonusHistoryItem = {
  prize: string;
  type: string;
  value: number;
  code: string;
  created_at: string;
};

export type DailyBonusResponse = {
  can_spin: boolean;
  next_spin_at: string | null;
  prizes: BonusPrize[];
  streak: number;
  streak_days: BonusStreakDay[];
  history: BonusHistoryItem[];
  prizes_disabled: boolean;
};

export type DailyBonusSpinResponse = {
  ok: boolean;
  prize_id: string;
  prize: BonusPrize;
  credited: number;
  // Промо-приз выдаётся личным одноразовым кодом; для приза «баланс» пусто.
  promo_code: string;
  streak: number;
  next_spin_at: string;
};

export async function fetchDailyBonus() {
  return apiFetch<DailyBonusResponse>("/v1/daily-bonus");
}

export async function spinDailyBonus() {
  return apiFetch<DailyBonusSpinResponse>("/v1/daily-bonus/spin", {
    method: "POST",
  });
}

export type Wallet = { id: string; currency: string; balance: number };
export type Transaction = {
  id: string;
  type: string;
  amount: number;
  description?: string;
  created_at: string;
};

export type BillingData = {
  wallets: Wallet[];
  selected_wallet: Wallet | null;
  transactions: Transaction[];
  credits_total: number;
  debits_total: number;
  available_currencies: string[];
};

export type TopupPaymentRecord = {
  id: string;
  status: string;
  amount: number;
  currency: string;
  credited_amount?: number | null;
  created_at: string;
};

export type TopupProvider = {
  id: string;
  code: string;
  name: string;
};

export type FreekassaMethod = {
  id: number;
  name: string;
};

export type TopupFormData = {
  wallets: Wallet[];
  selected_wallet: Wallet | null;
  payments: TopupPaymentRecord[];
  providers: TopupProvider[];
  enabled_providers: string[];
  freekassa_methods?: FreekassaMethod[];
};

export type CreateTopupParams = {
  walletId?: string;
  providerId: string;
  providerCode?: string;
  amount: string | number;
  promoCode?: string;
  paymentMethodId?: string;
};

export type CreateTopupResult = {
  payment_id: string;
  status: string;
  redirect_url?: string;
};

export type PaymentDetail = {
  id: string;
  amount: number;
  status: string;
  currency: string;
};

export async function fetchBilling(walletId?: string | null) {
  const q = walletId ? `?wallet_id=${encodeURIComponent(walletId)}` : "";
  return apiFetch<BillingData>(`/v1/billing${q}`);
}

export async function createWallet(currency = "RUB") {
  return apiFetch<{ id: string }>("/v1/billing/wallets/create", {
    method: "POST",
    body: JSON.stringify({ currency }),
  });
}

export async function fetchTopupForm() {
  return apiFetch<TopupFormData>("/v1/billing/topup");
}

export async function createTopup(params: CreateTopupParams) {
  const body: Record<string, unknown> = {
    amount: Number(params.amount),
    provider_id: params.providerId,
  };
  if (params.walletId) body.wallet_id = params.walletId;
  if (params.providerCode) body.provider = params.providerCode;
  if (params.promoCode) body.promo_code = params.promoCode;
  if (params.paymentMethodId) body.payment_method_id = params.paymentMethodId;

  return apiFetch<CreateTopupResult>("/v1/billing/topup/create", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export async function fetchPayment(id: string) {
  const data = await apiFetch<PaymentDetail | { payment?: PaymentDetail }>(
    `/v1/billing/payments/${id}`
  );
  if (data && typeof data === "object" && "payment" in data) {
    return data.payment ?? null;
  }
  return data as PaymentDetail;
}

export type SupportTicket = {
  id: string;
  subject: string;
  status: string;
  created_at: string;
  category?: string;
  priority?: string;
  last_message_at?: string;
  messages?: number;
  // Начало последней реплики и от кого она: список показывает, на чём стоит
  // обращение, а не только о чём оно.
  last_message?: string;
  last_from_staff?: boolean;
  notify?: boolean;
  service_kind?: string;
  service_id?: string;
  // Название услуги разрешается на сервере при чтении: храни его копией в
  // тикете — и после переименования сервера в поддержке осталось бы старое.
  service?: string;
};

export type SupportAttachment = {
  // id есть у файлов из таблицы вложений; у старых, описанных полем meta
  // сообщения, его нет — там идентификатором служит сам id сообщения.
  id?: string;
  name: string;
  size: number;
  content_type: string;
  url?: string;
};

export type SupportMessage = {
  id: string;
  body: string;
  is_staff: boolean;
  created_at: string;
  attachment?: SupportAttachment;
  // На сообщение теперь может приходиться несколько файлов.
  attachments?: SupportAttachment[];
};

export async function fetchSupportTickets() {
  return apiFetch<{ tickets: SupportTicket[] }>("/v1/support");
}

export type KBArticle = {
  id: string;
  slug: string;
  title: string;
  excerpt: string;
  body?: string;
  category: string;
  published: boolean;
  views: number;
  position: number;
  updated_at: string;
};

// Отдел и срочность отправляются вместе с обращением. До этого форма их
// спрашивала, а ручка принимала только тему и текст — выбор терялся молча.
export async function createSupportTicket(
  subject: string,
  body: string,
  options?: {
    category?: string;
    priority?: string;
    service_kind?: string;
    service_id?: string;
  }
) {
  return apiFetch<{ id: string }>("/v1/support", {
    method: "POST",
    body: JSON.stringify({
      subject,
      body,
      category: options?.category,
      priority: options?.priority,
      service_kind: options?.service_kind,
      service_id: options?.service_id,
    }),
  });
}

// Отделы поддержки настраиваются для каждой панели, поэтому список приходит с
// сервера, а не зашит в интерфейс.
export type SupportService = { kind: string; id: string; label: string };

export async function fetchSupportCreateForm() {
  return apiFetch<{
    departments: { id: string; name: string }[];
    services?: SupportService[];
    // Замеренное среднее время первого ответа по отделам за неделю. Пусто,
    // если отвечать ещё не приходилось.
    sla?: { category: string; minutes: number }[];
  }>("/v1/support/create");
}

export async function setTicketNotify(id: string, notify: boolean) {
  return apiFetch<{ notify: boolean }>(`/v1/support/${id}/notify`, {
    method: "PATCH",
    body: JSON.stringify({ notify }),
  });
}

export async function fetchKBArticles(params?: { category?: string; q?: string }) {
  const qs = new URLSearchParams();
  if (params?.category) qs.set("category", params.category);
  if (params?.q) qs.set("q", params.q);
  const suffix = qs.toString() ? `?${qs}` : "";
  return apiFetch<{ articles: KBArticle[] }>(`/v1/kb${suffix}`);
}

export async function fetchKBArticle(slug: string) {
  return apiFetch<KBArticle>(`/v1/kb/${encodeURIComponent(slug)}`);
}

export async function fetchAdminKBArticles() {
  return apiFetch<{ articles: KBArticle[] }>("/v1/admin/kb");
}

export async function saveAdminKBArticle(article: Partial<KBArticle>) {
  return apiFetch<{ id: string; slug: string }>("/v1/admin/kb", {
    method: "POST",
    body: JSON.stringify(article),
  });
}

export async function deleteAdminKBArticle(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/kb/${id}`, { method: "DELETE" });
}

export async function fetchSupportUnreadCount() {
  return apiFetch<{ count: number }>("/v1/support/unread-count");
}

export async function fetchSupportTicket(id: string) {
  return apiFetch<{ ticket_id: string; messages: SupportMessage[] }>(
    `/v1/support/${id}`
  );
}

export async function replySupportTicket(id: string, body: string) {
  return apiFetch<{ status: string }>(`/v1/support/${id}/reply`, {
    method: "POST",
    body: JSON.stringify({ body }),
  });
}

export async function uploadSupportAttachment(ticketId: string, file: File | File[]) {
  const token = getAccessToken();
  const form = new FormData();
  // Поле одно и то же для одного файла и для нескольких: сервер читает список.
  for (const f of Array.isArray(file) ? file : [file]) {
    form.append("file", f);
  }
  const res = await fetch(`${API_URL}/v1/support/${ticketId}/attachments`, {
    method: "POST",
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: form,
  });
  const text = await res.text();
  let data: { error?: string; message?: string } = {};
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {}
  }
  if (!res.ok) {
    throw new Error(
      data.error || data.message || t("errors.support.upload_failed", { code: res.status })
    );
  }
  return data as unknown as {
    status: string;
    message_id: string;
    filename: string;
    size: number;
    url: string;
    attachments?: SupportAttachment[];
  };
}

export async function downloadSupportAttachment(
  ticketId: string,
  messageId: string,
  fallbackName: string
) {
  const token = getAccessToken();
  const res = await fetch(
    `${API_URL}/v1/support/${ticketId}/attachments/${messageId}`,
    { headers: token ? { Authorization: `Bearer ${token}` } : {} }
  );
  if (!res.ok) {
    throw new Error(t("errors.support.attachment_download_failed"));
  }
  const blob = await res.blob();
  const blobUrl = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = blobUrl;
  a.download = fallbackName;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(blobUrl);
}

export async function closeSupportTicket(id: string) {
  return apiFetch<{ status: string }>(`/v1/support/${id}/close`, {
    method: "POST",
  });
}

export type NewsItem = {
  slug: string;
  title: string;
  excerpt?: string | null;
  published_at?: string | null;
  body?: string;
  image?: string | null;
  tag?: string;
  pinned?: boolean;
  // Считается сравнением даты публикации с отметкой пользователя, а не
  // хранится по каждой паре «читатель-новость».
  unread?: boolean;
};

export async function fetchNews() {
  return apiFetch<{ news: NewsItem[] }>("/v1/news");
}

export async function fetchNewsArticle(slug: string) {
  return apiFetch<NewsItem>(`/v1/news/${slug}`);
}

export async function markNewsRead() {
  return apiFetch<{ status: string }>("/v1/news/read", { method: "POST" });
}

export type MonitoringServerRow = {
  id: string;
  name: string;
  game_id: string;
  game_name: string;
  version: string;
  map: string;
  region: string;
  status: string;
  online: number;
  slots: number;
  ping: number;
  cpu: number;
  ram: number;
  ram_limit_mb: number;
  tps: number;
  uptime: number;
  ip: string;
  spark: number[];
  public_enabled: boolean;
  description: string;
  tags: string[];
  discord: string;
  website: string;
  votes: number;
};

export type MonitoringIndexResponse = {
  servers: MonitoringServerRow[];
  total: number;
  online: number;
  players: number;
  slots: number;
  avg_uptime: number;
  incidents_7d: number;
  updated_at: string;
};

export async function fetchMonitoring() {
  return apiFetch<MonitoringIndexResponse>("/v1/monitoring");
}

export type MonitoringPlayer = {
  name: string;
  score: number;
  ping: number;
  duration_sec: number;
};

export type MonitoringIncident = {
  id: number;
  title: string;
  level: "info" | "warn" | "bad";
  body: string;
  started_at: string;
  duration_sec: number;
  resolved: boolean;
};

export type MonitoringUptimeDay = {
  day: string;
  uptime: number;
  has_data: boolean;
};

export type MonitoringSettings = {
  public_enabled: boolean;
  title: string;
  description: string;
  tags: string[];
  discord: string;
  website: string;
  show_players: boolean;
  show_chart: boolean;
  show_incidents: boolean;
  show_address: boolean;
  show_version: boolean;
  votes: number;
};

export type MonitoringServerDetail = {
  server: MonitoringServerRow;
  players: MonitoringPlayer[];
  peak_24h: number;
  avg_24h: number;
  uptime_30d: number;
  uptime_days: MonitoringUptimeDay[];
  incidents: MonitoringIncident[];
  settings: MonitoringSettings;
  visits: { total: number; days: number[] };
  public_url: string;
  banner_url: string;
};

export async function fetchMonitoringServer(id: string) {
  return apiFetch<MonitoringServerDetail>(`/v1/monitoring/${id}`);
}

export type MonitoringTopItem = {
  rank: number;
  server_id: string;
  name: string;
  tagline: string;
  game_id: string;
  game_name: string;
  online: number;
  slots: number;
  uptime: number;
  votes: number;
  ip: string;
  spark: number[];
  tags: string[];
};

export async function fetchMonitoringTop(game?: string) {
  const qs = game ? `?game=${encodeURIComponent(game)}` : "";
  return apiFetch<{ items: MonitoringTopItem[]; games: string[] }>(`/v1/monitoring/top${qs}`);
}

export async function fetchPublicMonitoringTop(game?: string) {
  const qs = game ? `?game=${encodeURIComponent(game)}` : "";
  return apiFetch<{ items: MonitoringTopItem[]; games: string[] }>(
    `/v1/monitoring/public/top${qs}`
  );
}

export async function fetchMonitoringSettings(id: string) {
  return apiFetch<{
    settings: MonitoringSettings;
    public_url: string;
    banner_url: string;
  }>(`/v1/monitoring/${id}/settings`);
}

export async function updateMonitoringSettings(
  id: string,
  payload: Partial<MonitoringSettings>
) {
  return apiFetch<{ ok: boolean; settings: MonitoringSettings }>(
    `/v1/monitoring/${id}/settings`,
    { method: "PUT", body: JSON.stringify(payload) }
  );
}

export async function fetchMonitoringIncidents(id: string) {
  return apiFetch<{ items: MonitoringIncident[]; uptime_days: MonitoringUptimeDay[] }>(
    `/v1/monitoring/${id}/incidents`
  );
}

export type PublicMonitoringResponse = {
  server: MonitoringServerRow;
  settings: MonitoringSettings;
  players: MonitoringPlayer[];
  peak_24h: number;
  avg_24h: number;
  incidents: MonitoringIncident[];
  uptime_days: MonitoringUptimeDay[];
  show_chart: boolean;
  banner_url: string;
  public_url: string;
};

export async function fetchPublicMonitoringServer(id: string) {
  return apiFetch<PublicMonitoringResponse>(`/v1/monitoring/public/${id}`);
}

export async function voteForMonitoringServer(id: string) {
  return apiFetch<{ ok: boolean; votes: number }>(`/v1/monitoring/public/${id}/vote`, {
    method: "POST",
  });
}

export async function fetchPublicMonitoringLive(id: string) {
  return apiFetch<{ ok: boolean; item: { online?: number | null; max?: number | null; metrics?: Record<string, unknown> | null } }>(
    `/v1/monitoring/public/${id}/live`
  );
}

export async function fetchPublicMonitoringStats(id: string, days: number) {
  return apiFetch<{ ok: boolean; days: number; series: MonitoringStatsSeries }>(
    `/v1/monitoring/public/${id}/stats?days=${days}`
  );
}

export type RentPromoPreview = {
  valid: boolean;
  error?: string | null;
  discount?: number;
  base_cost?: number;
  final_cost?: number;
  promo_code?: string;
};

export type RentQuoteResponse = RentCatalog & {
  calculated_cost?: number;
  base_cost?: number;
  order?: Record<string, unknown>;
  promo_preview?: RentPromoPreview;
};

export type RentNode = {
  id: string;
  name: string;
  country?: string;
  code?: string;
  is_online?: boolean;
  servers_count?: number;
};

export type RentTariff = {
  id: string;
  name: string;
  slug?: string;
  game_id?: string;
  location_id?: string;
  price_monthly?: number;
  currency?: string;
  slots?: number | null;
  slots_min?: number | null;
  slots_max?: number | null;
  ram_mb?: number | null;
  disk_mb?: number | null;
  billing_type?: string;
  cpu_cores?: number | null;
  ram_gb?: number | null;
  disk_gb?: number | null;
  cpu_min?: number | null;
  cpu_max?: number | null;
  cpu_step?: number | null;
  ram_min?: number | null;
  ram_max?: number | null;
  ram_step?: number | null;
  disk_min?: number | null;
  disk_max?: number | null;
  disk_step?: number | null;
  price_per_cpu_core?: number | null;
  price_per_ram_gb?: number | null;
  price_per_disk_gb?: number | null;
  base_price_monthly?: number | null;
  rental_periods?: number[];
};

export type RentGameOption = {
  id: string;
  name: string;
  slug?: string;
  image?: string | null;
  min_price?: number | null;
  servers_count?: number;
};

export type RentGameVersionOption = {
  id: string;
  game_id: string;
  game_slug?: string;
  name: string;
  version?: string;
  source_type?: string;
};

export type RentCatalog = {
  games: RentGameOption[];
  tariffs: RentTariff[];
  nodes: RentNode[];
  game_versions?: RentGameVersionOption[];
};

export async function fetchRentServerForm() {
  return apiFetch<{
    games: {
      id: string;
      name: string;
      slug?: string;
      min_price?: number | null;
      servers_count?: number;
    }[];
    tariffs: RentTariff[];
    nodes: RentNode[];
    game_versions?: RentGameVersionOption[];
    calculated_cost?: number;
    base_cost?: number;
    promo_preview?: RentPromoPreview;
  }>("/v1/rent-server");
}

export async function fetchRentQuote(params: Record<string, string>): Promise<RentQuoteResponse> {
  const qs = new URLSearchParams(params).toString();
  const endpoint = qs ? `/v1/rent-server?${qs}` : "/v1/rent-server";
  return apiFetch<RentQuoteResponse>(endpoint);
}

export async function fetchRentCatalog(): Promise<RentCatalog> {
  const [form, tariffsRes, gamesRes] = await Promise.all([
    fetchRentServerForm(),
    fetchTariffsPublic(),
    fetchGames(),
  ]);
  const tariffById = new Map(
    (tariffsRes.tariffs as RentTariff[]).map((t) => [t.id, t])
  );
  const gameById = new Map(gamesRes.games.map((g) => [g.id, g]));
  return {
    games: form.games.map((g) => {
      const full = gameById.get(g.id);
      return {
        id: g.id,
        name: g.name,
        slug: full?.slug,
        image: full?.image ?? null,
        min_price: g.min_price ?? null,
        servers_count: g.servers_count ?? 0,
      };
    }),
    tariffs: form.tariffs.map((t) => ({
      ...t,
      ...(tariffById.get(t.id) ?? {}),
      id: t.id,
      name: t.name,
    })),
    nodes: form.nodes,
    game_versions: form.game_versions ?? [],
  };
}

export async function submitRentServer(data: {
  node_id: string;
  game_id?: string;
  game_version_id?: string;
  tariff_id?: string;
  name: string;
  period?: number;
  slots?: number;
  cpu_cores?: number;
  ram_gb?: number;
  disk_gb?: number;
  promo_code?: string;
  wallet_id?: string;
}) {
  return apiFetch<{ id: string; server_id: string; status: string }>("/v1/rent-server", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export type ServerFtpAccount = {
  id: string;
  username: string;
  password: string;
  status: "pending" | "active" | "failed" | "deleting";
  error_message?: string;
};

export type ServerFtpInfo = {
  accounts: ServerFtpAccount[];
  host: string;
  port: number;
  limit: number;
  root: string;
};

export async function fetchServerFtp(id: string) {
  return apiFetch<ServerFtpInfo>(`/v1/servers/${id}/ftp`);
}

export async function createServerFtpAccount(id: string, suffix = "") {
  return apiFetch<{ status: string; username: string; password: string }>(
    `/v1/servers/${id}/ftp`,
    { method: "POST", body: JSON.stringify({ suffix }) }
  );
}

export async function resetServerFtpPassword(id: string, username: string) {
  return apiFetch<{ status: string; username: string; password: string }>(
    `/v1/servers/${id}/ftp/reset-password`,
    { method: "POST", body: JSON.stringify({ username }) }
  );
}

export async function deleteServerFtpAccount(id: string, username: string) {
  return apiFetch<{ status: string; username: string }>(
    `/v1/servers/${id}/ftp/delete`,
    { method: "POST", body: JSON.stringify({ username }) }
  );
}

export async function reinstallServer(id: string) {
  return apiFetch<{ status: string }>(`/v1/servers/${id}/reinstall`, { method: "POST" });
}

export async function updateServerGame(id: string) {
  return apiFetch<{ status: string }>(`/v1/servers/${id}/update`, { method: "POST" });
}

export async function setServerAutoStart(id: string, enabled: boolean) {
  return apiFetch<{ auto_start: boolean }>(`/v1/servers/${id}/auto-start`, {
    method: "POST",
    body: JSON.stringify({ enabled }),
  });
}

export async function saveServerStartupParams(id: string, startupParams: string) {
  return apiFetch<{ status: string }>(`/v1/servers/${id}/startup-params`, {
    method: "POST",
    body: JSON.stringify({ startup_params: startupParams }),
  });
}

export async function switchServerVersion(id: string, versionId: string) {
  return apiFetch<{ status: string }>(`/v1/servers/${id}/version`, {
    method: "POST",
    body: JSON.stringify({ version_id: versionId }),
  });
}

export async function previewServerRenew(id: string, period: number, promoCode?: string) {
  return apiFetch<{
    period_days: number;
    price: number;
    currency: string;
    base_cost?: number;
    promo_preview?: RentPromoPreview;
  }>(`/v1/servers/${id}/renew/preview`, {
    method: "POST",
    body: JSON.stringify({ period, promo_code: promoCode ?? "" }),
  });
}

export async function renewServer(
  id: string,
  period: number,
  walletId?: string,
  promoCode?: string
) {
  return apiFetch<{ status: string }>(`/v1/servers/${id}/renew`, {
    method: "POST",
    body: JSON.stringify({ period, wallet_id: walletId ?? "", promo_code: promoCode ?? "" }),
  });
}

export type TariffOption = {
  id: string | number;
  name: string;
  billing_type?: string;
  currency?: string;
  base_price_monthly?: number;
  price_per_slot?: number;
  price_per_cpu_core?: number;
  price_per_ram_gb?: number;
  price_per_disk_gb?: number;
  min_slots?: number;
  max_slots?: number;
  cpu_min?: number;
  cpu_max?: number;
  cpu_step?: number;
  ram_min?: number;
  ram_max?: number;
  ram_step?: number;
  disk_min?: number;
  disk_max?: number;
  disk_step?: number;
};

export async function fetchServerTariffs(serverId: string) {
  return apiFetch<{ tariffs: TariffOption[] }>(`/v1/servers/${serverId}/tariffs`);
}

export async function changeServerTariff(serverId: string, tariffId: string, walletId?: string) {
  return apiFetch<{ status: string; charged?: number; currency?: string }>(
    `/v1/servers/${serverId}/tariff/change`,
    {
      method: "POST",
      body: JSON.stringify({ tariff_id: tariffId, wallet_id: walletId ?? "" }),
    }
  );
}

export async function previewServerTariffChange(serverId: string, tariffId: string) {
  return apiFetch<{
    ok: boolean;
    charged: number;
    currency: string;
    days_left: number;
    new_expires_at?: string | null;
    target_resources?: Record<string, unknown>;
    current_resources?: Record<string, unknown>;
  }>(`/v1/servers/${serverId}/tariff/change/preview`, {
    method: "POST",
    body: JSON.stringify({ tariff_id: tariffId }),
  });
}

export async function changeServerTariffResources(
  serverId: string,
  payload: { cpu_cores: number; ram_gb: number; disk_gb: number; slots: number },
  walletId?: string
) {
  return apiFetch<{ status: string; charged?: number; currency?: string }>(
    `/v1/servers/${serverId}/tariff/resources`,
    {
      method: "POST",
      body: JSON.stringify({ ...payload, wallet_id: walletId ?? "" }),
    }
  );
}

export async function previewServerTariffResources(
  serverId: string,
  payload: { cpu_cores: number; ram_gb: number; disk_gb: number; slots: number }
) {
  return apiFetch<{
    ok: boolean;
    charged: number;
    currency: string;
    days_left: number;
    new_expires_at?: string | null;
    target_resources?: Record<string, unknown>;
    current_resources?: Record<string, unknown>;
  }>(`/v1/servers/${serverId}/tariff/resources/preview`, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export type MonitoringStatsSeries = {
  labels: string[];
  online: (number | null)[];
  max: (number | null)[];
  cpu: (number | null)[];
  ram: (number | null)[];
  disk: (number | null)[];
  max_cap: number | null;
};

export async function fetchServerMonitoringStats(serverId: string, days: number) {
  return apiFetch<{ ok: boolean; days: number; series: MonitoringStatsSeries }>(
    `/v1/monitoring/${serverId}/stats?days=${days}`
  );
}

// Схема настроек приходит с бэкенда вместе со значениями: набор полей больше не
// описан на фронте, поэтому и адрес не содержит слага игры — какая это игра,
// сервер знает сам.
export async function fetchServerSettings(serverId: string) {
  return apiFetch<ServerSettingsSchema>(`/v1/servers/${serverId}/settings`);
}

// Отправляем ТОЛЬКО изменённые поля. Раньше форма слала все ключи разом, и
// пустое значение поля оседало в базе, навсегда закрывая собой то, что реально
// лежит в конфиге.
export async function saveServerSettings(serverId: string, values: Record<string, string>) {
  return apiFetch<SaveSettingsResponse>(`/v1/servers/${serverId}/settings`, {
    method: "POST",
    body: JSON.stringify({ values }),
  });
}

export async function fetchServerSettingsFile(serverId: string, fileId: string) {
  return apiFetch<SettingsFileContent>(`/v1/servers/${serverId}/settings/files/${fileId}`);
}

// sha256Base — содержимое, которое видел редактор. Если на ноде оно изменилось
// (игра переписала конфиг при старте, пока вкладка была открыта), сервер ответит
// 409, а не затрёт чужие правки молча.
export async function saveServerSettingsFile(
  serverId: string,
  fileId: string,
  content: string,
  sha256Base: string
) {
  return apiFetch<{ ok: boolean; sha256: string; backup?: string; restart_required: boolean; server_running: boolean }>(
    `/v1/servers/${serverId}/settings/files/${fileId}`,
    { method: "PUT", body: JSON.stringify({ content, sha256_base: sha256Base }) }
  );
}

export type HostingPlanSummary = {
  id: string;
  name: string;
  slug?: string;
  price_monthly: number;
  disk_mb?: number;
  disk_gb?: number;
  description?: string | null;
  has_ssl?: boolean;
  has_ssh?: boolean;
  has_cron?: boolean;
  has_backup?: boolean;
  rental_periods?: number[];
  discounts?: Record<string, number>;
  hosting_server?: { panel_type_label?: string } | null;
  features?: {
    sites?: number | null;
    databases?: number | null;
    email_domains?: number | null;
    ftp_accounts?: number | null;
  } | null;
};

export type HostingAccount = {
  id: string;
  username: string;
  domain?: string | null;
  primary_domain?: string | null;
  status: string;
  status_label?: string;
  status_color?: string;
  ip_address?: string | null;
  expires_at?: string | null;
  panel_password?: string | null;
  panel_login_url?: string | null;
  created_at?: string | null;
  hosting_plan?: {
    name: string;
    disk_gb?: number;
    disk_mb?: number;
    bandwidth_gb?: number;
    max_domains?: number;
    max_databases?: number;
    max_email_accounts?: number;
    has_ssl?: boolean;
    has_ssh?: boolean;
    has_cron?: boolean;
    has_backup?: boolean;
  } | null;
  hosting_server?: { panel_type_label?: string } | null;
  domains?: { id: string | number; name: string; status?: string; created_at?: string | null; is_primary?: boolean }[];
  databases?: { id: string | number; name: string; user?: string; status?: string; created_at?: string | null }[];
  emails?: { id: string | number; address: string; status?: string; created_at?: string | null }[];
};

export type HostingRentFormData = {
  plans: HostingPlanSummary[];
  wallets?: Wallet[];
};

export async function fetchMyHosting() {
  return apiFetch<{ accounts: HostingAccount[] }>("/v1/hosting/my");
}

export async function fetchHostingRentForm(): Promise<HostingRentFormData> {
  return apiFetch<HostingRentFormData>("/v1/hosting/rent");
}

export async function submitHostingRent(data: {
  plan_id: string;
  domain: string;
  period?: number;
  wallet_id?: string;
}) {
  // Домен и период отправляются отдельными полями: сервер принимает оба, но
  // получал только username. Из-за этого primary_domain у аккаунта оставался
  // пустым, а срок всегда выставлялся в 30 дней — сколько бы владелец ни
  // выбрал и ни увидел в итоге к оплате.
  return apiFetch<{ id: string }>("/v1/hosting/rent", {
    method: "POST",
    body: JSON.stringify({
      plan_id: data.plan_id,
      username: data.domain,
      domain: data.domain,
      period: data.period,
    }),
  });
}

export async function fetchHostingAccount(id: string): Promise<HostingAccount> {
  const res = await apiFetch<{ account?: HostingAccount } & HostingAccount>(
    `/v1/hosting/${id}`
  );
  return res.account ?? res;
}

export async function createHostingDomain(id: string, domain: string) {
  return apiFetch<{ id: string }>(`/v1/hosting/${id}/domains`, {
    method: "POST",
    body: JSON.stringify({ domain }),
  });
}

export async function createHostingDatabase(id: string, name: string, user?: string) {
  return apiFetch<{ id: string }>(`/v1/hosting/${id}/databases`, {
    method: "POST",
    body: JSON.stringify({ name, user: user || "" }),
  });
}

export async function createHostingEmail(id: string, address: string, password: string) {
  return apiFetch<{ id: string }>(`/v1/hosting/${id}/emails`, {
    method: "POST",
    body: JSON.stringify({ address, password }),
  });
}

export async function renewHostingAccount(id: string, period?: number) {
  return apiFetch<{ status: string }>(`/v1/hosting/${id}/renew`, {
    method: "POST",
    body: JSON.stringify({ period: period || 30 }),
  });
}

export async function changeHostingPassword(id: string, password: string) {
  return apiFetch<{ status: string }>(`/v1/hosting/${id}/change-password`, {
    method: "POST",
    body: JSON.stringify({ password }),
  });
}

export type Game = {
  id: string;
  slug: string;
  name: string;
  description?: string | null;
  image?: string | null;
  active?: boolean;
};

export type Feature = {
  title: string;
  description: string;
};

export async function fetchGames() {
  return apiFetch<{ games: Game[] }>("/v1/games");
}

export async function fetchGamesPublic() {
  const { games } = await fetchGames();
  return games;
}

export async function fetchGame(slug: string) {
  return apiFetch<Game>(`/v1/games/${slug}`);
}

export async function fetchFeatures() {
  return apiFetch<{ features: Feature[] }>("/v1/features");
}

export async function fetchTariffsPublic() {
  return apiFetch<{ tariffs: unknown[] }>("/v1/tariffs/public");
}

// Хвост запрашиваем явно: API принимает tail, но раньше его никто не слал, и
// журнал всегда обрывался на двухстах строках — для запуска игрового сервера
// это середина, без начала и без свежих событий.
export async function fetchServerLogs(serverId: string, tail = 1000) {
  return apiFetch<{ lines: string[] }>(
    `/v1/servers/${serverId}/logs?tail=${encodeURIComponent(String(tail))}`
  );
}

export type ServerInstallLog = {
  lines: string[];
  progress?: {
    percent?: number;
    stage?: string;
    message?: string;
    derived?: boolean;
  };
};

export async function fetchServerInstallLog(serverId: string) {
  return apiFetch<ServerInstallLog>(`/v1/servers/${serverId}/install-log`);
}

export async function fetchServerStatus(serverId: string) {
  return apiFetch<{
    ok: boolean;
    online: boolean;
    max_players: number;
    online_players: number;
    current_map?: string;
    players_online: { name: string; score?: number; ping?: number }[];
    runtime_status: string;
    provisioning_progress?: {
      percent?: number | string;
      message?: string;
      stage?: string;
    } | null;
    uptime?: string;
    started_at?: string;
    disk_used_mb?: number;
    disk_total_mb?: number;
  }>(`/v1/servers/${serverId}/status`);
}

export async function sendServerConsoleCommand(serverId: string, command: string) {
  return apiFetch<{ ok: boolean }>(`/v1/servers/${serverId}/console-command`, {
    method: "POST",
    body: JSON.stringify({ command }),
  });
}

export async function listServerMapsFolder(serverId: string) {
  return apiFetch<{ maps: string[] }>(`/v1/servers/${serverId}/maps-folder/list`, {
    method: "POST",
    body: JSON.stringify({}),
  });
}

export async function changeServerMap(serverId: string, map: string) {
  return apiFetch<{ ok: boolean; map: string }>(`/v1/servers/${serverId}/maps-folder/change`, {
    method: "POST",
    body: JSON.stringify({ map }),
  });
}

export async function fetchServerPlugins(serverId: string) {
  return apiFetch<{ items: ServerPluginItem[] }>(`/v1/servers/${serverId}/plugins`);
}

export type ServerPluginItem = {
  plugin: {
    id: string;
    slug: string;
    name: string;
    category?: string;
    version?: string;
    description?: string;
    image_url?: string;
    install_path?: string;
  };
  server_plugin: { installed: boolean; enabled: boolean; last_error?: string };
};

export async function installServerPlugin(serverId: string, pluginId: string) {
  return apiFetch<{ ok: boolean }>(`/v1/servers/${serverId}/plugins/${pluginId}/install`, {
    method: "POST",
    body: JSON.stringify({}),
  });
}

export async function uninstallServerPlugin(serverId: string, pluginId: string) {
  return apiFetch<{ ok: boolean }>(`/v1/servers/${serverId}/plugins/${pluginId}/uninstall`, {
    method: "POST",
    body: JSON.stringify({}),
  });
}

export async function toggleServerPlugin(serverId: string, pluginId: string, enabled: boolean) {
  return apiFetch<{ ok: boolean }>(`/v1/servers/${serverId}/plugins/${pluginId}/toggle`, {
    method: "POST",
    body: JSON.stringify({ enabled: enabled ? 1 : 0 }),
  });
}

export async function fetchServerMaps(serverId: string) {
  return apiFetch<{ items: ServerMapItem[] }>(`/v1/servers/${serverId}/maps`);
}

export type ServerMapItem = {
  map: { id: string; slug: string; name: string; category?: string; version?: string };
  server_map: { installed: boolean; is_active: boolean; last_error?: string };
  is_active?: boolean;
};

export async function installServerMap(serverId: string, mapId: string) {
  return apiFetch<{ ok: boolean }>(`/v1/servers/${serverId}/maps/${mapId}/install`, {
    method: "POST",
    body: JSON.stringify({}),
  });
}

export async function uninstallServerMap(serverId: string, mapId: string) {
  return apiFetch<{ ok: boolean }>(`/v1/servers/${serverId}/maps/${mapId}/uninstall`, {
    method: "POST",
    body: JSON.stringify({}),
  });
}

export async function activateServerMap(serverId: string, mapId: string) {
  return apiFetch<{ ok: boolean }>(`/v1/servers/${serverId}/maps/${mapId}/activate`, {
    method: "POST",
    body: JSON.stringify({}),
  });
}

export type ServerMysqlCatalog = {
  instance_key?: string;
  container?: string;
  port?: number;
  databases?: string[];
  users?: Array<{ username: string; host: string }>;
};

export type ServerMysqlInfo = Record<string, unknown> & {
  catalog?: ServerMysqlCatalog;
};

export async function fetchServerMysql(serverId: string) {
  return apiFetch<ServerMysqlInfo>(`/v1/servers/${serverId}/mysql`);
}

export async function resetServerMysqlPassword(serverId: string, mysqlInstanceKey?: string) {
  return apiFetch<{ ok: boolean }>(`/v1/servers/${serverId}/mysql/reset-password`, {
    method: "POST",
    body: JSON.stringify({ mysql_instance_key: mysqlInstanceKey }),
  });
}

export async function deleteServerMysql(serverId: string) {
  return apiFetch<{ ok: boolean }>(`/v1/servers/${serverId}/mysql/delete`, {
    method: "POST",
    body: JSON.stringify({}),
  });
}

export async function migrateServerMysql(serverId: string, targetKey: string) {
  return apiFetch<{ ok: boolean }>(`/v1/servers/${serverId}/mysql/migrate`, {
    method: "POST",
    body: JSON.stringify({ target_mysql_instance_key: targetKey }),
  });
}

export async function createServerMysqlDatabase(serverId: string, database: string) {
  return apiFetch<{ ok: boolean }>(`/v1/servers/${serverId}/mysql/database/create`, {
    method: "POST",
    body: JSON.stringify({ database }),
  });
}

export async function deleteServerMysqlDatabase(serverId: string, database: string) {
  return apiFetch<{ ok: boolean }>(`/v1/servers/${serverId}/mysql/database/delete`, {
    method: "POST",
    body: JSON.stringify({ database }),
  });
}

export async function createServerMysqlUser(serverId: string, username: string, password: string, database?: string) {
  return apiFetch<{ ok: boolean }>(`/v1/servers/${serverId}/mysql/user/create`, {
    method: "POST",
    body: JSON.stringify({ username, password, database }),
  });
}

export async function deleteServerMysqlUser(serverId: string, username: string) {
  return apiFetch<{ ok: boolean }>(`/v1/servers/${serverId}/mysql/user/delete`, {
    method: "POST",
    body: JSON.stringify({ username }),
  });
}

export async function resetServerMysqlUserPassword(serverId: string, username: string, password: string) {
  return apiFetch<{ ok: boolean }>(`/v1/servers/${serverId}/mysql/user/reset-password`, {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
}

export async function fetchServerCron(serverId: string) {
  return apiFetch<{
    jobs: { id: string; schedule: string; command: string; enabled: boolean }[];
  }>(`/v1/servers/${serverId}/cron/list`, { method: "POST" });
}

export async function createServerCronJob(serverId: string, schedule: string, command: string) {
  return apiFetch<{ id: string }>(`/v1/servers/${serverId}/cron/create`, {
    method: "POST",
    body: JSON.stringify({ schedule, command }),
  });
}

export async function deleteServerCronJob(serverId: string, jobId: string) {
  return apiFetch<{ status: string }>(`/v1/servers/${serverId}/cron/delete`, {
    method: "POST",
    body: JSON.stringify({ id: jobId }),
  });
}

export async function toggleServerCronJob(serverId: string, jobId: string, enabled: boolean) {
  return apiFetch<{ status: string }>(`/v1/servers/${serverId}/cron/toggle`, {
    method: "POST",
    body: JSON.stringify({ id: jobId, enabled }),
  });
}

export async function fetchServerFirewall(serverId: string) {
  return apiFetch<{
    rules: {
      id: string;
      protocol: string;
      port_from: number;
      enabled: boolean;
    }[];
  }>(`/v1/servers/${serverId}/firewall/list`, { method: "POST" });
}

export async function createServerFirewallRule(
  serverId: string,
  protocol: string,
  portFrom: number
) {
  return apiFetch<{ id: string }>(`/v1/servers/${serverId}/firewall/create`, {
    method: "POST",
    body: JSON.stringify({ protocol, port_from: portFrom }),
  });
}

export async function deleteServerFirewallRule(serverId: string, ruleId: string) {
  return apiFetch<{ status: string }>(`/v1/servers/${serverId}/firewall/delete`, {
    method: "POST",
    body: JSON.stringify({ id: ruleId }),
  });
}

export async function toggleServerFirewallRule(
  serverId: string,
  ruleId: string,
  enabled: boolean
) {
  return apiFetch<{ status: string }>(`/v1/servers/${serverId}/firewall/toggle`, {
    method: "POST",
    body: JSON.stringify({ id: ruleId, enabled }),
  });
}

export async function fetchServerFriends(serverId: string) {
  return apiFetch<{ friends: unknown[] }>(`/v1/servers/${serverId}/friends`);
}

export async function addServerFriend(serverId: string, email: string) {
  return apiFetch<{ status: string }>(`/v1/servers/${serverId}/friends/add`, {
    method: "POST",
    body: JSON.stringify({ email }),
  });
}

export async function removeServerFriend(serverId: string, friendId: string) {
  return apiFetch<{ status: string }>(`/v1/servers/${serverId}/friends/remove`, {
    method: "POST",
    body: JSON.stringify({ id: friendId }),
  });
}

export async function updateServerFriendPermissions(
  serverId: string,
  userId: string,
  permissions: ServerViewerPermissions
) {
  return apiFetch<{ status: string }>(`/v1/servers/${serverId}/friends/update`, {
    method: "POST",
    body: JSON.stringify({ user_id: userId, ...permissions }),
  });
}

export async function fetchServerDetail(id: string) {
  return apiFetch<DashboardServer & { game_id: string }>(`/v1/servers/${id}/detail`);
}

export async function fetchServerBackups(serverId: string) {
  return apiFetch<{ backups: BackupEntry[]; entries: BackupEntry[] }>(
    `/v1/servers/${serverId}/backups`
  );
}

export async function createServerBackup(serverId: string, name?: string) {
  return apiFetch<{ backup_id: string; status: string; filename: string }>(
    `/v1/servers/${serverId}/backups`,
    { method: "POST", body: JSON.stringify({ name: name ?? "" }) }
  );
}

export type ServerBackupSchedule = {
  enabled: boolean;
  frequency: "daily" | "weekly";
  hour_utc: number;
  day_of_week: number;
  keep_count: number;
  last_run_at?: string | null;
  last_error?: string | null;
};

export async function fetchServerBackupSchedule(serverId: string) {
  return apiFetch<ServerBackupSchedule>(`/v1/servers/${serverId}/backup-schedule`);
}

export async function saveServerBackupSchedule(
  serverId: string,
  data: Omit<ServerBackupSchedule, "last_run_at" | "last_error">
) {
  return apiFetch<ServerBackupSchedule>(
    `/v1/servers/${serverId}/backup-schedule`,
    { method: "PUT", body: JSON.stringify(data) }
  );
}

export async function deleteServerBackup(serverId: string, name: string) {
  return apiFetch<{ status: string }>(`/v1/servers/${serverId}/backups`, {
    method: "DELETE",
    body: JSON.stringify({ name }),
  });
}

export async function restoreServerBackup(serverId: string, name: string) {
  return apiFetch<{ status: string }>(`/v1/servers/${serverId}/backups/restore`, {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

export type BackupEntry = {
  name: string;
  is_dir: boolean;
  size: number;
};

export async function listServerFiles(serverId: string, path = "/") {
  return apiFetch<{ files: unknown[] }>(`/v1/servers/${serverId}/files/list`, {
    method: "POST",
    body: JSON.stringify({ path }),
  });
}

export async function readServerFile(serverId: string, path: string) {
  return apiFetch<{ path: string; content: string }>(
    `/v1/servers/${serverId}/files/read`,
    { method: "POST", body: JSON.stringify({ path }) }
  );
}

export async function writeServerFile(serverId: string, path: string, content: string) {
  return apiFetch<{ status: string }>(`/v1/servers/${serverId}/files/write`, {
    method: "POST",
    body: JSON.stringify({ path, content }),
  });
}

export async function mkdirServerDir(serverId: string, path: string) {
  return apiFetch<{ status: string }>(`/v1/servers/${serverId}/files/mkdir`, {
    method: "POST",
    body: JSON.stringify({ path }),
  });
}

export async function deleteServerPath(serverId: string, path: string) {
  return apiFetch<{ status: string }>(`/v1/servers/${serverId}/files/delete`, {
    method: "POST",
    body: JSON.stringify({ path }),
  });
}

export async function uploadServerFile(
  serverId: string,
  path: string,
  file: File,
  onProgress?: (percent: number) => void
) {
  const url = `${API_URL}/v1/servers/${serverId}/files/upload`;
  const token = getAccessToken();
  const form = new FormData();
  form.append("path", path);
  form.append("file", file);
  return new Promise<{ status: string; path?: string; size_bytes?: number }>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", url);
    if (token) xhr.setRequestHeader("Authorization", `Bearer ${token}`);
    xhr.upload.onprogress = (evt) => {
      if (!evt.lengthComputable || !onProgress) return;
      const percent = Math.max(0, Math.min(100, Math.round((evt.loaded / evt.total) * 100)));
      onProgress(percent);
    };
    xhr.onerror = () => reject(new Error("Upload failed"));
    xhr.onload = () => {
      let data: any = {};
      try {
        data = xhr.responseText ? JSON.parse(xhr.responseText) : {};
      } catch {
        data = {};
      }
      if (xhr.status < 200 || xhr.status >= 300) {
        reject(new Error(data.error ?? "Upload failed"));
        return;
      }
      resolve(data as { status: string; path?: string; size_bytes?: number });
    };
    xhr.send(form);
  });
}

export async function downloadServerFile(serverId: string, path: string, fallbackName?: string) {
  const token = getAccessToken();
  const url = `${API_URL}/v1/servers/${serverId}/files/download?path=${encodeURIComponent(path)}`;
  const res = await fetch(url, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) {
    let errMsg = "Download failed";
    try {
      const payload = await res.json();
      errMsg = payload.error ?? errMsg;
    } catch {}
    throw new Error(errMsg);
  }
  const blob = await res.blob();
  const cd = res.headers.get("Content-Disposition") ?? "";
  const matched = cd.match(/filename="([^"]+)"/i);
  const fileName = matched?.[1] || fallbackName || path.split("/").pop() || "download.bin";
  const blobUrl = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = blobUrl;
  a.download = fileName;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(blobUrl);
}

export type ServerPortEntry = {
  id: string;
  port: number;
  protocol: "tcp" | "udp" | "both";
  purpose?: string;
  is_primary?: boolean;
  created_at?: string;
};

export async function fetchServerPorts(serverId: string) {
  return apiFetch<{ ports: ServerPortEntry[]; primary_port?: number }>(`/v1/servers/${serverId}/ports/list`, {
    method: "POST",
    body: JSON.stringify({}),
  });
}

export async function createServerPort(
  serverId: string,
  payload: { port: number; protocol: "tcp" | "udp" | "both"; purpose?: string }
) {
  return apiFetch<{ id: string }>(`/v1/servers/${serverId}/ports/create`, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function deleteServerPort(serverId: string, id: string) {
  return apiFetch<{ status: string }>(`/v1/servers/${serverId}/ports/delete`, {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function adminFetch<T>(path: string): Promise<T> {
  const p = path.startsWith("/v1")
    ? path
    : `/v1/admin${path.startsWith("/") ? path : `/${path}`}`;
  return apiFetch<T>(p);
}

export type AdminDashboardStats = {
  servers: number;
  nodes: number;
  online_nodes: number;
  users: number;
  revenue_24h?: number;
  revenue_7d?: number;
  revenue_30d?: number;
  provisioning_installing?: number;
  provisioning_failed?: number;
  expiring_count?: number;
  failed_count?: number;
};

export async function fetchAdminDashboard() {
  return apiFetch<AdminDashboardStats>("/v1/admin/dashboard");
}

export type AdminDashboardNode = {
  id: string;
  name: string;
  country: string;
  code: string;
  fqdn: string;
  ip_address: string;
  status: string;
  daemon_status: string;
  is_online: boolean;
  servers_count: number;
  cpu_percent?: number;
  ram_percent?: number;
  ram_total?: string;
  disk_total?: string;
  disk_used?: string;
  measured_at?: string;
  daemon_last_seen_at?: string;
};

export async function fetchAdminDashboardNodes() {
  const res = await apiFetch<{ nodes: AdminDashboardNode[] }>(
    "/v1/admin/dashboard/nodes"
  );
  return res.nodes ?? [];
}

export type AdminLocationListItem = {
  id: string;
  node_id?: string;
  name: string;
  country: string;
  code: string;
  fqdn?: string;
  ip_address?: string;
  status?: string;
  node_status?: string;
  agent_status?: string;
  daemon_status?: string;
  is_online?: boolean;
  is_active: boolean;
  ssh_host?: string;
  ssh_user?: string;
  ssh_configured?: boolean;
  meta?: Record<string, unknown>;
  last_seen_at?: string;
  daemon_last_seen_at?: string;
  created_at?: string;
  servers_count: number;
  tariffs_count: number;
  maintenance_mode?: boolean;
  maintenance_reason?: string;
  maintenance_until?: string | null;
};

export type MysqlInstance = {
  key: string;
  engine: string;
  version: string;
  port: number;
  container: string;
  enabled: boolean;
};

export type AdminLocationDetail = {
  location: Record<string, unknown>;
  serverMetrics?: Record<string, unknown>;
  serviceStatuses?: Record<string, { state: string; error: string | null; label: string }>;
  metrics?: {
    cpu_usage?: { t?: string; v?: number; value?: number; measured_at?: string }[];
    ram_usage?: { t?: string; v?: number; value?: number; measured_at?: string }[];
  };
  cpuMetrics?: { t: string; v: number }[];
  ramMetrics?: { t: string; v: number }[];
  daemon?: Record<string, unknown>;
  metrics_stale?: boolean;
  sync_pending?: boolean;
};

export async function fetchAdminLocations() {
  return apiFetch<{ locations: AdminLocationListItem[] }>("/v1/admin/locations");
}

export async function fetchAdminLocation(id: string) {
  return apiFetch<AdminLocationDetail>(`/v1/admin/locations/${id}`);
}

export async function fetchAdminLocationEdit(id: string) {
  return apiFetch<{ location: Record<string, unknown> }>(
    `/v1/admin/locations/${id}/edit`
  );
}

export async function createAdminLocation(data: Record<string, unknown>) {
  return apiFetch<{ id: string; agent_token: string }>("/v1/admin/locations", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateAdminLocation(id: string, data: Record<string, unknown>) {
  return apiFetch<{ status: string; message?: string }>(`/v1/admin/locations/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function toggleAdminLocation(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/locations/${id}/toggle`, {
    method: "PATCH",
  });
}

export async function deleteAdminLocation(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/locations/${id}`, {
    method: "DELETE",
  });
}

export async function fetchAdminLocationSetup(id: string) {
  return apiFetch<{
    location: Record<string, unknown>;
    statuses: Record<string, string>;
    checks: { key: string; label: string; ok: boolean; hint?: string }[];
  }>(`/v1/admin/locations/${id}/setup`);
}

export async function fetchAdminLocationSetupStatus(id: string) {
  return apiFetch<{ log: string; completed: boolean; component?: string }>(
    `/v1/admin/locations/${id}/setup/status`
  );
}

export async function runAdminLocationSetupStep(id: string, step: string) {
  return apiFetch<{ success?: boolean; message?: string; status?: string }>(
    `/v1/admin/locations/${id}/setup/${step}`,
    { method: "POST" }
  );
}

export async function pullAdminLocationDaemon(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/locations/${id}/pull-daemon`, {
    method: "POST",
  });
}

export async function refreshAdminLocationDaemon(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/locations/${id}/daemon/refresh`, {
    method: "POST",
  });
}

export async function restartAdminLocationDaemon(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/locations/${id}/daemon/restart`, {
    method: "POST",
  });
}

export async function installAdminLocationDaemon(id: string) {
  return apiFetch<{ status: string; log?: string }>(`/v1/admin/locations/${id}/daemon`, {
    method: "POST",
  });
}

export async function fetchAdminLocationDaemon(id: string) {
  return apiFetch<Record<string, unknown>>(`/v1/admin/locations/${id}/daemon`);
}

export type AdminAgentListItem = {
  id: string;
  location_id: string;
  name?: string;
  code?: string;
  country?: string;
  region?: string;
  host?: string;
  status?: string;
  is_online?: boolean;
  version?: string;
  platform?: string;
  pid?: number;
  uptime_sec?: number;
  last_seen?: string;
  last_seen_human?: string;
  location?: { id: string; name: string; code: string; region?: string };
};

export async function fetchAdminAgents() {
  const res = await apiFetch<{ agents: AdminAgentListItem[]; daemons?: AdminAgentListItem[] }>(
    "/v1/admin/daemons"
  );
  return res.agents ?? res.daemons ?? [];
}

export async function fetchAdminAgent(id: string) {
  return apiFetch<{
    location: Record<string, unknown>;
    agent: Record<string, unknown>;
    daemon?: Record<string, unknown>;
    metrics?: {
      agent_cpu_usage?: { measured_at: string; value: number }[];
      agent_ram_usage?: { measured_at: string; value: number }[];
    };
  }>(`/v1/admin/daemons/${id}`);
}

export async function fetchAdminAgentLogs(id: string, tail = 200) {
  return apiFetch<{ logs: string[]; error?: string; hostname?: string; container_found?: boolean }>(
    `/v1/admin/daemons/${id}/logs?tail=${tail}`
  );
}

export async function fetchAdminAgentServers(id: string) {
  return apiFetch<{ servers: Record<string, unknown>[] }>(
    `/v1/admin/daemons/${id}/servers`
  );
}

export async function execAdminAgentCommand(id: string, cmd: string) {
  return apiFetch<{ stdout?: string; output?: string; error?: string; ok?: boolean }>(
    `/v1/admin/daemons/${id}/exec`,
    { method: "POST", body: JSON.stringify({ cmd }) }
  );
}

export async function refreshAdminAgent(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/daemons/${id}/refresh`, {
    method: "POST",
  });
}

export async function restartAdminAgent(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/daemons/${id}/restart`, {
    method: "POST",
  });
}

export async function installAdminAgent(id: string) {
  return apiFetch<{ status: string; log?: string }>(`/v1/admin/daemons/${id}/install`, {
    method: "POST",
  });
}

export async function fetchAdminGames() {
  return apiFetch<{ games: AdminGameListItem[] }>("/v1/admin/games");
}

export async function fetchAdminGame(id: string) {
  return apiFetch<{ game: AdminGameDetail }>(`/v1/admin/games/${id}`);
}

export async function fetchAdminGameEdit(id: string) {
  return apiFetch<{ game: AdminGameDetail }>(`/v1/admin/games/${id}/edit`);
}

export type AdminGameListItem = {
  id: string;
  name: string;
  slug: string;
  image?: string | null;
  is_active: boolean;
  is_visible: boolean;
  server_count?: number;
  tariff_count?: number;
  created_at?: string;
};

export type AdminGameDetail = AdminGameListItem & {
  description?: string | null;
  code?: string;
  query?: string;
  minport?: number;
  maxport?: number;
  default_startup_params?: string;
  status?: boolean;
  updated_at?: string;
  versions?: AdminGameVersion[];
};

export type AdminGameVersion = {
  id: string;
  name: string;
  version?: string;
  source_type: string;
  archive_url?: string;
  url?: string;
  docker_image?: string;
  steam_app_id?: number;
  steam_branch?: string;
  is_active: boolean;
  sort_order: number;
};

export async function createAdminGame(data: {
  name: string;
  slug: string;
  description?: string;
  code: string;
  query: string;
  minport?: number;
  maxport?: number;
  default_startup_params?: string;
  status?: boolean;
  is_active?: boolean;
}) {
  return apiFetch<{ ok: boolean; id: string }>("/v1/admin/games", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateAdminGame(
  id: string,
  data: Record<string, unknown>
) {
  return apiFetch<{ ok: boolean }>(`/v1/admin/games/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function deleteAdminGame(id: string) {
  return apiFetch<{ ok: boolean }>(`/v1/admin/games/${id}`, {
    method: "DELETE",
  });
}

export async function toggleAdminGame(id: string) {
  return apiFetch<{ ok: boolean; is_active: boolean }>(
    `/v1/admin/games/${id}/toggle`,
    { method: "POST" }
  );
}

export async function fetchAdminGameVersions(gameId: string) {
  return apiFetch<{ versions: AdminGameVersion[] }>(
    `/v1/admin/games/${gameId}/versions`
  );
}

export async function createAdminGameVersion(
  gameId: string,
  data: {
    name: string;
    source_type: "archive" | "steam" | "docker";
    url?: string;
    steam_app_id?: number;
    steam_branch?: string;
    docker_image?: string;
    is_active?: boolean;
    sort_order?: number;
  }
) {
  return apiFetch<{ ok: boolean; id: string }>(
    `/v1/admin/games/${gameId}/versions`,
    { method: "POST", body: JSON.stringify(data) }
  );
}

export async function deleteAdminGameVersion(gameId: string, versionId: string) {
  return apiFetch<{ status: string }>(
    `/v1/admin/games/${gameId}/versions/${versionId}`,
    { method: "DELETE" }
  );
}

export type Branding = {
  brand_name: string;
  logo_url: string;
  primary_color: string;
  // Палитра лендинга, набор его блоков и вариант меню пользователя. Настройки
  // задаются в админке, раньше до панели не доезжали и ни на что не влияли.
  colors?: Record<string, string>;
  blocks?: Record<string, boolean>;
  user_menu_variant?: "default" | "screenshot";
};

export async function fetchBranding() {
  const res = await fetch(`${API_URL}/v1/branding`);
  const data = await res.json();
  if (!res.ok) {
    throw new Error(data.error ?? "Request failed");
  }
  return data as Branding;
}

export type AdminTariff = {
  id: string;
  name: string;
  slug?: string;
  billing_type: string;
  is_available: boolean;
  position: number;
  min_slots: number;
  max_slots: number;
  cpu_cores: number;
  ram_gb: number;
  disk_gb: number;
  location_id?: string;
  game_id?: string;
  location?: { name: string } | null;
  game?: { name: string } | null;
  mysql_engine?: string | null;
  mysql_instance_key?: string | null;
  rental_periods?: number[];
  renewal_periods?: number[];
  price_per_slot?: number;
  price_per_cpu_core?: number;
  price_per_ram_gb?: number;
  price_per_disk_gb?: number;
  base_price_monthly?: number;
  cpu_min?: number | null;
  cpu_max?: number | null;
  cpu_step?: number | null;
  ram_min?: number | null;
  ram_max?: number | null;
  ram_step?: number | null;
  disk_min?: number | null;
  disk_max?: number | null;
  disk_step?: number | null;
  allow_antiddos?: boolean;
  antiddos_price?: number;
  cpu_shares?: number | null;
  discounts?: unknown;
  created_at?: string;
  updated_at?: string;
};

export type AdminTariffsPage = {
  data: AdminTariff[];
  current_page: number;
  last_page: number;
  per_page?: number;
  total?: number;
};

export type AdminTariffFormMeta = {
  locations: { id: string; name: string }[];
  games: { id: string; name: string }[];
};

export type AdminTariffsFilters = {
  search?: string;
  location_id?: string;
  game_id?: string;
  billing_type?: string;
  available?: string;
  per_page?: number;
};

export async function fetchAdminTariffs(
  page = 1,
  filters: AdminTariffsFilters = {}
) {
  const params = new URLSearchParams({ page: String(page) });
  Object.entries(filters).forEach(([key, value]) => {
    const val = String(value ?? "").trim();
    if (val) params.set(key, val);
  });
  return apiFetch<{ tariffs: AdminTariffsPage }>(
    `/v1/admin/tariffs?${params.toString()}`
  );
}

export async function fetchAdminTariffCreateForm() {
  return apiFetch<AdminTariffFormMeta>("/v1/admin/tariffs/create");
}

export async function fetchAdminTariff(id: string) {
  return apiFetch<{ tariff: AdminTariff }>(`/v1/admin/tariffs/${id}`);
}

export async function fetchAdminTariffEdit(id: string) {
  return apiFetch<{ tariff: AdminTariff } & AdminTariffFormMeta>(
    `/v1/admin/tariffs/${id}/edit`
  );
}

export async function createAdminTariff(data: Record<string, unknown>) {
  return apiFetch<{ ok: boolean }>("/v1/admin/tariffs", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateAdminTariff(
  id: string,
  data: Record<string, unknown>
) {
  return apiFetch<{ ok: boolean }>(`/v1/admin/tariffs/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function deleteAdminTariff(id: string) {
  return apiFetch<{ ok: boolean }>(`/v1/admin/tariffs/${id}`, {
    method: "DELETE",
  });
}

export async function duplicateAdminTariff(id: string) {
  return apiFetch<{ id: string }>(`/v1/admin/tariffs/${id}/duplicate`, {
    method: "POST",
  });
}

// Действие с файлом при установке или удалении плагина. Набор из шести видов —
// тот же, что понимает агент.
export type AdminPluginAction = {
  path: string;
  action:
    | "ensure_contains"
    | "append_lines"
    | "prepend_lines"
    | "remove_lines"
    | "write_file"
    | "replace_regex";
  create_if_missing?: boolean;
  pattern?: string;
  replacement?: string;
  lines?: string[];
};

export type AdminPlugin = {
  id: string;
  slug: string;
  name: string;
  category: string;
  version: string;
  description: string;
  archive_type: string;
  has_archive: boolean;
  archive_size: number;
  image_url: string;
  install_path: string;
  supported_games: string[];
  all_games: boolean;
  file_actions: AdminPluginAction[];
  uninstall_actions: AdminPluginAction[];
  restart_required: boolean;
  active: boolean;
};

export type AdminPluginInput = {
  id?: string;
  slug: string;
  name: string;
  category?: string;
  version?: string;
  description?: string;
  install_path?: string;
  all_games?: boolean;
  supported_games?: string[];
  file_actions?: AdminPluginAction[];
  uninstall_actions?: AdminPluginAction[];
  restart_required?: boolean;
  active?: boolean;
};

export type AdminMap = {
  id: string;
  slug: string;
  name: string;
  category: string;
  version: string;
  game_slug: string;
  archive_path: string;
  file_count: number;
  restart_required: boolean;
  active: boolean;
};

export type AdminMapInput = {
  id?: string;
  slug: string;
  name: string;
  category?: string;
  version?: string;
  game_slug?: string;
  restart_required?: boolean;
  active?: boolean;
};

export async function fetchAdminPlugins() {
  return apiFetch<{ plugins: AdminPlugin[] }>("/v1/admin/plugins");
}

export async function fetchAdminMaps() {
  return apiFetch<{ maps: AdminMap[] }>("/v1/admin/maps");
}

// uploadAdminFile — общая отправка multipart. apiFetch тут не подходит: он
// ставит Content-Type: application/json, а границу multipart проставляет сам
// браузер, и переопределять её нельзя.
async function uploadAdminFile<T>(path: string, form: FormData): Promise<T> {
  const headers: Record<string, string> = {};
  const token = getAccessToken();
  if (token) headers.Authorization = `Bearer ${token}`;
  const res = await fetch(API_URL + path, { method: "POST", headers, body: form });
  const text = await res.text();
  let data: Record<string, unknown> = {};
  if (text) {
    try {
      data = JSON.parse(text) as Record<string, unknown>;
    } catch {
      data = {};
    }
  }
  if (!res.ok) {
    throw new Error((data.error as string) ?? t("errors.upload.failed"));
  }
  return data as T;
}

export async function uploadAdminPluginArchive(id: string, file: File, archiveType?: string) {
  const form = new FormData();
  form.append("archive", file);
  if (archiveType) form.append("archive_type", archiveType);
  return uploadAdminFile<{ status: string; archive_type: string; size: number }>(
    `/v1/admin/plugins/${id}/archive`,
    form
  );
}

export async function uploadAdminPluginImage(id: string, file: File) {
  const form = new FormData();
  form.append("image", file);
  return uploadAdminFile<{ status: string; image_url: string }>(
    `/v1/admin/plugins/${id}/image`,
    form
  );
}

export async function uploadAdminMapArchive(id: string, file: File) {
  const form = new FormData();
  form.append("archive", file);
  return uploadAdminFile<{ status: string; size: number; file_count: number }>(
    `/v1/admin/maps/${id}/archive`,
    form
  );
}

export type AdminImageItem = {
  game: string;
  name: string;
  version: string;
  docker_image: string;
  repository: string;
  tag: string;
  default_tag: string;
  image_env: string;
  enabled: boolean;
  in_catalog: boolean;
  build_image: boolean;
  build_ready: number;
  build_failed: number;
  building: number;
  build_nodes: number;
};

export async function fetchAdminImages() {
  return apiFetch<{ images: AdminImageItem[] }>("/v1/admin/images");
}

export async function setAdminImageTag(payload: { game_slug: string; tag: string }) {
  return apiFetch<{ status: string; docker_image: string; tag: string }>("/v1/admin/images", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function setAdminImageBuild(payload: { game_slug: string; build: boolean }) {
  return apiFetch<{ ok: boolean; game: string; build_image: boolean }>(
    "/v1/admin/images/build-flag",
    { method: "POST", body: JSON.stringify(payload) }
  );
}

export async function buildAdminImages(payload: { node_id?: string } = {}) {
  return apiFetch<{ ok: boolean; nodes: number }>("/v1/admin/images/build", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export type AdminPayment = {
  id: string;
  user_id: string;
  user_email?: string;
  provider?: string;
  amount: number;
  currency?: string;
  status: string;
  provider_payment_id?: string;
  created_at: string;
  refunded_amount?: number;
};

export type AdminBillingFilters = {
  limit?: number;
  status?: string;
  provider?: string;
  days?: number;
  q?: string;
};

export async function fetchAdminBilling(
  filters?: number | AdminBillingFilters
) {
  const f: AdminBillingFilters =
    typeof filters === "number" ? { limit: filters } : (filters ?? {});
  const params = new URLSearchParams();
  if (f.limit) params.set("limit", String(f.limit));
  if (f.status) params.set("status", f.status);
  if (f.provider) params.set("provider", f.provider);
  if (f.days) params.set("days", String(f.days));
  if (f.q) params.set("q", f.q);
  const tail = params.toString() ? `?${params}` : "";
  return apiFetch<{ payments: AdminPayment[] }>(`/v1/admin/billing${tail}`);
}

export async function adjustAdminUserBalance(
  userId: string,
  data: { amount: number; currency?: string; comment: string }
) {
  return apiFetch<{ wallet_id: string; balance: number; currency: string }>(
    `/v1/users/${userId}/balance`,
    { method: "POST", body: JSON.stringify(data) }
  );
}

export type AdminServerListItem = Server & {
  owner_email?: string;
  user_email?: string;
  is_blocked?: boolean;
  blocked_reason?: string | null;
};

export async function fetchAdminServers() {
  return apiFetch<{ servers: AdminServerListItem[] }>("/v1/admin/servers");
}

export async function adminToggleServerBlock(id: string, blocked: boolean, reason?: string) {
  return apiFetch<{ blocked: boolean }>(`/v1/admin/servers/${id}/toggle-block`, {
    method: "POST",
    body: JSON.stringify({ blocked, reason: reason ?? "" }),
  });
}

export type AdminSupportTicket = SupportTicket & {
  priority: "low" | "normal" | "high" | "urgent";
  user_id: string;
  user_email: string;
  assigned_admin_id: string;
  assigned_admin_email: string;
  last_message_at: string;
  messages: number;
  awaiting_staff: boolean;
  // Сколько обращений у этого клиента всего: первое и двенадцатое разбирают
  // по-разному.
  user_tickets?: number;
  service_kind?: string;
  service_id?: string;
  service?: string;
};

export async function fetchAdminSupport(status?: string, category?: string) {
  const qs = new URLSearchParams();
  if (status) qs.set("status", status);
  if (category) qs.set("category", category);
  const tail = qs.toString() ? `?${qs}` : "";
  return apiFetch<{ tickets: AdminSupportTicket[] }>(`/v1/admin/support${tail}`);
}

export async function updateAdminSupportTicket(
  id: string,
  data: {
    status?: "open" | "pending" | "answered" | "closed";
    priority?: "low" | "normal" | "high" | "urgent";
    assign_to_me?: boolean;
  }
) {
  return apiFetch<{ status: string }>(`/v1/admin/support/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
}

export type AdminNewsImage = { id: string; url: string };

export type AdminNewsItem = {
  id: string;
  slug: string;
  title: string;
  excerpt?: string | null;
  body?: string | null;
  published_at?: string | null;
  active: boolean;
  image?: string | null;
  images: AdminNewsImage[];
  tag?: string;
  pinned?: boolean;
};

export type AdminNewsInput = {
  slug?: string;
  title?: string;
  excerpt?: string | null;
  body?: string;
  // Пустая строка снимает дату публикации, отсутствие поля её не трогает.
  published_at?: string | null;
  active?: boolean;
  tag?: string;
  pinned?: boolean;
};

export async function fetchAdminNews() {
  return apiFetch<{ news: AdminNewsItem[] }>("/v1/admin/news");
}

export async function createAdminNews(data: AdminNewsInput) {
  return apiFetch<{ id?: string; slug?: string }>("/v1/admin/news", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateAdminNews(id: string, data: AdminNewsInput) {
  return apiFetch<{ status: string }>(`/v1/admin/news/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function uploadAdminNewsImage(id: string, file: File) {
  const form = new FormData();
  form.append("image", file);
  return uploadAdminFile<{ id: string; url: string }>(`/v1/admin/news/${id}/images`, form);
}

export async function deleteAdminNewsImage(newsId: string, imageId: string) {
  return apiFetch<{ status: string }>(`/v1/admin/news/${newsId}/images/${imageId}`, {
    method: "DELETE",
  });
}

export async function createAdminMailing(data: { subject?: string; title?: string; body?: string }) {
  return apiFetch<{ id: string }>("/v1/admin/mailings", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function createAdminPlugin(data: AdminPluginInput) {
  return apiFetch<{ id?: string; status?: string }>("/v1/admin/plugins", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function createAdminMap(data: AdminMapInput) {
  return apiFetch<{ id?: string; status?: string }>("/v1/admin/maps", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function sendAdminMailing(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/mailings/${id}/send`, { method: "POST" });
}

export type AdminReadinessBlocker = {
  key: string;
  message: string;
  fallback?: string;
};

export type AdminReadiness = {
  ok: boolean;
  checks: {
    mail: { mailer: string; smtp_configured: boolean };
    oauth: { configured: Record<string, boolean>; any_ready: boolean };
    payments: { enabled_providers: number; providers_with_webhook: number };
  };
  blockers: AdminReadinessBlocker[];
};

export type AdminSeriesPoint = {
  ts: string;
  cpu: number | null;
  ram: number | null;
};

export type AdminDashboardSeries = {
  points: AdminSeriesPoint[];
  nodes: { id: string; name: string }[];
  hours: number;
  bucket_minutes: number;
  node_id: string;
};

export async function fetchAdminDashboardSeries(params?: {
  hours?: number;
  nodeId?: string;
}) {
  const sp = new URLSearchParams();
  if (params?.hours) sp.set("hours", String(params.hours));
  if (params?.nodeId) sp.set("node_id", params.nodeId);
  const tail = sp.toString() ? `?${sp}` : "";
  return apiFetch<AdminDashboardSeries>(`/v1/admin/dashboard/series${tail}`);
}

export async function fetchAdminReadiness() {
  return apiFetch<AdminReadiness>("/v1/admin/settings/readiness");
}

export async function fetchAdminSettings() {
  return apiFetch<AdminSettingsData>("/v1/admin/settings");
}

export async function saveAdminSettings(form: FormData) {
  const url = API_URL + "/v1/admin/settings";
  const headers: Record<string, string> = {};
  const token = getAccessToken();
  if (token) headers.Authorization = `Bearer ${token}`;
  const res = await fetch(url, { method: "POST", headers, body: form });
  const data = await res.json();
  if (!res.ok) {
    throw new Error(data.message ?? data.error ?? "Request failed");
  }
  return data as { ok?: boolean; message?: string };
}

export async function updateAdminSettings(settings: Record<string, unknown>) {
  return apiFetch<{ status: string; ok?: boolean; message?: string }>(
    "/v1/admin/settings",
    {
      method: "PATCH",
      body: JSON.stringify(settings),
    }
  );
}

export async function testAdminSettingsMail(testMailTo: string) {
  return apiFetch<{ ok?: boolean; message?: string }>(
    "/v1/admin/settings/test-mail",
    {
      method: "POST",
      body: JSON.stringify({ test_mail_to: testMailTo }),
    }
  );
}

export async function testAdminSettingsTelegram() {
  return apiFetch<{ ok?: boolean; message?: string }>(
    "/v1/admin/settings/test-telegram",
    { method: "POST", body: JSON.stringify({}) }
  );
}

export async function testAdminSettingsFilesStorage(
  payload: Record<string, unknown>
) {
  return apiFetch<{ ok?: boolean; message?: string }>(
    "/v1/admin/settings/test-files-storage",
    {
      method: "POST",
      body: JSON.stringify(payload),
    }
  );
}

export type AdminAppearanceData = {
  values?: Record<string, string>;
};

export async function fetchAdminAppearance() {
  return apiFetch<AdminAppearanceData>("/v1/admin/settings/appearance");
}

export async function saveAdminAppearance(payload: {
  default_template: string;
  template_colors: string;
  template_blocks: string;
  template_user_menu_variant: string;
}) {
  return apiFetch<{ ok?: boolean; message?: string }>(
    "/v1/admin/settings/appearance",
    {
      method: "POST",
      body: JSON.stringify(payload),
    }
  );
}

export function brandingUploadUrl(path: string): string {
  const trimmed = path.trim();
  if (!trimmed) return "";
  if (trimmed.startsWith("http://") || trimmed.startsWith("https://")) {
    return trimmed;
  }
  const name = trimmed
    .replace(/^\/+/, "")
    .replace(/^uploads\//, "")
    .replace(/^branding\//, "");
  if (!name || name.includes("..") || name.includes("/")) return "";
  return `${API_URL}/v1/uploads/branding/${name}`;
}

export type AdminGroupsResponse = {
  groups: Record<string, string>;
  keys: string[];
  matrix: Record<string, Record<string, boolean>>;
  permissionLabels: Record<string, string>;
};

export async function fetchAdminGroups() {
  return apiFetch<AdminGroupsResponse>("/v1/admin/groups");
}

export async function updateAdminGroups(
  permissions: Record<string, Record<string, number>>
) {
  return apiFetch<{ ok: boolean; message: string }>("/v1/admin/groups", {
    method: "PATCH",
    body: JSON.stringify({ permissions }),
  });
}

export async function fetchAdminLogs() {
  return apiFetch<{
    logs: { action: string; resource: string; created_at: string }[];
    limit?: number;
    search?: string;
    action?: string;
  }>("/v1/admin/logs");
}

export async function fetchAdminLogsFiltered(params?: {
  limit?: number;
  search?: string;
  action?: string;
}) {
  const qs = new URLSearchParams();
  if (params?.limit) qs.set("limit", String(params.limit));
  if (params?.search?.trim()) qs.set("search", params.search.trim());
  if (params?.action?.trim()) qs.set("action", params.action.trim());
  const tail = qs.toString() ? `?${qs.toString()}` : "";
  return apiFetch<{
    logs: { action: string; resource: string; created_at: string }[];
    limit?: number;
    search?: string;
    action?: string;
  }>(`/v1/admin/logs${tail}`);
}

export type AdminLogEvent = {
  id: string;
  action: string;
  resource: string;
  created_at: string;
};

export function subscribeAdminLogs(
  onEvent: (event: AdminLogEvent) => void,
  onError?: (err: Error) => void
): () => void {
  const controller = new AbortController();
  let stopped = false;
  let since: string | undefined;

  const run = async () => {
    while (!stopped) {
      try {
        const qs = new URLSearchParams({ stream: "1" });
        if (since) qs.set("since", since);
        const token = getAccessToken();
        const res = await fetch(`${API_URL}/v1/admin/logs?${qs.toString()}`, {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
          signal: controller.signal,
        });
        if (!res.ok || !res.body) {
          throw new Error(t("errors.logs.stream_unavailable", { code: res.status }));
        }
        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buffer = "";
        for (;;) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          let sep = buffer.indexOf("\n\n");
          while (sep >= 0) {
            const chunk = buffer.slice(0, sep);
            buffer = buffer.slice(sep + 2);
            for (const line of chunk.split("\n")) {
              if (!line.startsWith("data: ")) continue;
              try {
                const item = JSON.parse(line.slice(6)) as AdminLogEvent;
                since = item.created_at || since;
                onEvent(item);
              } catch {}
            }
            sep = buffer.indexOf("\n\n");
          }
        }
      } catch (err) {
        if (stopped || controller.signal.aborted) return;
        onError?.(
          err instanceof Error ? err : new Error(t("errors.logs.stream_interrupted"))
        );
      }
      if (stopped) return;
      await new Promise((resolve) => setTimeout(resolve, 3000));
    }
  };
  void run();

  return () => {
    stopped = true;
    controller.abort();
  };
}

export async function fetchAdminHostingServers() {
  return apiFetch<{ servers: unknown[] }>("/v1/admin/hosting/servers");
}

export type AdminHostingServerInput = {
  id?: string;
  name: string;
  hostname: string;
  ip_address?: string;
  port?: number;
  panel_type: string;
  api_url: string;
  api_username?: string;
  api_token?: string;
  use_ssl?: boolean;
  max_accounts?: number;
  active?: boolean;
  description?: string;
};

export type AdminHostingServerDetail = AdminHostingServerInput & {
  id: string;
  current_accounts?: number;
  has_api_token?: boolean;
};

export async function fetchAdminHostingServer(id: string) {
  return apiFetch<AdminHostingServerDetail>(`/v1/admin/hosting/servers/${id}`);
}

export async function createAdminHostingServer(data: AdminHostingServerInput) {
  return apiFetch<{ id: string }>("/v1/admin/hosting/servers", { method: "POST", body: JSON.stringify(data) });
}

// Тот же эндпоинт правит запись, если передан id: API умел это с самого начала,
// а в панели существовало только создание, и сервер приходилось заводить заново.
export async function updateAdminHostingServer(id: string, data: AdminHostingServerInput) {
  return apiFetch<{ id: string; status: string }>("/v1/admin/hosting/servers", {
    method: "POST",
    body: JSON.stringify({ ...data, id }),
  });
}

export async function deleteAdminHostingServer(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/hosting/servers/${id}`, { method: "DELETE" });
}

export async function fetchAdminHostingPlans() {
  return apiFetch<{ plans: unknown[] }>("/v1/admin/hosting/plans");
}

export type AdminHostingPlanInput = {
  id?: string;
  hosting_server_id?: string;
  name: string;
  panel_package_name: string;
  disk_mb?: number;
  bandwidth_mb?: number;
  max_domains?: number;
  max_databases?: number;
  max_email_accounts?: number;
  has_ssl?: boolean;
  has_ssh?: boolean;
  has_cron?: boolean;
  has_backup?: boolean;
  price_monthly?: number;
  active?: boolean;
};

export async function createAdminHostingPlan(data: AdminHostingPlanInput) {
  return apiFetch<{ id: string }>("/v1/admin/hosting/plans", { method: "POST", body: JSON.stringify(data) });
}

export async function updateAdminHostingPlan(id: string, data: AdminHostingPlanInput) {
  return apiFetch<{ id: string; status: string }>("/v1/admin/hosting/plans", {
    method: "POST",
    body: JSON.stringify({ ...data, id }),
  });
}

export async function deleteAdminHostingPlan(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/hosting/plans/${id}`, { method: "DELETE" });
}

export type AdminLanguageItem = { key: string; value: string };

export async function fetchTranslations() {
  return apiFetch<{ locale: string; messages: Record<string, string> }>(
    "/v1/translations/messages"
  );
}

export async function fetchAdminLanguage(locale = "ru") {
  return apiFetch<{ locale: string; messages: AdminLanguageItem[] }>(
    `/v1/admin/language?locale=${encodeURIComponent(locale)}`
  );
}

export async function updateAdminLanguage(
  locale: string,
  messages: Record<string, string>
) {
  return apiFetch<{ status: string }>("/v1/admin/language", {
    method: "PATCH",
    body: JSON.stringify({ locale, messages }),
  });
}

export type AdminMysqlInstance = {
  node_id: string;
  node_name: string;
  region?: string;
  key: string;
  name?: string;
  container: string;
  port: number;
  root_password?: string;
};

export async function fetchAdminMysqlInstances() {
  return apiFetch<{ instances: AdminMysqlInstance[]; count?: number }>(
    "/v1/admin/mysql"
  );
}

export async function createAdminMysqlInstance(payload: {
  node_id: string;
  key: string;
  name?: string;
  container: string;
  port: number;
  root_password?: string;
}) {
  return apiFetch<{ ok: boolean }>("/v1/admin/mysql/instance/create", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function updateAdminMysqlInstance(payload: {
  node_id: string;
  key: string;
  name?: string;
  container?: string;
  port?: number;
  root_password?: string;
}) {
  return apiFetch<{ ok: boolean }>("/v1/admin/mysql/instance/update", {
    method: "PATCH",
    body: JSON.stringify(payload),
  });
}

export async function deleteAdminMysqlInstance(nodeId: string, key: string) {
  return apiFetch<{ ok: boolean }>("/v1/admin/mysql/instance/delete", {
    method: "POST",
    body: JSON.stringify({ node_id: nodeId, key }),
  });
}

export async function createAdminBugReport(payload: {
  title: string;
  description: string;
  severity: "low" | "medium" | "high" | "critical";
  component: string;
  /** Где воспроизводится: нода или вся панель. */
  node?: string;
  /** Версии и браузер — сервер кладёт их в meta обращения. */
  environment?: Record<string, string>;
}) {
  return apiFetch<{
    ok: boolean;
    ticket_id: string;
    // Приоритет обращения, в который сервер перевёл критичность отчёта.
    priority: "low" | "normal" | "high" | "urgent";
    status: string;
    /** Дошёл ли отчёт до разработчика панели. */
    delivered?: boolean;
    /** Номер отчёта в консоли разработчика. */
    report_number?: number;
    /** Почему не дошёл — показываем автору, а не прячем. */
    delivery_error?: string;
  }>("/v1/admin/bug-report", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export type AdminBugReport = {
  id: string;
  /** Заголовок без служебного префикса «[BUG][HIGH][panel-ui]». */
  title: string;
  status: string;
  priority: "low" | "normal" | "high" | "urgent";
  severity: string;
  component: string;
  created_at: string;
  /** Отчёт создан текущим пользователем. */
  mine: boolean;
};

export async function fetchAdminBugReports() {
  return apiFetch<{ reports: AdminBugReport[]; mine: number }>(
    "/v1/admin/bug-report"
  );
}

// Пустой тип — акция без скидки: она может состоять из одного бонуса на
// пополнение.
export type AdminPromotionDiscount = "percent" | "fixed" | "";

// Где действует акция. Пустой список означает «везде».
export type AdminPromotionScope = "rent" | "renew" | "topup";

export type AdminPromotion = {
  id: string;
  title: string;
  code: string;
  type: AdminPromotionDiscount;
  value: number;
  active: boolean;
  starts_at: string | null;
  ends_at: string | null;
  max_uses: number | null;
  used_count: number;
  min_amount: number | null;
  only_new_users: boolean;
  description: string;
  applies_to: AdminPromotionScope[];
  bonus_percent: number;
  bonus_fixed: number;
  tariff_ids: string[];
  game_ids: string[];
  location_ids: string[];
  user_ids: string[];
};

export type AdminPromotionInput = {
  title?: string;
  code?: string;
  type?: AdminPromotionDiscount;
  value?: number;
  active?: boolean;
  starts_at?: string | null;
  ends_at?: string | null;
  max_uses?: number | null;
  min_amount?: number | null;
  only_new_users?: boolean;
  description?: string;
  applies_to?: AdminPromotionScope[];
  bonus_percent?: number;
  bonus_fixed?: number;
  tariff_ids?: string[];
  game_ids?: string[];
  location_ids?: string[];
  user_ids?: string[];
};

export async function fetchAdminPromotions() {
  return apiFetch<{ promotions: AdminPromotion[] }>("/v1/admin/promotions");
}

export async function createAdminPromotion(data: AdminPromotionInput) {
  return apiFetch<{ id: string }>("/v1/admin/promotions", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateAdminPromotion(id: string, data: AdminPromotionInput) {
  return apiFetch<{ status: string }>(`/v1/admin/promotions/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
}

export async function deleteAdminPromotion(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/promotions/${id}`, {
    method: "DELETE",
  });
}

export type AdminHostingAccount = {
  id: string;
  user_id: string;
  user_email: string;
  username: string;
  primary_domain: string;
  status: string;
  expires_at: string | null;
  suspended_at: string | null;
  plan: string;
  server: string;
  created_at: string;
};

export async function fetchAdminHostingAccountsList() {
  return apiFetch<{ accounts: AdminHostingAccount[] }>("/v1/admin/hosting/accounts");
}

export async function suspendAdminHostingAccount(id: string, reason?: string) {
  return apiFetch<{ status: string }>(`/v1/admin/hosting/accounts/${id}/suspend`, {
    method: "POST",
    body: JSON.stringify({ reason }),
  });
}

export async function unsuspendAdminHostingAccount(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/hosting/accounts/${id}/unsuspend`, {
    method: "POST",
  });
}

export async function deleteAdminNews(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/news/${id}`, { method: "DELETE" });
}

export async function deleteAdminMailing(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/mailings/${id}`, { method: "DELETE" });
}

export async function deleteAdminPlugin(id: string, force = false) {
  return apiFetch<{ status: string }>(
    `/v1/admin/plugins/${id}${force ? "?force=1" : ""}`,
    { method: "DELETE" }
  );
}

export async function deleteAdminMap(id: string, force = false) {
  return apiFetch<{ status: string }>(
    `/v1/admin/maps/${id}${force ? "?force=1" : ""}`,
    { method: "DELETE" }
  );
}

export async function fetchAdminMailings() {
  return apiFetch<{ mailings: unknown[] }>("/v1/admin/mailings");
}

export async function fetchAdminPaymentProviders() {
  return apiFetch<{ providers: unknown[] }>("/v1/admin/payment-providers");
}

export async function updateAdminPaymentProvider(
  id: string,
  data: { enabled?: boolean; config?: Record<string, unknown> }
) {
  return apiFetch<{ status: string }>(`/v1/admin/payment-providers/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
}


export async function fetchPanelVersion(): Promise<string> {
  try {
    const res = await fetch("/api/version", { cache: "no-store" });
    if (!res.ok) return "";
    const data = (await res.json()) as { version?: string };
    return data.version || "";
  } catch {
    return "";
  }
}

export type AdminJob = {
  id: string;
  type: string;
  type_label: string;
  status: "pending" | "running" | "completed" | "failed" | "cancelled";
  attempts: number;
  created_at: string;
  updated_at: string;
  age_seconds: number;
  idle_seconds: number;
  stuck: boolean;
  error: string;
  payload: Record<string, unknown>;
  result: Record<string, unknown>;
  server_name: string;
  node_name: string;
};

export type AdminJobsSummary = {
  pending: number;
  running: number;
  completed: number;
  failed: number;
  cancelled: number;
  stuck: number;
  oldest_pending_seconds: number;
};

export type AdminJobType = {
  type: string;
  label: string;
  total: number;
  pending: number;
  failed: number;
};

export type AdminJobsResponse = {
  jobs: AdminJob[];
  total: number;
  limit: number;
  offset: number;
  stuck_minutes: number;
  summary: AdminJobsSummary;
  types: AdminJobType[];
};

export async function fetchAdminJobs(params: {
  status?: string;
  type?: string;
  search?: string;
  limit?: number;
  offset?: number;
} = {}) {
  const q = new URLSearchParams();
  if (params.status && params.status !== "all") q.set("status", params.status);
  if (params.type && params.type !== "all") q.set("type", params.type);
  if (params.search) q.set("search", params.search);
  if (params.limit) q.set("limit", String(params.limit));
  if (params.offset) q.set("offset", String(params.offset));
  const suffix = q.toString() ? `?${q.toString()}` : "";
  return apiFetch<AdminJobsResponse>(`/v1/admin/jobs${suffix}`);
}

export async function retryAdminJob(id: string) {
  return apiFetch<{ status: string; type: string }>(`/v1/admin/jobs/${id}/retry`, {
    method: "POST",
  });
}

export async function cancelAdminJob(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/jobs/${id}/cancel`, {
    method: "POST",
  });
}

export async function deleteAdminJob(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/jobs/${id}`, {
    method: "DELETE",
  });
}

export async function retryAdminFailedJobs(type?: string) {
  return apiFetch<{ status: string; count: number }>(`/v1/admin/jobs/retry-failed`, {
    method: "POST",
    body: JSON.stringify({ type: type && type !== "all" ? type : "" }),
  });
}

export async function cleanupAdminJobs(days: number) {
  return apiFetch<{ status: string; count: number }>(`/v1/admin/jobs/cleanup`, {
    method: "POST",
    body: JSON.stringify({ days }),
  });
}

export type MigrationTargetNode = {
  id: string;
  name: string;
  fqdn: string;
  status: string;
  servers: number;
  metrics_known: boolean;
  ram_total_mb: number;
  ram_free_mb: number;
  disk_total_mb: number;
  disk_free_mb: number;
  cpu_percent: number;
  ram_percent: number;
  fits: boolean;
  reason: string;
};

export type MigrationTargets = {
  nodes: MigrationTargetNode[];
  current_node_id: string;
  game_id: string;
  need_ram_mb: number;
  need_disk_mb: number;
};

export type ServerMigration = {
  id: string;
  status: "pending" | "running" | "completed" | "failed";
  stage: string;
  error: string;
  bytes: number;
  remove_source: boolean;
  from_node: string;
  to_node: string;
  created_at: string;
  started_at: string | null;
  finished_at: string | null;
};

export async function fetchServerMigrationTargets(serverId: string) {
  return apiFetch<MigrationTargets>(`/v1/admin/servers/${serverId}/migration-targets`);
}

export async function fetchServerMigrations(serverId: string) {
  return apiFetch<{ migrations: ServerMigration[] }>(`/v1/admin/servers/${serverId}/migrations`);
}

export async function migrateServer(
  serverId: string,
  toNodeId: string,
  removeSource: boolean
) {
  return apiFetch<{ status: string; migration_id: string; to_node: string }>(
    `/v1/admin/servers/${serverId}/migrate`,
    {
      method: "POST",
      body: JSON.stringify({ to_node_id: toNodeId, remove_source: removeSource }),
    }
  );
}

export async function setNodeMaintenance(
  id: string,
  data: {
    enabled: boolean;
    reason?: string;
    until?: string;
    notify_owners?: boolean;
  }
) {
  return apiFetch<{ maintenance_mode: boolean; servers: number; notified: number }>(
    `/v1/admin/locations/${id}/maintenance`,
    { method: "POST", body: JSON.stringify(data) }
  );
}

export type RemoteBackup = {
  id: string;
  filename: string;
  size_bytes: number;
  remote_size: number;
  status: "none" | "pending" | "uploading" | "uploaded" | "failed" | "deleted";
  error: string | null;
  source: string;
  created_at: string;
  uploaded_at: string | null;
};

export async function fetchRemoteBackups(serverId: string) {
  return apiFetch<{ backups: RemoteBackup[]; available: boolean }>(
    `/v1/servers/${serverId}/backups/remote`
  );
}

export async function restoreRemoteBackup(
  serverId: string,
  backupId: string,
  restore = true
) {
  return apiFetch<{ status: string; restore: boolean }>(
    `/v1/servers/${serverId}/backups/remote/restore`,
    { method: "POST", body: JSON.stringify({ backup_id: backupId, restore }) }
  );
}

export async function deleteRemoteBackup(serverId: string, backupId: string) {
  return apiFetch<{ status: string }>(
    `/v1/servers/${serverId}/backups/remote/delete`,
    { method: "POST", body: JSON.stringify({ backup_id: backupId }) }
  );
}

export async function runNodeBulkAction(
  id: string,
  data: {
    action: "start" | "stop" | "restart" | "notify" | "extend";
    message?: string;
    days?: number;
    only_running?: boolean;
  }
) {
  return apiFetch<{ status: string; action: string; servers: number }>(
    `/v1/admin/locations/${id}/bulk`,
    { method: "POST", body: JSON.stringify(data) }
  );
}

export type NodeCapacityGame = {
  slug: string;
  name: string;
  ram_per_server_mb: number;
  disk_per_server_mb: number;
  fits: number;
  limited_by: string;
};

export type NodeCapacity = {
  node_id: string;
  node_name: string;
  servers: number;
  max_servers: number | null;
  slots_left: number | null;
  allocated_ram_mb: number;
  allocated_cpu: number;
  node_ram_mb: number;
  node_disk_mb: number;
  free_disk_mb: number;
  free_ram_mb: number;
  ram_overcommit: number;
  reserved_ram_mb: number;
  metrics_known: boolean;
  games: NodeCapacityGame[];
  // null означает «агент ещё не отчитался»: это не то же самое, что «квот нет».
  disk_quota: boolean | null;
};

export async function fetchNodeCapacity(id: string) {
  return apiFetch<NodeCapacity>(`/v1/admin/locations/${id}/capacity`);
}

export async function updateNodeCapacity(
  id: string,
  data: {
    max_servers?: number | null;
    ram_overcommit?: number;
    reserved_ram_mb?: number;
  }
) {
  return apiFetch<{ status: string }>(`/v1/admin/locations/${id}/capacity`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
}

export type AnalyticsPoint = {
  date: string;
  charges: number;
  topups: number;
  new_servers: number;
};

export type AnalyticsBreakdownRow = {
  label: string;
  servers: number;
  monthly: number;
};

export type AdminAnalytics = {
  days: number;
  revenue: {
    charges: number;
    charges_prev: number;
    topups: number;
    topups_prev: number;
    payers: number;
    arpu: number;
    charges_change_percent: number;
    topups_change_percent: number;
  };
  series: AnalyticsPoint[];
  sources: { source: string; label: string; count: number; amount: number }[];
  recurring: {
    monthly: number;
    active_servers: number;
    expiring_7d: number;
    avg_per_server: number;
  };
  customers: {
    total: number;
    registered: number;
    with_server: number;
    with_payment: number;
    server_conversion: number;
    payment_conversion: number;
  };
  churn: { expired: number; renewed: number; suspended: number };
  breakdown: {
    games: AnalyticsBreakdownRow[];
    locations: AnalyticsBreakdownRow[];
    tariffs: AnalyticsBreakdownRow[];
  };
};

export async function fetchAdminAnalytics(days: number) {
  return apiFetch<AdminAnalytics>(`/v1/admin/analytics?days=${days}`);
}

export async function downloadAdminAnalyticsCsv(days: number) {
  const token = getAccessToken();
  const res = await fetch(`${API_URL}/v1/admin/analytics/export.csv?days=${days}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) throw new Error(t("errors.analytics.export_failed"));
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `vortanix-analytics-${new Date().toISOString().slice(0, 10)}.csv`;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

export type ServerCharge = {
  source: string;
  description: string;
  amount: number;
  created_at: string;
};

export async function fetchServerCharges(serverId: string) {
  return apiFetch<{ charges: ServerCharge[]; total: number }>(
    `/v1/admin/servers/${serverId}/charges`
  );
}

export async function fetchRefundSupport() {
  return apiFetch<{ providers: Record<string, boolean> }>(
    "/v1/admin/billing/refund-support"
  );
}

export async function refundAdminPayment(
  paymentId: string,
  data: { amount?: number; reason?: string }
) {
  return apiFetch<{
    status: string;
    refunded_amount: number;
    refund_reference: string;
  }>(`/v1/admin/billing/payments/${paymentId}/refund`, {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function setServerAutoRenew(serverId: string, enabled: boolean) {
  return apiFetch<{ auto_renew: boolean }>(`/v1/servers/${serverId}/auto-renew`, {
    method: "POST",
    body: JSON.stringify({ enabled }),
  });
}

export function paymentReceiptUrl(paymentId: string): string {
  return `${API_URL}/v1/billing/payments/${paymentId}/receipt`;
}

export async function openPaymentReceipt(paymentId: string) {
  const token = getAccessToken();
  const res = await fetch(paymentReceiptUrl(paymentId), {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) {
    const text = await res.text();
    let message = t("errors.billing.receipt_failed");
    try {
      message = (JSON.parse(text) as { error?: string }).error ?? message;
    } catch {}
    throw new Error(message);
  }
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  window.open(url, "_blank", "noopener");
  setTimeout(() => URL.revokeObjectURL(url), 60_000);
}

export type NodeIPAddress = {
  id: string;
  address: string;
  status: "free" | "reserved" | "assigned";
  label: string;
  server_id: string | null;
  server_name: string;
  assigned_at: string | null;
};

export async function fetchNodeIPs(nodeId: string) {
  return apiFetch<{ addresses: NodeIPAddress[]; free: number }>(
    `/v1/admin/locations/${nodeId}/ips`
  );
}

export async function addNodeIPs(nodeId: string, addresses: string, label: string) {
  return apiFetch<{ added: number; skipped: number }>(
    `/v1/admin/locations/${nodeId}/ips`,
    { method: "POST", body: JSON.stringify({ addresses, label }) }
  );
}

export async function importNodeIPs(nodeId: string) {
  return apiFetch<{ added: number; found: number }>(
    `/v1/admin/locations/${nodeId}/ips/import`,
    { method: "POST" }
  );
}

export async function deleteNodeIP(nodeId: string, ipId: string) {
  return apiFetch<{ status: string }>(
    `/v1/admin/locations/${nodeId}/ips/${ipId}`,
    { method: "DELETE" }
  );
}

export async function assignServerIP(serverId: string, ipId: string) {
  return apiFetch<{ address: string; port: number; restart_required: boolean }>(
    `/v1/admin/servers/${serverId}/ip`,
    { method: "POST", body: JSON.stringify({ ip_id: ipId }) }
  );
}

export async function releaseServerIP(serverId: string) {
  return apiFetch<{ address: string; port: number; restart_required: boolean }>(
    `/v1/admin/servers/${serverId}/ip/release`,
    { method: "POST" }
  );
}

export type TrialStatus = {
  available: boolean;
  reason: string;
  hours: number;
  games: string[];
  cooldown_days: number;
  next_at: string | null;
};

export async function fetchTrialStatus() {
  return apiFetch<TrialStatus>("/v1/trial");
}

export async function createTrialServer(data: {
  game_id: string;
  node_id?: string;
  name?: string;
}) {
  return apiFetch<{ server_id: string; expires_at: string; hours: number }>(
    "/v1/trial",
    { method: "POST", body: JSON.stringify(data) }
  );
}

export type AdminAPIKey = {
  id: string;
  name: string;
  prefix: string;
  scopes: string[];
  last_used_at: string | null;
  expires_at: string | null;
  revoked_at: string | null;
  created_at: string;
  created_by: string;
  active: boolean;
};

export async function fetchAdminAPIKeys() {
  return apiFetch<{ keys: AdminAPIKey[]; all_scopes: string[] }>("/v1/admin/api-keys");
}

export async function createAdminAPIKey(data: {
  name: string;
  scopes: string[];
  expires_at?: string;
}) {
  return apiFetch<{ id: string; name: string; prefix: string; key: string }>(
    "/v1/admin/api-keys",
    { method: "POST", body: JSON.stringify(data) }
  );
}

export async function revokeAdminAPIKey(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/api-keys/${id}`, { method: "DELETE" });
}

export type AdminWebhook = {
  id: string;
  url: string;
  events: string[];
  active: boolean;
  description: string;
  created_at: string;
  failed_count: number;
  last_sent_at: string | null;
};

export type WebhookEvent = { key: string; label: string };

export type WebhookDelivery = {
  event: string;
  status: "pending" | "delivered" | "failed";
  attempts: number;
  response_code?: number;
  error: string | null;
  created_at: string;
  delivered_at: string | null;
};

export async function fetchAdminWebhooks() {
  return apiFetch<{ webhooks: AdminWebhook[]; events: WebhookEvent[] }>("/v1/admin/webhooks");
}

export async function createAdminWebhook(data: {
  url: string;
  events: string[];
  description?: string;
}) {
  return apiFetch<{ id: string; url: string; secret: string }>("/v1/admin/webhooks", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateAdminWebhook(
  id: string,
  data: { active?: boolean; events?: string[]; description?: string }
) {
  return apiFetch<{ status: string }>(`/v1/admin/webhooks/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
}

export async function deleteAdminWebhook(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/webhooks/${id}`, { method: "DELETE" });
}

export async function testAdminWebhook(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/webhooks/${id}/test`, { method: "POST" });
}

export async function fetchWebhookDeliveries(id: string) {
  return apiFetch<{ deliveries: WebhookDelivery[] }>(
    `/v1/admin/webhooks/${id}/deliveries`
  );
}

export type LoginAttempt = {
  email: string;
  ip: string;
  user_agent: string;
  success: boolean;
  reason: string;
  created_at: string;
};

export type SuspiciousIP = { ip: string; fails: number; last_at: string };

export type IPBlock = {
  id: string;
  ip: string;
  reason: string;
  auto: boolean;
  expires_at: string | null;
  created_at: string;
  created_by: string;
  active: boolean;
};

export async function fetchLoginAttempts(
  params: { failed?: boolean; search?: string } = {}
) {
  const q = new URLSearchParams();
  if (params.failed) q.set("failed", "1");
  if (params.search) q.set("search", params.search);
  const suffix = q.toString() ? `?${q.toString()}` : "";
  return apiFetch<{ attempts: LoginAttempt[]; suspicious: SuspiciousIP[] }>(
    `/v1/admin/security/logins${suffix}`
  );
}

export async function fetchIPBlocks() {
  return apiFetch<{ blocks: IPBlock[] }>("/v1/admin/security/ip-blocks");
}

export async function createIPBlock(data: {
  ip: string;
  reason?: string;
  hours?: number;
}) {
  return apiFetch<{ id: string; ip: string }>("/v1/admin/security/ip-blocks", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function deleteIPBlock(id: string) {
  return apiFetch<{ status: string }>(`/v1/admin/security/ip-blocks/${id}`, {
    method: "DELETE",
  });
}

export type SearchHit = {
  kind: "user" | "server" | "payment" | "ticket" | "location";
  id: string;
  title: string;
  subtitle: string;
  url: string;
};

export async function adminSearch(query: string) {
  return apiFetch<{ results: SearchHit[] }>(
    `/v1/admin/search?q=${encodeURIComponent(query)}`
  );
}
