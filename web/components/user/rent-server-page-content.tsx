"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { useMutation, useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Btn,
  EmptyState,
  Field,
  Panel,
  VX_FAINT,
  VX_INPUT,
  VX_MUTED,
  VX_SELECT,
  btnClass,
} from "@/components/vx/panel-ui";
import {
  createTrialServer,
  fetchBilling,
  fetchRentCatalog,
  fetchRentQuote,
  fetchTrialStatus,
  submitRentServer,
  type RentGameOption,
  type RentNode,
  type RentTariff,
} from "@/lib/api";
import { formatAmount } from "@/lib/format";
import { t } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

const STEPS = [
  { id: "game", num: 1, labelKey: "common.game" },
  { id: "config", num: 2, labelKey: "billing.rent.step_config" },
  { id: "confirm", num: 3, labelKey: "billing.rent.step_confirm" },
] as const;

type StepId = (typeof STEPS)[number]["id"];

const PERIODS = [15, 30, 60, 180] as const;

const CARD = "rounded-[14px] border border-[var(--vx-border)] bg-[var(--vx-card)]";
const INNER = "rounded-[12px] border border-[var(--vx-border)] bg-[var(--vx-bg)]";
const DIM = "text-[var(--vx-faint)]";

function money(value: number | null | undefined, currency = "RUB"): string {
  if (value == null || !Number.isFinite(value)) return "—";
  const symbol = currency === "RUB" ? "₽" : currency;
  return `${formatAmount(value, 0)} ${symbol}`;
}

function proRata(tariff: RentTariff | undefined, days: number): number | null {
  if (!tariff?.price_monthly) return null;
  return (tariff.price_monthly * days) / 30;
}

function tariffRange(
  tariff: RentTariff | undefined,
  field: "cpu" | "ram" | "disk",
  fallback: number
): { min: number; max: number; step: number; default: number } {
  if (!tariff) return { min: fallback, max: fallback * 2, step: 1, default: fallback };
  const minKey = `${field}_min` as keyof RentTariff;
  const maxKey = `${field}_max` as keyof RentTariff;
  const stepKey = `${field}_step` as keyof RentTariff;

  let defaultVal = fallback;
  if (field === "ram" && tariff.ram_gb != null) defaultVal = tariff.ram_gb;
  else if (field === "disk" && tariff.disk_gb != null) defaultVal = tariff.disk_gb;
  else if (field === "cpu" && tariff.cpu_cores != null) defaultVal = tariff.cpu_cores;
  else if (field === "ram" && tariff.ram_mb) defaultVal = Math.max(1, Math.round(tariff.ram_mb / 1024));
  else if (field === "disk" && tariff.disk_mb) defaultVal = Math.max(1, Math.round(tariff.disk_mb / 1024));

  const min = Number(tariff[minKey] ?? defaultVal);
  const max = Number(tariff[maxKey] ?? Math.max(defaultVal, min));
  const step = Number(tariff[stepKey] ?? 1) || 1;
  return { min, max: Math.max(max, min), step, default: defaultVal };
}

function tariffSpecs(tariff: RentTariff): { k: string; v: string }[] {
  const specs: { k: string; v: string }[] = [];
  if (tariff.cpu_cores != null) specs.push({ k: "CPU", v: `${tariff.cpu_cores}` });
  if (tariff.ram_mb) specs.push({ k: "RAM", v: `${Math.round(tariff.ram_mb / 1024)} GB` });
  if (tariff.disk_mb) {
    specs.push({ k: t("billing.rent.disk"), v: `${Math.round(tariff.disk_mb / 1024)} GB` });
  }
  if (tariff.slots != null) specs.push({ k: t("billing.rent.slots"), v: `${tariff.slots}` });
  return specs;
}

function GameCover({ game, className }: { game?: RentGameOption; className?: string }) {
  if (game?.image) {
    return (
      <Image
        src={game.image}
        alt=""
        width={320}
        height={180}
        unoptimized
        className={cn("h-full w-full object-cover", className)}
      />
    );
  }
  return (
    <span
      className={cn(
        "vx-cover flex h-full w-full items-center justify-center font-mono text-[10px] tracking-[0.08em]",
        DIM,
        className
      )}
    />
  );
}

function locationNote(node: RentNode): string {
  const parts = [node.country, node.code].filter(Boolean);
  const base = parts.join(" · ");
  const count = node.servers_count ?? 0;
  const load =
    count > 0 ? t("billing.rent.servers_count", { count }) : t("billing.rent.free");
  return base ? `${base} · ${load}` : load;
}

