"use client";

import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchAdminLegalSettings, saveAdminLegalSettings } from "@/lib/api";
import { useT } from "@/hooks/use-translations";

const KEY = ["admin-legal-settings"];

export function LegalSettings() {
  const t = useT();
  const qc = useQueryClient();
  const query = useQuery({ queryKey: KEY, queryFn: fetchAdminLegalSettings });
  const [loaded, setLoaded] = useState(false);
  const [required, setRequired] = useState(false);
  const [providers, setProviders] = useState<string[]>([]);
  const [cookieBanner, setCookieBanner] = useState(true);

  useEffect(() => {
    if (loaded || !query.data) return;
    setRequired(query.data.identification_required);
    setProviders(query.data.identification_providers);
    setCookieBanner(query.data.cookie_banner);
    setLoaded(true);
  }, [loaded, query.data]);

  const saveMut = useMutation({
    mutationFn: () =>
      saveAdminLegalSettings({
        identification_required: required,
        identification_providers: providers,
        cookie_banner: cookieBanner,
      }),
    onSuccess: (data) => {
      qc.setQueryData(KEY, data);
      toast.success(t("admin.legal.saved"));
    },
    onError: (e: Error) => toast.error(e.message || t("admin.legal.save_failed")),
  });

  if (query.isLoading || !query.data) {
    return <Skeleton className="h-64 w-full" />;
  }

  const toggleProvider = (code: string) =>
    setProviders((prev) => (prev.includes(code) ? prev.filter((c) => c !== code) : [...prev, code]));

  return (
    <div className="space-y-4">
      <div className="space-y-4 rounded-lg border bg-card p-4">
        <div className="text-sm font-medium">{t("admin.legal.identification_title")}</div>
        <label className="flex items-start gap-2 text-sm">
          <Checkbox className="mt-0.5" checked={required} onCheckedChange={(v) => setRequired(v === true)} />
          <span>
            {t("admin.legal.identification_required")}
            <span className="mt-0.5 block text-xs text-muted-foreground">
              {t("admin.legal.identification_required_hint")}
            </span>
          </span>
        </label>
        <div>
          <div className="text-sm">{t("admin.legal.identification_providers")}</div>
          <p className="mt-0.5 text-xs text-muted-foreground">{t("admin.legal.identification_providers_hint")}</p>
          <div className="mt-3 flex flex-wrap gap-2">
            {query.data.providers.map((provider) => (
              <label
                key={provider.code}
                className="flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs"
              >
                <Checkbox
                  checked={providers.includes(provider.code)}
                  onCheckedChange={() => toggleProvider(provider.code)}
                />
                <span>{provider.name}</span>
                {!provider.enabled && <Badge variant="outline">{t("admin.legal.provider_disabled")}</Badge>}
              </label>
            ))}
          </div>
        </div>
      </div>

      <div className="space-y-3 rounded-lg border bg-card p-4">
        <div className="text-sm font-medium">{t("admin.legal.cookie_title")}</div>
        <label className="flex items-center gap-2 text-sm">
          <Checkbox checked={cookieBanner} onCheckedChange={(v) => setCookieBanner(v === true)} />
          <span>{t("admin.legal.cookie_banner")}</span>
        </label>
      </div>

      <Button onClick={() => saveMut.mutate()} disabled={saveMut.isPending}>
        {t("common.save")}
      </Button>
    </div>
  );
}
