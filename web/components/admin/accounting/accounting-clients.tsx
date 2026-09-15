"use client";

import { useEffect, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
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
  fetchAccountingClientDocuments,
  openAccountingClientAct,
  openAccountingClientReconciliation,
  searchAccountingClients,
  type AccountingClient,
} from "@/lib/api";
import { money, monthTitle, yearToDate } from "@/lib/accounting-period";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";

export function AccountingClients() {
  const t = useT();
  const qc = useQueryClient();
  const [search, setSearch] = useState("");
  const [term, setTerm] = useState("");
  const [selected, setSelected] = useState<AccountingClient | null>(null);
  const [currency, setCurrency] = useState("");
  const [period, setPeriod] = useState(yearToDate);
  const [busy, setBusy] = useState("");

  useEffect(() => {
    const timer = setTimeout(() => setTerm(search.trim()), 300);
    return () => clearTimeout(timer);
  }, [search]);

  const clientsQuery = useQuery({
    queryKey: ["admin-accounting-clients", term],
    queryFn: () => searchAccountingClients(term),
  });

  const docsKey = ["admin-accounting-client-documents", selected?.id ?? "", currency];
  const docsQuery = useQuery({
    queryKey: docsKey,
    queryFn: () => fetchAccountingClientDocuments(selected?.id ?? "", currency || undefined),
    enabled: selected !== null,
  });
  const docs = docsQuery.data;
  const docCurrency = docs?.currency || currency;

  const run = async (key: string, action: () => Promise<void>) => {
    setBusy(key);
    try {
      await action();
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : t("admin.accounting.open_failed"));
    } finally {
      setBusy("");
    }
  };

  const clients = clientsQuery.data?.clients ?? [];
  const payer = docs?.payer;

  return (
    <div className="grid gap-4 lg:grid-cols-[340px_1fr]">
      <div className="rounded-lg border bg-card">
        <div className="space-y-1 border-b p-4">
          <Label>{t("admin.accounting.clients_search")}</Label>
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t("admin.accounting.clients_placeholder")}
          />
        </div>
        {clientsQuery.isLoading ? (
          <div className="p-4">
            <Skeleton className="h-24 w-full" />
          </div>
        ) : clients.length === 0 ? (
          <div className="p-6 text-center text-sm text-muted-foreground">
            {t("admin.accounting.clients_empty")}
          </div>
        ) : (
          <div className="max-h-[520px] divide-y overflow-y-auto">
            {clients.map((client) => (
              <button
                key={client.id}
                type="button"
                onClick={() => {
                  setSelected(client);
                  setCurrency("");
                }}
                className={cn(
                  "block w-full px-4 py-3 text-left transition-colors hover:bg-muted/50",
                  selected?.id === client.id && "bg-muted"
                )}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="truncate text-sm font-medium">{client.name}</span>
                  <Badge variant="outline">{t(`admin.accounting.payer.${client.payer_type}`)}</Badge>
                </div>
                <div className="mt-0.5 truncate text-xs text-muted-foreground">
                  {client.email}
                  {client.inn ? ` · ${client.inn}` : ""}
                </div>
              </button>
            ))}
          </div>
        )}
      </div>

      <div className="rounded-lg border bg-card p-4">
        {!selected ? (
          <div className="py-16 text-center text-sm text-muted-foreground">
            {t("admin.accounting.clients_pick")}
          </div>
        ) : docsQuery.isLoading || !docs ? (
          <Skeleton className="h-48 w-full" />
        ) : (
          <div className="space-y-6">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <div className="text-base font-semibold">{selected.name}</div>
                <div className="text-xs text-muted-foreground">{selected.email}</div>
              </div>
              {docs.currencies.length > 1 ? (
                <Select value={docCurrency} onValueChange={setCurrency}>
                  <SelectTrigger className="w-[110px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {docs.currencies.map((c) => (
                      <SelectItem key={c} value={c}>
                        {c}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : null}
            </div>

            <div>
              <div className="mb-2 text-sm font-medium">{t("admin.accounting.client_requisites")}</div>
              {payer && payer.payer_type !== "person" ? (
                <dl className="grid gap-x-4 gap-y-1 text-sm sm:grid-cols-[160px_1fr]">
                  <dt className="text-muted-foreground">{t(`admin.accounting.payer.${payer.payer_type}`)}</dt>
                  <dd>{payer.legal_name}</dd>
                  <dt className="text-muted-foreground">{t("admin.accounting.field.inn")}</dt>
                  <dd className="font-mono">{payer.inn || "—"}</dd>
                  {payer.kpp ? (
                    <>
                      <dt className="text-muted-foreground">{t("admin.accounting.field.kpp")}</dt>
                      <dd className="font-mono">{payer.kpp}</dd>
                    </>
                  ) : null}
                  {payer.ogrn ? (
                    <>
                      <dt className="text-muted-foreground">{t("admin.accounting.field.ogrn")}</dt>
                      <dd className="font-mono">{payer.ogrn}</dd>
                    </>
                  ) : null}
                  {payer.address ? (
                    <>
                      <dt className="text-muted-foreground">{t("admin.accounting.field.address")}</dt>
                      <dd>{payer.address}</dd>
                    </>
                  ) : null}
                </dl>
              ) : (
                <p className="text-sm text-muted-foreground">{t("admin.accounting.client_no_requisites")}</p>
              )}
            </div>

            <div>
              <div className="mb-2 text-sm font-medium">{t("admin.accounting.acts_title")}</div>
              {docs.months.length === 0 ? (
                <p className="text-sm text-muted-foreground">{t("admin.accounting.acts_empty")}</p>
              ) : (
                <div className="divide-y rounded-lg border">
                  {docs.months.map((m) => (
                    <div key={m.period} className="flex flex-wrap items-center justify-between gap-3 px-3 py-2">
                      <div>
                        <div className="text-sm">
                          {monthTitle(m.period)}
                          {m.number !== null ? (
                            <span className="ml-2 font-mono text-xs text-muted-foreground">
                              {t("admin.accounting.act_number", { number: m.number })}
                            </span>
                          ) : null}
                        </div>
                        <div className="text-xs text-muted-foreground">
                          {t("admin.accounting.act_meta", {
                            count: m.count,
                            amount: money(m.amount),
                            currency: docCurrency,
                          })}
                        </div>
                      </div>
                      {m.closed ? (
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={busy !== ""}
                          onClick={() =>
                            void run(`act-${m.period}`, async () => {
                              await openAccountingClientAct(selected.id, m.period, docCurrency);
                              await qc.invalidateQueries({ queryKey: docsKey });
                            })
                          }
                        >
                          {t("admin.accounting.act_open")}
                        </Button>
                      ) : (
                        <span className="text-xs text-muted-foreground">{t("admin.accounting.act_pending")}</span>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div>
              <div className="mb-2 text-sm font-medium">{t("admin.accounting.reconciliation_title")}</div>
              <div className="flex flex-wrap items-end gap-3">
                <div className="space-y-1">
                  <Label>{t("admin.accounting.from")}</Label>
                  <Input
                    type="date"
                    className="w-[160px]"
                    value={period.from}
                    onChange={(e) => setPeriod((prev) => ({ ...prev, from: e.target.value }))}
                  />
                </div>
                <div className="space-y-1">
                  <Label>{t("admin.accounting.to")}</Label>
                  <Input
                    type="date"
                    className="w-[160px]"
                    value={period.to}
                    onChange={(e) => setPeriod((prev) => ({ ...prev, to: e.target.value }))}
                  />
                </div>
                <Button
                  variant="outline"
                  disabled={busy !== "" || !period.from || !period.to}
                  onClick={() =>
                    void run("reconciliation", () =>
                      openAccountingClientReconciliation(selected.id, period.from, period.to, docCurrency)
                    )
                  }
                >
                  {t("admin.accounting.reconciliation_open")}
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
