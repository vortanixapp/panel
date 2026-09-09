import { checkBootstrapped } from "@/lib/setup-gate";

export const dynamic = "force-dynamic";

export async function GET() {
  return Response.json(
    { bootstrapped: await checkBootstrapped() },
    { headers: { "Cache-Control": "no-store" } },
  );
}
