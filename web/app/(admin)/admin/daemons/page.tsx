import { Suspense } from "react";
import { AgentsPage } from "@/components/admin/agents/agents-page";

export default function AdminDaemonsPage() {
  return (
    <Suspense>
      <AgentsPage />
    </Suspense>
  );
}
