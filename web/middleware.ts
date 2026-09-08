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
};

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
