"use client";

import Link from "next/link";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  createAdminMailing,
  deleteAdminMailing,
  fetchAdminMailings,
  sendAdminMailing,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";

type Mailing = { id: string; subject: string; status: string };

export default function AdminMailingsPage() {
  const t = useT();
  const qc = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: ["admin-mailings"],
    queryFn: async () => (await fetchAdminMailings()).mailings as Mailing[],
  });
  const mailings = data ?? [];

  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ subject: "", body: "" });

  const createMut = useMutation({
    mutationFn: (d: { subject: string; body: string }) => createAdminMailing({ subject: d.subject, body: d.body }),
    onSuccess: () => {
      toast.success(t("admin.mailings.created"));
      setShowForm(false);
      setForm({ subject: "", body: "" });
      qc.invalidateQueries({ queryKey: ["admin-mailings"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const sendMut = useMutation({
    mutationFn: (id: string) => sendAdminMailing(id),
    onSuccess: () => {
      toast.success(t("admin.mailings.queued"));
      qc.invalidateQueries({ queryKey: ["admin-mailings"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteAdminMailing(id),
    onSuccess: () => {
      toast.success(t("admin.mailings.deleted"));
      qc.invalidateQueries({ queryKey: ["admin-mailings"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  function onCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!form.subject.trim()) return;
    createMut.mutate(form);
  }

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("admin.mailings.title")}</h1>
          <p className="text-sm text-muted-foreground">{t("admin.mailings.subtitle")}</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" asChild>
            <Link href="/admin">{t("layout.back_arrow")}</Link>
          </Button>
          <Button onClick={() => setShowForm((s) => !s)}>
            {showForm ? t("common.hide") : t("admin.mailings.create")}
          </Button>
        </div>
      </div>

      {showForm && (
        <form onSubmit={onCreate} className="mb-6 space-y-3 rounded-lg border bg-card p-4">
          <div>
            <Label>{t("admin.mailings.subject")}</Label>
            <Input value={form.subject} onChange={(e) => setForm({ ...form, subject: e.target.value })} required />
          </div>
          <div>
            <Label>{t("admin.mailings.body")}</Label>
            <Textarea value={form.body} onChange={(e) => setForm({ ...form, body: e.target.value })} className="min-h-[120px]" />
          </div>
          <Button type="submit" disabled={createMut.isPending}>
            {t("admin.mailings.create_submit")}
          </Button>
        </form>
      )}

      <div className="rounded-lg border bg-card">
        {isLoading ? <div className="p-6 text-sm">{t("common.loading")}</div> : mailings.length === 0 ? (
          <div className="p-8 text-center text-sm text-muted-foreground">
            {t("admin.mailings.empty")}
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/30">
                <th className="px-4 py-2 text-left">{t("admin.mailings.subject")}</th>
                <th className="px-4 py-2 text-left">{t("common.status")}</th>
                <th className="px-4 py-2 text-right">{t("common.actions")}</th>
              </tr>
            </thead>
            <tbody>
              {mailings.map((m) => (
                <tr key={m.id} className="border-b last:border-0">
                  <td className="px-4 py-2">{m.subject}</td>
                  <td className="px-4 py-2"><span className="rounded bg-muted px-2 py-0.5 text-xs">{m.status}</span></td>
                  <td className="px-4 py-2 text-right space-x-2">
                    <Button size="sm" variant="secondary" onClick={() => sendMut.mutate(m.id)} disabled={sendMut.isPending || m.status === "queued" || m.status === "sent"}>
                      {t("common.send")}
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        if (
                          !confirm(
                            t("admin.mailings.delete_confirm", { subject: m.subject })
                          )
                        )
                          return;
                        deleteMut.mutate(m.id);
                      }}
                      disabled={deleteMut.isPending || m.status === "sending"}
                    >
                      {t("common.delete")}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </PageShell>
  );
}
