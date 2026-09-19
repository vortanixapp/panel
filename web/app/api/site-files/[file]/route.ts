import { serverApiURL } from "@/lib/runtime-config";

export const dynamic = "force-dynamic";

const PASS_HEADERS = ["content-type", "content-security-policy", "x-content-type-options", "cache-control"];

async function load(file: string) {
  try {
    const res = await fetch(`${serverApiURL()}/v1/site/files/${encodeURIComponent(file)}`, {
      cache: "no-store",
      signal: AbortSignal.timeout(3_000),
    });
    return res.ok ? res : null;
  } catch {
    return null;
  }
}

function notFound() {
  return new Response("Not found", {
    status: 404,
    headers: { "Content-Type": "text/plain; charset=utf-8", "Cache-Control": "no-store" },
  });
}

async function serve(params: Promise<{ file: string }>, withBody: boolean) {
  const { file } = await params;
  const res = await load(file);
  if (!res) return notFound();
  const headers = new Headers();
  for (const name of PASS_HEADERS) {
    const value = res.headers.get(name);
    if (value) headers.set(name, value);
  }
  return new Response(withBody ? await res.arrayBuffer() : null, { status: 200, headers });
}

export async function GET(_request: Request, { params }: { params: Promise<{ file: string }> }) {
  return serve(params, true);
}

export async function HEAD(_request: Request, { params }: { params: Promise<{ file: string }> }) {
  return serve(params, false);
}
