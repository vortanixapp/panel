export type StoredAccount = {
  id: string;
  email: string;
  role: string;
  isCurrent: boolean;
  avatarUrl: string | null;
};

export type AccessClaims = {
  user_id: string;
  email: string;
  role: string;
  exp?: number;
};

export const USER_COOKIE = "vtx_user";

function readCookie(name: string): string {
  if (typeof document === "undefined") return "";
  for (const part of document.cookie.split(";")) {
    const [key, ...rest] = part.trim().split("=");
    if (key === name) return decodeURIComponent(rest.join("="));
  }
  return "";
}

export function csrfToken(): string {
  return readCookie("vtx_csrf");
}

export function currentAccount(): AccessClaims | null {
  return parseAccountCookie(readCookie(USER_COOKIE));
}

export function parseAccountCookie(raw: string): AccessClaims | null {
  if (!raw) return null;
  try {
    const normalized = raw.replace(/-/g, "+").replace(/_/g, "/");
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
      exp: claims.exp,
    };
  } catch {
    return null;
  }
}

export function accountInitials(email: string, name?: string | null): string {
  const words = (name ?? "").trim().split(/\s+/).filter(Boolean);
  if (words.length > 1) {
    return (words[0].charAt(0) + words[1].charAt(0)).toUpperCase();
  }
  if (words.length === 1) {
    return words[0].slice(0, 2).toUpperCase();
  }
  const local = email.split("@")[0] ?? email;
  return local.slice(0, 2).toUpperCase();
}
