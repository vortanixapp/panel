/**
 * Список аккаунтов, между которыми можно переключаться без пароля.
 *
 * Хранит рядом с активной сессией сессии остальных аккаунтов, куда человек уже
 * входил в этом браузере. Раньше сессия была ровно одна: чтобы попасть в другой
 * аккаунт, нужно было выйти и ввести пароль заново, а админ, нажавший «войти как
 * пользователь», обратно не возвращался вовсе — ключи `vortanix_impersonator_*`
 * записывались и не читались ни одной строкой кода.
 *
 * Наполняется само. Токен доступа несёт `user_id`, `email`, `role` и
 * `tenant_slug`, поэтому единственная точка подключения — `setTokens` в
 * `lib/api.ts`: любой способ входа (пароль, 2FA, соцсеть, Telegram, регистрация,
 * вход под пользователем) уже проходит через неё и попадает в список без правок
 * на своей стороне.
 */

const ACCOUNTS_KEY = "vortanix_accounts";

/**
 * Сколько аккаунтов держим. Каждая запись — живой токен обновления в
 * localStorage, то есть цена одной чужой XSS. Ограничение не даёт списку расти
 * бесконечно на общих машинах; при переполнении вытесняется тот, кем дольше
 * всех не пользовались.
 */
const MAX_ACCOUNTS = 6;

export type StoredAccount = {
  /** user_id из токена — он же ключ записи. */
  id: string;
  email: string;
  role: string;
  /** Арендатор аккаунта: у соседней записи он может быть другим. */
  tenantSlug: string;
  accessToken: string;
  refreshToken: string;
  addedAt: number;
  lastUsedAt: number;
  /**
   * Аккаунт, из-под которого сюда вошли. Ставится при входе под пользователем:
   * по нему меню показывает «вернуться к себе».
   */
  viaId?: string;
  viaEmail?: string;
  /**
   * Сессия не отозвалась при переключении — нужен пароль. Запись не удаляем:
   * человек хотел попасть именно в этот аккаунт, и он должен остаться в списке
   * с понятной пометкой, а не исчезнуть без объяснений.
   */
  needsSignIn?: boolean;
};

function read(): StoredAccount[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(ACCOUNTS_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    // Чужую или устаревшую запись молча отбрасываем: список аккаунтов не та
    // вещь, из-за которой панель должна падать при старте.
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
    // Приватный режим или переполненное хранилище. Переключение аккаунтов —
    // удобство: без него панель работает как раньше, с одной сессией.
  }
}

/** Список для меню: сверху те, кем пользовались недавнее. */
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

/**
 * Разбор полезной нагрузки токена доступа.
 *
 * Подпись не проверяется и проверять её здесь нечем: значение идёт только на
 * подписи в меню и на выбор арендатора при переключении, а всё, что решает
 * доступ, всё равно проверяет сервер на каждом запросе.
 */
export function decodeAccess(token: string): AccessClaims | null {
  try {
    const payload = token.split(".")[1];
    if (!payload) return null;
    const normalized = payload.replace(/-/g, "+").replace(/_/g, "/");
    // atob не знает UTF-8: кириллическая почта без этого превращалась в кашу.
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

/**
 * Запомнить пару токенов как аккаунт.
 *
 * Зовётся из `setTokens`, то есть на каждом входе и на каждом продлении сессии.
 * Владелец определяется по самому токену, поэтому продление активного аккаунта
 * обновляет его запись, а вход в новый — заводит соседнюю, не затирая первую.
 */
export function rememberTokens(accessToken: string, refreshToken: string) {
  const claims = decodeAccess(accessToken);
  if (!claims) return;

  const list = read();
  const existing = list.find((a) => a.id === claims.user_id);
  // Строго больше всех остальных. Просто Date.now() недостаточно: при
  // переполнении списка вытесняется запись с наименьшей отметкой, а две
  // записи, тронутые в одну миллисекунду, сравниваются в произвольном порядке —
  // и выбыть мог тот самый аккаунт, в который только что вошли.
  const now = Math.max(Date.now(), ...list.map((a) => a.lastUsedAt + 1));

  if (existing) {
    existing.email = claims.email || existing.email;
    existing.role = claims.role || existing.role;
    existing.tenantSlug = claims.tenant_slug || existing.tenantSlug;
    existing.accessToken = accessToken;
    // Пустой refresh приходит у входов, которые его не выдают (Telegram, вход
    // под пользователем). Затирать им рабочий токен нельзя — иначе аккаунт
    // становится непереключаемым после первого же такого входа.
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

/** Пометить, из-под какого аккаунта вошли: для возврата к себе. */
export function markEnteredVia(userID: string, via: { id: string; email: string }) {
  const list = read();
  const target = list.find((a) => a.id === userID);
  if (!target) return;
  target.viaId = via.id;
  target.viaEmail = via.email;
  write(list);
}

/** Сессия не откликнулась — оставляем запись, но требуем пароль. */
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
    // см. write()
  }
}

/** Инициалы для кружка аватара. */
export function accountInitials(email: string): string {
  const local = email.split("@")[0] ?? email;
  return local.slice(0, 2).toUpperCase();
}
