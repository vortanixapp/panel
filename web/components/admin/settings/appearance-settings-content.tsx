"use client";

import Image from "next/image";
import Link from "next/link";
import { useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ImageIcon, LayoutGrid, RotateCcw, Trash2, Upload } from "lucide-react";
import { toast } from "sonner";

import {
  FieldGrid,
  Segmented,
  SelectField,
  SettingsCard,
  TextAreaField,
  TextField,
  ToggleRow,
} from "@/components/admin/settings/settings-ui";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { useBrand } from "@/context/brand-provider";
import {
  APPEARANCE_FONTS,
  DEFAULT_PANEL_ACCENT,
  isHexColor,
  readableTextOn,
  type AppearanceAccent,
} from "@/lib/appearance";
import {
  brandingUploadUrl,
  deleteAdminAppearanceAsset,
  fetchAdminAppearance,
  saveAdminAppearance,
  uploadAdminAppearanceAsset,
  type AdminAppearance,
  type AppearanceAssetKind,
} from "@/lib/api";
import { cn } from "@/lib/utils";
import { useT } from "@/hooks/use-translations";

type Tab = "colors" | "brand" | "landing" | "css";

const QUERY_KEY = ["admin-appearance"] as const;

const LINK_FIELDS = [
  { key: "telegram", label: "Telegram", labelKey: "", placeholder: "https://t.me/example" },
  { key: "discord", label: "Discord", labelKey: "", placeholder: "https://discord.gg/example" },
  { key: "vk", label: "VK", labelKey: "", placeholder: "https://vk.com/example" },
  { key: "support", label: "", labelKey: "admin.appearance.links.support", placeholder: "https://example.com/support" },
  { key: "email", label: "", labelKey: "admin.appearance.links.email", placeholder: "support@example.com" },
  { key: "offer", label: "", labelKey: "admin.appearance.links.offer", placeholder: "https://example.com/offer" },
  { key: "privacy", label: "", labelKey: "admin.appearance.links.privacy", placeholder: "https://example.com/privacy" },
] as const;

const RADIUS_PREVIEW: Record<AdminAppearance["radius"], string> = {
  strict: "6px",
  standard: "10px",
  soft: "16px",
};

function fontStyle(id: string): React.CSSProperties | undefined {
  const variable = APPEARANCE_FONTS.find((f) => f.id === id)?.variable;
  return variable ? { fontFamily: `var(${variable})` } : undefined;
}

export function AppearanceSettingsContent() {
  const t = useT();
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: QUERY_KEY, queryFn: fetchAdminAppearance });

  return (
    <PageShell variant="admin">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">
          {t("admin.appearance.title")}
        </h1>
        <p className="text-sm text-muted-foreground">
          {t("admin.appearance.subtitle")}
        </p>
      </div>
      {query.isError ? (
        <p className="text-sm text-destructive">{(query.error as Error).message}</p>
      ) : query.data ? (
        <AppearanceEditor
          initial={query.data.appearance}
          onSaved={(appearance) => queryClient.setQueryData(QUERY_KEY, { appearance })}
        />
      ) : (
        <Skeleton className="h-96 w-full rounded-2xl" />
      )}
    </PageShell>
  );
}

