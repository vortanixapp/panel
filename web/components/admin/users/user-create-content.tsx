"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { useCreateUser } from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";

const inputCls =
  "block w-full rounded-xl border border-border bg-muted px-4 py-2.5 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary";

export function UserCreateContent() {
  const t = useT();
  const router = useRouter();
  const createUser = useCreateUser();
  const [error, setError] = useState("");
  const [f, setF] = useState({
    name: "",
    email: "",
    password: "",
    password_confirmation: "",
    role: "user",
  });
  const set = (k: string, v: string) => setF((p) => ({ ...p, [k]: v }));

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    try {
      await createUser.mutateAsync({
        name: f.name,
        email: f.email,
        password: f.password,
        password_confirmation: f.password_confirmation,
        role: f.role as "user" | "admin" | "support",
      });
      toast.success(t("admin.users.created"));
      router.push("/admin/users");
    } catch (err) {
      const msg = err instanceof Error ? err.message : t("common.error");
      setError(msg);
      toast.error(msg);
    }
  };

  return (
    <PageShell variant="admin">
      <div className="space-y-6">
        <div className="relative overflow-hidden rounded-2xl border border-border bg-gradient-to-br from-primary/10 via-card to-card p-6">
          <div className="pointer-events-none absolute top-0 right-0 h-64 w-64 translate-x-1/2 -translate-y-1/2 rounded-full bg-primary/10 blur-3xl" />
          <div className="relative flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br from-primary to-primary/70 text-primary-foreground shadow-lg shadow-primary/25">
                <i className="ri-user-add-line text-2xl" />
              </div>
              <div>
                <h1 className="text-xl font-bold text-foreground lg:text-2xl">
                  {t("admin.users.create.title")}
                </h1>
                <p className="text-sm text-muted-foreground">
                  {t("admin.users.create.subtitle")}
                </p>
              </div>
            </div>
            <Link
              href="/admin/users"
              className="inline-flex items-center gap-2 rounded-xl bg-muted px-4 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted/80"
            >
              <i className="ri-arrow-left-line" /> {t("common.back")}
            </Link>
          </div>
        </div>

        {error && (
          <div className="rounded-xl border border-rose-500/20 bg-rose-500/10 px-4 py-3 text-sm text-rose-500">
            {error}
          </div>
        )}

        <div className="rounded-2xl border border-border bg-card p-6">
          <form onSubmit={submit} className="space-y-5">
            <div className="grid gap-5 md:grid-cols-2">
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">
                  {t("admin.users.name")}
                </label>
                <input
                  type="text"
                  value={f.name}
                  onChange={(e) => set("name", e.target.value)}
                  required
                  className={inputCls}
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">Email</label>
                <input
                  type="email"
                  value={f.email}
                  onChange={(e) => set("email", e.target.value)}
                  required
                  className={inputCls}
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">
                  {t("common.password")}
                </label>
                <input
                  type="password"
                  value={f.password}
                  onChange={(e) => set("password", e.target.value)}
                  required
                  className={inputCls}
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">
                  {t("admin.users.password_confirm")}
                </label>
                <input
                  type="password"
                  value={f.password_confirmation}
                  onChange={(e) => set("password_confirmation", e.target.value)}
                  required
                  className={inputCls}
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">
                  {t("admin.users.role")}
                </label>
                <select
                  value={f.role}
                  onChange={(e) => set("role", e.target.value)}
                  className={inputCls}
                >
                  <option value="user">{t("admin.users.role.user")}</option>
                  <option value="support">
                    {t("admin.users.role.support_full")}
                  </option>
                  <option value="admin">
                    {t("admin.users.role.admin_short")}
                  </option>
                </select>
              </div>
            </div>
            <div className="flex items-center justify-end gap-3 border-t border-border pt-4">
              <Link
                href="/admin/users"
                className="px-4 py-2 text-sm text-muted-foreground hover:text-foreground"
              >
                {t("common.cancel")}
              </Link>
              <Button type="submit" disabled={createUser.isPending}>
                <i className="ri-save-line mr-2" />
                {createUser.isPending
                  ? t("common.creating")
                  : t("common.create")}
              </Button>
            </div>
          </form>
        </div>
      </div>
    </PageShell>
  );
}
