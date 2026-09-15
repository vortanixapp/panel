"use client";

import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchAdminLegalConsents } from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

const KINDS = ["offer", "privacy", "consent"] as const;

export function LegalConsents() {
  const t = useT();
  const [search, setSearch] = useState("");
  const [term, setTerm] = useState("");
  const [kind, setKind] = useState("all");
  const [page, setPage] = useState(1);

  useEffect(() => {
    const timer = setTimeout(() => {
      setTerm(search.trim());
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const query = useQuery({
    queryKey: ["admin-legal-consents", term, kind, page],
    queryFn: () => fetchAdminLegalConsents({ q: term, kind: kind === "all" ? "" : kind, page }),
    placeholderData: (prev) => prev,
  });
  const data = query.data;
  const pages = data ? Math.max(1, Math.ceil(data.total / data.per_page)) : 1;

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <Input
          className="max-w-xs"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t("admin.legal.consents_search")}
        />
        <Select
          value={kind}
          onValueChange={(value) => {
            setKind(value);
            setPage(1);
          }}
        >
          <SelectTrigger className="w-[260px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t("admin.legal.consents_all")}</SelectItem>
            {KINDS.map((value) => (
              <SelectItem key={value} value={value}>
                {t(`admin.legal.kind.${value}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="rounded-lg border bg-card">
        {query.isLoading ? (
          <div className="p-4">
            <Skeleton className="h-32 w-full" />
          </div>
        ) : (data?.consents.length ?? 0) === 0 ? (
          <div className="p-8 text-center text-sm text-muted-foreground">
            {t("admin.legal.consents_empty")}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-xs text-muted-foreground">
                  <th className="px-4 py-2 text-left font-normal">{t("admin.legal.col_date")}</th>
                  <th className="px-4 py-2 text-left font-normal">{t("admin.legal.col_user")}</th>
                  <th className="px-4 py-2 text-left font-normal">{t("admin.legal.col_document")}</th>
                  <th className="px-4 py-2 text-left font-normal">{t("admin.legal.col_version")}</th>
                  <th className="px-4 py-2 text-left font-normal">{t("admin.legal.col_action")}</th>
                  <th className="px-4 py-2 text-left font-normal">{t("admin.legal.col_source")}</th>
                  <th className="px-4 py-2 text-left font-normal">{t("admin.legal.col_ip")}</th>
                </tr>
              </thead>
              <tbody>
                {(data?.consents ?? []).map((consent) => (
                  <tr key={consent.id} className="border-t" title={consent.user_agent}>
                    <td className="px-4 py-2 whitespace-nowrap">
                      {new Date(consent.created_at).toLocaleString(localeTag())}
                    </td>
                    <td className="px-4 py-2">{consent.email}</td>
                    <td className="px-4 py-2">{t(`admin.legal.kind.${consent.kind}`)}</td>
                    <td className="px-4 py-2">{consent.version}</td>
                    <td className="px-4 py-2">{t(`admin.legal.action.${consent.action}`)}</td>
                    <td className="px-4 py-2 text-muted-foreground">{consent.source}</td>
                    <td className="px-4 py-2 font-mono text-xs">{consent.ip || "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {data ? (
        <div className="flex flex-wrap items-center justify-between gap-2 text-sm text-muted-foreground">
          <span>{t("admin.legal.page_info", { page: data.page, pages, total: data.total })}</span>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
              {t("admin.legal.prev")}
            </Button>
            <Button variant="outline" size="sm" disabled={page >= pages} onClick={() => setPage((p) => p + 1)}>
              {t("admin.legal.next")}
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
