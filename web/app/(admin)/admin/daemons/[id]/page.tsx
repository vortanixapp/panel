import { Suspense } from "react";
import { AgentShell } from "@/components/admin/agents/agent-shell";

export default function AdminAgentDetailPage() {
  return (
    <Suspense>
      <AgentShell />
    </Suspense>
  );
}
