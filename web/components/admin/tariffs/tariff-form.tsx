"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

const PERIODS = [15, 30, 60, 180];

const RANGE_ROWS = [
  { key: "cpu", labelKey: "admin.tariff_form.range.cpu" },
  { key: "ram", labelKey: "admin.tariff_form.range.ram" },
  { key: "disk", labelKey: "admin.tariff_form.range.disk" },
] as const;

export type TariffFormState = {
  name: string;
  location_id: string;
  game_id: string;
  billing_type: string;
  mysql_engine: string;
  mysql_instance_key: string;
  rental_periods: number[];
  renewal_periods: number[];
  price_per_cpu_core: number;
  price_per_ram_gb: number;
  price_per_disk_gb: number;
  price_per_slot: number;
  min_slots: number;
  max_slots: number;
  base_price_monthly: number;
  position: number;
  is_available: boolean;
  allow_antiddos: boolean;
  antiddos_price: number;
  discounts: string;
  cpu_min: string;
  cpu_max: string;
  cpu_step: string;
  ram_min: string;
  ram_max: string;
  ram_step: string;
  disk_min: string;
  disk_max: string;
  disk_step: string;
  cpu_cores: number;
  cpu_shares: string;
  ram_gb: number;
  disk_gb: number;
};

export const defaultTariffFormState: TariffFormState = {
  name: "",
  location_id: "",
  game_id: "",
  billing_type: "resources",
  mysql_engine: "",
  mysql_instance_key: "",
  rental_periods: [],
  renewal_periods: [],
  price_per_cpu_core: 0,
  price_per_ram_gb: 0,
  price_per_disk_gb: 0,
  price_per_slot: 0,
  min_slots: 1,
  max_slots: 100,
  base_price_monthly: 0,
  position: 0,
  is_available: true,
  allow_antiddos: false,
  antiddos_price: 0,
  discounts: "",
  cpu_min: "",
  cpu_max: "",
  cpu_step: "",
  ram_min: "",
  ram_max: "",
  ram_step: "",
  disk_min: "",
  disk_max: "",
  disk_step: "",
  cpu_cores: 1,
  cpu_shares: "",
  ram_gb: 1,
  disk_gb: 10,
};

type Props = {
  form: TariffFormState;
  setForm: React.Dispatch<React.SetStateAction<TariffFormState>>;
  locations: { id: string; name: string }[];
  games: { id: string; name: string }[];
  saving: boolean;
  submitLabel: string;
  onSubmit: (e: React.FormEvent) => void;
  dirty?: boolean;
  onCancel?: () => void;
};

export function estimateMonthlyPrice(form: TariffFormState): number {
  const num = (v: string | number) => {
    const parsed = typeof v === "number" ? v : Number.parseFloat(v);
    return Number.isFinite(parsed) ? parsed : 0;
  };
  const base = num(form.base_price_monthly);
  if (form.billing_type === "slots") {
    return base + num(form.min_slots) * num(form.price_per_slot);
  }
  const cpu = form.cpu_min !== "" ? num(form.cpu_min) : num(form.cpu_cores);
  const ram = form.ram_min !== "" ? num(form.ram_min) : num(form.ram_gb);
  const disk = form.disk_min !== "" ? num(form.disk_min) : num(form.disk_gb);
  return (
    base +
    cpu * num(form.price_per_cpu_core) +
    ram * num(form.price_per_ram_gb) +
    disk * num(form.price_per_disk_gb)
  );
}

