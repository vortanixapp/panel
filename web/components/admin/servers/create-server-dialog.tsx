"use client";

import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  createAdminServer,
  fetchAdminGames,
  fetchAdminTariffs,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useAdminLocations } from "@/hooks/use-queries";
import { useT } from "@/hooks/use-translations";

const SELECT_CLASS =
  "flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm";

export function CreateServerDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();

  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [gameId, setGameId] = useState("");
  const [locationId, setLocationId] = useState("");
  const [tariffId, setTariffId] = useState("");
  const [period, setPeriod] = useState(30);
  const [error, setError] = useState("");

  const locationsQuery = useAdminLocations();
  const gamesQuery = useQuery({
    queryKey: queryKeys.adminGames,
    queryFn: fetchAdminGames,
    enabled: open,
  });

  const locations = useMemo(
    () => locationsQuery.data?.locations ?? [],
    [locationsQuery.data]
  );
  const games = useMemo(
    () => (gamesQuery.data?.games ?? []).filter((g) => g.is_active),
    [gamesQuery.data]
  );

  const tariffsQuery = useQuery({
    queryKey: ["admin-servers", "create-tariffs", gameId, locationId],
    queryFn: () =>
      fetchAdminTariffs(1, {
        game_id: gameId,
        location_id: locationId,
        per_page: 100,
      }),
    enabled: open && !!gameId,
  });
  const tariffs = tariffsQuery.data?.tariffs.data ?? [];

  useEffect(() => {
    if (!open) return;
    if (!gameId && games[0]) setGameId(games[0].id);
    if (!locationId && locations[0]) setLocationId(locations[0].id);
  }, [open, games, locations, gameId, locationId]);

  useEffect(() => {
    setTariffId("");
  }, [gameId, locationId]);

  const createMutation = useMutation({
    mutationFn: () =>
      createAdminServer({
        email: email.trim(),
        node_id: locationId,
        game_id: gameId,
        tariff_id: tariffId || undefined,
        name: name.trim(),
        period,
      }),
    onSuccess: (res) => {
      toast.success(t("servers.admin.create.created", { email: res.owner_email }));
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminServers });
      setName("");
      setEmail("");
      setTariffId("");
      setError("");
      onOpenChange(false);
    },
    onError: (err) =>
      setError(err instanceof Error ? err.message : t("common.error")),
  });

  function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (!email.trim() || !name.trim() || !gameId || !locationId) {
      setError(t("servers.admin.create.fill_required"));
      return;
    }
    createMutation.mutate();
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("servers.admin.create.title")}</DialogTitle>
          <DialogDescription>{t("servers.admin.create.description")}</DialogDescription>
        </DialogHeader>

        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="admin-create-owner">
              {t("servers.admin.create.owner")}
            </Label>
            <Input
              id="admin-create-owner"
              type="email"
              placeholder="client@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="admin-create-name">{t("common.name")}</Label>
            <Input
              id="admin-create-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="admin-create-game">{t("common.game")}</Label>
              <select
                id="admin-create-game"
                className={SELECT_CLASS}
                value={gameId}
                onChange={(e) => setGameId(e.target.value)}
                required
              >
                {games.length === 0 ? (
                  <option value="">{t("servers.admin.create.no_games")}</option>
                ) : (
                  games.map((g) => (
                    <option key={g.id} value={g.id}>
                      {g.name}
                    </option>
                  ))
                )}
              </select>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="admin-create-location">
                {t("servers.admin.list.col_location")}
              </Label>
              <select
                id="admin-create-location"
                className={SELECT_CLASS}
                value={locationId}
                onChange={(e) => setLocationId(e.target.value)}
                required
              >
                {locations.length === 0 ? (
                  <option value="">{t("servers.admin.no_nodes")}</option>
                ) : (
                  locations.map((l) => (
                    <option key={l.id} value={l.id}>
                      {l.name}
                    </option>
                  ))
                )}
              </select>
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="admin-create-tariff">
                {t("servers.admin.list.col_tariff")}
              </Label>
              <select
                id="admin-create-tariff"
                className={SELECT_CLASS}
                value={tariffId}
                onChange={(e) => setTariffId(e.target.value)}
              >
                <option value="">{t("servers.admin.create.tariff_default")}</option>
                {tariffs.map((tariff) => (
                  <option key={tariff.id} value={tariff.id}>
                    {tariff.name}
                  </option>
                ))}
              </select>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="admin-create-period">
                {t("servers.admin.create.period")}
              </Label>
              <Input
                id="admin-create-period"
                type="number"
                min={1}
                max={365}
                value={period}
                onChange={(e) => setPeriod(Number(e.target.value) || 30)}
              />
            </div>
          </div>

          <p className="text-xs text-muted-foreground">
            {t("servers.admin.create.free_hint")}
          </p>

          {error && <p className="text-sm text-destructive">{error}</p>}

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? t("common.saving") : t("common.create")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
