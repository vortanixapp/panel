"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import {
  createAdminPromotion,
  deleteAdminPromotion,
  fetchAdminPromotions,
  updateAdminPromotion,
  type AdminPromotion,
  type AdminPromotionInput,
  type AdminPromotionDiscount,
  type AdminPromotionScope,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

type FormState = {
  id?: string;
  title: string;
  code: string;
  type: AdminPromotionDiscount;
  value: string;
  active: boolean;
  starts_at: string;
  ends_at: string;
  max_uses: string;
  min_amount: string;
  only_new_users: boolean;
  description: string;
  applies_to: AdminPromotionScope[];
  bonus_percent: string;
  bonus_fixed: string;
  tariff_ids: string;
  game_ids: string;
  location_ids: string;
  user_ids: string;
};

const NO_DISCOUNT = "__none__";

export const PROMO_SCOPES: { value: AdminPromotionScope; labelKey: string }[] = [
  { value: "rent", labelKey: "admin.promo.scope.rent" },
  { value: "renew", labelKey: "admin.promo.scope.renew" },
  { value: "topup", labelKey: "admin.promo.scope.topup" },
];

const emptyForm: FormState = {
  title: "",
  code: "",
  type: "percent",
  value: "10",
  active: true,
  starts_at: "",
  ends_at: "",
  max_uses: "",
  min_amount: "",
  only_new_users: false,
  description: "",
  applies_to: [],
  bonus_percent: "",
  bonus_fixed: "",
  tariff_ids: "",
  game_ids: "",
  location_ids: "",
  user_ids: "",
};