export function TariffForm({
  form,
  setForm,
  locations,
  games,
  saving,
  submitLabel,
  onSubmit,
  dirty = false,
  onCancel,
}: Props) {
  const t = useT();
  const isSlots = form.billing_type === "slots";
  const set = <K extends keyof TariffFormState>(
    key: K,
    value: TariffFormState[K]
  ) => setForm((p) => ({ ...p, [key]: value }));

  const togglePeriod = (
    field: "rental_periods" | "renewal_periods",
    value: number
  ) =>
    setForm((p) => {
      const arr = p[field];
      return {
        ...p,
        [field]: arr.includes(value)
          ? arr.filter((x) => x !== value)
          : [...arr, value].sort((a, b) => a - b),
      };
    });

  const estimate = estimateMonthlyPrice(form);

  return (
    <form onSubmit={onSubmit} className="grid items-start gap-4 xl:grid-cols-3">
      <Card
        title={t("admin.tariff_form.basic")}
        description={t("admin.tariff_form.basic_hint")}
      >
        <Field label={t("admin.tariff_form.name")}>
          <Input
            value={form.name}
            onChange={(e) => set("name", e.target.value)}
            required
            placeholder={t("admin.tariff_form.name_placeholder")}
            className={inputClass}
          />
        </Field>

        <div className="grid gap-3.5 sm:grid-cols-2">
          <Field label={t("common.location")}>
            <NativeSelect
              value={form.location_id}
              onChange={(v) => set("location_id", v)}
              required
              placeholder={t("admin.tariff_form.pick_location")}
              options={locations.map((l) => ({ value: l.id, label: l.name }))}
            />
          </Field>
          <Field label={t("common.game")}>
            <NativeSelect
              value={form.game_id}
              onChange={(v) => set("game_id", v)}
              required
              placeholder={t("admin.tariff_form.pick_game")}
              options={games.map((g) => ({ value: g.id, label: g.name }))}
            />
          </Field>
        </div>

        <Field label={t("admin.tariff_form.billing_type")}>
          <div className="flex gap-1 rounded-xl border bg-muted/40 p-1">
            {[
              {
                id: "resources",
                label: t("admin.tariff_form.pay_resources"),
              },
              { id: "slots", label: t("admin.tariff_form.pay_slots") },
            ].map((opt) => (
              <button
                key={opt.id}
                type="button"
                onClick={() => set("billing_type", opt.id)}
                className={cn(
                  "h-8 flex-1 rounded-lg px-3.5 text-[13px] whitespace-nowrap transition-colors",
                  form.billing_type === opt.id
                    ? "bg-primary font-medium text-primary-foreground"
                    : "text-muted-foreground hover:text-foreground"
                )}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </Field>

        <Section title={t("admin.tariff_form.database")}>
          <Field label={t("admin.tariff_form.mysql_version")}>
            <NativeSelect
              value={form.mysql_engine}
              onChange={(v) => set("mysql_engine", v)}
              placeholder={t("admin.tariff_form.mysql_default")}
              options={[
                { value: "mysql80", label: "MySQL 8.0" },
                { value: "mysql57", label: "MySQL 5.7" },
                { value: "mariadb", label: "MariaDB" },
              ]}
            />
          </Field>
          <Field label="MySQL instance key">
            <Input
              value={form.mysql_instance_key}
              onChange={(e) => set("mysql_instance_key", e.target.value)}
              placeholder="mysql80-main"
              className={cn(inputClass, "font-mono")}
            />
          </Field>
        </Section>

        <Section>
          <Field label={t("admin.tariff_form.rental_periods")}>
            <div className="flex flex-wrap gap-2">
              {PERIODS.map((d) => (
                <PeriodChip
                  key={d}
                  days={d}
                  active={form.rental_periods.includes(d)}
                  onClick={() => togglePeriod("rental_periods", d)}
                />
              ))}
            </div>
          </Field>
          <Field label={t("admin.tariff_form.renewal_periods")}>
            <div className="flex flex-wrap gap-2">
              {PERIODS.map((d) => (
                <PeriodChip
                  key={d}
                  days={d}
                  active={form.renewal_periods.includes(d)}
                  onClick={() => togglePeriod("renewal_periods", d)}
                />
              ))}
            </div>
          </Field>
        </Section>
      </Card>

      <Card
        title={t("admin.tariff_form.resources")}
        description={t("admin.tariff_form.resources_hint")}
      >
        <div className="flex items-center justify-between gap-4 rounded-xl border bg-muted/30 px-4 py-3.5">
          <div className="space-y-0.5">
            <div className="text-[13px] font-medium">Anti-DDoS</div>
            <div className="text-xs text-muted-foreground">
              {t("admin.tariff_form.antiddos_hint")}
            </div>
          </div>
          <Switch
            checked={form.allow_antiddos}
            onCheckedChange={(v) => set("allow_antiddos", v)}
          />
        </div>

        {form.allow_antiddos && (
          <Field label={t("admin.tariff_form.antiddos_price")}>
            <Input
              type="number"
              step="0.01"
              min={0}
              value={form.antiddos_price}
              onChange={(e) => set("antiddos_price", +e.target.value)}
              className={cn(inputClass, "font-mono")}
            />
          </Field>
        )}

        <Field label={t("admin.tariff_form.discounts")}>
          <textarea
            value={form.discounts}
            onChange={(e) => set("discounts", e.target.value)}
            rows={2}
            placeholder='{"30": 5, "180": 15}'
            className="min-h-[68px] w-full resize-y rounded-lg border border-input bg-transparent px-3 py-2.5 font-mono text-[13px] leading-relaxed outline-none transition-[color,box-shadow] placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:bg-input/30"
          />
        </Field>

        {!isSlots && (
          <Section title={t("admin.tariff_form.ranges")}>
            <div className="grid grid-cols-[1fr_repeat(3,64px)] items-center gap-2 text-center text-[11px] text-muted-foreground">
              <span />
              <span>Min</span>
              <span>Max</span>
              <span>Step</span>
            </div>
            {RANGE_ROWS.map((row) => (
              <div
                key={row.key}
                className="grid grid-cols-[1fr_repeat(3,64px)] items-center gap-2"
              >
                <Label className="text-xs font-normal">{t(row.labelKey)}</Label>
                {(["min", "max", "step"] as const).map((bound) => {
                  const field = `${row.key}_${bound}` as keyof TariffFormState;
                  return (
                    <Input
                      key={bound}
                      type="number"
                      min={0}
                      value={form[field] as string}
                      onChange={(e) => set(field, e.target.value as never)}
                      className="h-[34px] rounded-md px-2 text-center font-mono text-[13px] md:text-[13px]"
                    />
                  );
                })}
              </div>
            ))}
          </Section>
        )}

        <Section
          title={t("admin.tariff_form.container_limits")}
          hint={t("admin.tariff_form.cpu_shares_hint")}
        >
          <div className="grid gap-3.5 sm:grid-cols-2">
            <Field label="CPU cores">
              <Input
                type="number"
                min={0}
                value={form.cpu_cores}
                onChange={(e) => set("cpu_cores", +e.target.value)}
                required
                className={cn(inputClass, "font-mono")}
              />
            </Field>
            <Field label="CPU shares" muted={form.cpu_cores !== 0}>
              <Input
                type="number"
                min={2}
                value={form.cpu_shares}
                onChange={(e) => set("cpu_shares", e.target.value)}
                disabled={form.cpu_cores !== 0}
                placeholder="1024"
                className={cn(inputClass, "font-mono")}
              />
            </Field>
            <Field label={t("admin.tariff_form.ram_gb")}>
              <Input
                type="number"
                min={1}
                value={form.ram_gb}
                onChange={(e) => set("ram_gb", +e.target.value)}
                required
                className={cn(inputClass, "font-mono")}
              />
            </Field>
            <Field label={t("admin.tariff_form.disk_gb")}>
              <Input
                type="number"
                min={1}
                value={form.disk_gb}
                onChange={(e) => set("disk_gb", +e.target.value)}
                required
                className={cn(inputClass, "font-mono")}
              />
            </Field>
          </div>
        </Section>
      </Card>

      <div className="flex flex-col gap-4">
        <Card
          title={t("admin.billing.title")}
          description={
            isSlots
              ? t("admin.tariff_form.billing_slots_hint")
              : t("admin.tariff_form.billing_resources_hint")
          }
        >
          {!isSlots ? (
            <div className="space-y-3.5">
              <PriceRow
                label={t("admin.tariff_form.price_cpu")}
                value={form.price_per_cpu_core}
                onChange={(v) => set("price_per_cpu_core", v)}
              />
              <PriceRow
                label={t("admin.tariff_form.price_ram")}
                value={form.price_per_ram_gb}
                onChange={(v) => set("price_per_ram_gb", v)}
              />
              <PriceRow
                label={t("admin.tariff_form.price_disk")}
                value={form.price_per_disk_gb}
                onChange={(v) => set("price_per_disk_gb", v)}
              />
              <div className="border-t pt-3.5">
                <PriceRow
                  strong
                  label={t("admin.tariff_form.base_price")}
                  value={form.base_price_monthly}
                  onChange={(v) => set("base_price_monthly", v)}
                />
              </div>
            </div>
          ) : (
            <div className="space-y-3.5">
              <PriceRow
                label={t("admin.tariff_form.price_slot")}
                value={form.price_per_slot}
                onChange={(v) => set("price_per_slot", v)}
              />
              <div className="grid gap-3.5 border-t pt-3.5 sm:grid-cols-2">
                <Field label={t("admin.tariff_form.min_slots")}>
                  <Input
                    type="number"
                    min={1}
                    value={form.min_slots}
                    onChange={(e) => set("min_slots", +e.target.value)}
                    className="h-[34px] rounded-md font-mono text-[13px] md:text-[13px]"
                  />
                </Field>
                <Field label={t("admin.tariff_form.max_slots")}>
                  <Input
                    type="number"
                    min={1}
                    value={form.max_slots}
                    onChange={(e) => set("max_slots", +e.target.value)}
                    className="h-[34px] rounded-md font-mono text-[13px] md:text-[13px]"
                  />
                </Field>
              </div>
            </div>
          )}

          <div className="flex items-baseline justify-between gap-3.5 rounded-xl border bg-muted/30 px-4 py-3.5">
            <span className="text-xs text-muted-foreground">
              {t("admin.tariff_form.estimate_hint")}
            </span>
            <span className="font-mono text-base">
              {t("admin.tariff_form.estimate_value", {
                value: Math.round(estimate).toLocaleString(localeTag()),
              })}
            </span>
          </div>
        </Card>

        <Card title={t("common.status")}>
          <div className="flex items-center justify-between gap-3.5">
            <Label className="text-[13px] font-normal">
              {t("admin.tariff_form.position")}
            </Label>
            <Input
              type="number"
              min={0}
              value={form.position}
              onChange={(e) => set("position", +e.target.value)}
              className="h-[34px] w-24 rounded-md text-right font-mono text-[13px] md:text-[13px]"
            />
          </div>
          <div className="flex items-center justify-between gap-4 rounded-xl border bg-muted/30 px-4 py-3.5">
            <div className="space-y-0.5">
              <div className="text-[13px] font-medium">
                {t("admin.tariff_form.availability")}
              </div>
              <div className="text-xs text-muted-foreground">
                {t("admin.tariff_form.availability_hint")}
              </div>
            </div>
            <Switch
              checked={form.is_available}
              onCheckedChange={(v) => set("is_available", v)}
            />
          </div>
        </Card>

        <div className="space-y-3 rounded-2xl border bg-muted/30 px-5 py-4">
          <div className="flex items-center gap-2.5">
            <span
              className={cn(
                "size-1.5 rounded-full",
                dirty ? "bg-amber-500" : "bg-muted-foreground/40"
              )}
            />
            <span className="text-[13px] text-muted-foreground">
              {dirty
                ? t("admin.tariff_form.dirty")
                : t("admin.tariff_form.clean")}
            </span>
          </div>
          <div className="flex gap-2.5">
            {onCancel && (
              <Button
                type="button"
                variant="outline"
                className="h-[38px] flex-1 text-[13px]"
                onClick={onCancel}
                disabled={saving}
              >
                {t("admin.settings.discard")}
              </Button>
            )}
            <Button
              type="submit"
              className="h-[38px] flex-[2] text-[13px]"
              disabled={saving}
            >
              {saving ? t("common.saving") : submitLabel}
            </Button>
          </div>
        </div>
      </div>
    </form>
  );
}

const inputClass = "h-[38px] rounded-lg text-[13px] md:text-[13px]";

function Card({
  title,
  description,
  children,
}: {
  title?: string;
  description?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="flex flex-col gap-5 rounded-2xl border bg-card px-5 py-5 sm:px-6">
      {title && (
        <div className="space-y-1">
          <div className="text-[15px] leading-none font-semibold">{title}</div>
          {description && (
            <div className="text-xs text-muted-foreground">{description}</div>
          )}
        </div>
      )}
      {children}
    </section>
  );
}

function Section({
  title,
  hint,
  children,
}: {
  title?: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-3.5 border-t pt-4.5">
      {title && (
        <div className="space-y-0.5">
          <div className="font-mono text-[11px] tracking-wider text-muted-foreground uppercase">
            {title}
          </div>
          {hint && (
            <div className="text-xs text-muted-foreground/70">{hint}</div>
          )}
        </div>
      )}
      {children}
    </div>
  );
}

function Field({
  label,
  muted,
  children,
}: {
  label: string;
  muted?: boolean;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-[7px]">
      <Label
        className={cn(
          "text-xs font-normal text-muted-foreground",
          muted && "text-muted-foreground/60"
        )}
      >
        {label}
      </Label>
      {children}
    </div>
  );
}

function PriceRow({
  label,
  value,
  onChange,
  strong,
}: {
  label: string;
  value: number;
  onChange: (next: number) => void;
  strong?: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-3.5">
      <Label
        className={cn(
          "text-[13px] font-normal",
          !strong && "text-muted-foreground"
        )}
      >
        {label}
      </Label>
      <Input
        type="number"
        step="0.01"
        min={0}
        value={value}
        onChange={(e) => onChange(+e.target.value)}
        className="h-[34px] w-24 rounded-md text-right font-mono text-[13px] md:text-[13px]"
      />
    </div>
  );
}

function PeriodChip({
  days,
  active,
  onClick,
}: {
  days: number;
  active: boolean;
  onClick: () => void;
}) {
  const t = useT();
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "h-[30px] rounded-lg border px-3 font-mono text-xs transition-colors",
        active
          ? "border-transparent bg-primary text-primary-foreground"
          : "text-muted-foreground hover:text-foreground"
      )}
    >
      {t("admin.tariff_form.days_short", { days })}
    </button>
  );
}

