"use client";

import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Btn,
  Field,
  InfoRow,
  Panel,
  VX_FAINT,
  VX_INPUT,
  VX_MONO_LABEL,
  VX_MUTED,
  VX_SELECT,
} from "@/components/vx/panel-ui";
import {
  changeServerTariff,
  changeServerTariffResources,
  fetchBilling,
  fetchServerTariffs,
  previewServerRenew,
  previewServerTariffChange,
  previewServerTariffResources,
  renewServer,
  setServerAutoRenew,
} from "@/lib/api";
import { formatAmount } from "@/lib/format";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useServerDetail } from "@/hooks/use-queries";
import { getLocale } from "@/lib/i18n";
import { useT } from "@/hooks/use-translations";

export function ServerTariffTab() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const { data: server } = useServerDetail(id);

  const [renewPeriod, setRenewPeriod] = useState(30);
  const [autoRenew, setAutoRenew] = useState(false);
  const [promoCode, setPromoCode] = useState("");
  const [debouncedPromo, setDebouncedPromo] = useState("");
  const [selectedTariffId, setSelectedTariffId] = useState("");
  const [cpuCores, setCpuCores] = useState(1);
  const [ramGb, setRamGb] = useState(1);
  const [diskGb, setDiskGb] = useState(10);
  const [slots, setSlots] = useState(10);
  const [walletId, setWalletId] = useState("");

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedPromo(promoCode.trim()), 250);
    return () => clearTimeout(timer);
  }, [promoCode]);

  const limits = useMemo(() => server?.limits ?? {}, [server?.limits]);
  const tariff = server?.tariff;

  useEffect(() => {
    if (!server) return;
    setCpuCores(Number(limits.cpu_cores ?? limits.cpu ?? 1));
    setRamGb(
      Number(
        limits.ram_gb ??
          (limits.memory_mb
            ? Math.round(Number(limits.memory_mb) / 1024)
            : tariff?.ram_mb
              ? Math.round(tariff.ram_mb / 1024)
              : 1)
      )
    );
    setDiskGb(
      Number(
        limits.disk_gb ??
          (limits.disk_mb
            ? Math.round(Number(limits.disk_mb) / 1024)
            : tariff?.disk_mb
              ? Math.round(tariff.disk_mb / 1024)
              : 10)
      )
    );
    setSlots(Number(limits.slots ?? tariff?.slots ?? 10));
    if (tariff?.id) setSelectedTariffId(String(tariff.id));
  }, [server, limits, tariff?.id, tariff?.ram_mb, tariff?.disk_mb, tariff?.slots]);

  const { data: billing } = useQuery({ queryKey: queryKeys.billing(), queryFn: () => fetchBilling() });
  useEffect(() => {
    if (!walletId && billing?.selected_wallet?.id) setWalletId(billing.selected_wallet.id);
  }, [walletId, billing?.selected_wallet?.id]);

  const tariffsQuery = useQuery({
    queryKey: ["server-tariffs", id],
    queryFn: () => fetchServerTariffs(id),
    enabled: !!id,
  });

  const renewPreview = useQuery({
    queryKey: ["server-tariff-renew-preview", id, renewPeriod, debouncedPromo],
    queryFn: () => previewServerRenew(id, renewPeriod, debouncedPromo || undefined),
    enabled: !!id,
  });

  useEffect(() => {
    if (server?.auto_renew !== undefined) setAutoRenew(Boolean(server.auto_renew));
  }, [server?.auto_renew]);

  const autoRenewMutation = useMutation({
    mutationFn: (enabled: boolean) => setServerAutoRenew(id, enabled),
    onSuccess: (res) => {
      setAutoRenew(res.auto_renew);
      toast.success(
        res.auto_renew
          ? t("servers.tariff.autorenew_on")
          : t("servers.tariff.autorenew_off")
      );
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverDetail(id) });
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("common.error")),
  });

  const renewMutation = useMutation({
    mutationFn: () =>
      renewServer(id, renewPeriod, walletId || billing?.selected_wallet?.id, debouncedPromo || undefined),
    onSuccess: () => {
      toast.success(t("servers.overview.renewed"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverDetail(id) });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.overview.renew_error")
      ),
  });

  const changeTariffMutation = useMutation({
    mutationFn: (tariffId: string) =>
      changeServerTariff(id, tariffId, walletId || billing?.selected_wallet?.id),
    onSuccess: (data) => {
      toast.success(
        data.charged
          ? t("servers.tariff.changed_charged", {
              amount: data.charged,
              currency: data.currency ?? "",
            })
          : t("servers.tariff.changed")
      );
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverDetail(id) });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.tariff.change_error")
      ),
  });

  const resourcesMutation = useMutation({
    mutationFn: () =>
      changeServerTariffResources(
        id,
        { cpu_cores: cpuCores, ram_gb: ramGb, disk_gb: diskGb, slots },
        walletId || billing?.selected_wallet?.id
      ),
    onSuccess: (data) => {
      toast.success(
        data.charged
          ? t("servers.tariff.resources_charged", {
              amount: data.charged,
              currency: data.currency ?? "",
            })
          : t("servers.tariff.resources_updated")
      );
      void queryClient.invalidateQueries({ queryKey: queryKeys.serverDetail(id) });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("servers.tariff.resources_error")
      ),
  });

  const renewalPeriods = server?.tariff?.renewal_periods?.length
    ? server.tariff.renewal_periods
    : [15, 30, 60, 180];

  const availableTariffs = tariffsQuery.data?.tariffs ?? [];
  const activeTariff = useMemo(
    () => availableTariffs.find((item) => String(item.id) === String(tariff?.id)),
    [availableTariffs, tariff?.id]
  );

  useEffect(() => {
    if (!renewalPeriods.includes(renewPeriod)) setRenewPeriod(renewalPeriods[0] ?? 30);
  }, [renewalPeriods, renewPeriod]);

  const tariffChangePreview = useQuery({
    queryKey: ["server-tariff-change-preview", id, selectedTariffId],
    queryFn: () => previewServerTariffChange(id, selectedTariffId),
    enabled: !!id && !!selectedTariffId && selectedTariffId !== String(tariff?.id ?? ""),
  });

  const resourcesPreview = useQuery({
    queryKey: ["server-tariff-resources-preview", id, cpuCores, ramGb, diskGb, slots],
    queryFn: () =>
      previewServerTariffResources(id, { cpu_cores: cpuCores, ram_gb: ramGb, disk_gb: diskGb, slots }),
    enabled: !!id,
  });

  if (!server) return null;

  const expiresAt = server.expires_at ? new Date(server.expires_at) : null;
  const daysLeft = expiresAt
    ? Math.max(0, Math.ceil((expiresAt.getTime() - Date.now()) / 86_400_000))
    : null;

  const billingType = tariff?.billing_type ?? activeTariff?.billing_type ?? "resources";
  const showResourceSliders = billingType === "resources" || billingType === "slots";
  const promoError = renewPreview.data?.promo_preview?.error;

  return (
    <div className="grid grid-cols-1 items-start gap-[18px] lg:grid-cols-2">
      <Panel
        title={t("servers.tariff.renew_title")}
        bodyClassName="flex flex-col gap-3.5 p-[18px]"
      >
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
          {renewalPeriods.map((p) => (
            <button
              key={p}
              type="button"
              onClick={() => setRenewPeriod(p)}
              className={cn(
                "h-[34px] rounded-[9px] border text-[12.5px] font-medium transition-colors",
                renewPeriod === p
                  ? "border-[var(--vx-fg-strong)] bg-[var(--vx-fg-strong)] text-[var(--vx-on-fill)]"
                  : "border-[var(--vx-border-2)] bg-[var(--vx-inset)] text-[var(--vx-fg)] hover:border-[var(--vx-border-strong)]"
              )}
            >
              {t("servers.tariff.days", { days: p })}
            </button>
          ))}
        </div>

        <Field label={t("servers.tariff.promo")}>
          <input
            className={VX_INPUT}
            value={promoCode}
            onChange={(e) => setPromoCode(e.target.value)}
            placeholder="VORTANIX10"
          />
        </Field>
        {promoError && <div className="text-[11.5px] text-[var(--vx-warn)]">{promoError}</div>}

        {(billing?.wallets?.length ?? 0) > 1 && (
          <Field label={t("servers.tariff.wallet")}>
            <select
              className={cn(VX_SELECT, "w-full")}
              value={walletId}
              onChange={(e) => setWalletId(e.target.value)}
            >
              {(billing?.wallets ?? []).map((w) => (
                <option key={w.id} value={w.id}>
                  {w.currency}: {formatAmount(w.balance)}
                </option>
              ))}
            </select>
          </Field>
        )}

        <div className="rounded-[10px] border border-[var(--vx-border-2)] bg-[var(--vx-elevated)] px-4 py-3.5">
          <div className="flex items-center justify-between text-[13px] font-medium">
            <span>{t("servers.tariff.due")}</span>
            <span className="font-mono">
              {renewPreview.data
                ? `${formatAmount(renewPreview.data.price)} ${renewPreview.data.currency}`
                : "—"}
            </span>
          </div>
          {billing?.selected_wallet && (
            <div className={cn("mt-1.5 text-[11.5px]", VX_FAINT)}>
              {t("servers.tariff.balance", {
                amount: formatAmount(billing.selected_wallet.balance),
                currency: billing.selected_wallet.currency,
              })}
            </div>
          )}
        </div>

        <Btn
          tone="primary"
          className="h-9 w-full"
          onClick={() => renewMutation.mutate()}
          disabled={renewMutation.isPending}
        >
          {renewMutation.isPending
            ? t("servers.overview.paying")
            : t("servers.overview.renew")}
        </Btn>

        <label className="flex items-start gap-2.5">
          <input
            type="checkbox"
            checked={autoRenew}
            onChange={(e) => autoRenewMutation.mutate(e.target.checked)}
            disabled={autoRenewMutation.isPending}
            className="mt-0.5 size-4 accent-[var(--vx-fg-strong)]"
          />
          <span className="text-[12.5px]">
            {t("servers.tariff.autorenew_label")}
            <span className={cn("mt-0.5 block text-[11.5px]", VX_FAINT)}>
              {t("servers.tariff.autorenew_hint")}
            </span>
          </span>
        </label>
      </Panel>

      <Panel
        title={t("servers.tariff.resources_title")}
        bodyClassName="flex flex-col gap-4 p-[18px]"
      >
        <div>
          <InfoRow k={t("common.tariff")} v={tariff?.name ?? "—"} />
          {tariff?.price_monthly != null && (
            <InfoRow
              k={t("servers.tariff.row_price")}
              v={t("servers.tariff.price_per_month", {
                amount: formatAmount(tariff.price_monthly),
                currency: tariff.currency ?? "RUB",
              })}
            />
          )}
          {expiresAt && (
            <InfoRow
              k={t("servers.tariff.row_rented_until")}
              v={expiresAt.toLocaleDateString(getLocale() === "en" ? "en-GB" : "ru")}
            />
          )}
          {daysLeft !== null && (
            <InfoRow k={t("servers.tariff.row_days_left")} v={String(daysLeft)} />
          )}
        </div>

        {availableTariffs.length > 1 && (
          <div className="border-t border-[var(--vx-border)] pt-3.5">
            <div className={VX_MONO_LABEL}>{t("servers.tariff.change")}</div>
            <select
              className={cn(VX_SELECT, "mt-2.5 w-full")}
              value={selectedTariffId}
              onChange={(e) => setSelectedTariffId(e.target.value)}
            >
              {availableTariffs.map((item) => (
                <option key={String(item.id)} value={String(item.id)}>
                  {item.name}
                </option>
              ))}
            </select>
            <Btn
              className="mt-2.5 w-full"
              disabled={changeTariffMutation.isPending || selectedTariffId === String(tariff?.id ?? "")}
              onClick={() => changeTariffMutation.mutate(selectedTariffId)}
            >
              {changeTariffMutation.isPending
                ? t("servers.tariff.applying")
                : t("servers.tariff.change")}
            </Btn>
            {tariffChangePreview.data && (
              <p className={cn("mt-2 text-[11.5px]", VX_FAINT)}>
                {t("servers.tariff.preview")}{" "}
                {tariffChangePreview.data.charged > 0
                  ? t("servers.tariff.preview_charge", {
                      amount: formatAmount(tariffChangePreview.data.charged),
                      currency: tariffChangePreview.data.currency,
                    })
                  : t("servers.tariff.preview_free")}
                {t("servers.tariff.preview_days_left", {
                  days: Math.ceil(tariffChangePreview.data.days_left),
                })}
              </p>
            )}
          </div>
        )}

        {showResourceSliders && (
          <div className="flex flex-col gap-4 border-t border-[var(--vx-border)] pt-3.5">
            {billingType === "resources" && (
              <>
                <ResourceSlider
                  label="CPU"
                  value={cpuCores}
                  display={String(cpuCores)}
                  min={activeTariff?.cpu_min ?? 1}
                  max={activeTariff?.cpu_max ?? 16}
                  step={activeTariff?.cpu_step ?? 1}
                  onChange={setCpuCores}
                />
                <ResourceSlider
                  label="RAM"
                  value={ramGb}
                  display={`${ramGb} GB`}
                  min={activeTariff?.ram_min ?? 1}
                  max={activeTariff?.ram_max ?? 64}
                  step={activeTariff?.ram_step ?? 1}
                  onChange={setRamGb}
                />
                <ResourceSlider
                  label={t("servers.tariff.slider_disk")}
                  value={diskGb}
                  display={`${diskGb} GB`}
                  min={activeTariff?.disk_min ?? 10}
                  max={activeTariff?.disk_max ?? 500}
                  step={activeTariff?.disk_step ?? 10}
                  onChange={setDiskGb}
                />
              </>
            )}
            <ResourceSlider
              label={t("servers.tariff.slider_slots")}
              value={slots}
              display={String(slots)}
              min={activeTariff?.min_slots ?? 1}
              max={activeTariff?.max_slots ?? 100}
              step={1}
              onChange={setSlots}
            />

            <div
              className={cn(
                "rounded-[10px] border border-[var(--vx-border-2)] bg-[var(--vx-elevated)] px-3.5 py-3 text-[12px]",
                VX_MUTED
              )}
            >
              {t("servers.tariff.preview")}{" "}
              {resourcesPreview.data
                ? `${
                    resourcesPreview.data.charged > 0
                      ? t("servers.tariff.preview_charge", {
                          amount: formatAmount(resourcesPreview.data.charged),
                          currency: resourcesPreview.data.currency,
                        })
                      : t("servers.tariff.preview_free")
                  }${t("servers.tariff.preview_days_left", {
                    days: Math.ceil(resourcesPreview.data.days_left),
                  })}`
                : t("servers.tariff.calculating")}
            </div>

            <Btn
              className="h-9 w-full"
              disabled={resourcesMutation.isPending}
              onClick={() => resourcesMutation.mutate()}
            >
              {resourcesMutation.isPending
                ? t("servers.tariff.applying")
                : t("servers.tariff.apply_resources")}
            </Btn>
          </div>
        )}
      </Panel>
    </div>
  );
}

function ResourceSlider({
  label,
  value,
  display,
  min,
  max,
  step,
  onChange,
}: {
  label: string;
  value: number;
  display: string;
  min: number;
  max: number;
  step: number;
  onChange: (v: number) => void;
}) {
  return (
    <div className="flex flex-col gap-[7px]">
      <div className="flex items-center justify-between text-[12px]">
        <span className={VX_MUTED}>{label}</span>
        <span className="font-mono text-[12px] font-medium">{display}</span>
      </div>
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="w-full accent-[var(--vx-fg-strong)]"
      />
    </div>
  );
}