function toDateInput(iso: string | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function fmtDate(iso: string | null): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleString(localeTag(), {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function idsToText(ids: string[] | undefined): string {
  return (ids ?? []).join(", ");
}

function textToIds(v: string): string[] {
  return v
    .split(/[\s,]+/)
    .map((s) => s.trim())
    .filter(Boolean);
}

function toForm(p: AdminPromotion): FormState {
  return {
    id: p.id,
    title: p.title,
    code: p.code,
    type: p.type,
    value: String(p.value),
    active: p.active,
    starts_at: toDateInput(p.starts_at),
    ends_at: toDateInput(p.ends_at),
    max_uses: p.max_uses === null ? "" : String(p.max_uses),
    min_amount: p.min_amount === null ? "" : String(p.min_amount),
    only_new_users: p.only_new_users,
    description: p.description,
    applies_to: p.applies_to ?? [],
    bonus_percent: p.bonus_percent ? String(p.bonus_percent) : "",
    bonus_fixed: p.bonus_fixed ? String(p.bonus_fixed) : "",
    tariff_ids: idsToText(p.tariff_ids),
    game_ids: idsToText(p.game_ids),
    location_ids: idsToText(p.location_ids),
    user_ids: idsToText(p.user_ids),
  };
}

function numOrNull(v: string): number | null {
  const t = v.trim();
  if (t === "") return null;
  const n = Number(t);
  return Number.isFinite(n) ? n : null;
}

function toPayload(f: FormState): AdminPromotionInput {
  return {
    title: f.title.trim() || f.code.trim() || "Автоматическая акция",
    code: f.code.trim(),
    type: f.type,
    value: Number(f.value) || 0,
    active: f.active,
    starts_at: f.starts_at || null,
    ends_at: f.ends_at || null,
    max_uses: numOrNull(f.max_uses),
    min_amount: numOrNull(f.min_amount),
    only_new_users: f.only_new_users,
    description: f.description.trim(),
    applies_to: f.applies_to,
    bonus_percent: Number(f.bonus_percent) || 0,
    bonus_fixed: Number(f.bonus_fixed) || 0,
    tariff_ids: textToIds(f.tariff_ids),
    game_ids: textToIds(f.game_ids),
    location_ids: textToIds(f.location_ids),
    user_ids: textToIds(f.user_ids),
  };
}

export function PromoPageContent() {
  const t = useT();
  const qc = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: ["admin-promo"],
    queryFn: async () => (await fetchAdminPromotions()).promotions,
  });
  const promos = data ?? [];

  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<FormState>(emptyForm);

  const invalidate = () => qc.invalidateQueries({ queryKey: ["admin-promo"] });

  const saveMut = useMutation({
    mutationFn: async (f: FormState) => {
      if (f.id) {
        await updateAdminPromotion(f.id, toPayload(f));
        return;
      }
      await createAdminPromotion(toPayload(f));
    },
    onSuccess: () => {
      toast.success(
        form.id ? t("admin.promo.updated") : t("admin.promo.created")
      );
      setShowForm(false);
      setForm(emptyForm);
      invalidate();
    },
    onError: (e: Error) => toast.error(e.message || t("common.save_failed")),
  });

  const toggleMut = useMutation({
    mutationFn: (p: AdminPromotion) =>
      updateAdminPromotion(p.id, { active: !p.active }),
    onSuccess: invalidate,
    onError: (e: Error) => toast.error(e.message || t("admin.promo.toggle_failed")),
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteAdminPromotion(id),
    onSuccess: () => {
      toast.success(t("admin.promo.deleted"));
      invalidate();
    },
    onError: (e: Error) => toast.error(e.message || t("common.delete_failed")),
  });

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.code.trim() && !form.title.trim()) {
      toast.error(t("admin.promo.need_title"));
      return;
    }
    const hasDiscount = form.type !== "" && Number(form.value) > 0;
    const hasBonus = Number(form.bonus_percent) > 0 || Number(form.bonus_fixed) > 0;
    if (!hasDiscount && !hasBonus) {
      toast.error(t("admin.promo.need_value"));
      return;
    }
    saveMut.mutate(form);
  }

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("admin.promo.title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.promo.subtitle")}
          </p>
        </div>
        <Button
          onClick={() => {
            setForm(emptyForm);
            setShowForm(true);
          }}
        >
          + {t("admin.promo.create")}
        </Button>
      </div>

      {showForm && (
        <form onSubmit={onSubmit} className="mb-6 space-y-4 rounded-lg border bg-card p-4">
          <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
            <div className="space-y-1">
              <Label>{t("admin.promo.code_label")}</Label>
              <Input
                value={form.code}
                onChange={(e) => setForm({ ...form, code: e.target.value })}
                placeholder="SUMMER25"
              />
            </div>
            <div className="space-y-1">
              <Label>{t("common.name")}</Label>
              <Input
                value={form.title}
                onChange={(e) => setForm({ ...form, title: e.target.value })}
                placeholder={t("admin.promo.title_placeholder")}
              />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.promo.discount_type")}</Label>
              <Select
                value={form.type === "" ? NO_DISCOUNT : form.type}
                onValueChange={(v) =>
                  setForm({ ...form, type: v === NO_DISCOUNT ? "" : (v as AdminPromotionDiscount) })
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="percent">
                    {t("admin.promo.type.percent")}
                  </SelectItem>
                  <SelectItem value="fixed">
                    {t("admin.promo.type.fixed")}
                  </SelectItem>
                  <SelectItem value={NO_DISCOUNT}>
                    {t("admin.promo.type.none")}
                  </SelectItem>
                </SelectContent>
              </Select>
              <p className="text-[11px] text-muted-foreground">
                {t("admin.promo.type_hint")}
              </p>
            </div>
            <div className="space-y-1">
              <Label>{t("admin.promo.value")}</Label>
              <Input
                type="number"
                min="0"
                step="0.01"
                value={form.value}
                onChange={(e) => setForm({ ...form, value: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.promo.starts_at")}</Label>
              <Input
                type="datetime-local"
                value={form.starts_at}
                onChange={(e) => setForm({ ...form, starts_at: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.promo.ends_at")}</Label>
              <Input
                type="datetime-local"
                value={form.ends_at}
                onChange={(e) => setForm({ ...form, ends_at: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.promo.max_uses")}</Label>
              <Input
                type="number"
                min="1"
                value={form.max_uses}
                onChange={(e) => setForm({ ...form, max_uses: e.target.value })}
                placeholder={t("admin.promo.no_limit")}
              />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.promo.min_amount")}</Label>
              <Input
                type="number"
                min="0"
                step="0.01"
                value={form.min_amount}
                onChange={(e) => setForm({ ...form, min_amount: e.target.value })}
                placeholder={t("admin.promo.no_restriction")}
              />
            </div>
            <div className="space-y-1 md:col-span-2">
              <Label>{t("admin.promo.scope")}</Label>
              <div className="flex flex-wrap gap-2 pt-1">
                {PROMO_SCOPES.map((s) => {
                  const picked = form.applies_to.includes(s.value);
                  return (
                    <button
                      key={s.value}
                      type="button"
                      onClick={() =>
                        setForm({
                          ...form,
                          applies_to: picked
                            ? form.applies_to.filter((x) => x !== s.value)
                            : [...form.applies_to, s.value],
                        })
                      }
                      className={`rounded-full border px-3 py-1 text-xs transition-colors ${
                        picked ? "border-primary bg-primary/10 text-primary" : "text-muted-foreground"
                      }`}
                    >
                      {t(s.labelKey)}
                    </button>
                  );
                })}
              </div>
              <p className="text-[11px] text-muted-foreground">
                {t("admin.promo.scope_hint")}
              </p>
            </div>

            <div className="space-y-1">
              <Label>{t("admin.promo.bonus_percent")}</Label>
              <Input
                type="number"
                min="0"
                step="0.01"
                value={form.bonus_percent}
                placeholder="0"
                onChange={(e) => setForm({ ...form, bonus_percent: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.promo.bonus_fixed")}</Label>
              <Input
                type="number"
                min="0"
                step="0.01"
                value={form.bonus_fixed}
                placeholder="0"
                onChange={(e) => setForm({ ...form, bonus_fixed: e.target.value })}
              />
              <p className="text-[11px] text-muted-foreground">
                {t("admin.promo.bonus_hint")}
              </p>
            </div>

            <div className="space-y-1">
              <Label>{t("admin.promo.only_tariffs")}</Label>
              <Input
                value={form.tariff_ids}
                placeholder={t("admin.promo.ids_placeholder")}
                onChange={(e) => setForm({ ...form, tariff_ids: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.promo.only_games")}</Label>
              <Input
                value={form.game_ids}
                placeholder={t("admin.promo.ids_placeholder")}
                onChange={(e) => setForm({ ...form, game_ids: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.promo.only_locations")}</Label>
              <Input
                value={form.location_ids}
                placeholder={t("admin.promo.ids_placeholder")}
                onChange={(e) => setForm({ ...form, location_ids: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label>{t("admin.promo.only_users")}</Label>
              <Input
                value={form.user_ids}
                placeholder={t("admin.promo.ids_placeholder")}
                onChange={(e) => setForm({ ...form, user_ids: e.target.value })}
              />
              <p className="text-[11px] text-muted-foreground">
                {t("admin.promo.personal_hint")}
              </p>
            </div>

            <div className="space-y-1 md:col-span-2">
              <Label>{t("common.description")}</Label>
              <Input
                value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
              />
            </div>
          </div>
          <div className="flex flex-wrap gap-6">
            <div className="flex items-center gap-2">
              <Switch
                checked={form.active}
                onCheckedChange={(v) => setForm({ ...form, active: v })}
              />
              <Label className="!mb-0">{t("admin.promo.active")}</Label>
            </div>
            <div className="flex items-center gap-2">
              <Switch
                checked={form.only_new_users}
                onCheckedChange={(v) => setForm({ ...form, only_new_users: v })}
              />
              <Label className="!mb-0">{t("admin.promo.only_new_users")}</Label>
            </div>
          </div>
          <div className="flex gap-2">
            <Button type="submit" disabled={saveMut.isPending}>
              {form.id ? t("common.save") : t("common.create")}
            </Button>
            <Button type="button" variant="ghost" onClick={() => setShowForm(false)}>
              {t("common.cancel")}
            </Button>
          </div>
        </form>
      )}

      <div className="rounded-lg border bg-card">
        {isLoading ? (
          <div className="space-y-3 p-6">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
          </div>
        ) : promos.length === 0 ? (
          <div className="p-10 text-center text-sm text-muted-foreground">
            {t("admin.promo.empty")}
          </div>
        ) : (
          <div className="divide-y">
            {promos.map((p) => {
              const exhausted = p.max_uses !== null && p.used_count >= p.max_uses;
              const expired = p.ends_at !== null && new Date(p.ends_at) < new Date();
              return (
                <div
                  key={p.id}
                  className="flex flex-wrap items-center justify-between gap-4 p-4"
                >
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono font-medium">
                        {p.code || t("admin.promo.no_code")}
                      </span>
                      <span className="text-sm text-muted-foreground">
                        {p.type === ""
                          ? t("admin.promo.no_discount")
                          : p.type === "percent"
                            ? `−${p.value}%`
                            : `−${p.value} ₽`}
                      </span>
                      {(p.bonus_percent > 0 || p.bonus_fixed > 0) && (
                        <span className="text-sm text-emerald-600">
                          {t("admin.promo.bonus")}{" "}
                          {[
                            p.bonus_percent > 0 ? `+${p.bonus_percent}%` : "",
                            p.bonus_fixed > 0 ? `+${p.bonus_fixed} ₽` : "",
                          ]
                            .filter(Boolean)
                            .join(t("admin.promo.bonus_join"))}
                        </span>
                      )}
                      {(p.applies_to ?? []).map((scope) => {
                        const known = PROMO_SCOPES.find((s) => s.value === scope);
                        return (
                          <Badge key={scope} variant="outline">
                            {known ? t(known.labelKey) : scope}
                          </Badge>
                        );
                      })}
                    </div>
                    <div className="mt-1 text-xs text-muted-foreground">
                      {p.title || t("admin.promo.no_title")} ·{" "}
                      {fmtDate(p.starts_at)} — {fmtDate(p.ends_at)} ·{" "}
                      {t("admin.promo.uses", { count: p.used_count })}
                      {p.max_uses !== null
                        ? t("admin.promo.uses_of", { count: p.max_uses })
                        : ""}
                      {p.min_amount !== null
                        ? t("admin.promo.min_from", { amount: p.min_amount })
                        : ""}
                      {p.only_new_users ? t("admin.promo.tag_new") : ""}
                      {(p.user_ids ?? []).length > 0
                        ? t("admin.promo.tag_personal")
                        : ""}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {p.active ? (
                      <Badge className="bg-emerald-500/10 text-emerald-600 ring-1 ring-inset ring-emerald-500/20">
                        {t("admin.promo.active")}
                      </Badge>
                    ) : (
                      <Badge variant="outline">{t("admin.promo.inactive")}</Badge>
                    )}
                    {expired && (
                      <Badge variant="outline">{t("admin.promo.expired")}</Badge>
                    )}
                    {exhausted && (
                      <Badge variant="outline">{t("admin.promo.exhausted")}</Badge>
                    )}
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => toggleMut.mutate(p)}
                      disabled={toggleMut.isPending}
                    >
                      {p.active ? t("common.disable") : t("common.enable")}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setForm(toForm(p));
                        setShowForm(true);
                      }}
                    >
                      {t("common.edit")}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        if (
                          !confirm(
                            t("admin.promo.delete_confirm", {
                              code: p.code || t("admin.promo.without_code"),
                            })
                          )
                        )
                          return;
                        deleteMut.mutate(p.id);
                      }}
                      disabled={deleteMut.isPending}
                    >
                      {t("common.delete")}
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </PageShell>
  );
}
