import {
  API_URL,
  clearAuthAndForget,
  setTokens,
} from "@/lib/api";
import { postLoginPath } from "@/lib/auth-redirect";
import { t } from "@/lib/i18n";
import {
  findAccount,
  forgetAccount,
  listAccounts,
  markNeedsSignIn,
  type StoredAccount,
} from "@/lib/accounts";

export type SwitchOutcome =
  | { ok: true; path: string }
  | { ok: false; reason: "needs-sign-in"; email: string }
  | { ok: false; reason: "error"; message: string };

async function refreshForeignSession(
  account: StoredAccount
): Promise<{ access_token: string; refresh_token: string }> {
  const res = await fetch(API_URL + "/v1/auth/refresh", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(account.tenantSlug ? { "X-Tenant-Slug": account.tenantSlug } : {}),
    },
    body: JSON.stringify({ refresh_token: account.refreshToken }),
  });
  const text = await res.text();
  let data: { access_token?: string; refresh_token?: string; error?: string } = {};
  if (text) {
    try {
      data = JSON.parse(text) as typeof data;
    } catch {
      data = {};
    }
  }
  if (!res.ok || !data.access_token) {
    throw new Error(data.error ?? "session expired");
  }
  return {
    access_token: data.access_token,
    refresh_token: data.refresh_token ?? "",
  };
}

export async function prepareSwitch(id: string): Promise<SwitchOutcome> {
  const account = findAccount(id);
  if (!account) {
    return { ok: false, reason: "error", message: t("layout.account.not_found") };
  }
  if (account.needsSignIn || !account.refreshToken) {
    return { ok: false, reason: "needs-sign-in", email: account.email };
  }

  let tokens: { access_token: string; refresh_token: string };
  try {
    tokens = await refreshForeignSession(account);
  } catch (err) {
    const message = err instanceof Error ? err.message : "";
    if (message === "session expired" || message === "сессия закрыта") {
      markNeedsSignIn(id);
      return { ok: false, reason: "needs-sign-in", email: account.email };
    }
    return {
      ok: false,
      reason: "error",
      message: t("layout.account.server_unreachable"),
    };
  }

  setTokens(tokens.access_token, tokens.refresh_token);

  return { ok: true, path: postLoginPath(account.role) };
}

export function goToAccount(path: string) {
  window.location.replace(path);
}

export function nextAccountAfterLogout(currentId: string | null): StoredAccount | null {
  return listAccounts().find((a) => a.id !== currentId && !a.needsSignIn) ?? null;
}

export function forgetEveryAccount() {
  for (const a of listAccounts()) forgetAccount(a.id);
  clearAuthAndForget();
}
