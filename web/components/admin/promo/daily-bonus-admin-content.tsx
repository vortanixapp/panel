"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
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
import { Switch } from "@/components/ui/switch";
import {
  createAdminBonusPrize,
  deleteAdminBonusPrize,
  fetchAdminDailyBonus,
  updateAdminBonusPrize,
  type AdminBonusPrize,
  type AdminBonusPrizeInput,
  type AdminBonusPrizeType,
} from "@/lib/api";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

const QUERY_KEY = ["admin-daily-bonus"] as const;

const TYPES: AdminBonusPrizeType[] = ["balance", "promo_rent", "promo_renew", "promo_game", "promo_hosting"];

const PALETTE = ["#3a3b3d", "#555658", "#757678", "#9a9b9d", "#c9cacc", "#e8a03c", "#4f7cff", "#2fb67c"];

type FormState = {
  id?: string;
  label: string;
  type: AdminBonusPrizeType;
  value: string;
  discount_type: "percent" | "fixed";
  weight: string;
  color: string;
  icon: string;
  duration_hours: string;
  active: boolean;
};

const EMPTY: FormState = {
  label: "",
  type: "balance",
  value: "50",
  discount_type: "percent",
  weight: "10",
  color: PALETTE[0],
  icon: "ri-coin-line",
  duration_hours: "",
  active: true,
};

function toForm(p: AdminBonusPrize): FormState {
  return {
    id: p.id,
    label: p.label,
    type: p.type,
    value: String(p.value),
    discount_type: p.discount_type === "fixed" ? "fixed" : "percent",
    weight: String(p.weight),
    color: p.color || PALETTE[0],
    icon: p.icon,
    duration_hours: p.duration_hours ? String(p.duration_hours) : "",
    active: p.active,
  };
}

function toPayload(f: FormState): AdminBonusPrizeInput {
  const promo = f.type !== "balance";
  return {
    label: f.label.trim(),
    type: f.type,
    value: Number(f.value) || 0,
    discount_type: promo ? f.discount_type : "",
    weight: Math.max(0, Math.round(Number(f.weight) || 0)),
    color: f.color.trim(),
    icon: f.icon.trim(),
    duration_hours: promo ? Math.max(0, Math.round(Number(f.duration_hours) || 0)) : 0,
    active: f.active,
  };
}

function num(v: number): string {
  return new Intl.NumberFormat(localeTag(), { maximumFractionDigits: 2 }).format(v);
}