export function RentServerPageContent() {
  useT();
  const router = useRouter();
  const [step, setStep] = useState<StepId>("game");
  const [gameId, setGameId] = useState("");
  const [nodeId, setNodeId] = useState("");
  const [tariffId, setTariffId] = useState("");
  const [gameVersionId, setGameVersionId] = useState("");
  const [period, setPeriod] = useState("30");
  const [name, setName] = useState("");
  const [search, setSearch] = useState("");

  const [slots, setSlots] = useState(10);
  const [cpuCores, setCpuCores] = useState(1);
  const [ramGb, setRamGb] = useState(1);
  const [diskGb, setDiskGb] = useState(10);
  const [walletId, setWalletId] = useState("");
  const [promoCode, setPromoCode] = useState("");
  const [quoteParams, setQuoteParams] = useState<Record<string, string>>({});
  const recalcTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.rentCatalog,
    queryFn: fetchRentCatalog,
  });

  const quoteQuery = useQuery({
    queryKey: queryKeys.rentServer(quoteParams),
    queryFn: () => fetchRentQuote(quoteParams),
    enabled: !!quoteParams.tariff_id && !!quoteParams.period,
    placeholderData: (prev) => prev,
  });

  const scheduleQuote = useCallback((params: Record<string, string>) => {
    if (recalcTimerRef.current) clearTimeout(recalcTimerRef.current);
    recalcTimerRef.current = setTimeout(() => setQuoteParams(params), 150);
  }, []);

  const { data: billing } = useQuery({
    queryKey: queryKeys.billing(),
    queryFn: () => fetchBilling(),
  });

  const { data: trialStatus } = useQuery({
    queryKey: ["trial-status"],
    queryFn: fetchTrialStatus,
  });

  const trialMutation = useMutation({
    mutationFn: () =>
      createTrialServer({
        game_id: selectedGame?.slug ?? gameId,
        node_id: nodeId || undefined,
      }),
    onSuccess: (res) => {
      toast.success(t("billing.rent.trial_created", { hours: res.hours }));
      router.push(`/servers/${res.server_id}`);
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("billing.rent.trial_failed")
      ),
  });

  const createMutation = useMutation({
    mutationFn: submitRentServer,
    onSuccess: (res) => {
      toast.success(t("billing.rent.created"));
      router.push(`/servers/${res.server_id ?? res.id}`);
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : t("billing.rent.create_failed")
      ),
  });

  const games = data?.games ?? [];
  const allTariffs = data?.tariffs ?? [];
  const nodes = data?.nodes ?? [];

  const selectedGame = useMemo(() => games.find((g) => g.id === gameId), [games, gameId]);
  const selectedNode = useMemo(() => nodes.find((n) => n.id === nodeId), [nodes, nodeId]);

  const tariffs = useMemo(() => {
    if (!gameId) return allTariffs;
    return allTariffs.filter((t) => !t.game_id || t.game_id === gameId);
  }, [allTariffs, gameId]);

  const selectedTariff = useMemo(() => tariffs.find((t) => t.id === tariffId), [tariffs, tariffId]);
  const versions = useMemo(
    () =>
      (data?.game_versions ?? []).filter(
        (v) => v.game_id === gameId || v.game_slug === selectedGame?.slug
      ),
    [data?.game_versions, gameId, selectedGame?.slug]
  );

  const filteredGames = useMemo(() => {
    const q = search.trim().toLowerCase();
    return q ? games.filter((g) => g.name.toLowerCase().includes(q)) : games;
  }, [games, search]);

  const isSlotsTariff = selectedTariff?.billing_type === "slots";
  const isResourcesTariff = selectedTariff?.billing_type === "resources";
  const cpuRange = tariffRange(selectedTariff, "cpu", 1);
  const ramRange = tariffRange(selectedTariff, "ram", 1);
  const diskRange = tariffRange(selectedTariff, "disk", 10);
  const slotsMin = selectedTariff?.slots_min ?? 2;
  const slotsMax = selectedTariff?.slots_max ?? selectedTariff?.slots ?? 32;
  const currency = selectedTariff?.currency ?? "RUB";
  const promoPreview = quoteQuery.data?.promo_preview;
  const total =
    quoteQuery.data?.calculated_cost ??
    promoPreview?.final_cost ??
    proRata(selectedTariff, Number(period) || 30);
  const baseCost = promoPreview?.base_cost ?? quoteQuery.data?.base_cost ?? null;
  const discount = promoPreview?.valid ? (promoPreview.discount ?? 0) : 0;
  const isRecalculating = quoteQuery.isFetching;

  const wallet =
    (billing?.wallets ?? []).find((w) => w.id === (walletId || billing?.selected_wallet?.id)) ??
    billing?.selected_wallet;
  const balanceAfter = wallet && total != null ? Number(wallet.balance) - total : null;
  const notEnough = balanceAfter != null && balanceAfter < 0;

  useEffect(() => {
    if (!tariffId || !period) return;
    const params: Record<string, string> = {
      tariff_id: tariffId,
      period,
      game_id: gameId,
      location_id: nodeId,
    };
    if (isSlotsTariff) params.slots = String(slots);
    if (isResourcesTariff) {
      params.cpu_cores = String(cpuCores);
      params.ram_gb = String(ramGb);
      params.disk_gb = String(diskGb);
    }
    if (promoCode.trim()) params.promo_code = promoCode.trim();
    if (gameVersionId) params.game_version_id = gameVersionId;
    scheduleQuote(params);
  }, [
    tariffId,
    period,
    gameId,
    nodeId,
    slots,
    cpuCores,
    ramGb,
    diskGb,
    promoCode,
    gameVersionId,
    isSlotsTariff,
    isResourcesTariff,
    scheduleQuote,
  ]);

  useEffect(() => {
    if (!versions.length) {
      setGameVersionId("");
      return;
    }
    if (!versions.some((v) => v.id === gameVersionId)) setGameVersionId(versions[0]?.id ?? "");
  }, [versions, gameVersionId]);

  useEffect(() => {
    if (!tariffId) return;
    if (!tariffs.some((t) => t.id === tariffId)) setTariffId("");
  }, [tariffs, tariffId]);

  function selectTariff(tariff: RentTariff) {
    setTariffId(tariff.id);
    if (tariff.billing_type === "slots") {
      const min = tariff.slots_min ?? 2;
      const max = tariff.slots_max ?? tariff.slots ?? 32;
      setSlots(Math.min(max, Math.max(min, slots)));
    }
    if (tariff.billing_type === "resources") {
      const cpu = tariffRange(tariff, "cpu", 1);
      const ram = tariffRange(tariff, "ram", 1);
      const disk = tariffRange(tariff, "disk", 10);
      setCpuCores(Math.min(cpu.max, Math.max(cpu.min, cpu.default)));
      setRamGb(Math.min(ram.max, Math.max(ram.min, ram.default)));
      setDiskGb(Math.min(disk.max, Math.max(disk.min, disk.default)));
    }
  }

  const canLeaveGame = !!gameId;
  const canLeaveConfig = !!nodeId && !!tariffId && !!name.trim();
  const stepIdx = STEPS.findIndex((s) => s.id === step);

  function goNext() {
    if (step === "game" && canLeaveGame) setStep("config");
    else if (step === "config" && canLeaveConfig) setStep("confirm");
  }

  function goBack() {
    if (step === "confirm") setStep("config");
    else if (step === "config") setStep("game");
  }

  function handleCreate() {
    if (!nodeId || !name.trim()) return;
    createMutation.mutate({
      node_id: nodeId,
      game_id: gameId || undefined,
      game_version_id: gameVersionId || undefined,
      tariff_id: tariffId || undefined,
      name: name.trim(),
      period: Number(period) || 30,
      slots: isSlotsTariff ? slots : undefined,
      cpu_cores: isResourcesTariff ? cpuCores : undefined,
      ram_gb: isResourcesTariff ? ramGb : undefined,
      disk_gb: isResourcesTariff ? diskGb : undefined,
      promo_code: promoCode.trim() || undefined,
      wallet_id: walletId || billing?.selected_wallet?.id,
    });
  }

  const summaryRows: [string, string][] = [
    [t("common.game"), selectedGame?.name ?? "—"],
    [t("common.location"), selectedNode?.name ?? "—"],
    [t("common.tariff"), selectedTariff?.name ?? "—"],
    [t("common.period"), t("billing.hosting.days", { days: period })],
    ...(versions.length > 0
      ? ([
          [
            t("billing.rent.version"),
            versions.find((v) => v.id === gameVersionId)?.name ?? "—",
          ],
        ] as [string, string][])
      : []),
    ...(isSlotsTariff
      ? ([[t("billing.rent.slots"), String(slots)]] as [string, string][])
      : []),
    ...(isResourcesTariff
      ? ([
          ["CPU", `${cpuCores} ${t("billing.rent.cores")}`],
          ["RAM", `${ramGb} GB`],
          [t("billing.rent.disk"), `${diskGb} GB`],
        ] as [string, string][])
      : []),
    [t("common.name"), name.trim() || "—"],
  ];

  const priceRows: { label: string; value: string; tone?: "discount" }[] = [
    {
      label: t("billing.rent.tariff_x_days", { days: period }),
      value: money(baseCost ?? total, currency),
    },
    ...(discount > 0
      ? [
          {
            label: t("billing.topup.promo"),
            value: `−${money(discount, currency)}`,
            tone: "discount" as const,
          },
        ]
      : []),
  ];

  const ctaDisabled = createMutation.isPending || !canLeaveConfig || notEnough;

  return (
    <PageShell variant="user">
      <div className="font-panel flex flex-col gap-5 pb-24 lg:pb-0">
        <div className="flex flex-wrap items-center gap-6">
          <div className="min-w-0">
            <h1 className="m-0 text-[22px] font-bold">{t("billing.rent.title")}</h1>
            <p className={cn("mt-1 text-[13px]", VX_MUTED)}>
              {t("billing.rent.subtitle")}
            </p>
          </div>
          <div
            className={cn(
              "ml-auto flex items-center gap-1.5 rounded-[10px] border border-[var(--vx-border)] bg-[var(--vx-card)] p-1",
              "overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
            )}
          >
            {STEPS.map((s, idx) => {
              const active = s.id === step;
              const reachable =
                idx === 0 || (idx === 1 && canLeaveGame) || (idx === 2 && canLeaveGame && canLeaveConfig);
              return (
                <button
                  key={s.id}
                  type="button"
                  disabled={!reachable}
                  onClick={() => setStep(s.id)}
                  className={cn(
                    "inline-flex h-8 shrink-0 items-center gap-2 rounded-[8px] px-3 text-[13px] font-medium transition-colors",
                    active
                      ? "bg-[var(--vx-tint)] text-[var(--vx-fg-strong)]"
                      : reachable
                        ? cn(VX_MUTED, "hover:text-[var(--vx-fg)]")
                        : cn(DIM, "cursor-not-allowed")
                  )}
                >
                  <span
                    className={cn(
                      "inline-flex h-5 w-5 items-center justify-center rounded-full font-mono text-[11px]",
                      active ? "bg-[var(--vx-fg-strong)] text-[var(--vx-on-fill)]" : "bg-[var(--vx-tint)] text-[var(--vx-muted)]"
                    )}
                  >
                    {s.num}
                  </span>
                  {t(s.labelKey)}
                </button>
              );
            })}
          </div>
        </div>

        {isLoading ? (
          <div className="grid grid-cols-1 gap-5 lg:grid-cols-[minmax(0,1fr)_340px]">
            <Skeleton className="h-[520px] rounded-[14px]" />
            <Skeleton className="h-[320px] rounded-[14px]" />
          </div>
        ) : (
          <div className="grid grid-cols-1 items-start gap-5 lg:grid-cols-[minmax(0,1fr)_340px]">
            <div className="flex flex-col gap-4">
              {step === "game" && trialStatus?.available && selectedGame && (
                <div
                  className={cn(
                    CARD,
                    "flex flex-wrap items-center justify-between gap-3 px-5 py-4"
                  )}
                >
                  <div>
                    <div className="text-[14px] font-semibold">
                      {t("billing.rent.trial_title")}
                    </div>
                    <div className={cn("mt-0.5 text-[12.5px]", VX_MUTED)}>
                      {t("billing.rent.trial_hint", {
                        game: selectedGame.name,
                        hours: trialStatus.hours,
                      })}
                    </div>
                  </div>
                  <Btn
                    tone="primary"
                    onClick={() => trialMutation.mutate()}
                    disabled={trialMutation.isPending}
                  >
                    {trialMutation.isPending
                      ? t("billing.rent.trial_creating")
                      : t("billing.rent.trial_submit")}
                  </Btn>
                </div>
              )}

              {step === "game" && (
                <div className={CARD}>
                  <div className="flex flex-wrap items-center gap-3 border-b border-[var(--vx-border)] px-5 py-4">
                    <span className="text-[15px] font-semibold">
                      {t("billing.rent.choose_game")}
                    </span>
                    <span className={cn("text-[12px]", VX_MUTED)}>
                      {t("billing.rent.games_available", { count: games.length })}
                    </span>
                    <div className="ml-auto flex h-8 min-w-[220px] items-center gap-2 rounded-[8px] border border-[var(--vx-border-2)] bg-[var(--vx-bg)] px-3">
                      <i className={cn("ri-search-line text-[14px]", VX_MUTED)} />
                      <input
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        placeholder={t("billing.rent.search_games", {
                          count: games.length,
                        })}
                        className="w-full bg-transparent text-[13px] text-[var(--vx-fg)] outline-none placeholder:text-[var(--vx-muted)]"
                      />
                    </div>
                  </div>

                  {filteredGames.length === 0 ? (
                    <EmptyState>
                      {games.length === 0
                        ? t("billing.rent.no_games")
                        : t("common.not_found")}
                    </EmptyState>
                  ) : (
                    <div className="grid grid-cols-2 gap-3.5 p-5 sm:grid-cols-3 xl:grid-cols-4">
                      {filteredGames.map((game) => {
                        const selected = gameId === game.id;
                        return (
                          <button
                            key={game.id}
                            type="button"
                            onClick={() => setGameId(game.id)}
                            className={cn(
                              "flex flex-col overflow-hidden rounded-[12px] border bg-[var(--vx-bg)] text-left transition-colors",
                              selected
                                ? "border-[var(--vx-fg-strong)]"
                                : "border-[var(--vx-border)] hover:border-[var(--vx-border-strong)]"
                            )}
                          >
                            <span className="relative block aspect-video border-b border-[var(--vx-border)]">
                              <GameCover game={game} />
                              {selected && (
                                <span className="absolute top-2 right-2 inline-flex h-[22px] w-[22px] items-center justify-center rounded-full bg-[var(--vx-fg-strong)] text-[var(--vx-on-fill)]">
                                  <i className="ri-check-line text-[14px]" />
                                </span>
                              )}
                            </span>
                            <span className="flex flex-col gap-1.5 p-3">
                              <span className="truncate text-[13px] font-semibold text-[var(--vx-fg)]">
                                {game.name}
                              </span>
                              <span className="flex items-baseline gap-1.5">
                                <span className={cn("text-[12px]", VX_MUTED)}>
                                  {t("billing.rent.from")}
                                </span>
                                <span className="font-mono text-[13px] text-[var(--vx-fg)]">
                                  {game.min_price != null ? money(game.min_price) : "—"}
                                </span>
                                <span className={cn("text-[11px]", VX_MUTED)}>
                                  {t("billing.rent.per_month")}
                                </span>
                              </span>
                              <span className={cn("text-[11px]", DIM)}>
                                {t("billing.rent.servers_here", {
                                  count: game.servers_count ?? 0,
                                })}
                              </span>
                            </span>
                          </button>
                        );
                      })}
                    </div>
                  )}
                </div>
              )}

              {step === "config" && (
                <>
                  <div className={CARD}>
                    <div className="flex flex-wrap items-center gap-3 border-b border-[var(--vx-border)] px-5 py-4">
                      <span className="text-[15px] font-semibold">{t("common.tariff")}</span>
                      <span className={cn("text-[12px]", VX_MUTED)}>
                        {t("billing.rent.prices_30d")}
                      </span>
                    </div>
                    {tariffs.length === 0 ? (
                      <div className="flex flex-col items-start gap-2.5 p-5">
                        <i className="ri-inbox-line text-[22px] text-[var(--vx-ghost)]" />
                        <span className="text-[14px] font-semibold">
                          {t("billing.rent.no_tariffs")}
                        </span>
                        <span className={cn("text-[13px]", VX_MUTED)}>
                          {t("billing.rent.no_tariffs_hint")}
                        </span>
                        <a href="/support" className={btnClass("default", "sm")}>
                          {t("billing.rent.contact_support")}
                        </a>
                      </div>
                    ) : (
                      <div className="grid grid-cols-1 gap-3.5 p-5 sm:grid-cols-2 xl:grid-cols-4">
                        {tariffs.map((tariff) => {
                          const selected = tariffId === tariff.id;
                          const specs = tariffSpecs(tariff);
                          return (
                            <button
                              key={tariff.id}
                              type="button"
                              onClick={() => selectTariff(tariff)}
                              className={cn(
                                "flex flex-col gap-2.5 rounded-[12px] border bg-[var(--vx-bg)] p-4 text-left transition-colors",
                                selected
                                  ? "border-[var(--vx-fg-strong)]"
                                  : "border-[var(--vx-border)] hover:border-[var(--vx-border-strong)]"
                              )}
                            >
                              <span className="flex items-center gap-2">
                                <span className="text-[14px] font-semibold">{tariff.name}</span>
                                {tariff.billing_type === "resources" && (
                                  <span className="rounded-[4px] bg-[var(--vx-fg-strong)] px-1.5 py-0.5 text-[10px] tracking-[0.06em] text-[var(--vx-on-fill)] uppercase">
                                    {t("billing.rent.custom_config")}
                                  </span>
                                )}
                              </span>
                              <span className={cn("flex flex-col gap-1.5 text-[12px]", VX_MUTED)}>
                                {specs.map((spec) => (
                                  <span key={spec.k} className="flex justify-between gap-2">
                                    <span>{spec.k}</span>
                                    <span className="font-mono text-[var(--vx-fg)]">{spec.v}</span>
                                  </span>
                                ))}
                              </span>
                              <span className="mt-auto border-t border-[var(--vx-border)] pt-2.5 font-mono text-[15px] text-[var(--vx-fg)]">
                                {money(tariff.price_monthly, tariff.currency ?? "RUB")}
                              </span>
                            </button>
                          );
                        })}
                      </div>
                    )}

                    {isResourcesTariff && (
                      <div className="grid grid-cols-1 gap-5 px-5 pb-5 sm:grid-cols-3">
                        <ResourceSlider
                          label="CPU"
                          unit={t("billing.rent.cores")}
                          value={cpuCores}
                          range={cpuRange}
                          onChange={setCpuCores}
                        />
                        <ResourceSlider
                          label="RAM"
                          unit="GB"
                          value={ramGb}
                          range={ramRange}
                          onChange={setRamGb}
                        />
                        <ResourceSlider
                          label={t("billing.rent.disk")}
                          unit="GB"
                          value={diskGb}
                          range={diskRange}
                          onChange={setDiskGb}
                        />
                      </div>
                    )}
                  </div>

                  <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                    <div className={cn(CARD, "flex flex-col gap-4 p-5")}>
                      <span className="text-[15px] font-semibold">{t("common.server")}</span>
                      <Field label={t("common.name")}>
                        <input
                          className={cn(VX_INPUT, "h-9")}
                          value={name}
                          onChange={(e) => setName(e.target.value)}
                          placeholder={t("billing.rent.name_placeholder")}
                        />
                      </Field>
                      {versions.length > 0 && (
                        <Field label={t("billing.rent.game_version")}>
                          <select
                            className={cn(VX_SELECT, "h-9 w-full")}
                            value={gameVersionId}
                            onChange={(e) => setGameVersionId(e.target.value)}
                          >
                            {versions.map((v) => (
                              <option key={v.id} value={v.id}>
                                {v.name}
                                {v.source_type ? ` (${v.source_type})` : ""}
                              </option>
                            ))}
                          </select>
                        </Field>
                      )}
                      {isSlotsTariff && (
                        <Field
                          label={t("billing.rent.slots_range", {
                            min: slotsMin,
                            max: slotsMax,
                          })}
                        >
                          <input
                            type="number"
                            min={slotsMin}
                            max={slotsMax}
                            className={cn(VX_INPUT, "h-9")}
                            value={slots}
                            onChange={(e) => setSlots(Number(e.target.value))}
                          />
                        </Field>
                      )}
                    </div>

                    <div className={cn(CARD, "flex flex-col gap-4 p-5")}>
                      <span className="text-[15px] font-semibold">{t("common.location")}</span>
                      {nodes.length === 0 ? (
                        <EmptyState>{t("billing.rent.no_locations")}</EmptyState>
                      ) : (
                        <div className="flex flex-col gap-2">
                          {nodes.map((node) => {
                            const selected = nodeId === node.id;
                            const online = node.is_online !== false;
                            return (
                              <button
                                key={node.id}
                                type="button"
                                onClick={() => setNodeId(node.id)}
                                className={cn(
                                  "flex items-center justify-between gap-3 rounded-[10px] border bg-[var(--vx-bg)] px-3 py-2.5 transition-colors",
                                  selected
                                    ? "border-[var(--vx-fg-strong)]"
                                    : "border-[var(--vx-border)] hover:border-[var(--vx-border-strong)]"
                                )}
                              >
                                <span className="flex min-w-0 flex-col gap-0.5 text-left">
                                  <span className="truncate text-[13px] font-medium">
                                    {node.name}
                                  </span>
                                  <span className={cn("truncate text-[11px]", DIM)}>
                                    {locationNote(node)}
                                  </span>
                                </span>
                                <span
                                  className={cn(
                                    "shrink-0 font-mono text-[12px]",
                                    online ? "text-[var(--vx-ok)]" : "text-[var(--vx-warn)]"
                                  )}
                                >
                                  {online
                                    ? t("billing.rent.node_available")
                                    : t("billing.rent.node_offline")}
                                </span>
                              </button>
                            );
                          })}
                        </div>
                      )}

                      <span className="text-[15px] font-semibold">{t("common.period")}</span>
                      <div className="grid grid-cols-4 gap-2">
                        {PERIODS.map((days) => {
                          const selected = period === String(days);
                          const cost = proRata(selectedTariff, days);
                          return (
                            <button
                              key={days}
                              type="button"
                              onClick={() => setPeriod(String(days))}
                              className={cn(
                                "flex flex-col items-center gap-0.5 rounded-[10px] border bg-[var(--vx-bg)] px-2 py-2.5 transition-colors",
                                selected
                                  ? "border-[var(--vx-fg-strong)]"
                                  : "border-[var(--vx-border)] hover:border-[var(--vx-border-strong)]"
                              )}
                            >
                              <span className="text-[13px] font-medium">
                                {t("billing.rent.days_short", { days })}
                              </span>
                              <span className={cn("font-mono text-[11px]", VX_MUTED)}>
                                {cost != null ? money(cost, currency) : "—"}
                              </span>
                            </button>
                          );
                        })}
                      </div>
                    </div>
                  </div>

                  <div className={cn(CARD, "flex flex-wrap items-center gap-3 px-5 py-4")}>
                    <i className={cn("ri-price-tag-3-line", VX_MUTED)} />
                    <input
                      className={cn(VX_INPUT, "h-9 min-w-[160px] flex-1")}
                      value={promoCode}
                      onChange={(e) => setPromoCode(e.target.value.toUpperCase())}
                      placeholder={t("billing.topup.promo")}
                    />
                    {(billing?.wallets?.length ?? 0) > 1 && (
                      <select
                        className={cn(VX_SELECT, "h-9")}
                        value={walletId || billing?.selected_wallet?.id || ""}
                        onChange={(e) => setWalletId(e.target.value)}
                      >
                        {(billing?.wallets ?? []).map((w) => (
                          <option key={w.id} value={w.id}>
                            {t("billing.rent.wallet_option", {
                              currency: w.currency,
                              balance: formatAmount(w.balance),
                            })}
                          </option>
                        ))}
                      </select>
                    )}
                    {promoPreview?.error && (
                      <span className="inline-flex items-center gap-1.5 text-[12px] text-[var(--vx-danger)]">
                        <i className="ri-error-warning-line" />
                        {promoPreview.error}
                      </span>
                    )}
                    {promoPreview?.valid && discount > 0 && (
                      <span className="inline-flex items-center gap-1.5 text-[12px] text-[var(--vx-ok)]">
                        <i className="ri-check-line" />
                        {t("billing.rent.discount", {
                          amount: money(discount, currency),
                        })}
                      </span>
                    )}
                  </div>
                </>
              )}

              {step === "confirm" && (
                <div className={CARD}>
                  <div className="border-b border-[var(--vx-border)] px-5 py-4 text-[15px] font-semibold">
                    {t("billing.rent.review_order")}
                  </div>
                  <div className="px-5 pt-2 pb-5">
                    {summaryRows.map(([label, value]) => (
                      <div
                        key={label}
                        className="flex justify-between gap-4 border-b border-[var(--vx-elevated)] py-3 text-[13px] last:border-b-0"
                      >
                        <span className={VX_MUTED}>{label}</span>
                        <span className="truncate text-right font-medium">{value}</span>
                      </div>
                    ))}
                    <div className={cn("mt-4 flex items-center gap-2.5 text-[12px]", VX_MUTED)}>
                      <i className="ri-shield-check-line" />
                      <span>
                        {t("billing.rent.charge_note", {
                          currency: wallet?.currency ?? "RUB",
                        })}
                      </span>
                    </div>
                  </div>
                </div>
              )}

              <div className="flex flex-wrap gap-3">
                {stepIdx > 0 && (
                  <Btn className="h-[38px] rounded-[8px] border-[var(--vx-border-strong)] bg-transparent" onClick={goBack}>
                    <i className="ri-arrow-left-line" />
                    {t("common.back")}
                  </Btn>
                )}
                {stepIdx < 2 && (
                  <Btn
                    tone="primary"
                    className="h-[38px] rounded-[8px] px-4"
                    onClick={goNext}
                    disabled={step === "game" ? !canLeaveGame : !canLeaveConfig}
                  >
                    {t("common.next")}
                    <i className="ri-arrow-right-line" />
                  </Btn>
                )}
                {step === "confirm" && (
                  <Btn
                    tone="primary"
                    className="h-[38px] rounded-[8px] px-[18px]"
                    onClick={handleCreate}
                    disabled={ctaDisabled}
                  >
                    <i className="ri-flashlight-line" />
                    {createMutation.isPending
                      ? t("billing.rent.creating")
                      : t("billing.rent.pay_and_deploy")}
                  </Btn>
                )}
              </div>
            </div>

            <div
              className={cn(CARD, "flex flex-col gap-3 p-5 lg:sticky lg:top-24")}
            >
              <div className="flex items-center justify-between">
                <span className="text-[15px] font-semibold">{t("billing.rent.summary")}</span>
                <span className={cn("font-mono text-[11px]", DIM)}>
                  {t("billing.rent.step_of", { current: stepIdx + 1 })}
                </span>
              </div>

              {!selectedGame ? (
                <>
                  <div className="flex flex-col items-center gap-2.5 py-5 text-center">
                    <i className="ri-gamepad-line text-[22px] text-[var(--vx-ghost)]" />
                    <span className={cn("text-[13px]", VX_MUTED)}>
                      {t("billing.rent.pick_game_hint")}
                    </span>
                  </div>
                  <div className="flex flex-col gap-2">
                    {["70%", "45%", "60%"].map((w) => (
                      <span key={w} className="h-2.5 rounded-[6px] bg-[var(--vx-elevated)]" style={{ width: w }} />
                    ))}
                  </div>
                </>
              ) : (
                <>
                  <div className={cn(INNER, "flex items-center gap-3 p-3")}>
                    <span className="block h-8 w-14 shrink-0 overflow-hidden rounded-[6px]">
                      <GameCover game={selectedGame} />
                    </span>
                    <span className="flex min-w-0 flex-col">
                      <span className="truncate text-[13px] font-semibold">
                        {selectedGame.name}
                      </span>
                      <span className={cn("truncate text-[11px]", VX_MUTED)}>
                        {selectedTariff?.name ?? t("billing.rent.no_tariff_selected")}
                      </span>
                    </span>
                  </div>

                  <div className="flex flex-col gap-2.5 text-[13px]">
                    {priceRows.map((row) => (
                      <span key={row.label} className="flex justify-between gap-3">
                        <span className={VX_MUTED}>{row.label}</span>
                        <span
                          className={cn(
                            "font-mono",
                            row.tone === "discount" ? "text-[var(--vx-ok)]" : "text-[var(--vx-fg)]"
                          )}
                        >
                          {row.value}
                        </span>
                      </span>
                    ))}
                    <span className="flex justify-between gap-3">
                      <span className={VX_MUTED}>{t("common.location")}</span>
                      <span className="truncate font-mono text-[var(--vx-fg)]">
                        {selectedNode?.name ?? "—"}
                      </span>
                    </span>
                  </div>

                  <div className="flex items-baseline justify-between border-t border-[var(--vx-border)] pt-3">
                    <span className={cn("text-[13px]", VX_MUTED)}>
                      {isRecalculating
                        ? t("billing.rent.recalculating")
                        : t("billing.rent.to_pay")}
                    </span>
                    <span className="font-mono text-[24px] font-semibold">
                      {money(total, currency)}
                    </span>
                  </div>

                  {wallet && (
                    <div className={cn("flex items-center justify-between text-[12px]", VX_MUTED)}>
                      <span>{t("billing.rent.balance_after")}</span>
                      <span
                        className={cn(
                          "font-mono",
                          notEnough ? "text-[var(--vx-danger)]" : "text-[var(--vx-ok)]"
                        )}
                      >
                        {money(balanceAfter, wallet.currency)}
                      </span>
                    </div>
                  )}

                  {notEnough ? (
                    <a href="/billing/topup" className={cn(btnClass("primary"), "h-[38px] w-full rounded-[8px]")}>
                      {t("billing.history.topup_cta")}
                    </a>
                  ) : (
                    <Btn
                      tone="primary"
                      className="h-[38px] w-full rounded-[8px]"
                      onClick={step === "confirm" ? handleCreate : goNext}
                      disabled={step === "confirm" ? ctaDisabled : !canLeaveGame}
                    >
                      <i className="ri-flashlight-line" />
                      {step === "confirm"
                        ? createMutation.isPending
                          ? t("billing.rent.creating")
                          : t("billing.rent.pay_and_deploy")
                        : t("common.next")}
                    </Btn>
                  )}

                  <span className={cn("text-center text-[11px]", DIM)}>
                    {t("billing.rent.footnote")}
                  </span>
                </>
              )}
            </div>
          </div>
        )}
      </div>

      {!isLoading && selectedGame && (
        <div className="fixed inset-x-0 bottom-0 z-40 flex items-center gap-3.5 border-t border-[var(--vx-border)] bg-[rgba(10,11,13,0.94)] px-4 py-3.5 backdrop-blur-md lg:hidden">
          <span className="flex flex-col">
            <span className={cn("text-[11px]", VX_MUTED)}>
              {t("billing.rent.to_pay_for", { days: period })}
            </span>
            <span className="font-mono text-[20px] font-semibold">{money(total, currency)}</span>
          </span>
          <Btn
            tone="primary"
            className="h-11 flex-1 rounded-[10px] text-[15px]"
            onClick={step === "confirm" ? handleCreate : goNext}
            disabled={
              step === "confirm" ? ctaDisabled : step === "game" ? !canLeaveGame : !canLeaveConfig
            }
          >
            {step === "confirm"
              ? createMutation.isPending
                ? t("billing.rent.creating")
                : t("billing.rent.pay")
              : t("common.next")}
          </Btn>
        </div>
      )}
    </PageShell>
  );
}

function ResourceSlider({
  label,
  unit,
  value,
  range,
  onChange,
}: {
  label: string;
  unit: string;
  value: number;
  range: { min: number; max: number; step: number };
  onChange: (v: number) => void;
}) {
  return (
    <div className={cn(INNER, "flex flex-col gap-2 p-3.5")}>
      <span className="flex items-baseline justify-between">
        <span className={cn("text-[12px]", VX_MUTED)}>{label}</span>
        <span className="font-mono text-[14px]">
          {value} {unit}
        </span>
      </span>
      <input
        type="range"
        min={range.min}
        max={range.max}
        step={range.step}
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="w-full accent-[var(--vx-fg-strong)]"
      />
      <span className={cn("font-mono text-[11px]", DIM)}>
        {range.min}–{range.max} {unit}
      </span>
    </div>
  );
}
