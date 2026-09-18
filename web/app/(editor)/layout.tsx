import { AdminGuard } from "@/components/auth/admin-guard";
import { EditorSessionGate } from "@/components/site/editor/session-gate";

export default function EditorLayout({ children }: { children: React.ReactNode }) {
  return (
    <AdminGuard>
      <EditorSessionGate>{children}</EditorSessionGate>
    </AdminGuard>
  );
}
