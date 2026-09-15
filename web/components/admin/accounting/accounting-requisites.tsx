"use client";

import { useEffect, useState, type ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

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
import { Skeleton } from "@/components/ui/skeleton";
import {
  fetchAccountingRequisites,
  saveAccountingRequisites,
  type AccountingRequisites,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";

const KEY = ["admin-accounting-requisites"];
const EMPTY_OPTION = "empty";

const EMPTY: AccountingRequisites = {
  legal_form: "",
  name: "",
  full_name: "",
  inn: "",
  kpp: "",
  ogrn: "",
  address: "",
  email: "",
  phone: "",
  bank_name: "",
  bank_bik: "",
  bank_account: "",
  bank_corr_account: "",
  signer_name: "",
  signer_position: "",
  tax_system: "",
  vat: "",
  receipt_mode: "full_payment",
  receipt_item: "",
  timezone: "Europe/Moscow",
};

function Field({ label, hint, children, wide }: { label: string; hint?: string; children: ReactNode; wide?: boolean }) {
  return (
    <div className={wide ? "space-y-1 sm:col-span-2" : "space-y-1"}>
      <Label>{label}</Label>
      {children}
      {hint ? <p className="text-xs text-muted-foreground">{hint}</p> : null}
    </div>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="rounded-lg border bg-card p-4">
      <div className="mb-3 text-sm font-medium">{title}</div>
      <div className="grid gap-3 sm:grid-cols-2">{children}</div>
    </div>
  );
}

export function AccountingRequisitesForm() {
  const t = useT();
  const qc = useQueryClient();
  const query = useQuery({ queryKey: KEY, queryFn: fetchAccountingRequisites });
  const [form, setForm] = useState<AccountingRequisites>(EMPTY);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    if (loaded || !query.data) return;
    setForm(query.data.requisites);
    setLoaded(true);
  }, [loaded, query.data]);

  function setField<K extends keyof AccountingRequisites>(key: K, value: AccountingRequisites[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  const saveMut = useMutation({
    mutationFn: () => saveAccountingRequisites(form),
    onSuccess: (data) => {
      qc.setQueryData(KEY, data);
      setForm(data.requisites);
      toast.success(t("admin.accounting.saved"));
      void qc.invalidateQueries({ queryKey: ["admin-accounting-summary"] });
    },
    onError: (e: Error) => toast.error(e.message || t("admin.accounting.save_failed")),
  });

  if (query.isLoading || !query.data) {
    return <Skeleton className="h-96 w-full" />;
  }

  const options = query.data;
  const individual = form.legal_form === "ip" || form.legal_form === "npd";
  const npd = form.tax_system === "npd";

  const text = (key: keyof AccountingRequisites, extra?: { placeholder?: string; mono?: boolean; digits?: number }) => (
    <Input
      value={form[key]}
      placeholder={extra?.placeholder}
      className={extra?.mono ? "font-mono" : undefined}
      maxLength={extra?.digits}
      inputMode={extra?.digits ? "numeric" : undefined}
      onChange={(e) =>
        setField(key, extra?.digits ? e.target.value.replace(/\D/g, "") : e.target.value)
      }
    />
  );

  return (
    <form
      className="space-y-4"
      onSubmit={(e) => {
        e.preventDefault();
        saveMut.mutate();
      }}
    >
      <div className="flex flex-wrap items-center gap-3">
        {options.ready ? (
          <Badge variant="secondary">{t("admin.accounting.ready")}</Badge>
        ) : (
          <Badge variant="outline">{t("admin.accounting.not_ready")}</Badge>
        )}
      </div>

      <Section title={t("admin.accounting.section_company")}>
        <Field label={t("admin.accounting.field.legal_form")}>
          <Select
            value={form.legal_form || EMPTY_OPTION}
            onValueChange={(v) => {
              const legalForm = v === EMPTY_OPTION ? "" : v;
              setForm((prev) => ({
                ...prev,
                legal_form: legalForm,
                kpp: legalForm === "ip" || legalForm === "npd" ? "" : prev.kpp,
                tax_system: legalForm === "npd" ? "npd" : prev.tax_system,
                vat: legalForm === "npd" ? "none" : prev.vat,
              }));
            }}
          >
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={EMPTY_OPTION}>{t("admin.accounting.tax_system.empty")}</SelectItem>
              {options.legal_forms.map((v) => (
                <SelectItem key={v} value={v}>
                  {t(`admin.accounting.legal_form.${v}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
        <Field label={t("admin.accounting.field.name")}>
          {text("name", { placeholder: t("admin.accounting.field.name_placeholder") })}
        </Field>
        <Field label={t("admin.accounting.field.full_name")} wide>
          {text("full_name")}
        </Field>
        <Field label={t("admin.accounting.field.inn")}>{text("inn", { mono: true, digits: 12 })}</Field>
        {!individual ? (
          <Field label={t("admin.accounting.field.kpp")}>
            <Input
              value={form.kpp}
              className="font-mono"
              maxLength={9}
              onChange={(e) => setField("kpp", e.target.value.toUpperCase())}
            />
          </Field>
        ) : null}
        <Field label={t("admin.accounting.field.ogrn")}>{text("ogrn", { mono: true, digits: 15 })}</Field>
        <Field label={t("admin.accounting.field.address")} wide>
          {text("address")}
        </Field>
        <Field label={t("admin.accounting.field.email")}>{text("email")}</Field>
        <Field label={t("admin.accounting.field.phone")}>{text("phone")}</Field>
      </Section>

      <Section title={t("admin.accounting.section_bank")}>
        <Field label={t("admin.accounting.field.bank_name")} wide>
          {text("bank_name")}
        </Field>
        <Field label={t("admin.accounting.field.bank_bik")}>{text("bank_bik", { mono: true, digits: 9 })}</Field>
        <Field label={t("admin.accounting.field.bank_account")}>
          {text("bank_account", { mono: true, digits: 20 })}
        </Field>
        <Field label={t("admin.accounting.field.bank_corr_account")}>
          {text("bank_corr_account", { mono: true, digits: 20 })}
        </Field>
      </Section>

      <Section title={t("admin.accounting.section_signer")}>
        <Field label={t("admin.accounting.field.signer_position")}>
          {text("signer_position", { placeholder: t("admin.accounting.field.signer_position_placeholder") })}
        </Field>
        <Field label={t("admin.accounting.field.signer_name")}>
          {text("signer_name", { placeholder: t("admin.accounting.field.signer_name_placeholder") })}
        </Field>
      </Section>

      <Section title={t("admin.accounting.section_tax")}>
        <Field label={t("admin.accounting.field.tax_system")}>
          <Select
            value={form.tax_system || EMPTY_OPTION}
            onValueChange={(v) => {
              const system = v === EMPTY_OPTION ? "" : v;
              setForm((prev) => ({ ...prev, tax_system: system, vat: system === "npd" ? "none" : prev.vat }));
            }}
          >
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={EMPTY_OPTION}>{t("admin.accounting.tax_system.empty")}</SelectItem>
              {options.tax_systems.map((v) => (
                <SelectItem key={v} value={v}>
                  {t(`admin.accounting.tax_system.${v}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
        <Field label={t("admin.accounting.field.vat")}>
          <Select value={form.vat || "none"} disabled={npd} onValueChange={(v) => setField("vat", v)}>
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {options.vat_rates.map((v) => (
                <SelectItem key={v} value={v}>
                  {v === "none" ? t("admin.accounting.vat.none") : t("admin.accounting.vat.rate", { rate: v })}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
        <Field label={t("admin.accounting.field.timezone")} hint={t("admin.accounting.field.timezone_hint")} wide>
          {text("timezone", { placeholder: "Europe/Moscow" })}
        </Field>
      </Section>

      <Section title={t("admin.accounting.section_receipts")}>
        {npd ? (
          <p className="text-sm text-muted-foreground sm:col-span-2">{t("admin.accounting.npd_hint")}</p>
        ) : null}
        <Field label={t("admin.accounting.field.receipt_mode")} hint={t("admin.accounting.receipt_mode_hint")} wide>
          <Select value={form.receipt_mode || "full_payment"} onValueChange={(v) => setField("receipt_mode", v)}>
            <SelectTrigger className="w-full sm:w-[280px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {options.receipt_modes.map((v) => (
                <SelectItem key={v} value={v}>
                  {t(`admin.accounting.receipt_mode.${v}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
        <Field label={t("admin.accounting.field.receipt_item")} wide>
          <Input
            value={form.receipt_item}
            maxLength={128}
            onChange={(e) => setField("receipt_item", e.target.value)}
          />
        </Field>
        <p className="text-xs text-muted-foreground sm:col-span-2">{t("admin.accounting.receipts_hint")}</p>
      </Section>

      <Button type="submit" disabled={saveMut.isPending}>
        {t("common.save")}
      </Button>
    </form>
  );
}
