"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Skeleton } from "@/components/ui/skeleton";
import { IdentificationNotice } from "@/components/user/identification-notice";
import { promptAction } from "@/components/action-dialog";
import {
  Btn,
  EmptyState,
  Field,
  VX_INPUT,
  VX_MUTED,
  VX_SELECT,
  btnClass,
} from "@/components/vx/panel-ui";
import {
  createProject,
  createTrialServer,
  fetchBilling,
  fetchProjects,
  fetchRentCatalog,
  fetchRentQuote,
  fetchTrialStatus,
  submitRentServer,
  type RentGameOption,
  type RentNode,
  type RentPriceLine,
  type RentTariff,
} from "@/lib/api";
import { formatAmount } from "@/lib/format";
import { t } from "@/lib/i18n";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import {
  allowedPeriods,
  fitRange,
  periodCost,
  slotRange,
  tariffRange,
} from "@/lib/tariff-pricing";
import { usePublicSettings } from "@/context/brand-provider";
import { useT } from "@/hooks/use-translations";
import { useLocationPing } from "@/hooks/use-location-ping";

const PERIODS = [15, 30, 60, 180] as const;
const HOURS_PER_MONTH = 720;

type PriceUnit = "hour" | "day" | "month";

const CARD = "rounded-[14px] border border-[var(--vx-border)] bg-[var(--vx-card)]";
const INNER = "rounded-[12px] border border-[var(--vx-border)] bg-[var(--vx-bg)]";
const DIM = "text-[var(--vx-faint)]";

function money(value: number | null | undefined, currency = "RUB"): string {
  if (value == null || !Number.isFinite(value)) return "—";
  const symbol = currency === "RUB" ? "₽" : currency;
  const digits = Math.abs(value) < 1000 ? 3 : 0;
  const fixed = formatAmount(value, digits);
  const text = fixed.includes(".") ? fixed.replace(/0+$/, "").replace(/\.$/, "") : fixed;
  return `${text} ${symbol}`;
}

function priceLineLabel(line: RentPriceLine): string {
  switch (line.key) {
    case "tariff":
      return t("billing.rent.line_tariff");
    case "base":
      return t("billing.rent.line_base");
    case "cpu":
      return "CPU";
    case "ram":
      return "RAM";
    case "disk":
      return t("billing.rent.disk");
    case "slots":
      return t("billing.rent.slots");
    case "antiddos":
      return t("billing.rent.line_antiddos");
    case "period_discount":
      return t("billing.rent.line_period_discount", { percent: line.percent ?? 0 });
    default:
      return line.key;
  }
}

