"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
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
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import {
  abuseCaseAction,
  createAbuseCase,
  fetchAbuseCase,
  fetchAbuseCases,
  type AbuseCase,
  type AbuseCaseInput,
  type AbuseSource,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";
import { cn } from "@/lib/utils";

const SOURCES: AbuseSource[] = ["rkn", "court", "police", "copyright", "abuse", "other"];
const FILTERS = ["open", "new", "notified", "restricted", "resolved", "rejected", "all"] as const;

const EMPTY_INPUT: AbuseCaseInput = {
  source: "rkn",
  reference: "",
  subject: "",
  description: "",
  target: "",
  server_id: "",
  user_email: "",
  received_at: "",
  deadline_hours: 0,
};

function fmt(iso: string | null) {
  if (!iso) return "—";
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? iso : date.toLocaleString(localeTag());
}

function statusVariant(status: AbuseCase["status"]): "secondary" | "outline" | "destructive" {
  if (status === "new") return "destructive";
  if (status === "resolved" || status === "rejected") return "outline";
  return "secondary";
}

export function AbusePageContent() {
  const t = useT();
  const qc = useQueryClient();
  const [status, setStatus] = useState<string>("open");
  const [search, setSearch] = useState("");
  const [term, setTerm] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState<AbuseCaseInput>(EMPTY_INPUT);
  const [message, setMessage] = useState("");

  useEffect(() => {
    const timer = setTimeout(() => setTerm(search.trim()), 300);
    return () => clearTimeout(timer);
  }, [search]);

  const listQuery = useQuery({
    queryKey: ["admin-abuse", status, term],
    queryFn: () => fetchAbuseCases(status, term),
    refetchInterval: 60_000,
  });
  const detailQuery = useQuery({
    queryKey: ["admin-abuse-case", selectedId],
    queryFn: () => fetchAbuseCase(selectedId ?? ""),
    enabled: selectedId !== null,
  });

  const createMut = useMutation({
    mutationFn: () => createAbuseCase(form),
    onSuccess: (res) => {
      toast.success(t("admin.abuse.created", { number: res.case.number }));
      setForm(EMPTY_INPUT);
      setCreating(false);
      qc.setQueryData(["admin-abuse-case", res.case.id], res);
      setSelectedId(res.case.id);
      void qc.invalidateQueries({ queryKey: ["admin-abuse"] });
    },
    onError: (e: Error) => toast.error(e.message || t("admin.abuse.create_failed")),
  });

  const actionMut = useMutation({
    mutationFn: (action: string) => abuseCaseAction(selectedId ?? "", action, message),
    onSuccess: (res) => {
      qc.setQueryData(["admin-abuse-case", res.case.id], res);
      setMessage("");
      toast.success(t("admin.abuse.action_done"));
      void qc.invalidateQueries({ queryKey: ["admin-abuse"] });
    },
    onError: (e: Error) => toast.error(e.message || t("admin.abuse.action_failed")),
  });

  const setField = (patch: Partial<AbuseCaseInput>) => setForm((prev) => ({ ...prev, ...patch }));
  const cases = listQuery.data?.cases ?? [];
  const detail = detailQuery.data;
  const current = detail?.case;
  const closed = current ? current.status === "resolved" || current.status === "rejected" : false;

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("admin.abuse.title")}</h1>
          <p className="text-sm text-muted-foreground">{t("admin.abuse.subtitle")}</p>
          {listQuery.data ? (
            <p className="mt-1 text-xs text-muted-foreground">
              {t("admin.abuse.open_count", { open: listQuery.data.open, overdue: listQuery.data.overdue })}
            </p>
          ) : null}
        </div>
        <Button onClick={() => setCreating((v) => !v)}>{t("admin.abuse.new")}</Button>
      </div>

      {creating ? (
        <form
          className="mb-6 grid gap-3 rounded-lg border bg-card p-4 sm:grid-cols-2"
          onSubmit={(e) => {
            e.preventDefault();
            createMut.mutate();
          }}
        >
          <div className="space-y-1">
            <Label>{t("admin.abuse.field.source")}</Label>
            <Select value={form.source} onValueChange={(v) => setField({ source: v as AbuseSource })}>
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {SOURCES.map((source) => (
                  <SelectItem key={source} value={source}>
                    {t(`admin.abuse.source.${source}`)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1">
            <Label>{t("admin.abuse.field.reference")}</Label>
            <Input value={form.reference} onChange={(e) => setField({ reference: e.target.value })} />
          </div>
          <div className="space-y-1 sm:col-span-2">
            <Label>{t("admin.abuse.field.subject")}</Label>
            <Input value={form.subject} onChange={(e) => setField({ subject: e.target.value })} />
          </div>
          <div className="space-y-1 sm:col-span-2">
            <Label>{t("admin.abuse.field.description")}</Label>
            <Textarea rows={4} value={form.description} onChange={(e) => setField({ description: e.target.value })} />
          </div>
          <div className="space-y-1">
            <Label>{t("admin.abuse.field.target")}</Label>
            <Input value={form.target} onChange={(e) => setField({ target: e.target.value })} />
          </div>
          <div className="space-y-1">
            <Label>{t("admin.abuse.field.server_id")}</Label>
            <Input
              className="font-mono"
              value={form.server_id}
              onChange={(e) => setField({ server_id: e.target.value.trim() })}
            />
          </div>
          <div className="space-y-1">
            <Label>{t("admin.abuse.field.user_email")}</Label>
            <Input value={form.user_email} onChange={(e) => setField({ user_email: e.target.value })} />
          </div>
          <div className="space-y-1">
            <Label>{t("admin.abuse.field.received_at")}</Label>
            <Input
              type="datetime-local"
              value={form.received_at}
              onChange={(e) => setField({ received_at: e.target.value })}
            />
          </div>
          <div className="space-y-1">
            <Label>{t("admin.abuse.field.deadline_hours")}</Label>
            <Input
              type="number"
              min={0}
              value={form.deadline_hours}
              onChange={(e) => setField({ deadline_hours: Math.max(0, Number(e.target.value) || 0) })}
            />
            <p className="text-xs text-muted-foreground">{t("admin.abuse.field.deadline_hint")}</p>
          </div>
          <div className="sm:col-span-2">
            <Button type="submit" disabled={createMut.isPending || !form.subject.trim()}>
              {t("admin.abuse.create")}
            </Button>
          </div>
        </form>
      ) : null}

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <Select value={status} onValueChange={setStatus}>
              <SelectTrigger className="w-[200px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {FILTERS.map((filter) => (
                  <SelectItem key={filter} value={filter}>
                    {t(`admin.abuse.filter.${filter}`)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Input
              className="max-w-xs"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t("admin.abuse.search")}
            />
          </div>
          <div className="rounded-lg border bg-card">
            {listQuery.isLoading ? (
              <div className="p-4">
                <Skeleton className="h-32 w-full" />
              </div>
            ) : cases.length === 0 ? (
              <div className="p-8 text-center text-sm text-muted-foreground">{t("admin.abuse.empty")}</div>
            ) : (
              <div className="divide-y">
                {cases.map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => setSelectedId(item.id)}
                    className={cn(
                      "block w-full px-4 py-3 text-left transition-colors hover:bg-muted/50",
                      selectedId === item.id && "bg-muted"
                    )}
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono text-xs text-muted-foreground">
                        {t("admin.abuse.number", { number: item.number })}
                      </span>
                      <Badge variant={statusVariant(item.status)}>{t(`admin.abuse.status.${item.status}`)}</Badge>
                      {item.overdue ? <Badge variant="destructive">{t("admin.abuse.overdue")}</Badge> : null}
                      <span className="text-xs text-muted-foreground">{t(`admin.abuse.source.${item.source}`)}</span>
                    </div>
                    <div className="mt-1 truncate text-sm font-medium">{item.subject}</div>
                    <div className="mt-0.5 truncate text-xs text-muted-foreground">
                      {[item.server_name, item.user_email, item.target].filter(Boolean).join(" · ") || "—"}
                      {item.deadline_at ? ` · ${t("admin.abuse.deadline", { date: fmt(item.deadline_at) })}` : ""}
                    </div>
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>

        <div className="rounded-lg border bg-card p-4">
          {!selectedId ? (
            <div className="py-16 text-center text-sm text-muted-foreground">{t("admin.abuse.pick")}</div>
          ) : detailQuery.isLoading || !current || !detail ? (
            <Skeleton className="h-64 w-full" />
          ) : (
            <div className="space-y-4">
              <div>
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-mono text-xs text-muted-foreground">
                    {t("admin.abuse.number", { number: current.number })}
                  </span>
                  <Badge variant={statusVariant(current.status)}>{t(`admin.abuse.status.${current.status}`)}</Badge>
                  {current.overdue ? <Badge variant="destructive">{t("admin.abuse.overdue")}</Badge> : null}
                </div>
                <h2 className="mt-2 text-lg font-semibold">{current.subject}</h2>
                <p className="mt-1 text-xs text-muted-foreground">
                  {t(`admin.abuse.source.${current.source}`)}
                  {current.reference ? ` · ${current.reference}` : ""}
                  {` · ${t("admin.abuse.received", { date: fmt(current.received_at) })}`}
                  {current.deadline_at ? ` · ${t("admin.abuse.deadline", { date: fmt(current.deadline_at) })}` : ""}
                </p>
                {current.description ? (
                  <p className="mt-3 text-sm whitespace-pre-line">{current.description}</p>
                ) : null}
              </div>

              <dl className="grid gap-x-4 gap-y-1 text-sm sm:grid-cols-[140px_1fr]">
                {current.target ? (
                  <>
                    <dt className="text-muted-foreground">{t("admin.abuse.field.target")}</dt>
                    <dd className="break-all">{current.target}</dd>
                  </>
                ) : null}
                <dt className="text-muted-foreground">{t("admin.abuse.server")}</dt>
                <dd>
                  {current.server_id ? (
                    <Link href={`/admin/servers/${current.server_id}`} className="hover:underline">
                      {current.server_name || current.server_id}
                    </Link>
                  ) : (
                    "—"
                  )}
                </dd>
                <dt className="text-muted-foreground">{t("admin.abuse.client")}</dt>
                <dd>
                  {current.user_id ? (
                    <Link href={`/admin/users/${current.user_id}`} className="hover:underline">
                      {current.user_email || current.user_id}
                    </Link>
                  ) : (
                    "—"
                  )}
                </dd>
                {current.resolution ? (
                  <>
                    <dt className="text-muted-foreground">{t("admin.abuse.resolution")}</dt>
                    <dd className="whitespace-pre-line">{current.resolution}</dd>
                  </>
                ) : null}
              </dl>

              <div className="space-y-2">
                <Label>{t("admin.abuse.message")}</Label>
                <Textarea rows={3} value={message} onChange={(e) => setMessage(e.target.value)} />
                <div className="flex flex-wrap gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={closed || !current.user_id || actionMut.isPending}
                    onClick={() => actionMut.mutate("notify")}
                  >
                    {t("admin.abuse.action.notify")}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={closed || !current.server_id || actionMut.isPending}
                    onClick={() => actionMut.mutate("restrict")}
                  >
                    {t("admin.abuse.action.restrict")}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={!current.server_id || actionMut.isPending}
                    onClick={() => actionMut.mutate("lift")}
                  >
                    {t("admin.abuse.action.lift")}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={closed || actionMut.isPending}
                    onClick={() => actionMut.mutate("resolve")}
                  >
                    {t("admin.abuse.action.resolve")}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={closed || actionMut.isPending}
                    onClick={() => actionMut.mutate("reject")}
                  >
                    {t("admin.abuse.action.reject")}
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    disabled={actionMut.isPending}
                    onClick={() => actionMut.mutate("note")}
                  >
                    {t("admin.abuse.action.note")}
                  </Button>
                </div>
              </div>

              <div>
                <div className="mb-2 text-sm font-medium">{t("admin.abuse.events")}</div>
                <ol className="space-y-2 border-l pl-4">
                  {detail.events.map((event, index) => (
                    <li key={`${event.at}-${index}`} className="text-sm">
                      <div className="text-xs text-muted-foreground">
                        {fmt(event.at)} · {event.actor || "—"} · {t(`admin.abuse.action.${event.action}`)}
                      </div>
                      {event.note ? <div className="whitespace-pre-line">{event.note}</div> : null}
                    </li>
                  ))}
                </ol>
              </div>
            </div>
          )}
        </div>
      </div>
    </PageShell>
  );
}
