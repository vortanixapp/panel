"use client";

import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, RotateCcw, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { confirmAction } from "@/components/action-dialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { useT } from "@/hooks/use-translations";
import { cn } from "@/lib/utils";
import {
  updateAdminGame,
  type AdminGameRuntime,
  type AdminGameRuntimeVersion,
} from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

export function runtimeTabLabel(runtime: AdminGameRuntime) {
  return runtime.kind === "php" ? "PHP" : "Java";
}

export function GameRuntimeTab({
  gameId,
  runtime,
}: {
  gameId: string;
  runtime: AdminGameRuntime;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [rows, setRows] = useState<AdminGameRuntimeVersion[]>(runtime.versions);

  useEffect(() => {
    setRows(runtime.versions);
  }, [runtime.versions]);

  const saveMut = useMutation({
    mutationFn: (list: AdminGameRuntimeVersion[]) =>
      updateAdminGame(gameId, { runtime_versions: list }),
    onSuccess: () => {
      toast.success(t("common.saved"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminGameEdit(gameId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminGame(gameId) });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const setRow = (index: number, patch: Partial<AdminGameRuntimeVersion>) =>
    setRows((prev) => prev.map((row, i) => (i === index ? { ...row, ...patch } : row)));

  const addRow = () =>
    setRows((prev) => [...prev, { version: "", url: "", enabled: true }]);

  const removeRow = (index: number) =>
    setRows((prev) => prev.filter((_, i) => i !== index));

  const onSave = () => {
    const list = rows
      .map((row) => ({
        version: row.version.trim(),
        url: row.url.trim(),
        enabled: row.enabled,
      }))
      .filter((row) => row.version !== "");
    if (list.length === 0) {
      toast.error(t("admin.games.runtime.need_version"));
      return;
    }
    saveMut.mutate(list);
  };

  const onReset = async () => {
    const ok = await confirmAction(
      t("admin.games.runtime.reset_text", { list: runtime.defaults.join(", ") }),
      {
        title: t("admin.games.runtime.reset_title"),
        confirmText: t("admin.games.runtime.reset_ok"),
      }
    );
    if (!ok) return;
    saveMut.mutate([]);
  };

  const full = rows.length >= runtime.version_limit;
  const cols = runtime.custom_source
    ? "md:grid-cols-[minmax(110px,150px)_1fr_120px]"
    : "md:grid-cols-[minmax(110px,220px)_120px]";

  return (
    <div className="grid gap-4 lg:grid-cols-3">
      <Card className="lg:col-span-2">
        <CardHeader className="flex flex-row items-center justify-between gap-3 space-y-0">
          <CardTitle className="text-base">
            {t("admin.games.runtime.list_title", { kind: runtimeTabLabel(runtime) })}
          </CardTitle>
          <Button type="button" variant="outline" size="sm" onClick={addRow} disabled={full}>
            <Plus className="mr-1.5 size-4" />
            {t("admin.games.runtime.add")}
          </Button>
        </CardHeader>
        <CardContent className="space-y-4">
          {rows.length === 0 && (
            <p className="text-sm text-muted-foreground">
              {t("admin.games.runtime.empty")}
            </p>
          )}

          {rows.length > 0 && (
            <div className="rounded-lg border border-border">
              <div className={cn("hidden gap-3 border-b border-border px-3 py-2 text-xs text-muted-foreground md:grid", cols)}>
                <span>{t("admin.games.runtime.version")}</span>
                {runtime.custom_source && <span>{t("admin.games.runtime.url")}</span>}
                <span>{t("admin.games.runtime.enabled")}</span>
              </div>

              <div className="divide-y divide-border">
                {rows.map((row, index) => (
                  <div key={index} className={cn("grid gap-3 p-3 md:items-center", cols)}>
                    <div className="space-y-1.5">
                      <Label htmlFor={`runtime-version-${index}`} className="md:hidden">
                        {t("admin.games.runtime.version")}
                      </Label>
                      <Input
                        id={`runtime-version-${index}`}
                        value={row.version}
                        placeholder={runtime.defaults[0] ?? ""}
                        onChange={(e) => setRow(index, { version: e.target.value })}
                      />
                    </div>

                    {runtime.custom_source && (
                      <div className="space-y-1.5">
                        <Label htmlFor={`runtime-url-${index}`} className="md:hidden">
                          {t("admin.games.runtime.url")}
                        </Label>
                        <Input
                          id={`runtime-url-${index}`}
                          value={row.url}
                          placeholder={t("admin.games.runtime.url_placeholder")}
                          onChange={(e) => setRow(index, { url: e.target.value })}
                        />
                      </div>
                    )}

                    <div className="flex items-center justify-between gap-3">
                      <div className="flex items-center gap-2 text-sm">
                        <Switch
                          checked={row.enabled}
                          aria-label={t("admin.games.runtime.enabled")}
                          onCheckedChange={(on) => setRow(index, { enabled: on })}
                        />
                        <span className="md:hidden">{t("admin.games.runtime.enabled")}</span>
                      </div>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        aria-label={t("common.delete")}
                        onClick={() => removeRow(index)}
                      >
                        <Trash2 className="size-4 text-destructive" />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="flex flex-wrap items-center gap-2">
            <Button type="button" onClick={onSave} disabled={saveMut.isPending}>
              {saveMut.isPending ? t("common.saving") : t("common.save")}
            </Button>
            {runtime.customized && (
              <Button
                type="button"
                variant="outline"
                onClick={onReset}
                disabled={saveMut.isPending}
              >
                <RotateCcw className="mr-1.5 size-4" />
                {t("admin.games.runtime.reset")}
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("admin.games.runtime.help_title")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-[13px] leading-relaxed text-muted-foreground">
          <p>{t("admin.games.runtime.help_list")}</p>
          <p>
            {t("admin.games.runtime.help_default", {
              list: runtime.defaults.join(", "),
            })}
          </p>
          {runtime.custom_source && (
            <>
              <p>{t("admin.games.runtime.help_url")}</p>
              <p>{t("admin.games.runtime.help_restart")}</p>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
