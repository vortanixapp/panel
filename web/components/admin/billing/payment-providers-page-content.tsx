"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CreditCard } from "lucide-react";
import { toast } from "sonner";

import { PageShell } from "@/components/layout/page-shell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  fetchAdminPaymentProviders,
  fetchAdminSettings,
  updateAdminPaymentProvider,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";

type ProviderItem = {
  id: string;
  key: string;
  name: string;
  enabled: boolean;
  configured: boolean;
  supported: boolean;
  fields: Record<string, { label?: string; type?: string; default?: string; options?: Record<string, string> }>;
  config: Record<string, string>;
};

export function PaymentProvidersPageContent() {
  const t = useT();
  const queryClient = useQueryClient();
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editConfig, setEditConfig] = useState<Record<string, string>>({});

  const { data: provData, isLoading: provLoading } = useQuery({
    queryKey: queryKeys.adminPaymentProviders,
    queryFn: fetchAdminPaymentProviders,
  });

  const { data: settingsData } = useQuery({
    queryKey: queryKeys.adminSettings,
    queryFn: fetchAdminSettings,
  });

  const items: ProviderItem[] = useMemo(() => {
    const raw = (provData?.providers ?? []) as any[];
    const spRaw = (settingsData?.payment_providers ?? settingsData?.paymentProviders ?? []) as any;
    const settingsProviders: any[] = Array.isArray(spRaw) ? spRaw : Object.values(spRaw || {});

    return raw.map((p) => {
      const key = String(p.code ?? p.key ?? p.id ?? "");
      const match = settingsProviders.find((s: any) => String(s.key) === key);
      const cfg: Record<string, string> = match?.config || {};
      const hasVal = Object.values(cfg).some((v) => v != null && String(v).trim() !== "");
      const fields = (match?.fields || {}) as ProviderItem["fields"];
      return {
        id: String(p.id),
        key,
        name: String(p.name || key),
        enabled: !!p.enabled,
        configured: hasVal,
        supported: !!match?.supported,
        fields,
        config: cfg,
      };
    });
  }, [provData, settingsData]);

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.adminPaymentProviders });
    queryClient.invalidateQueries({ queryKey: queryKeys.adminSettings });
  };

  const toggleMut = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      updateAdminPaymentProvider(id, { enabled }),
    onSuccess: async () => {
      await invalidate();
      toast.success(t("admin.psp.status_updated"));
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.psp.status_failed")),
    onSettled: () => setActionLoading(null),
  });

  const configMut = useMutation({
    mutationFn: ({ id, config }: { id: string; config: Record<string, string> }) =>
      updateAdminPaymentProvider(id, { config }),
    onSuccess: async () => {
      await invalidate();
      toast.success(t("admin.psp.config_saved"));
      setEditingId(null);
      setEditConfig({});
    },
    onError: (e: Error) =>
      toast.error(e.message || t("admin.psp.config_failed")),
    onSettled: () => setActionLoading(null),
  });

  async function handleToggle(p: ProviderItem) {
    setActionLoading(p.id);
    await toggleMut.mutateAsync({ id: p.id, enabled: !p.enabled });
  }

  function startConfigure(p: ProviderItem) {
    setEditingId(p.id);
    setEditConfig({ ...(p.config || {}) });
  }

  function updateEditField(fk: string, val: string) {
    setEditConfig((prev) => ({ ...prev, [fk]: val }));
  }

  async function saveConfig(id: string) {
    setActionLoading(id);
    await configMut.mutateAsync({ id, config: editConfig });
  }

  function cancelEdit() {
    setEditingId(null);
    setEditConfig({});
  }

  const isLoading = provLoading;

  return (
    <PageShell variant="admin">
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            {t("admin.psp.title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.psp.subtitle")}
          </p>
        </div>
      </div>

      <div className="rounded-lg border bg-card">
        {isLoading ? (
          <div className="space-y-3 p-6">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
          </div>
        ) : items.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <CreditCard className="mb-3 h-10 w-10 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">
              {t("admin.psp.empty")}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              {t("admin.psp.empty_hint")}
            </p>
          </div>
        ) : (
          <div className="space-y-3 p-4">
            {items.map((p) => {
              const isEditing = editingId === p.id;
              const isActing = actionLoading === p.id;
              return (
                <div
                  key={p.id}
                  className="rounded-xl border border-border bg-muted/20 p-4 transition-all hover:border-primary/20"
                >
                  <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                    <div className="flex flex-wrap items-center gap-4">
                      <div>
                        <div className="text-base font-bold text-foreground">{p.name}</div>
                        <div className="text-[10px] font-mono uppercase tracking-[2px] text-muted-foreground">
                          {p.key}
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        {p.enabled ? (
                          <Badge className="bg-emerald-500/10 text-emerald-600 hover:bg-emerald-500/10 ring-1 ring-inset ring-emerald-500/20">
                            {t("admin.settings.providers.enabled")}
                          </Badge>
                        ) : (
                          <Badge variant="outline">
                            {t("admin.settings.providers.disabled")}
                          </Badge>
                        )}
                        {!p.configured && (
                          <Badge className="bg-amber-500/10 text-amber-600 hover:bg-amber-500/10 ring-1 ring-inset ring-amber-500/20">
                            {t("admin.psp.not_configured")}
                          </Badge>
                        )}
                        {p.supported ? (
                          <Badge className="bg-sky-500/10 text-sky-600 hover:bg-sky-500/10 ring-1 ring-inset ring-sky-500/20">
                            {t("admin.psp.supported")}
                          </Badge>
                        ) : (
                          <Badge variant="outline" className="text-muted-foreground">
                            {t("admin.psp.unsupported")}
                          </Badge>
                        )}
                      </div>
                    </div>

                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={isActing}
                        onClick={() => handleToggle(p)}
                      >
                        {p.enabled ? t("common.disable") : t("common.enable")}
                      </Button>
                      <Button
                        variant="default"
                        size="sm"
                        disabled={isActing}
                        onClick={() => startConfigure(p)}
                      >
                        {p.configured
                          ? t("common.edit")
                          : t("admin.psp.configure")}
                      </Button>
                    </div>
                  </div>

                  {isEditing && (
                    <div className="mt-4 border-t pt-4">
                      <div className="mb-2 text-xs font-semibold text-muted-foreground">
                        {t("admin.psp.config_of", { name: p.name })}
                      </div>
                      {Object.keys(p.fields).length > 0 ? (
                        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                          {Object.entries(p.fields).map(([fk, fdef]) => {
                            const label = (fdef as any).label || fk;
                            const typ = (fdef as any).type || "text";
                            const isPass = typ === "password";
                            return (
                              <div key={fk} className="space-y-1">
                                <Label className="text-xs text-muted-foreground">{label}</Label>
                                <Input
                                  type={isPass ? "password" : "text"}
                                  value={editConfig[fk] ?? ""}
                                  onChange={(e) => updateEditField(fk, e.target.value)}
                                  placeholder={(fdef as any).default || ""}
                                  disabled={isActing}
                                />
                              </div>
                            );
                          })}
                        </div>
                      ) : (
                        <div className="text-xs text-muted-foreground">
                          {t("admin.psp.no_fields")}
                          <textarea
                            className="mt-2 w-full rounded border bg-background p-2 font-mono text-xs"
                            rows={3}
                            value={JSON.stringify(editConfig, null, 2)}
                            onChange={(e) => {
                              try {
                                setEditConfig(JSON.parse(e.target.value) || {});
                              } catch {}
                            }}
                          />
                        </div>
                      )}
                      <div className="mt-3 flex gap-2">
                        <Button size="sm" disabled={isActing} onClick={() => saveConfig(p.id)}>
                          {t("common.save")}
                        </Button>
                        <Button size="sm" variant="ghost" onClick={cancelEdit} disabled={isActing}>
                          {t("common.cancel")}
                        </Button>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </PageShell>
  );
}
