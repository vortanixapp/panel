"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useMutation, useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  clearAuth,
  deleteOwnAccount,
  downloadAccountData,
  fetchAccountIdentification,
  fetchAccountLegal,
} from "@/lib/api";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";

function fmtDate(iso: string | null | undefined) {
  if (!iso) return "—";
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? iso : date.toLocaleString(localeTag());
}

export function AccountData() {
  const t = useT();
  const router = useRouter();
  const legalQuery = useQuery({ queryKey: ["account-legal"], queryFn: fetchAccountLegal });
  const identQuery = useQuery({
    queryKey: ["account-identification"],
    queryFn: fetchAccountIdentification,
  });
  const [exporting, setExporting] = useState(false);
  const [confirmEmail, setConfirmEmail] = useState("");

  const deleteMut = useMutation({
    mutationFn: () => deleteOwnAccount(confirmEmail.trim()),
    onSuccess: () => {
      toast.success(t("settings.data.deleted"));
      clearAuth();
      router.replace("/");
    },
    onError: (e: Error) => toast.error(e.message || t("settings.data.delete_failed")),
  });

  const exportData = async () => {
    setExporting(true);
    try {
      await downloadAccountData();
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : t("settings.data.export_failed"));
    } finally {
      setExporting(false);
    }
  };

  const consents = legalQuery.data?.consents ?? [];
  const ident = identQuery.data;

  return (
    <div className="space-y-4">
      <div className="rounded-2xl border border-border bg-card p-5">
        <h3 className="text-sm font-semibold">{t("settings.data.consents_title")}</h3>
        <p className="mt-1 text-xs text-muted-foreground">{t("settings.data.consents_hint")}</p>
        {consents.length === 0 ? (
          <p className="mt-4 text-sm text-muted-foreground">{t("settings.data.consents_empty")}</p>
        ) : (
          <div className="mt-4 space-y-2">
            {consents.map((consent) => (
              <div
                key={consent.kind}
                className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-border bg-muted/30 px-4 py-3 text-sm"
              >
                <a
                  href={`/legal/${consent.kind}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="font-medium hover:underline"
                >
                  {consent.title || consent.kind}
                </a>
                <span className="text-xs text-muted-foreground">
                  {t(
                    consent.action === "accepted"
                      ? "settings.data.consent_accepted"
                      : "settings.data.consent_withdrawn",
                    { version: consent.version, date: fmtDate(consent.at) }
                  )}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="rounded-2xl border border-border bg-card p-5">
        <h3 className="text-sm font-semibold">{t("settings.data.identification_title")}</h3>
        <p className="mt-1 text-sm text-muted-foreground">
          {ident?.identified
            ? t("settings.data.identified", { date: fmtDate(ident.identified_at) })
            : ident && ident.methods.length > 0
              ? t("settings.data.not_identified_methods", { methods: ident.methods.join(", ") })
              : t("settings.data.not_identified")}
        </p>
      </div>

      <div className="rounded-2xl border border-border bg-card p-5">
        <h3 className="text-sm font-semibold">{t("settings.data.export_title")}</h3>
        <p className="mt-1 text-xs text-muted-foreground">{t("settings.data.export_hint")}</p>
        <Button
          className="mt-4"
          variant="outline"
          disabled={exporting}
          onClick={() => void exportData()}
        >
          {exporting ? t("common.loading") : t("settings.data.export")}
        </Button>
      </div>

      <div className="rounded-2xl border border-destructive/40 bg-card p-5">
        <h3 className="text-sm font-semibold text-destructive">{t("settings.data.delete_title")}</h3>
        <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
          {t("settings.data.delete_hint")}
        </p>
        <div className="mt-4 grid max-w-md gap-2">
          <Label htmlFor="delete-confirm-email">{t("settings.data.delete_confirm_label")}</Label>
          <Input
            id="delete-confirm-email"
            type="email"
            autoComplete="off"
            value={confirmEmail}
            onChange={(e) => setConfirmEmail(e.target.value)}
          />
          <Button
            variant="destructive"
            disabled={!confirmEmail.trim() || deleteMut.isPending}
            onClick={() => {
              if (window.confirm(t("settings.data.delete_confirm"))) deleteMut.mutate();
            }}
          >
            {t("settings.data.delete")}
          </Button>
        </div>
      </div>
    </div>
  );
}
