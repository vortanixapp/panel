import { forgetSavedAccount, savedAccounts, switchAccount } from "@/lib/api";
import { postLoginPath } from "@/lib/auth-redirect";
import { t } from "@/lib/i18n";
import type { StoredAccount } from "@/lib/accounts";

export type SwitchOutcome =
  | { ok: true; path: string }
  | { ok: false; reason: "needs-sign-in"; email: string }
  | { ok: false; reason: "error"; message: string };

export async function loadAccounts(): Promise<StoredAccount[]> {
  try {
    const res = await savedAccounts();
    return res.accounts.map((a) => ({
      id: a.id,
      email: a.email,
      role: a.role,
      isCurrent: a.is_current,
    }));
  } catch {
    return [];
  }
}

export async function prepareSwitch(id: string, email = ""): Promise<SwitchOutcome> {
  try {
    const res = await switchAccount(id);
    return { ok: true, path: postLoginPath(res.user.role) };
  } catch (err) {
    const message = err instanceof Error ? err.message : "";
    if (
      message.includes("войдите в него заново") ||
      message.includes("недоступна") ||
      message.includes("sign in")
    ) {
      return { ok: false, reason: "needs-sign-in", email };
    }
    return {
      ok: false,
      reason: "error",
      message: message || t("layout.account.server_unreachable"),
    };
  }
}

export function goToAccount(path: string) {
  window.location.replace(path);
}

export async function nextAccountAfterLogout(
  currentId: string | null
): Promise<StoredAccount | null> {
  const list = await loadAccounts();
  return list.find((a) => a.id !== currentId) ?? null;
}

export async function forgetEveryAccount() {
  try {
    await forgetSavedAccount({ all: true });
  } catch {
  }
}