export function DailyBonusAdminContent() {
  const t = useT();
  const qc = useQueryClient();
  const query = useQuery({ queryKey: QUERY_KEY, queryFn: fetchAdminDailyBonus });
  const [form, setForm] = useState<FormState | null>(null);
  const [deleting, setDeleting] = useState<AdminBonusPrize | null>(null);

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: QUERY_KEY });
    void qc.invalidateQueries({ queryKey: ["daily-bonus"] });
  };

  const save = useMutation({
    mutationFn: async (f: FormState) => {
      if (f.id) await updateAdminBonusPrize(f.id, toPayload(f));
      else await createAdminBonusPrize(toPayload(f));
    },
    onSuccess: (_, f) => {
      toast.success(f.id ? t("admin.bonus.updated") : t("admin.bonus.created"));
      setForm(null);
      invalidate();
    },
    onError: (e: Error) => toast.error(e.message || t("common.save_failed")),
  });

  const toggle = useMutation({
    mutationFn: (p: AdminBonusPrize) => updateAdminBonusPrize(p.id, { active: !p.active }),
    onSuccess: invalidate,
    onError: (e: Error) => toast.error(e.message || t("common.save_failed")),
  });

  const remove = useMutation({
    mutationFn: (id: string) => deleteAdminBonusPrize(id),
    onSuccess: () => {
      toast.success(t("admin.bonus.deleted"));
      setDeleting(null);
      invalidate();
    },
    onError: (e: Error) => toast.error(e.message || t("common.delete_failed")),
  });

  const prizes = query.data?.prizes ?? [];
  const stats = query.data?.stats;
  const playable = prizes.filter((p) => p.active && p.weight > 0 && p.value > 0);

  const valueText = (p: Pick<AdminBonusPrize, "type" | "value" | "discount_type">) =>
    p.type === "balance"
      ? t("admin.bonus.value_balance", { value: num(p.value) })
      : p.discount_type === "fixed"
        ? t("admin.bonus.value_fixed", { value: num(p.value) })
        : t("admin.bonus.value_percent", { value: num(p.value) });

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!form) return;
    if (!form.label.trim()) {
      toast.error(t("admin.bonus.need_label"));
      return;
    }
    if (!(Number(form.value) > 0)) {
      toast.error(t("admin.bonus.need_value"));
      return;
    }
    save.mutate(form);
  }

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("admin.bonus.title")}</h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.bonus.subtitle", { hours: query.data?.cooldown_hours ?? 24 })}
          </p>
        </div>
        <Button onClick={() => setForm({ ...EMPTY, color: PALETTE[prizes.length % PALETTE.length] })}>
          <Plus />
          {t("admin.bonus.add")}
        </Button>
      </div>

      {stats && (
        <div className="mb-4 grid grid-cols-2 gap-px overflow-hidden rounded-lg border bg-border md:grid-cols-4">
          {[
            [t("admin.bonus.stats.spins_24h"), num(stats.spins_24h)],
            [t("admin.bonus.stats.spins_7d"), num(stats.spins_7d)],
            [t("admin.bonus.stats.players_7d"), num(stats.players_7d)],
            [t("admin.bonus.stats.balance_7d"), t("admin.bonus.value_balance", { value: num(stats.balance_7d) })],
          ].map(([label, value]) => (
            <div key={label} className="bg-card px-4 py-3">
              <div className="text-[12px] text-muted-foreground">{label}</div>
              <div className="mt-1 text-lg font-semibold tabular-nums">{value}</div>
            </div>
          ))}
        </div>
      )}

      {!query.isLoading && playable.length === 0 && (
        <div className="mb-4 rounded-lg border border-amber-500/40 bg-amber-500/5 px-4 py-3 text-[13px] text-amber-600 dark:text-amber-300">
          {t("admin.bonus.disabled_hint")}
        </div>
      )}

      <div className="overflow-x-auto rounded-lg border bg-card">
        {query.isLoading ? (
          <div className="space-y-2 p-4">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : prizes.length === 0 ? (
          <p className="px-4 py-10 text-center text-sm text-muted-foreground">{t("admin.bonus.empty")}</p>
        ) : (
          <table className="w-full min-w-[760px] text-sm">
            <thead className="border-b bg-muted/40 text-left text-[11px] tracking-wider text-muted-foreground uppercase">
              <tr>
                <th className="px-4 py-2.5 font-normal">{t("admin.bonus.col.prize")}</th>
                <th className="px-4 py-2.5 font-normal">{t("admin.bonus.col.type")}</th>
                <th className="px-4 py-2.5 font-normal">{t("admin.bonus.col.value")}</th>
                <th className="px-4 py-2.5 text-right font-normal">{t("admin.bonus.col.weight")}</th>
                <th className="px-4 py-2.5 text-right font-normal">{t("admin.bonus.col.chance")}</th>
                <th className="px-4 py-2.5 text-right font-normal">{t("admin.bonus.col.wins")}</th>
                <th className="px-4 py-2.5 font-normal">{t("admin.bonus.col.active")}</th>
                <th className="w-24" />
              </tr>
            </thead>
            <tbody>
              {prizes.map((p) => {
                const broken = p.value <= 0;
                return (
                  <tr key={p.id} className={cn("border-b last:border-0", !p.active && "opacity-60")}>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2.5">
                        <span
                          className="flex size-7 shrink-0 items-center justify-center rounded-md text-white"
                          style={{ background: p.color || PALETTE[0] }}
                        >
                          {p.icon && <i className={cn(p.icon, "text-[14px]")} />}
                        </span>
                        <div className="min-w-0">
                          <div className="truncate font-medium">{p.label}</div>
                          {broken && (
                            <div className="text-[11.5px] text-rose-500">{t("admin.bonus.broken")}</div>
                          )}
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">{t(`admin.bonus.type.${p.type}`)}</td>
                    <td className="px-4 py-3">
                      {valueText(p)}
                      {p.type !== "balance" && p.duration_hours > 0 && (
                        <div className="text-[11.5px] text-muted-foreground">
                          {t("admin.bonus.valid_for", { hours: p.duration_hours })}
                        </div>
                      )}
                    </td>
                    <td className="px-4 py-3 text-right font-mono tabular-nums">{p.weight}</td>
                    <td className="px-4 py-3 text-right font-mono tabular-nums">
                      {p.chance > 0 ? `${num(p.chance)}%` : "—"}
                    </td>
                    <td className="px-4 py-3 text-right font-mono tabular-nums">{p.wins}</td>
                    <td className="px-4 py-3">
                      <Switch
                        checked={p.active}
                        disabled={toggle.isPending}
                        onCheckedChange={() => toggle.mutate(p)}
                        aria-label={t("admin.bonus.col.active")}
                      />
                    </td>
                    <td className="px-2 py-3 text-right whitespace-nowrap">
                      <Button variant="ghost" size="icon" aria-label={t("common.edit")} onClick={() => setForm(toForm(p))}>
                        <Pencil />
                      </Button>
                      <Button variant="ghost" size="icon" aria-label={t("common.delete")} onClick={() => setDeleting(p)}>
                        <Trash2 />
                      </Button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>
      <p className="mt-3 text-[12px] leading-relaxed text-muted-foreground">{t("admin.bonus.weight_hint")}</p>

      <Dialog open={form !== null} onOpenChange={(open) => !open && !save.isPending && setForm(null)}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{form?.id ? t("admin.bonus.edit_title") : t("admin.bonus.add")}</DialogTitle>
          </DialogHeader>
          {form && (
            <form id="bonus-prize-form" onSubmit={submit} className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1 sm:col-span-2">
                <Label>{t("admin.bonus.field.label")}</Label>
                <Input
                  value={form.label}
                  maxLength={60}
                  onChange={(e) => setForm({ ...form, label: e.target.value })}
                  placeholder={t("admin.bonus.field.label_placeholder")}
                />
              </div>
              <div className="space-y-1">
                <Label>{t("admin.bonus.col.type")}</Label>
                <Select
                  value={form.type}
                  onValueChange={(v) => setForm({ ...form, type: v as AdminBonusPrizeType })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {TYPES.map((type) => (
                      <SelectItem key={type} value={type}>
                        {t(`admin.bonus.type.${type}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              {form.type !== "balance" && (
                <div className="space-y-1">
                  <Label>{t("admin.bonus.field.discount_type")}</Label>
                  <Select
                    value={form.discount_type}
                    onValueChange={(v) => setForm({ ...form, discount_type: v as "percent" | "fixed" })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="percent">{t("admin.promo.type.percent")}</SelectItem>
                      <SelectItem value="fixed">{t("admin.promo.type.fixed")}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              )}
              <div className="space-y-1">
                <Label>
                  {form.type === "balance"
                    ? t("admin.bonus.field.amount")
                    : form.discount_type === "percent"
                      ? t("admin.bonus.field.percent")
                      : t("admin.bonus.field.amount")}
                </Label>
                <Input
                  type="number"
                  min="0"
                  step="0.01"
                  value={form.value}
                  onChange={(e) => setForm({ ...form, value: e.target.value })}
                />
              </div>
              {form.type !== "balance" && (
                <div className="space-y-1">
                  <Label>{t("admin.bonus.field.duration")}</Label>
                  <Input
                    type="number"
                    min="0"
                    step="1"
                    value={form.duration_hours}
                    onChange={(e) => setForm({ ...form, duration_hours: e.target.value })}
                    placeholder={t("admin.promo.no_limit")}
                  />
                </div>
              )}
              <div className="space-y-1">
                <Label>{t("admin.bonus.col.weight")}</Label>
                <Input
                  type="number"
                  min="0"
                  step="1"
                  value={form.weight}
                  onChange={(e) => setForm({ ...form, weight: e.target.value })}
                />
              </div>
              <div className="space-y-1">
                <Label>{t("admin.bonus.field.icon")}</Label>
                <div className="flex items-center gap-2">
                  <Input
                    value={form.icon}
                    onChange={(e) => setForm({ ...form, icon: e.target.value })}
                    placeholder="ri-coin-line"
                  />
                  <span
                    className="flex size-9 shrink-0 items-center justify-center rounded-md text-white"
                    style={{ background: form.color }}
                  >
                    {form.icon && <i className={form.icon} />}
                  </span>
                </div>
              </div>
              <div className="space-y-1 sm:col-span-2">
                <Label>{t("admin.bonus.field.color")}</Label>
                <div className="flex flex-wrap items-center gap-2">
                  {PALETTE.map((c) => (
                    <button
                      key={c}
                      type="button"
                      aria-label={c}
                      onClick={() => setForm({ ...form, color: c })}
                      className={cn(
                        "size-7 rounded-md border-2",
                        form.color.toLowerCase() === c ? "border-foreground" : "border-transparent"
                      )}
                      style={{ background: c }}
                    />
                  ))}
                  <Input
                    className="h-8 w-28 font-mono text-xs"
                    value={form.color}
                    onChange={(e) => setForm({ ...form, color: e.target.value })}
                  />
                </div>
              </div>
              <label className="flex items-center gap-2 text-sm sm:col-span-2">
                <Switch checked={form.active} onCheckedChange={(v) => setForm({ ...form, active: v })} />
                {t("admin.bonus.field.active")}
              </label>
              {form.type !== "balance" && (
                <p className="text-[12px] text-muted-foreground sm:col-span-2">{t("admin.bonus.promo_hint")}</p>
              )}
            </form>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setForm(null)} disabled={save.isPending}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" form="bonus-prize-form" disabled={save.isPending}>
              {t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(open) => !open && !remove.isPending && setDeleting(null)}
        title={t("admin.bonus.delete_title")}
        desc={t("admin.bonus.delete_desc", { label: deleting?.label ?? "" })}
        cancelBtnText={t("common.cancel")}
        confirmText={t("common.delete")}
        destructive
        isLoading={remove.isPending}
        handleConfirm={() => deleting && remove.mutate(deleting.id)}
      />
    </PageShell>
  );
}
