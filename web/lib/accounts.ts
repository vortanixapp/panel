const ACCOUNTS_KEY = "vortanix_accounts";

const MAX_ACCOUNTS = 6;

export type StoredAccount = {
  id: string;
  email: string;
  role: string;
  tenantSlug: string;
  accessToken: string;
  refreshToken: string;
  addedAt: number;
  lastUsedAt: number;
  viaId?: string;
  viaEmail?: string;
  needsSignIn?: boolean;
};

function read(): StoredAccount[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(ACCOUNTS_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(
      (a): a is StoredAccount =>
        !!a &&
        typeof a === "object" &&
        typeof (a as StoredAccount).id === "string" &&
        typeof (a as StoredAccount).refreshToken === "string"
    );
  } catch {
    return [];
  }
}

function write(list: StoredAccount[]) {
  try {
    const trimmed = [...list]
      .sort((a, b) => b.lastUsedAt - a.lastUsedAt)
      .slice(0, MAX_ACCOUNTS);
    localStorage.setItem(ACCOUNTS_KEY, JSON.stringify(trimmed));
  } catch {
  }
}

export function listAccounts(): StoredAccount[] {
  return read().sort((a, b) => b.lastUsedAt - a.lastUsedAt);
}

export function findAccount(id: string): StoredAccount | undefined {
  return read().find((a) => a.id === id);
}

export type AccessClaims = {
  user_id: string;
  email: string;
  role: string;
  tenant_slug: string;
  session_id?: string;
  exp?: number;
};

export function decodeAccess(token: string): AccessClaims | null {
  try {
    const payload = token.split(".")[1];
    if (!payload) return null;
    const normalized = payload.replace(/-/g, "+").replace(/_/g, "/");
    const json = decodeURIComponent(
      atob(normalized)
        .split("")
        .map((c) => "%" + c.charCodeAt(0).toString(16).padStart(2, "0"))
        .join("")
    );
    const claims = JSON.parse(json) as Partial<AccessClaims>;
    if (!claims.user_id) return null;
    return {
      user_id: claims.user_id,
      email: claims.email ?? "",
      role: claims.role ?? "user",
      tenant_slug: claims.tenant_slug ?? "",
      session_id: claims.session_id,
      exp: claims.exp,
    };
  } catch {
    return null;
  }
}

export function rememberTokens(accessToken: string, refreshToken: string) {
  const claims = decodeAccess(accessToken);
  if (!claims) return;

  const list = read();
  const existing = list.find((a) => a.id === claims.user_id);
  const now = Math.max(Date.now(), ...list.map((a) => a.lastUsedAt + 1));

  if (existing) {
    existing.email = claims.email || existing.email;
    existing.role = claims.role || existing.role;
    existing.tenantSlug = claims.tenant_slug || existing.tenantSlug;
    existing.accessToken = accessToken;
    if (refreshToken) existing.refreshToken = refreshToken;
    existing.lastUsedAt = now;
    existing.needsSignIn = false;
  } else {
    list.push({
      id: claims.user_id,
      email: claims.email,
      role: claims.role,
      tenantSlug: claims.tenant_slug,
      accessToken,
      refreshToken,
      addedAt: now,
      lastUsedAt: now,
    });
  }
  write(list);
}

export function markEnteredVia(userID: string, via: { id: string; email: string }) {
  const list = read();
  const target = list.find((a) => a.id === userID);
  if (!target) return;
  target.viaId = via.id;
  target.viaEmail = via.email;
  write(list);
}

export function markNeedsSignIn(id: string) {
  const list = read();
  const target = list.find((a) => a.id === id);
  if (!target) return;
  target.needsSignIn = true;
  target.accessToken = "";
  target.refreshToken = "";
  write(list);
}

export function forgetAccount(id: string) {
  write(read().filter((a) => a.id !== id));
}

export function forgetAllAccounts() {
  try {
    localStorage.removeItem(ACCOUNTS_KEY);
  } catch {
  }
}

export function accountInitials(email: string): string {
  const local = email.split("@")[0] ?? email;
  return local.slice(0, 2).toUpperCase();
}