function tariffSpecs(tariff: RentTariff): { k: string; v: string }[] {
  const specs: { k: string; v: string }[] = [];
  if (tariff.cpu_cores) specs.push({ k: "CPU", v: `${tariff.cpu_cores}` });
  if (tariff.ram_gb) specs.push({ k: "RAM", v: `${tariff.ram_gb} GB` });
  if (tariff.disk_gb) specs.push({ k: t("billing.rent.disk"), v: `${tariff.disk_gb} GB` });
  if (tariff.max_slots && tariff.billing_type !== "resources") {
    specs.push({
      k: t("billing.rent.slots"),
      v:
        tariff.billing_type === "slots"
          ? `${tariff.min_slots ?? 1}–${tariff.max_slots}`
          : `${tariff.max_slots}`,
    });
  }
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

function pingColor(ms: number): string {
  if (ms <= 60) return "text-[var(--vx-ok)]";
  if (ms <= 130) return "text-[var(--vx-warn)]";
  return "text-[var(--vx-danger)]";
}

function PingValue({ value }: { value: number | null | undefined }) {
  if (value === undefined) {
    return (
      <span className={cn("font-mono text-[11px]", DIM)}>
        {t("billing.rent.ping_measuring")}
      </span>
    );
  }
  if (value === null) {
    return (
      <span className={cn("font-mono text-[11px]", DIM)}>
        {t("billing.rent.ping_unknown")}
      </span>
    );
  }
  return (
    <span className={cn("font-mono text-[11px]", pingColor(value))}>
      {t("billing.rent.ping_value", { ms: value })}
    </span>
  );
}

function Section({
  num,
  title,
  hint,
  aside,
  children,
}: {
  num: number;
  title: string;
  hint?: string;
  aside?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section className={CARD}>
      <div className="flex flex-wrap items-center gap-2.5 border-b border-[var(--vx-border)] px-5 py-4">
        <span className="inline-flex h-6 w-6 items-center justify-center rounded-full bg-[var(--vx-tint)] font-mono text-[12px] text-[var(--vx-fg)]">
          {num}
        </span>
        <span className="text-[15px] font-semibold">{title}</span>
        {hint && <span className={cn("text-[12px]", VX_MUTED)}>{hint}</span>}
        {aside && <div className="ml-auto flex items-center gap-2">{aside}</div>}
      </div>
      {children}
    </section>
  );
}

export function RentServerPageContent() {
  useT();
  const { rentMaxBatch: MAX_SERVERS } = usePublicSettings();
  const router = useRouter();
  const queryClient = useQueryClient();
  const [gameId, setGameId] = useState("");
  const [nodeId, setNodeId] = useState("");
  const [tariffId, setTariffId] = useState("");
  const [gameVersionId, setGameVersionId] = useState("");
  const [period, setPeriod] = useState("30");
  const [name, setName] = useState("");
  const [comment, setComment] = useState("");
  const [projectId, setProjectId] = useState("");
  const [search, setSearch] = useState("");
  const [count, setCount] = useState(1);
  const [deleteProtection, setDeleteProtection] = useState(false);
  const [autoRenew, setAutoRenew] = useState(false);
  const [unit, setUnit] = useState<PriceUnit>("day");

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

  const projectsQuery = useQuery({
    queryKey: queryKeys.projects,
    queryFn: fetchProjects,
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

  const games = data?.games ?? [];
  const allTariffs = data?.tariffs ?? [];
  const nodes = data?.nodes ?? [];

  const selectedGame = useMemo(() => games.find((g) => g.id === gameId), [games, gameId]);
  const selectedNode = useMemo(() => nodes.find((n) => n.id === nodeId), [nodes, nodeId]);

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
      toast.error(err instanceof Error ? err.message : t("billing.rent.trial_failed")),
  });

  const createMutation = useMutation({
    mutationFn: submitRentServer,
    onSuccess: (res) => {
      if (res.partial_error) toast.warning(res.partial_error);
      toast.success(
        (res.count ?? 1) > 1
          ? t("billing.rent.created_many", { count: res.count ?? 1 })
          : t("billing.rent.created")
      );
      void queryClient.invalidateQueries({ queryKey: queryKeys.projects });
      router.push(`/servers/${res.server_id ?? res.id}`);
    },
    onError: (err) =>
      toast.error(err instanceof Error ? err.message : t("billing.rent.create_failed")),
  });

  const pingTargets = useMemo(
    () =>
      nodes
        .filter((n) => n.is_online !== false && n.ping_host && n.ping_port)
        .map((n) => ({ id: n.id, host: String(n.ping_host), port: Number(n.ping_port) })),
    [nodes]
  );
  const pings = useLocationPing(pingTargets);

  const tariffs = useMemo(
    () =>
      gameId
        ? allTariffs.filter(
            (item) =>
              (!item.game_id || item.game_id === gameId) &&
              (!nodeId || !item.location_id || item.location_id === nodeId)
          )
        : [],
    [allTariffs, gameId, nodeId]
  );

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
  const slotsRange = slotRange(selectedTariff, slots);
  const periods = allowedPeriods(selectedTariff?.rental_periods, PERIODS);
  const currency = selectedTariff?.currency ?? "RUB";
  const promoPreview = quoteQuery.data?.promo_preview;
  const perServer =
    quoteQuery.data?.calculated_cost ??
    promoPreview?.final_cost ??
    periodCost(selectedTariff, Number(period) || 30);
  const total =
    quoteQuery.data?.total_cost ?? (perServer != null ? perServer * count : null);
  const discount = promoPreview?.valid ? (promoPreview.discount ?? 0) : 0;
  const breakdown = quoteQuery.data?.breakdown ?? [];
  const isRecalculating = quoteQuery.isFetching;

  const hourly =
    quoteQuery.data?.payment_mode === "hourly" || selectedTariff?.payment_mode === "hourly";
  const hourlyRate = quoteQuery.data?.hourly_rate ?? 0;
  const unitFactor = unit === "hour" ? 1 : unit === "day" ? 24 : HOURS_PER_MONTH;
  const rateInUnit = hourlyRate * unitFactor * count;
  const prepaidHours = quoteQuery.data?.prepaid_hours ?? 24;

  const wallet =
    (billing?.wallets ?? []).find((w) => w.id === (walletId || billing?.selected_wallet?.id)) ??
    billing?.selected_wallet;
  const firstCharge = Math.floor(hourlyRate * 100 + 1e-6) / 100 * count;
  const minBalance = hourly ? hourlyRate * prepaidHours * count : 0;
  const hoursLeft =
    hourly && hourlyRate > 0 && wallet
      ? Math.floor(Number(wallet.balance) / (hourlyRate * count))
      : null;
  const balanceAfter =
    wallet && total != null ? Number(wallet.balance) - (hourly ? firstCharge : total) : null;
  const notEnough = hourly
    ? !!wallet && Number(wallet.balance) < minBalance
    : balanceAfter != null && balanceAfter < 0;
  const projects = projectsQuery.data?.projects ?? [];

  useEffect(() => {
    if (!tariffId || !period) return;
    const params: Record<string, string> = {
      tariff_id: tariffId,
      period,
      game_id: gameId,
      location_id: nodeId,
      count: String(count),
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
    count,
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
    const nextPeriods = allowedPeriods(tariff.rental_periods, PERIODS);
    if (!nextPeriods.includes(Number(period))) setPeriod(String(nextPeriods[0]));
    if (tariff.billing_type === "slots") {
      setSlots(fitRange(slotRange(tariff, slots), slots));
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

  const missing = !gameId
    ? t("billing.rent.need_game")
    : !nodeId
      ? t("billing.rent.need_location")
      : !tariffId
        ? t("billing.rent.need_tariff")
        : !name.trim()
          ? t("billing.rent.need_name")
          : "";
  const ready = missing === "";
  const ctaDisabled = createMutation.isPending || !ready || notEnough;

  async function createNewProject() {
    const title = await promptAction(t("projects.create_prompt"), {
      title: t("projects.create_title"),
      confirmText: t("common.create"),
      placeholder: t("projects.name_placeholder"),
    });
    if (!title || !title.trim()) return;
    try {
      const res = await createProject({ name: title.trim() });
      await queryClient.invalidateQueries({ queryKey: queryKeys.projects });
      setProjectId(res.id);
      toast.success(t("projects.created"));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("projects.create_failed"));
    }
  }

  function handleCreate() {
    if (!ready) return;
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
      project_id: projectId || undefined,
      comment: comment.trim() || undefined,
      delete_protection: deleteProtection,
      auto_renew: autoRenew,
      count,
    });
  }

  return (
    <PageShell variant="user">
      <div className="font-panel flex flex-col gap-5 pb-28 lg:pb-0">
        <IdentificationNotice />
        <div className="min-w-0">
          <h1 className="m-0 text-[22px] font-bold">{t("billing.rent.title")}</h1>
          <p className={cn("mt-1 text-[13px]", VX_MUTED)}>{t("billing.rent.subtitle")}</p>
        </div>

        {isLoading ? (
          <div className="grid grid-cols-1 gap-5 lg:grid-cols-[minmax(0,1fr)_340px]">
            <Skeleton className="h-[620px] rounded-[14px]" />
            <Skeleton className="h-[360px] rounded-[14px]" />
          </div>
        ) : (
          <div className="grid grid-cols-1 items-start gap-5 lg:grid-cols-[minmax(0,1fr)_340px]">
            <div className="flex flex-col gap-4">
              <Section
                num={1}
                title={t("billing.rent.choose_game")}
                hint={t("billing.rent.games_available", { count: games.length })}
                aside={
                  <div className="flex h-8 min-w-[200px] items-center gap-2 rounded-[8px] border border-[var(--vx-border-2)] bg-[var(--vx-bg)] px-3">
                    <i className={cn("ri-search-line text-[14px]", VX_MUTED)} />
                    <input
                      value={search}
                      onChange={(e) => setSearch(e.target.value)}
                      placeholder={t("billing.rent.search_games", { count: games.length })}
                      className="w-full bg-transparent text-[13px] text-[var(--vx-fg)] outline-none placeholder:text-[var(--vx-muted)]"
                    />
                  </div>
                }
              >
                {filteredGames.length === 0 ? (
                  <EmptyState>
                    {games.length === 0 ? t("billing.rent.no_games") : t("common.not_found")}
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
                              {t("billing.rent.servers_here", { count: game.servers_count ?? 0 })}
                            </span>
                          </span>
                        </button>
                      );
                    })}
                  </div>
                )}

                {versions.length > 0 && (
                  <div className="border-t border-[var(--vx-border)] px-5 py-4">
                    <Field label={t("billing.rent.game_version")}>
                      <select
                        className={cn(VX_SELECT, "h-9 w-full max-w-[320px]")}
                        value={gameVersionId}
                        onChange={(e) => setGameVersionId(e.target.value)}
                      >
                        {versions.map((v) => (
                          <option key={v.id} value={v.id}>
                            {v.name}
                          </option>
                        ))}
                      </select>
                    </Field>
                  </div>
                )}

                {trialStatus?.available && selectedGame && (
                  <div className="flex flex-wrap items-center justify-between gap-3 border-t border-[var(--vx-border)] px-5 py-4">
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
              </Section>

              <Section
                num={2}
                title={t("common.location")}
                hint={pingTargets.length > 0 ? t("billing.rent.ping_hint") : undefined}
              >
                {nodes.length === 0 ? (
                  <EmptyState>{t("billing.rent.no_locations")}</EmptyState>
                ) : (
                  <div className="grid grid-cols-1 gap-3 p-5 sm:grid-cols-2 xl:grid-cols-3">
                    {nodes.map((node) => {
                      const selected = nodeId === node.id;
                      const online = node.is_online !== false;
                      return (
                        <button
                          key={node.id}
                          type="button"
                          onClick={() => setNodeId(node.id)}
                          className={cn(
                            "flex items-center justify-between gap-3 rounded-[10px] border bg-[var(--vx-bg)] px-3.5 py-3 transition-colors",
                            selected
                              ? "border-[var(--vx-fg-strong)]"
                              : "border-[var(--vx-border)] hover:border-[var(--vx-border-strong)]"
                          )}
                        >
                          <span className="flex min-w-0 flex-col gap-0.5 text-left">
                            <span className="truncate text-[13px] font-medium">{node.name}</span>
                            <span className={cn("truncate text-[11px]", DIM)}>
                              {locationNote(node)}
                            </span>
                          </span>
                          <span className="flex shrink-0 flex-col items-end gap-0.5">
                            <span
                              className={cn(
                                "font-mono text-[12px]",
                                online ? "text-[var(--vx-ok)]" : "text-[var(--vx-warn)]"
                              )}
                            >
                              {online
                                ? t("billing.rent.node_available")
                                : t("billing.rent.node_offline")}
                            </span>
                            {online && node.ping_host && <PingValue value={pings[node.id]} />}
                          </span>
                        </button>
                      );
                    })}
                  </div>
                )}
              </Section>

              <Section
                num={3}
                title={t("billing.rent.step_config")}
                hint={t("billing.rent.prices_30d")}
                aside={
                  <div className="flex items-center gap-2">
                    <span className={cn("text-[12px]", VX_MUTED)}>
                      {t("billing.rent.server_count")}
                    </span>
                    <div className="flex h-8 items-center gap-1 rounded-[8px] border border-[var(--vx-border-2)] bg-[var(--vx-bg)] px-1">
                      <button
                        type="button"
                        className="flex h-6 w-6 items-center justify-center rounded-[6px] text-[var(--vx-muted)] transition-colors hover:text-[var(--vx-fg)] disabled:opacity-40"
                        onClick={() => setCount((n) => Math.max(1, n - 1))}
                        disabled={count <= 1}
                        aria-label={t("billing.rent.server_count_less")}
                      >
                        <i className="ri-subtract-line text-[15px]" />
                      </button>
                      <span className="w-5 text-center font-mono text-[13px]">{count}</span>
                      <button
                        type="button"
                        className="flex h-6 w-6 items-center justify-center rounded-[6px] text-[var(--vx-muted)] transition-colors hover:text-[var(--vx-fg)] disabled:opacity-40"
                        onClick={() => setCount((n) => Math.min(MAX_SERVERS, n + 1))}
                        disabled={count >= MAX_SERVERS}
                        aria-label={t("billing.rent.server_count_more")}
                      >
                        <i className="ri-add-line text-[15px]" />
                      </button>
                    </div>
                  </div>
                }
              >
                {!gameId ? (
                  <EmptyState>{t("billing.rent.pick_game_first")}</EmptyState>
                ) : tariffs.length === 0 ? (
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
                          <span className="flex flex-wrap items-start gap-x-2 gap-y-1.5">
                            <span className="min-w-0 text-[14px] leading-snug font-semibold [overflow-wrap:anywhere]">
                              {tariff.name}
                            </span>
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
                            {money(
                              tariff.price_from ?? tariff.price_monthly,
                              tariff.currency ?? "RUB"
                            )}
                          </span>
                        </button>
                      );
                    })}
                  </div>
                )}

                {isResourcesTariff && (
                  <div className="grid grid-cols-1 gap-5 border-t border-[var(--vx-border)] px-5 py-5 sm:grid-cols-3">
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

                {isSlotsTariff && (
                  <div className="border-t border-[var(--vx-border)] px-5 py-5">
                    <div className="max-w-[320px]">
                      <ResourceSlider
                        label={t("billing.rent.slots")}
                        unit=""
                        value={slots}
                        range={{
                          min: slotsRange.min,
                          max: slotsRange.max,
                          step: slotsRange.step,
                        }}
                        onChange={setSlots}
                      />
                    </div>
                  </div>
                )}
              </Section>

              <Section
                num={4}
                title={hourly ? t("billing.rent.hourly_title") : t("common.period")}
                hint={hourly ? t("billing.rent.hourly_note") : undefined}
              >
                <div className="flex flex-col gap-4 p-5">
                  {hourly ? (
                    <div className="flex flex-col gap-3">
                      <div className="flex w-fit gap-1 rounded-[8px] border border-[var(--vx-border)] bg-[var(--vx-bg)] p-[3px]">
                        {(
                          [
                            { id: "hour" as const, label: t("billing.rent.unit_hour") },
                            { id: "day" as const, label: t("billing.rent.unit_day") },
                            { id: "month" as const, label: t("billing.rent.unit_month") },
                          ] satisfies { id: PriceUnit; label: string }[]
                        ).map((item) => (
                          <button
                            key={item.id}
                            type="button"
                            onClick={() => setUnit(item.id)}
                            className={cn(
                              "h-7 rounded-[6px] px-3 text-[12px] font-medium transition-colors",
                              unit === item.id
                                ? "bg-[var(--vx-tint)] text-[var(--vx-fg)]"
                                : cn(VX_MUTED, "hover:text-[var(--vx-fg)]")
                            )}
                          >
                            {item.label}
                          </button>
                        ))}
                      </div>
                      <div className={cn(INNER, "flex flex-wrap items-baseline gap-x-3 gap-y-1 p-3.5")}>
                        <span className="font-mono text-[20px] font-semibold">
                          {money(rateInUnit, currency)}
                        </span>
                        <span className={cn("text-[12px]", VX_MUTED)}>
                          {unit === "hour"
                            ? t("billing.rent.per_hour")
                            : unit === "day"
                              ? t("billing.rent.per_day")
                              : t("billing.rent.per_month")}
                        </span>
                        <span className={cn("ml-auto text-[12px]", DIM)}>
                          {t("billing.rent.min_balance", {
                            hours: prepaidHours,
                            amount: money(minBalance, currency),
                          })}
                        </span>
                      </div>
                    </div>
                  ) : (
                  <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                    {periods.map((days) => {
                      const selected = period === String(days);
                      const cost = periodCost(selectedTariff, days);
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
                  )}

                  <div className="flex flex-wrap items-center gap-3">
                    {!hourly && (
                      <>
                        <i className={cn("ri-price-tag-3-line", VX_MUTED)} />
                        <input
                          className={cn(VX_INPUT, "h-9 min-w-[160px] flex-1")}
                          value={promoCode}
                          onChange={(e) => setPromoCode(e.target.value.toUpperCase())}
                          placeholder={t("billing.topup.promo")}
                        />
                      </>
                    )}
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
                    {!hourly && promoPreview?.error && (
                      <span className="inline-flex items-center gap-1.5 text-[12px] text-[var(--vx-danger)]">
                        <i className="ri-error-warning-line" />
                        {promoPreview.error}
                      </span>
                    )}
                    {!hourly && promoPreview?.valid && discount > 0 && (
                      <span className="inline-flex items-center gap-1.5 text-[12px] text-[var(--vx-ok)]">
                        <i className="ri-check-line" />
                        {t("billing.rent.discount", { amount: money(discount, currency) })}
                      </span>
                    )}
                  </div>
                </div>
              </Section>

              <Section num={5} title={t("billing.rent.safety")}>
                <div className="grid grid-cols-1 gap-3 p-5 sm:grid-cols-2">
                  <OptionToggle
                    icon="ri-shield-keyhole-line"
                    title={t("billing.rent.delete_protection")}
                    note={t("billing.rent.delete_protection_note")}
                    checked={deleteProtection}
                    onChange={setDeleteProtection}
                  />
                  <OptionToggle
                    icon="ri-refresh-line"
                    title={t("billing.rent.auto_renew")}
                    note={t("billing.rent.auto_renew_note")}
                    checked={autoRenew}
                    onChange={setAutoRenew}
                  />
                </div>
              </Section>

              <Section num={6} title={t("billing.rent.server_info")}>
                <div className="grid grid-cols-1 gap-4 p-5 sm:grid-cols-2">
                  <Field label={t("common.name")}>
                    <input
                      className={cn(VX_INPUT, "h-9")}
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      placeholder={t("billing.rent.name_placeholder")}
                    />
                  </Field>
                  <Field label={t("billing.rent.project")}>
                    <div className="flex items-center gap-2">
                      <select
                        className={cn(VX_SELECT, "h-9 w-full")}
                        value={projectId}
                        onChange={(e) => setProjectId(e.target.value)}
                      >
                        <option value="">{t("billing.rent.no_project")}</option>
                        {projects.map((p) => (
                          <option key={p.id} value={p.id}>
                            {p.name}
                          </option>
                        ))}
                      </select>
                      <Btn className="h-9 shrink-0" onClick={() => void createNewProject()}>
                        <i className="ri-add-line" />
                      </Btn>
                    </div>
                  </Field>
                  <div className="sm:col-span-2">
                    <Field label={t("billing.rent.comment")}>
                      <input
                        className={cn(VX_INPUT, "h-9")}
                        value={comment}
                        onChange={(e) => setComment(e.target.value)}
                        placeholder={t("billing.rent.comment_placeholder")}
                      />
                    </Field>
                  </div>
                  {count > 1 && (
                    <span className={cn("text-[12px] sm:col-span-2", VX_MUTED)}>
                      {t("billing.rent.name_suffix_hint", { name: name.trim() || "server" })}
                    </span>
                  )}
                </div>
              </Section>
            </div>

            <div className={cn(CARD, "flex flex-col gap-3 p-5 lg:sticky lg:top-24")}>
              <div className="flex items-center justify-between">
                <span className="text-[15px] font-semibold">{t("billing.rent.summary")}</span>
                {isRecalculating && (
                  <span className={cn("font-mono text-[11px]", DIM)}>
                    {t("billing.rent.recalculating")}
                  </span>
                )}
              </div>

              <div className={cn(INNER, "flex items-center gap-3 p-3")}>
                <span className="block h-8 w-14 shrink-0 overflow-hidden rounded-[6px]">
                  <GameCover game={selectedGame} />
                </span>
                <span className="flex min-w-0 flex-col">
                  <span className="truncate text-[13px] font-semibold">
                    {selectedGame?.name ?? t("billing.rent.pick_game_hint")}
                  </span>
                  <span className={cn("truncate text-[11px]", VX_MUTED)}>
                    {selectedTariff?.name ?? t("billing.rent.no_tariff_selected")}
                  </span>
                </span>
              </div>

              <div className="flex flex-col gap-2 text-[13px]">
                <SummaryRow label={t("common.location")} value={selectedNode?.name ?? "—"} />
                {hourly ? (
                  <SummaryRow
                    label={t("billing.rent.payment_mode")}
                    value={t("billing.rent.mode_hourly")}
                  />
                ) : (
                  <SummaryRow
                    label={t("common.period")}
                    value={t("billing.hosting.days", { days: period })}
                  />
                )}
                {breakdown.map((line) => (
                  <SummaryRow
                    key={line.key}
                    label={priceLineLabel(line)}
                    note={
                      line.qty && line.unit
                        ? t("billing.rent.line_qty", {
                            qty: line.qty,
                            unit: money(line.unit * unitFactor, currency),
                          })
                        : undefined
                    }
                    value={money(line.amount * (hourly ? unitFactor : 1), currency)}
                    tone={line.amount < 0 ? "discount" : undefined}
                  />
                ))}
                {discount > 0 && (
                  <SummaryRow
                    label={t("billing.topup.promo")}
                    value={`−${money(discount, currency)}`}
                    tone="discount"
                  />
                )}
                {count > 1 && (
                  <SummaryRow
                    label={t("billing.rent.server_count")}
                    value={t("billing.rent.times", { count })}
                  />
                )}
              </div>

              <div className="flex items-baseline justify-between border-t border-[var(--vx-border)] pt-3">
                <span className={cn("text-[13px]", VX_MUTED)}>
                  {hourly
                    ? unit === "hour"
                      ? t("billing.rent.per_hour")
                      : unit === "day"
                        ? t("billing.rent.per_day")
                        : t("billing.rent.per_month")
                    : t("billing.rent.to_pay")}
                </span>
                <span className="font-mono text-[24px] font-semibold">
                  {money(hourly ? rateInUnit : total, currency)}
                </span>
              </div>

              {hourly && (
                <>
                  <SummaryRow
                    label={t("billing.rent.charge_now")}
                    value={money(firstCharge, currency)}
                  />
                  {hoursLeft != null && (
                    <SummaryRow
                      label={t("billing.rent.balance_lasts")}
                      value={t("billing.rent.hours_short", { hours: hoursLeft })}
                    />
                  )}
                </>
              )}

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
                <a
                  href="/billing#topup"
                  className={cn(btnClass("primary"), "h-[38px] w-full rounded-[8px]")}
                >
                  {t("billing.history.topup_cta")}
                </a>
              ) : (
                <Btn
                  tone="primary"
                  className="h-[38px] w-full rounded-[8px]"
                  onClick={handleCreate}
                  disabled={ctaDisabled}
                >
                  <i className="ri-flashlight-line" />
                  {createMutation.isPending
                    ? t("billing.rent.creating")
                    : t("billing.rent.pay_and_deploy")}
                </Btn>
              )}

              <span className={cn("text-center text-[11px]", DIM)}>
                {missing || t("billing.rent.footnote")}
              </span>
            </div>
          </div>
        )}
      </div>

      {!isLoading && (
        <div className="fixed inset-x-0 bottom-0 z-40 flex items-center gap-3.5 border-t border-[var(--vx-border)] bg-[rgba(10,11,13,0.94)] px-4 py-3.5 backdrop-blur-md lg:hidden">
          <span className="flex flex-col">
            <span className={cn("text-[11px]", VX_MUTED)}>
              {hourly ? t("billing.rent.per_day") : t("billing.rent.to_pay_for", { days: period })}
            </span>
            <span className="font-mono text-[20px] font-semibold">
              {money(hourly ? hourlyRate * 24 * count : total, currency)}
            </span>
          </span>
          <Btn
            tone="primary"
            className="h-11 flex-1 rounded-[10px] text-[15px]"
            onClick={handleCreate}
            disabled={ctaDisabled}
          >
            {createMutation.isPending ? t("billing.rent.creating") : t("billing.rent.pay")}
          </Btn>
        </div>
      )}
    </PageShell>
  );
}