function AppearanceEditor({
  initial,
  onSaved,
}: {
  initial: AdminAppearance;
  onSaved: (appearance: AdminAppearance) => void;
}) {
  const t = useT();
  const { refresh } = useBrand();
  const [tab, setTab] = useState<Tab>("colors");
  const [draft, setDraft] = useState<AdminAppearance>(initial);
  const [baseline, setBaseline] = useState<AdminAppearance>(initial);

  const dirty = useMemo(
    () => JSON.stringify(draft) !== JSON.stringify(baseline),
    [draft, baseline]
  );
  const invalidColor = Object.values(draft.accent).some(
    (color) => color !== "" && !isHexColor(color)
  );

  const update = (patch: Partial<AdminAppearance>) =>
    setDraft((d) => ({ ...d, ...patch }));
  const setAccent = (key: keyof AppearanceAccent, value: string) =>
    setDraft((d) => ({ ...d, accent: { ...d.accent, [key]: value } }));
  const setLink = (key: string, value: string) =>
    setDraft((d) => ({ ...d, links: { ...d.links, [key]: value } }));

  const save = useMutation({
    mutationFn: () => saveAdminAppearance(draft),
    onSuccess: async ({ appearance }) => {
      setDraft(appearance);
      setBaseline(appearance);
      onSaved(appearance);
      await refresh();
      toast.success(t("admin.appearance.saved"));
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const syncAssets = async (appearance: AdminAppearance, message: string) => {
    const assets = {
      logo: appearance.logo,
      logo_dark: appearance.logo_dark,
      icon: appearance.icon,
    };
    setDraft((d) => ({ ...d, ...assets }));
    setBaseline((b) => ({ ...b, ...assets }));
    await refresh();
    toast.success(message);
  };

  const upload = useMutation({
    mutationFn: ({ kind, file }: { kind: AppearanceAssetKind; file: File }) =>
      uploadAdminAppearanceAsset(kind, file),
    onSuccess: ({ appearance }) =>
      syncAssets(appearance, t("admin.appearance.asset_saved")),
    onError: (e: Error) => toast.error(e.message),
  });

  const removeAsset = useMutation({
    mutationFn: (kind: AppearanceAssetKind) => deleteAdminAppearanceAsset(kind),
    onSuccess: ({ appearance }) =>
      syncAssets(appearance, t("admin.appearance.asset_removed")),
    onError: (e: Error) => toast.error(e.message),
  });

  const assetBusy = upload.isPending || removeAsset.isPending;
  const panelColor = isHexColor(draft.accent.panel)
    ? draft.accent.panel
    : DEFAULT_PANEL_ACCENT;
  const landingColor = isHexColor(draft.accent.landing)
    ? draft.accent.landing
    : panelColor;
  const panelDarkOn = draft.accent.panel_dark !== "";
  const landingOn = draft.accent.landing !== "";
  const landingDarkOn = draft.accent.landing_dark !== "";

  const fontOptions = APPEARANCE_FONTS.map((f) => ({
    value: f.id,
    label: f.id === "default" ? t("admin.appearance.font.default") : f.label,
  }));

  return (
    <div className="flex flex-col gap-4">
      <Segmented<Tab>
        className="w-fit max-w-full"
        items={[
          { id: "colors", label: t("admin.appearance.tab.colors") },
          { id: "brand", label: t("admin.appearance.tab.brand") },
          { id: "landing", label: t("admin.appearance.tab.landing") },
          { id: "css", label: t("admin.appearance.tab.css") },
        ]}
        value={tab}
        onChange={setTab}
      />

      {tab === "colors" && (
        <>
          <SettingsCard
            title={t("admin.appearance.accent.panel_title")}
            description={t("admin.appearance.accent.panel_hint")}
          >
            <div className="grid gap-4">
              <FieldGrid>
                <ColorField
                  label={
                    panelDarkOn
                      ? t("admin.appearance.accent.light")
                      : t("admin.appearance.accent.color")
                  }
                  value={draft.accent.panel}
                  fallback={DEFAULT_PANEL_ACCENT}
                  onChange={(v) => setAccent("panel", v)}
                />
                {panelDarkOn && (
                  <ColorField
                    label={t("admin.appearance.accent.dark")}
                    value={draft.accent.panel_dark}
                    fallback={panelColor}
                    onChange={(v) => setAccent("panel_dark", v)}
                  />
                )}
              </FieldGrid>
              <ToggleRow
                label={t("admin.appearance.accent.separate_dark")}
                hint={t("admin.appearance.accent.separate_dark_hint")}
                checked={panelDarkOn}
                onCheckedChange={(on) => setAccent("panel_dark", on ? panelColor : "")}
              />
              <PreviewPair
                light={panelColor}
                dark={isHexColor(draft.accent.panel_dark) ? draft.accent.panel_dark : panelColor}
              />
            </div>
          </SettingsCard>

          <SettingsCard
            title={t("admin.appearance.accent.landing_title")}
            description={t("admin.appearance.accent.landing_hint")}
          >
            <div className="grid gap-4">
              <ToggleRow
                label={t("admin.appearance.accent.landing_custom")}
                hint={t("admin.appearance.accent.landing_custom_hint")}
                checked={landingOn}
                onCheckedChange={(on) =>
                  setDraft((d) => ({
                    ...d,
                    accent: {
                      ...d.accent,
                      landing: on ? panelColor : "",
                      landing_dark: on ? d.accent.landing_dark : "",
                    },
                  }))
                }
              />
              {landingOn && (
                <>
                  <FieldGrid>
                    <ColorField
                      label={
                        landingDarkOn
                          ? t("admin.appearance.accent.light")
                          : t("admin.appearance.accent.color")
                      }
                      value={draft.accent.landing}
                      fallback={panelColor}
                      onChange={(v) => setAccent("landing", v)}
                    />
                    {landingDarkOn && (
                      <ColorField
                        label={t("admin.appearance.accent.dark")}
                        value={draft.accent.landing_dark}
                        fallback={landingColor}
                        onChange={(v) => setAccent("landing_dark", v)}
                      />
                    )}
                  </FieldGrid>
                  <ToggleRow
                    label={t("admin.appearance.accent.separate_dark")}
                    hint={t("admin.appearance.accent.separate_dark_hint")}
                    checked={landingDarkOn}
                    onCheckedChange={(on) =>
                      setAccent("landing_dark", on ? landingColor : "")
                    }
                  />
                  <PreviewPair
                    light={landingColor}
                    dark={
                      isHexColor(draft.accent.landing_dark)
                        ? draft.accent.landing_dark
                        : landingColor
                    }
                  />
                </>
              )}
            </div>
          </SettingsCard>

          <SettingsCard
            title={t("admin.appearance.theme.title")}
            description={t("admin.appearance.theme.hint")}
          >
            <div className="grid gap-4">
              <div className="grid gap-2">
                <span className="text-xs text-muted-foreground">
                  {t("admin.appearance.theme.default")}
                </span>
                <Segmented<AdminAppearance["theme"]>
                  className="w-fit max-w-full"
                  items={[
                    { id: "dark", label: t("admin.appearance.theme.dark") },
                    { id: "light", label: t("admin.appearance.theme.light") },
                    { id: "system", label: t("admin.appearance.theme.system") },
                  ]}
                  value={draft.theme}
                  onChange={(theme) => update({ theme })}
                />
              </div>
              <ToggleRow
                label={t("admin.appearance.theme.locked")}
                hint={t("admin.appearance.theme.locked_hint")}
                checked={draft.theme_locked}
                onCheckedChange={(theme_locked) => update({ theme_locked })}
              />
              <div className="grid gap-2">
                <span className="text-xs text-muted-foreground">
                  {t("admin.appearance.user_menu")}
                </span>
                <Segmented<AdminAppearance["user_menu"]>
                  className="w-fit max-w-full"
                  items={[
                    { id: "default", label: t("admin.appearance.user_menu.default") },
                    { id: "screenshot", label: t("admin.appearance.user_menu.screenshot") },
                  ]}
                  value={draft.user_menu}
                  onChange={(user_menu) => update({ user_menu })}
                />
              </div>
            </div>
          </SettingsCard>

          <SettingsCard
            title={t("admin.appearance.radius.title")}
            description={t("admin.appearance.radius.hint")}
          >
            <div className="flex flex-wrap items-center gap-5">
              <Segmented<AdminAppearance["radius"]>
                className="w-fit max-w-full"
                items={[
                  { id: "strict", label: t("admin.appearance.radius.strict") },
                  { id: "standard", label: t("admin.appearance.radius.standard") },
                  { id: "soft", label: t("admin.appearance.radius.soft") },
                ]}
                value={draft.radius}
                onChange={(radius) => update({ radius })}
              />
              <div className="flex items-center gap-3">
                {(["strict", "standard", "soft"] as const).map((radius) => (
                  <div
                    key={radius}
                    className={cn(
                      "h-10 w-16 border-2 transition-colors",
                      draft.radius === radius
                        ? "border-primary bg-primary/10"
                        : "border-border bg-muted/40"
                    )}
                    style={{ borderRadius: RADIUS_PREVIEW[radius] }}
                  />
                ))}
              </div>
            </div>
          </SettingsCard>

          <SettingsCard
            title={t("admin.appearance.font.title")}
            description={t("admin.appearance.font.hint")}
          >
            <FieldGrid>
              {(["font_panel", "font_landing"] as const).map((field) => (
                <div key={field} className="flex flex-col gap-2">
                  <SelectField
                    label={
                      field === "font_panel"
                        ? t("admin.appearance.font.panel")
                        : t("admin.appearance.font.landing")
                    }
                    value={draft[field]}
                    options={fontOptions}
                    onChange={(v) =>
                      update({ [field]: v } as Partial<AdminAppearance>)
                    }
                  />
                  <p
                    className="rounded-lg border bg-muted/30 px-3 py-2.5 text-[15px]"
                    style={fontStyle(draft[field])}
                  >
                    {t("admin.appearance.font.sample")}
                  </p>
                </div>
              ))}
            </FieldGrid>
          </SettingsCard>
        </>
      )}

      {tab === "brand" && (
        <SettingsCard
          title={t("admin.appearance.brand.title")}
          description={t("admin.appearance.brand.hint")}
        >
          <div className="grid gap-5">
            <FieldGrid>
              <TextField
                label={t("admin.settings.project.name")}
                value={draft.name}
                onChange={(name) => update({ name })}
                placeholder="Vortanix"
              />
            </FieldGrid>
            <div className="grid gap-5 md:grid-cols-3">
              <AssetField
                label={t("admin.appearance.brand.logo")}
                hint={t("admin.appearance.brand.logo_hint")}
                path={draft.logo}
                busy={assetBusy}
                onUpload={(file) => upload.mutate({ kind: "logo", file })}
                onRemove={() => removeAsset.mutate("logo")}
              />
              <AssetField
                dark
                label={t("admin.appearance.brand.logo_dark")}
                hint={t("admin.appearance.brand.logo_dark_hint")}
                path={draft.logo_dark}
                busy={assetBusy}
                onUpload={(file) => upload.mutate({ kind: "logo_dark", file })}
                onRemove={() => removeAsset.mutate("logo_dark")}
              />
              <AssetField
                label={t("admin.appearance.brand.icon")}
                hint={t("admin.appearance.brand.icon_hint")}
                path={draft.icon}
                busy={assetBusy}
                onUpload={(file) => upload.mutate({ kind: "icon", file })}
                onRemove={() => removeAsset.mutate("icon")}
              />
            </div>
          </div>
        </SettingsCard>
      )}

      {tab === "landing" && (
        <>
          <SettingsCard
            title={t("admin.appearance.template.title")}
            description={t("admin.appearance.template.hint")}
          >
            <Button asChild variant="outline" className="w-fit">
              <Link href="/admin/template">
                <LayoutGrid className="size-4" />
                {t("admin.appearance.template.open")}
              </Link>
            </Button>
          </SettingsCard>

          <SettingsCard
            title={t("admin.appearance.links.title")}
            description={t("admin.appearance.links.hint")}
          >
            <FieldGrid>
              {LINK_FIELDS.map((field) => (
                <TextField
                  key={field.key}
                  mono={field.key !== "email"}
                  label={field.label || t(field.labelKey)}
                  value={draft.links[field.key] ?? ""}
                  placeholder={field.placeholder}
                  onChange={(v) => setLink(field.key, v)}
                />
              ))}
            </FieldGrid>
          </SettingsCard>
        </>
      )}

      {tab === "css" && (
        <SettingsCard
          title={t("admin.appearance.css.title")}
          description={t("admin.appearance.css.hint")}
        >
          <div className="grid gap-3">
            <p className="rounded-lg border border-amber-500/30 bg-amber-500/5 px-3.5 py-2.5 text-[13px] text-amber-600 dark:text-amber-500">
              {t("admin.appearance.css.warning")}
            </p>
            <TextAreaField
              label="CSS"
              mono
              rows={16}
              value={draft.custom_css}
              onChange={(custom_css) => update({ custom_css })}
              placeholder={".font-landing h1 {\n  letter-spacing: -0.02em;\n}"}
            />
          </div>
        </SettingsCard>
      )}

      <div className="sticky bottom-4 z-10 flex flex-wrap items-center justify-end gap-3 rounded-xl border bg-card/95 px-4 py-3 shadow-sm backdrop-blur">
        <span
          className={cn(
            "me-auto text-sm",
            invalidColor ? "text-destructive" : "text-muted-foreground"
          )}
        >
          {invalidColor
            ? t("admin.appearance.invalid_color")
            : dirty
              ? t("admin.appearance.unsaved")
              : ""}
        </span>
        <Button
          variant="outline"
          disabled={!dirty || save.isPending}
          onClick={() => setDraft(baseline)}
        >
          {t("admin.appearance.discard")}
        </Button>
        <Button
          disabled={!dirty || invalidColor || save.isPending}
          onClick={() => save.mutate()}
        >
          {save.isPending ? t("common.saving") : t("common.save")}
        </Button>
      </div>
    </div>
  );
}

function ColorField({
  label,
  value,
  fallback,
  onChange,
}: {
  label: string;
  value: string;
  fallback: string;
  onChange: (value: string) => void;
}) {
  const t = useT();
  const current = isHexColor(value) ? value : fallback;
  const invalid = value !== "" && !isHexColor(value);

  return (
    <div className="flex flex-col gap-[7px]">
      <span className="text-xs text-muted-foreground">{label}</span>
      <div className="flex items-center gap-2">
        <label
          className="relative size-[38px] shrink-0 cursor-pointer overflow-hidden rounded-lg border"
          style={{ background: current }}
        >
          <input
            type="color"
            value={current}
            aria-label={label}
            onChange={(e) => onChange(e.target.value)}
            className="absolute inset-0 size-full cursor-pointer opacity-0"
          />
        </label>
        <Input
          value={value}
          placeholder={fallback}
          onChange={(e) => onChange(e.target.value.trim())}
          className={cn(
            "h-[38px] rounded-lg font-mono text-[13px]",
            invalid && "border-destructive focus-visible:border-destructive"
          )}
        />
        {value && (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-[38px] shrink-0"
            title={t("admin.appearance.reset")}
            aria-label={t("admin.appearance.reset")}
            onClick={() => onChange("")}
          >
            <RotateCcw className="size-4" />
          </Button>
        )}
      </div>
    </div>
  );
}

function PreviewPair({ light, dark }: { light: string; dark: string }) {
  return (
    <div className="grid gap-3 lg:grid-cols-2">
      <AccentPreview color={light} tone="light" />
      <AccentPreview color={dark} tone="dark" />
    </div>
  );
}

function AccentPreview({ color, tone }: { color: string; tone: "light" | "dark" }) {
  const t = useT();
  const fg = readableTextOn(color);
  const dark = tone === "dark";

  return (
    <div
      className={cn(
        "flex flex-wrap items-center gap-3 rounded-xl border px-4 py-3.5",
        dark
          ? "border-[#1c1d1f] bg-[#0a0b0d] text-[#e8e9eb]"
          : "border-[#e4e6e9] bg-white text-[#16171a]"
      )}
    >
      <span className="text-[11px] tracking-wide uppercase opacity-60">
        {dark ? t("admin.appearance.accent.dark") : t("admin.appearance.accent.light")}
      </span>
      <span
        className="rounded-md px-3.5 py-1.5 text-[13px] font-medium"
        style={{ background: color, color: fg }}
      >
        {t("admin.appearance.preview.button")}
      </span>
      <span
        className="text-[13px] font-medium underline underline-offset-4"
        style={{ color }}
      >
        {t("admin.appearance.preview.link")}
      </span>
      <span
        className="rounded-full border px-2.5 py-0.5 text-xs"
        style={{ borderColor: color, color }}
      >
        {t("admin.appearance.preview.badge")}
      </span>
      <span
        className="relative inline-flex h-5 w-9 shrink-0 rounded-full"
        style={{ background: color }}
      >
        <span
          className="absolute top-0.5 right-0.5 size-4 rounded-full"
          style={{ background: fg }}
        />
      </span>
    </div>
  );
}

function AssetField({
  label,
  hint,
  path,
  busy,
  dark = false,
  onUpload,
  onRemove,
}: {
  label: string;
  hint: string;
  path: string;
  busy: boolean;
  dark?: boolean;
  onUpload: (file: File) => void;
  onRemove: () => void;
}) {
  const t = useT();
  const inputRef = useRef<HTMLInputElement>(null);
  const url = path ? brandingUploadUrl(path) : "";

  return (
    <div className="flex flex-col gap-2.5">
      <span className="text-xs text-muted-foreground">{label}</span>
      <div
        className={cn(
          "flex h-24 items-center justify-center overflow-hidden rounded-xl border border-dashed p-3",
          dark ? "bg-[#0a0b0d]" : "bg-white"
        )}
      >
        {url ? (
          <Image
            src={url}
            alt={label}
            width={240}
            height={72}
            unoptimized
            className="h-auto max-h-full w-auto max-w-full object-contain"
          />
        ) : (
          <ImageIcon className="size-5 text-muted-foreground" />
        )}
      </div>
      <div className="flex flex-wrap gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={busy}
          onClick={() => inputRef.current?.click()}
        >
          <Upload className="size-3.5" />
          {url ? t("admin.appearance.brand.replace") : t("admin.appearance.brand.choose")}
        </Button>
        {url && (
          <Button type="button" variant="ghost" size="sm" disabled={busy} onClick={onRemove}>
            <Trash2 className="size-3.5" />
            {t("admin.appearance.brand.remove")}
          </Button>
        )}
      </div>
      <p className="text-xs text-muted-foreground">{hint}</p>
      <input
        ref={inputRef}
        type="file"
        accept=".png,.jpg,.jpeg,.webp,.svg,.ico"
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0];
          e.target.value = "";
          if (file) onUpload(file);
        }}
      />
    </div>
  );
}
