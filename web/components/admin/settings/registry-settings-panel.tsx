"use client";

import { useMemo, useState } from "react";
import type { ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { confirmAction } from "@/components/action-dialog";
import { PageShell } from "@/components/layout/page-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { useT } from "@/hooks/use-translations";
import { ApiError, fetchAdminSettingsRegistry, saveAdminSettingsRegistry } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import {
  FieldGrid,
  SelectField,
  SettingsCard,
  SettingsSaveBar,
  ToggleRow,
} from "./settings-ui";
import type { RegistryGroup, RegistryItem } from "./types";

type Translate = ReturnType<typeof useT>;

type Props = {
  group: RegistryGroup;
  header: ReactNode;
  draft: Record<string, string>;
  onDraftChange: (next: Record<string, string>) => void;
};

function itemLabel(t: Translate, item: RegistryItem) {
  const base = t(`admin.settings.reg.${item.key}`);
  if (base === `admin.settings.reg.${item.key}`) return item.key;
  return base;
}

function itemHint(t: Translate, item: RegistryItem) {
  const key = `admin.settings.reg.${item.key}.hint`;
  const text = t(key);
  return text === key ? "" : text;
}

function unitLabel(t: Translate, unit: string) {
  return unit ? t(`admin.settings.reg.unit.${unit}`) : "";
}

function parseList(value: string) {
  return value
    .split(/[\s,;]+/)
    .filter(Boolean)
    .map((part) => Number(part));
}

function validate(t: Translate, item: RegistryItem, value: string): string {
  const text = value.trim();
  if (item.kind === "int") {
    if (!/^-?\d+$/.test(text)) return t("admin.settings.reg.error.number");
    const n = Number(text);
    if (n < item.min || n > item.max) {
      return t("admin.settings.reg.error.range", { min: item.min, max: item.max });
    }
  }
  if (item.kind === "intlist") {
    const list = parseList(text);
    if (list.length === 0 || list.some((n) => !Number.isInteger(n))) {
      return t("admin.settings.reg.error.list");
    }
    if (list.some((n) => n < item.min || n > item.max)) {
      return t("admin.settings.reg.error.range", { min: item.min, max: item.max });
    }
  }
  if (item.kind === "string" && text.length > (item.max > 0 ? item.max : 500)) {
    return t("admin.settings.reg.error.length", { max: item.max > 0 ? item.max : 500 });
  }
  return "";
}

function RegistryField({
  item,
  value,
  error,
  onChange,
  t,
}: {
  item: RegistryItem;
  value: string;
  error: string;
  onChange: (next: string) => void;
  t: Translate;
}) {
  const label = itemLabel(t, item);
  const hint = itemHint(t, item);
  const unit = unitLabel(t, item.unit);
  const modified = value !== item.default;

  if (item.kind === "bool") {
    return (
      <ToggleRow
        label={label}
        hint={hint || undefined}
        checked={value === "1"}
        onCheckedChange={(checked) => onChange(checked ? "1" : "0")}
        accent={modified}
      />
    );
  }

  if (item.kind === "enum") {
    return (
      <SelectField
        label={label}
        value={value}
        onChange={onChange}
        options={(item.options ?? []).map((option) => ({
          value: option,
          label: t(`admin.settings.reg.${item.key}.option.${option}`) === `admin.settings.reg.${item.key}.option.${option}`
            ? option
            : t(`admin.settings.reg.${item.key}.option.${option}`),
        }))}
      />
    );
  }

  const range =
    item.kind === "int"
      ? t("admin.settings.reg.range", {
          min: item.min,
          max: item.max,
          unit,
          default: item.default,
        })
      : t("admin.settings.reg.default", { default: item.default });

  return (
    <div className="flex flex-col gap-[7px]">
      <Label
        className={cn(
          "text-xs font-normal",
          modified ? "text-amber-500" : "text-muted-foreground"
        )}
      >
        {label}
        {unit ? `, ${unit}` : ""}
      </Label>
      <div className="flex items-center gap-2">
        <Input
          value={value}
          onChange={(e) => onChange(e.target.value)}
          inputMode={item.kind === "int" ? "numeric" : undefined}
          autoComplete="off"
          aria-invalid={error ? true : undefined}
          className={cn(
            "h-[38px] rounded-lg font-mono text-[13px] shadow-none md:text-[13px]",
            modified && "border-amber-500/40 focus-visible:border-amber-500",
            error && "border-destructive focus-visible:border-destructive"
          )}
        />
        {modified ? (
          <Button
            type="button"
            variant="ghost"
            className="h-[38px] shrink-0 px-2.5 text-xs text-muted-foreground"
            onClick={() => onChange(item.default)}
          >
            {t("admin.settings.reg.reset_field")}
          </Button>
        ) : null}
      </div>
      {error ? (
        <span className="text-[11px] text-destructive">{error}</span>
      ) : (
        <span className="text-[11px] text-muted-foreground">
          {hint ? `${hint} · ` : ""}
          {range}
        </span>
      )}
    </div>
  );
}

export function RegistrySettingsPanel({ group, header, draft, onDraftChange }: Props) {
  const t = useT();
  const queryClient = useQueryClient();
  const [serverErrors, setServerErrors] = useState<Record<string, string>>({});

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.adminSettingsRegistry,
    queryFn: fetchAdminSettingsRegistry,
    staleTime: 30 * 1000,
  });

  const items = useMemo(
    () => (data?.items ?? []).filter((item) => item.group === group),
    [data, group]
  );

  const sections = useMemo(() => {
    const order: string[] = [];
    items.forEach((item) => {
      if (!order.includes(item.section)) order.push(item.section);
    });
    return order;
  }, [items]);

  const valueOf = (item: RegistryItem) => draft[item.key] ?? item.value;

  const changed = items.filter(
    (item) => draft[item.key] !== undefined && draft[item.key] !== item.value
  );

  const errors = useMemo(() => {
    const out: Record<string, string> = {};
    items.forEach((item) => {
      if (draft[item.key] === undefined) return;
      const message = validate(t, item, draft[item.key]);
      if (message) out[item.key] = message;
    });
    return out;
  }, [items, draft, t]);

  const hasErrors = Object.keys(errors).length > 0;

  const setValue = (item: RegistryItem, next: string) => {
    const copy = { ...draft };
    if (next === item.value) delete copy[item.key];
    else copy[item.key] = next;
    onDraftChange(copy);
    if (serverErrors[item.key]) {
      const rest = { ...serverErrors };
      delete rest[item.key];
      setServerErrors(rest);
    }
  };

  const saveMutation = useMutation({
    mutationFn: saveAdminSettingsRegistry,
    onSuccess: (res, variables) => {
      queryClient.setQueryData(queryKeys.adminSettingsRegistry, res);
      const copy = { ...draft };
      Object.keys(variables).forEach((key) => delete copy[key]);
      onDraftChange(copy);
      setServerErrors({});
      toast.success(t("admin.settings.saved"));
    },
    onError: (err: Error) => {
      if (err instanceof ApiError && err.status === 422) {
        const fields = (err.data?.fields ?? {}) as Record<string, string>;
        setServerErrors(fields);
      }
      toast.error(err.message || t("common.save_failed"));
    },
  });

  const save = () => {
    const values: Record<string, string> = {};
    changed.forEach((item) => {
      const next = draft[item.key].trim();
      values[item.key] = next === item.default ? "" : next;
    });
    saveMutation.mutate(values);
  };

  const resetGroup = async () => {
    const ok = await confirmAction(t("admin.settings.reg.reset_all_confirm"), {
      title: t("admin.settings.reg.reset_all"),
      confirmText: t("admin.settings.reg.reset_all"),
      destructive: true,
    });
    if (!ok) return;
    const next = { ...draft };
    items.forEach((item) => {
      if (item.value !== item.default) next[item.key] = item.default;
      else delete next[item.key];
    });
    onDraftChange(next);
  };

  const dirty = changed.length > 0;
  const groupTitle = t(`admin.settings.tab.${group}`);
  const groupDescriptionKey = `admin.settings.reg.group.${group}`;
  const groupDescription =
    t(groupDescriptionKey) === groupDescriptionKey ? "" : t(groupDescriptionKey);

  if (isLoading) {
    return (
      <PageShell variant="admin">
        <div className="w-full space-y-6">
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-11 w-full" />
          <Skeleton className="h-96 w-full" />
        </div>
      </PageShell>
    );
  }

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-6 pb-24">
        {header}

        {groupDescription ? (
          <p className="text-sm text-muted-foreground">{groupDescription}</p>
        ) : null}

        {items.length === 0 ? (
          <SettingsCard>
            <div className="text-sm text-muted-foreground">
              {t("admin.settings.reg.empty")}
            </div>
          </SettingsCard>
        ) : (
          sections.map((section) => {
            const sectionItems = items.filter((item) => item.section === section);
            const toggles = sectionItems.filter((item) => item.kind === "bool");
            const fields = sectionItems.filter((item) => item.kind !== "bool");
            const descKey = `admin.settings.reg.section.${group}.${section}.description`;
            const desc = t(descKey);
            return (
              <SettingsCard
                key={section}
                title={t(`admin.settings.reg.section.${group}.${section}`)}
                description={desc === descKey ? undefined : desc}
              >
                <div className="space-y-4">
                  {toggles.length > 0 ? (
                    <div className="space-y-3">
                      {toggles.map((item) => (
                        <RegistryField
                          key={item.key}
                          item={item}
                          value={valueOf(item)}
                          error=""
                          onChange={(next) => setValue(item, next)}
                          t={t}
                        />
                      ))}
                    </div>
                  ) : null}
                  {fields.length > 0 ? (
                    <FieldGrid cols={3}>
                      {fields.map((item) => (
                        <RegistryField
                          key={item.key}
                          item={item}
                          value={valueOf(item)}
                          error={errors[item.key] ?? serverErrors[item.key] ?? ""}
                          onChange={(next) => setValue(item, next)}
                          t={t}
                        />
                      ))}
                    </FieldGrid>
                  ) : null}
                </div>
              </SettingsCard>
            );
          })
        )}
      </div>

      <SettingsSaveBar
        dirty={dirty}
        message={
          dirty
            ? t("admin.settings.dirty", { tab: groupTitle })
            : t("admin.settings.clean", { tab: groupTitle })
        }
        saving={saveMutation.isPending}
        discardLabel={t("admin.settings.discard")}
        saveLabel={t("common.save")}
        savingLabel={t("common.saving")}
        onDiscard={() => {
          const next = { ...draft };
          items.forEach((item) => delete next[item.key]);
          onDraftChange(next);
          setServerErrors({});
        }}
        onSave={save}
        saveDisabled={!dirty || hasErrors}
        extra={
          <Button
            type="button"
            variant="outline"
            className="h-9 text-[13px]"
            onClick={() => void resetGroup()}
            disabled={saveMutation.isPending}
          >
            {t("admin.settings.reg.reset_all")}
          </Button>
        }
      />
    </PageShell>
  );
}
