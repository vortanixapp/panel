import { serverRuntimeConfig } from "@/lib/runtime-config";

export const dynamic = "force-dynamic";

export function GET() {
  return Response.json(
    { status: "ok", version: serverRuntimeConfig().version },
    { headers: { "Cache-Control": "no-store" } }
  );
}