function NativeSelect({
  value,
  onChange,
  options,
  placeholder,
  required,
}: {
  value: string;
  onChange: (next: string) => void;
  options: { value: string; label: string }[];
  placeholder: string;
  required?: boolean;
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      required={required}
      className="h-[38px] w-full rounded-lg border border-input bg-transparent px-2.5 text-[13px] outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:bg-input/30"
    >
      <option value="">{placeholder}</option>
      {options.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
    </select>
  );
}

export function tariffFormToPayload(form: TariffFormState): Record<string, unknown> {
  return { ...form };
}

export function adminTariffToFormState(t: {
  name?: string;
  location_id?: string;
  game_id?: string;
  billing_type?: string;
  mysql_engine?: string | null;
  mysql_instance_key?: string | null;
  rental_periods?: number[];
  renewal_periods?: number[];
  price_per_cpu_core?: number;
  price_per_ram_gb?: number;
  price_per_disk_gb?: number;
  price_per_slot?: number;
  min_slots?: number;
  max_slots?: number;
  base_price_monthly?: number;
  position?: number;
  is_available?: boolean;
  allow_antiddos?: boolean;
  antiddos_price?: number;
  discounts?: unknown;
  cpu_min?: number | null;
  cpu_max?: number | null;
  cpu_step?: number | null;
  ram_min?: number | null;
  ram_max?: number | null;
  ram_step?: number | null;
  disk_min?: number | null;
  disk_max?: number | null;
  disk_step?: number | null;
  cpu_cores?: number;
  cpu_shares?: number | null;
  ram_gb?: number;
  disk_gb?: number;
}): TariffFormState {
  return {
    name: t.name || "",
    location_id: t.location_id || "",
    game_id: t.game_id || "",
    billing_type: t.billing_type || "resources",
    mysql_engine: t.mysql_engine || "",
    mysql_instance_key: t.mysql_instance_key || "",
    rental_periods: Array.isArray(t.rental_periods) ? t.rental_periods : [],
    renewal_periods: Array.isArray(t.renewal_periods) ? t.renewal_periods : [],
    price_per_cpu_core: t.price_per_cpu_core ?? 0,
    price_per_ram_gb: t.price_per_ram_gb ?? 0,
    price_per_disk_gb: t.price_per_disk_gb ?? 0,
    price_per_slot: t.price_per_slot ?? 0,
    min_slots: t.min_slots ?? 1,
    max_slots: t.max_slots ?? 100,
    base_price_monthly: t.base_price_monthly ?? 0,
    position: t.position ?? 0,
    is_available: !!(t.is_available ?? true),
    allow_antiddos: !!t.allow_antiddos,
    antiddos_price: t.antiddos_price ?? 0,
    discounts:
      typeof t.discounts === "string"
        ? t.discounts
        : JSON.stringify(t.discounts || {}),
    cpu_min: t.cpu_min != null ? String(t.cpu_min) : "",
    cpu_max: t.cpu_max != null ? String(t.cpu_max) : "",
    cpu_step: t.cpu_step != null ? String(t.cpu_step) : "",
    ram_min: t.ram_min != null ? String(t.ram_min) : "",
    ram_max: t.ram_max != null ? String(t.ram_max) : "",
    ram_step: t.ram_step != null ? String(t.ram_step) : "",
    disk_min: t.disk_min != null ? String(t.disk_min) : "",
    disk_max: t.disk_max != null ? String(t.disk_max) : "",
    disk_step: t.disk_step != null ? String(t.disk_step) : "",
    cpu_cores: t.cpu_cores ?? 1,
    cpu_shares: t.cpu_shares != null ? String(t.cpu_shares) : "",
    ram_gb: t.ram_gb ?? 1,
    disk_gb: t.disk_gb ?? 10,
  };
}
