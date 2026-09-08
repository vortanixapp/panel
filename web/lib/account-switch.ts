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

/**
 * Переключение между сохранёнными аккаунтами.
 *
 * Порядок здесь важнее самого действия: сессию соседнего аккаунта проверяем
 * ДО того, как подменить рабочие токены. Наоборот было бы хуже всего — человек
 * оказался бы на форме входа, потеряв и тот аккаунт, в котором только что
 * работал: активную сессию мы уже затёрли, а новая не поднялась.
 */

export type SwitchOutcome =
  | { ok: true; path: string }
  | { ok: false; reason: "needs-sign-in"; email: string }
  | { ok: false; reason: "error"; message: string };

/**
 * Продлить чужую сессию, не трогая активную.
 *
 * Мимо apiFetch намеренно: тот подставляет арендатора текущего аккаунта и по
 * дороге успевает обновить активную сессию. У соседней записи арендатор может
 * быть другим, и заголовок нужен её собственный — иначе сервер пойдёт искать
 * пользователя не в той базе и ответит «user not found».
 */
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

/**
 * Подготовить переход в аккаунт. Активные токены подменяются только после
 * успешной проверки; вызывающему остаётся увести браузер по возвращённому пути.
 */
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
    // Сеть отвалилась — сессия, скорее всего, жива, и запись портить нельзя.
    // Отзыв сессии сервер называет прямо, и только тогда просим пароль.
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

  // Арендатор — раньше токенов: заголовок X-Tenant-Slug читается из этого же
  // хранилища, и запрос сразу после переключения должен уйти в нужную базу.
  setTokens(tokens.access_token, tokens.refresh_token);

  return { ok: true, path: postLoginPath(account.role) };
}

/**
 * Уйти в аккаунт полной загрузкой страницы.
 *
 * Не router.push: к личности привязаны живой веб-сокет, кэш запросов и цикл
 * продления сессии. Мягкий переход оставил бы сокет открытым от имени
 * предыдущего аккаунта, а на экране — его данные до первого ответа сервера.
 * Перезагрузка снимает всё это разом и стоит доли секунды.
 */
export function goToAccount(path: string) {
  window.location.replace(path);
}

/** Куда возвращаться после выхода: следующий сохранённый аккаунт или вход. */
export function nextAccountAfterLogout(currentId: string | null): StoredAccount | null {
  return listAccounts().find((a) => a.id !== currentId && !a.needsSignIn) ?? null;
}

/** Выйти из всех сохранённых аккаунтов сразу. */
export function forgetEveryAccount() {
  for (const a of listAccounts()) forgetAccount(a.id);
  clearAuthAndForget();
}
