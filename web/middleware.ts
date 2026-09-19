import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const LEGACY_REDIRECTS: Record<string, string> = {
  "/my-servers": "/servers",
  "/my-hosting": "/hosting/my",
  "/rent-hosting": "/hosting/rent",
  "/account": "/settings/account",
  "/admin/vortanix-daemons": "/admin/daemons",
  "/admin/promotions": "/admin/promo",
  "/auth/login": "/login",
  "/auth/register": "/register",
  "/auth/forgot-password": "/forgot-password",
  "/auth/reset-password": "/reset-password",
  "/auth/two-factor-challenge": "/two-factor-challenge",
  "/billing/topup": "/billing",
};

const PAYMENT_PATH = /^\/billing(?:\/topup)?\/payment\/([A-Za-z0-9-]{1,64})\/?$/;

const REFERRAL_COOKIE = "vtx_ref";
const REFERRAL_CODE = /^[A-Za-z0-9]{4,32}$/;
const REFERRAL_MAX_AGE = 60 * 60 * 24 * 30;

let bootstrapped = false;
let checkedAt = 0;
const NEGATIVE_TTL_MS = 3_000;

async function isBootstrapped(origin: string): Promise<boolean> {
  if (bootstrapped) return true;
  if (Date.now() - checkedAt < NEGATIVE_TTL_MS) return false;
  try {
    const res = await fetch(`${origin}/api/setup-state`, {
      cache: "no-store",
      signal: AbortSignal.timeout(3_000),
    });
    if (!res.ok) return true;
    const data = (await res.json()) as { bootstrapped?: boolean };
    checkedAt = Date.now();
    if (data.bootstrapped) bootstrapped = true;
    return Boolean(data.bootstrapped);
  } catch {
    return true;
  }
}

function servedBeforeSetup(pathname: string) {
  return (
    pathname === "/setup" ||
    pathname.startsWith("/api/") ||
    pathname.startsWith("/_next/") ||
    pathname === "/favicon.ico"
  );
}

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  if (LEGACY_REDIRECTS[pathname]) {
    const url = request.nextUrl.clone();
    url.pathname = LEGACY_REDIRECTS[pathname];
    return NextResponse.redirect(url);
  }

  const ref = request.nextUrl.searchParams.get("ref");
  if (ref !== null && !pathname.startsWith("/api/")) {
    const url = request.nextUrl.clone();
    url.searchParams.delete("ref");
    const res = NextResponse.redirect(url);
    if (REFERRAL_CODE.test(ref)) {
      res.cookies.set(REFERRAL_COOKIE, ref.toLowerCase(), {
        path: "/",
        maxAge: REFERRAL_MAX_AGE,
        sameSite: "lax",
        secure: request.nextUrl.protocol === "https:",
      });
    }
    return res;
  }

  const payment = PAYMENT_PATH.exec(pathname);
  if (payment) {
    const url = request.nextUrl.clone();
    url.pathname = "/billing";
    url.search = `?payment=${payment[1]}`;
    return NextResponse.redirect(url);
  }

  if (pathname.startsWith("/admin/servers/") && pathname.includes("/manage")) {
    const url = request.nextUrl.clone();
    url.pathname = pathname.replace("/manage", "");
    return NextResponse.redirect(url);
  }

  if (!servedBeforeSetup(pathname) && !(await isBootstrapped(request.nextUrl.origin))) {
    const url = request.nextUrl.clone();
    url.pathname = "/setup";
    url.search = "";
    return NextResponse.redirect(url);
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|.*\\.[\\w]+$).*)"],
};
