"use client";

import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Palette } from "lucide-react";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { fetchAdminAppearance, saveAdminAppearance } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { useT } from "@/hooks/use-translations";

type AppearanceTab = "landing" | "template";

// Подписи храним ключами: список читается на уровне модуля, и готовый текст
// застыл бы на языке, который стоял в момент загрузки страницы.
const TABS: { id: AppearanceTab; labelKey: string }[] = [
  { id: "landing", labelKey: "admin.appearance.tab.landing" },
  { id: "template", labelKey: "admin.appearance.tab.template" },
];

const LANDING_COLOR_FIELDS: { key: string; label: string; fallback: string }[] =
  [
    { key: "primary", label: "Primary", fallback: "#f97316" },
    { key: "secondary", label: "Secondary", fallback: "#1f2937" },
    { key: "accent", label: "Accent", fallback: "#fb923c" },
    { key: "background", label: "Background", fallback: "#0b0b0c" },
    { key: "foreground", label: "Foreground", fallback: "#f8fafc" },
    { key: "card", label: "Card", fallback: "#18181b" },
    { key: "muted", label: "Muted", fallback: "#27272a" },
    { key: "border", label: "Border", fallback: "#3f3f46" },
  ];

function parseColors(raw: string): Record<string, string> {
  const source = String(raw || "").trim();
  if (!source) return {};

  try {
    const parsed = JSON.parse(source) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {};
    }

    return Object.entries(parsed as Record<string, unknown>).reduce<
      Record<string, string>
    >((acc, [k, v]) => {
      if (typeof v === "string") {
        acc[k] = v;
      }
      return acc;
    }, {});
  } catch {
    return {};
  }
}

function normalizeHex(value: string, fallback: string): string {
  const v = String(value || "").trim();
  return /^#([\da-fA-F]{6})$/.test(v) ? v : fallback;
}

