"use client";

import { useMemo, useState } from "react";
import { Loader2, Plus } from "lucide-react";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { DataTable } from "@/components/data-table";
import { getUsersColumns } from "@/components/users/users-columns";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { PasswordInput } from "@/components/password-input";
import type { PanelUser } from "@/lib/api";
import { isOwnerRole } from "@/lib/rbac";
import type { PanelVariant } from "@/lib/panel-paths";
import {
  useCreateUser,
  useDeleteUser,
  useMe,
  useUsers,
} from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";

export function UsersPage({ variant = "admin" }: { variant?: PanelVariant }) {
  const t = useT();
  const { data: me, isLoading: meLoading } = useMe();
  const { data: users = [], isLoading, isError } = useUsers();
  const createUser = useCreateUser();
  const deleteUser = useDeleteUser();

  const [open, setOpen] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState<"user" | "admin">("user");
  const [userToDelete, setUserToDelete] = useState<PanelUser | null>(null);

  const columns = useMemo(
    () =>
      getUsersColumns(
        {
          onDelete: setUserToDelete,
          deletePending: deleteUser.isPending,
        },
        t
      ),
    [deleteUser.isPending, t]
  );

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    try {
      await createUser.mutateAsync({ email, password, role });
      toast.success(t("admin.users.created"));
      setOpen(false);
      setEmail("");
      setPassword("");
      setRole("user");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("admin.users.create_failed")
      );
    }
  }

  async function confirmDelete() {
    if (!userToDelete) return;
    try {
      await deleteUser.mutateAsync(userToDelete.id);
      toast.success(t("admin.users.deleted"));
      setUserToDelete(null);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("admin.users.delete_failed")
      );
    }
  }

  if (meLoading || !me) {
    return (
      <PageShell variant={variant}>
        <Skeleton className="h-64 w-full" />
      </PageShell>
    );
  }

  return (
    <PageShell variant={variant}>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("common.users")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.users.accounts_subtitle")}
          </p>
        </div>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus />
              {t("common.add")}
            </Button>
          </DialogTrigger>
          <DialogContent>
            <form onSubmit={handleCreate}>
              <DialogHeader>
                <DialogTitle>{t("admin.users.create.title")}</DialogTitle>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="user-email">{t("common.email")}</Label>
                  <Input
                    id="user-email"
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    required
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="user-password">{t("common.password")}</Label>
                  <PasswordInput
                    id="user-password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    minLength={8}
                    required
                  />
                </div>
                <div className="grid gap-2">
                  <Label>{t("admin.users.role")}</Label>
                  <Select
                    value={role}
                    onValueChange={(v) => setRole(v as "user" | "admin")}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="user">
                        {t("admin.users.role.user")}
                      </SelectItem>
                      {isOwnerRole(me.role) && (
                        <SelectItem value="admin">
                          {t("admin.users.role.admin")}
                        </SelectItem>
                      )}
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <DialogFooter>
                <Button type="submit" disabled={createUser.isPending}>
                  {createUser.isPending && <Loader2 className="animate-spin" />}
                  {t("common.create")}
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {isError && (
        <p className="mb-4 text-sm text-destructive">
          {t("admin.users.list_load_failed")}
        </p>
      )}

      {isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <DataTable
          columns={columns}
          data={users}
          searchPlaceholder={t("admin.users.search_by_email")}
          searchKey="email"
          filters={[
            {
              columnId: "role",
              title: t("admin.users.role"),
              options: [
                { label: t("admin.users.role.owner"), value: "owner" },
                { label: t("admin.users.role.admin"), value: "admin" },
                { label: t("admin.users.role.user"), value: "user" },
              ],
            },
            {
              columnId: "status",
              title: t("common.status"),
              options: [
                { label: t("admin.users.status.active"), value: "active" },
              ],
            },
          ]}
          emptyMessage={t("admin.users.empty")}
        />
      )}

      <AlertDialog
        open={!!userToDelete}
        onOpenChange={(open) => !open && setUserToDelete(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t("admin.users.delete_confirm_title")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("admin.users.delete_confirm_text", {
                email: userToDelete?.email ?? "",
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteUser.isPending ? t("common.deleting") : t("common.delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </PageShell>
  );
}
