import { RUNTIME_CONFIG_GLOBAL, serverRuntimeConfig } from "@/lib/runtime-config";

export const dynamic = "force-dynamic";

export function GET() {
  const config = serverRuntimeConfig();
  const body = `window.${RUNTIME_CONFIG_GLOBAL}=${JSON.stringify(config)};`;
  return new Response(body, {
    headers: {
      "Content-Type": "application/javascript; charset=utf-8",
      "Cache-Control": "no-store",
    },
  });
}