export function AppearancePageContent() {
  const t = useT();
  const queryClient = useQueryClient();
  const [tab, setTab] = useState<AppearanceTab>("landing");
  const [values, setValues] = useState<Record<string, string>>({});

  const landingColors = useMemo(
    () => parseColors(values["app.site.template.colors"] || ""),
    [values]
  );

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminAppearance,
    queryFn: fetchAdminAppearance,
    staleTime: 60 * 1000,
  });

  useEffect(() => {
    if (data?.values) {
      setValues(data.values);
    }
  }, [data]);

  const saveMutation = useMutation({
    mutationFn: saveAdminAppearance,
    onSuccess: () => {
      toast.success(t("admin.appearance.saved"));
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminAppearance });
      void queryClient.invalidateQueries({ queryKey: queryKeys.adminSettings });
    },
    onError: (err: Error) => toast.error(err.message || t("common.save_failed")),
  });

  const updateValue = (key: string, val: string) => {
    setValues((prev) => ({ ...prev, [key]: val }));
  };

  const updateLandingColor = (key: string, val: string) => {
    setValues((prev) => {
      const parsed = parseColors(prev["app.site.template.colors"] || "");
      parsed[key] = val;
      return {
        ...prev,
        "app.site.template.colors": JSON.stringify(parsed, null, 2),
      };
    });
  };

  const save = () => {
    saveMutation.mutate({
      default_template: values["app.site.default_template"] || "",
      template_colors: values["app.site.template.colors"] || "",
      template_blocks: values["app.site.template.blocks"] || "",
      template_user_menu_variant:
        values["app.site.template.user_menu_variant"] || "default",
    });
  };

  if (isLoading) {
    return (
      <PageShell title={t("admin.appearance.title")}>
        <div className="space-y-4">
          <Skeleton className="h-24 w-full rounded-2xl" />
          <Skeleton className="h-96 w-full rounded-2xl" />
        </div>
      </PageShell>
    );
  }

  return (
    <PageShell title={t("admin.appearance.title")}>
      <div className="space-y-6">
        <Card className="overflow-hidden border-primary/20 bg-gradient-to-br from-primary/10 via-card to-card">
          <CardContent className="flex items-center gap-4 p-6">
            <div className="flex size-12 items-center justify-center rounded-2xl bg-gradient-to-br from-primary to-primary/70 text-primary-foreground shadow-lg shadow-primary/25">
              <Palette className="size-6" />
            </div>
            <div className="min-w-0 flex-1">
              <h1 className="text-xl font-bold lg:text-2xl">
                {t("admin.appearance.title")}
              </h1>
              <p className="text-sm text-muted-foreground">
                {t("admin.appearance.subtitle")}
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="space-y-6 p-4">
            <div className="grid w-full max-w-xs grid-cols-1 gap-1 rounded-2xl border bg-muted p-1 text-sm">
              {/* Параметр назван item, а не t: имя t занято функцией перевода. */}
              {TABS.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => setTab(item.id)}
                  className={cn(
                    "inline-flex w-full items-center justify-center rounded-xl px-3 py-2 text-center leading-tight transition-all",
                    tab === item.id
                      ? "bg-background font-semibold shadow-sm"
                      : "text-muted-foreground hover:text-foreground"
                  )}
                >
                  {t(item.labelKey)}
                </button>
              ))}
            </div>

            {tab === "landing" && (
              <>
                <div className="rounded-2xl border bg-muted/50 p-4">
                  <div className="text-xs font-semibold">
                    {t("admin.appearance.palette.title")}
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {t("admin.appearance.palette.hint")}
                  </p>
                  <div className="mt-3 grid gap-3 md:grid-cols-2">
                    {LANDING_COLOR_FIELDS.map((field) => {
                      const value = landingColors[field.key] || "";
                      return (
                        <div
                          key={field.key}
                          className="rounded-xl border bg-card/60 p-3"
                        >
                          <div className="mb-2 text-xs font-medium">
                            {field.label}
                          </div>
                          <div className="flex items-center gap-2">
                            <input
                              type="color"
                              value={normalizeHex(value, field.fallback)}
                              onChange={(e) =>
                                updateLandingColor(field.key, e.target.value)
                              }
                              className="h-10 w-14 cursor-pointer rounded border bg-transparent p-1"
                            />
                            <Input
                              value={value}
                              onChange={(e) =>
                                updateLandingColor(field.key, e.target.value)
                              }
                              placeholder={field.fallback}
                            />
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>

                <div className="rounded-2xl border bg-muted/50 p-4">
                  <div className="text-xs font-semibold">
                    {t("admin.appearance.colors.title")}
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {t("admin.appearance.colors.hint")}
                  </p>
                  <textarea
                    value={values["app.site.template.colors"] || ""}
                    onChange={(e) =>
                      updateValue("app.site.template.colors", e.target.value)
                    }
                    rows={8}
                    className="mt-3 flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-sm"
                    placeholder='{"primary":"#3b82f6","secondary":"#1f2937"}'
                  />
                </div>

                <div className="rounded-2xl border bg-muted/50 p-4">
                  <div className="text-xs font-semibold">
                    {t("admin.appearance.blocks.title")}
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {t("admin.appearance.blocks.hint")}
                  </p>
                  <textarea
                    value={values["app.site.template.blocks"] || ""}
                    onChange={(e) =>
                      updateValue("app.site.template.blocks", e.target.value)
                    }
                    rows={10}
                    className="mt-3 flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-sm"
                    placeholder='{"hero":true,"features":true,"faq":true}'
                  />
                </div>
              </>
            )}

            {tab === "template" && (
              <div className="rounded-2xl border bg-muted/50 p-4">
                <div className="text-xs font-semibold">
                  {t("admin.appearance.template.title")}
                </div>
                <div className="mt-3 grid gap-4 md:grid-cols-2">
                  <div className="space-y-1">
                    <Label className="text-xs text-muted-foreground">
                      {t("admin.settings.project.default_template")}
                    </Label>
                    <Input
                      value={values["app.site.default_template"] || ""}
                      onChange={(e) =>
                        updateValue("app.site.default_template", e.target.value)
                      }
                      placeholder="default"
                    />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs text-muted-foreground">
                      {t("admin.appearance.user_menu_variant")}
                    </Label>
                    <select
                      value={
                        values["app.site.template.user_menu_variant"] ||
                        "default"
                      }
                      onChange={(e) =>
                        updateValue(
                          "app.site.template.user_menu_variant",
                          e.target.value
                        )
                      }
                      className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm"
                    >
                      <option value="default">
                        {t("admin.appearance.user_menu.default")}
                      </option>
                      <option value="screenshot">
                        {t("admin.appearance.user_menu.screenshot")}
                      </option>
                    </select>
                  </div>
                </div>
              </div>
            )}

            <Button
              type="button"
              onClick={save}
              disabled={saveMutation.isPending}
            >
              {saveMutation.isPending ? t("common.saving") : t("common.save")}
            </Button>
          </CardContent>
        </Card>
      </div>
    </PageShell>
  );
}