function SummaryRow({
  label,
  value,
  note,
  tone,
}: {
  label: string;
  value: string;
  note?: string;
  tone?: "discount";
}) {
  return (
    <span className="flex items-baseline justify-between gap-3">
      <span className={cn("min-w-0 truncate", VX_MUTED)}>
        {label}
        {note && <span className={cn("ml-1.5 font-mono text-[11px]", DIM)}>{note}</span>}
      </span>
      <span
        className={cn(
          "shrink-0 font-mono",
          tone === "discount" ? "text-[var(--vx-ok)]" : "text-[var(--vx-fg)]"
        )}
      >
        {value}
      </span>
    </span>
  );
}

function OptionToggle({
  icon,
  title,
  note,
  checked,
  onChange,
}: {
  icon: string;
  title: string;
  note: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onChange(!checked)}
      aria-pressed={checked}
      className={cn(
        "flex items-start gap-3 rounded-[12px] border bg-[var(--vx-bg)] p-3.5 text-left transition-colors",
        checked
          ? "border-[var(--vx-fg-strong)]"
          : "border-[var(--vx-border)] hover:border-[var(--vx-border-strong)]"
      )}
    >
      <span
        className={cn(
          "flex h-8 w-8 shrink-0 items-center justify-center rounded-[8px]",
          checked ? "bg-[var(--vx-fg-strong)] text-[var(--vx-on-fill)]" : "bg-[var(--vx-tint)]"
        )}
      >
        <i className={cn(icon, "text-[16px]")} />
      </span>
      <span className="flex min-w-0 flex-col gap-0.5">
        <span className="text-[13px] font-medium">{title}</span>
        <span className={cn("text-[11.5px] leading-[1.45]", VX_MUTED)}>{note}</span>
      </span>
    </button>
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
