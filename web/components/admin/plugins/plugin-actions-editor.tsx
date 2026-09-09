"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { AdminPluginAction } from "@/lib/api";
import { useT } from "@/hooks/use-translations";

export const PLUGIN_ACTION_KINDS: {
  value: AdminPluginAction["action"];
  labelKey: string;
  hintKey: string;
}[] = [
  {
    value: "ensure_contains",
    labelKey: "admin.plugins.action.ensure_contains",
    hintKey: "admin.plugins.action.ensure_contains_hint",
  },
  {
    value: "append_lines",
    labelKey: "admin.plugins.action.append_lines",
    hintKey: "admin.plugins.action.append_lines_hint",
  },
  {
    value: "prepend_lines",
    labelKey: "admin.plugins.action.prepend_lines",
    hintKey: "admin.plugins.action.prepend_lines_hint",
  },
  {
    value: "remove_lines",
    labelKey: "admin.plugins.action.remove_lines",
    hintKey: "admin.plugins.action.remove_lines_hint",
  },
  {
    value: "write_file",
    labelKey: "admin.plugins.action.write_file",
    hintKey: "admin.plugins.action.write_file_hint",
  },
  {
    value: "replace_regex",
    labelKey: "admin.plugins.action.replace_regex",
    hintKey: "admin.plugins.action.replace_regex_hint",
  },
];

export function emptyPluginAction(): AdminPluginAction {
  return { path: "", action: "ensure_contains", create_if_missing: true, lines: [] };
}

export function PluginActionsEditor({
  title,
  hint,
  actions,
  onChange,
}: {
  title: string;
  hint: string;
  actions: AdminPluginAction[];
  onChange: (next: AdminPluginAction[]) => void;
}) {
  const t = useT();

  function update(index: number, patch: Partial<AdminPluginAction>) {
    onChange(actions.map((a, i) => (i === index ? { ...a, ...patch } : a)));
  }

  return (
    <div className="rounded-lg border bg-card p-4">
      <div className="mb-3 flex items-start justify-between gap-4">
        <div>
          <div className="text-sm font-semibold">{title}</div>
          <p className="text-xs text-muted-foreground">{hint}</p>
        </div>
        <Button type="button" variant="outline" size="sm" onClick={() => onChange([...actions, emptyPluginAction()])}>
          + {t("admin.plugins.action.add")}
        </Button>
      </div>

      {actions.length === 0 ? (
        <p className="text-xs text-muted-foreground">
          {t("admin.plugins.action.empty")}
        </p>
      ) : (
        <div className="space-y-3">
          {actions.map((action, index) => {
            const kind = PLUGIN_ACTION_KINDS.find((k) => k.value === action.action);
            return (
              <div key={index} className="rounded-md border bg-background p-3">
                <div className="grid gap-3 md:grid-cols-2">
                  <div>
                    <Label className="text-xs">
                      {t("admin.plugins.action.file")}
                    </Label>
                    <Input
                      value={action.path}
                      placeholder="addons/amxmodx/configs/plugins.ini"
                      onChange={(e) => update(index, { path: e.target.value })}
                    />
                    <p className="mt-1 text-[11px] text-muted-foreground">
                      {t("admin.plugins.action.file_hint")}
                    </p>
                  </div>
                  <div>
                    <Label className="text-xs">
                      {t("admin.plugins.action.what")}
                    </Label>
                    <Select
                      value={action.action}
                      onValueChange={(v) => update(index, { action: v as AdminPluginAction["action"] })}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {PLUGIN_ACTION_KINDS.map((k) => (
                          <SelectItem key={k.value} value={k.value}>
                            {t(k.labelKey)}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    {kind && (
                      <p className="mt-1 text-[11px] text-muted-foreground">
                        {t(kind.hintKey)}
                      </p>
                    )}
                  </div>
                </div>

                {action.action === "replace_regex" ? (
                  <div className="mt-3 grid gap-3 md:grid-cols-2">
                    <div>
                      <Label className="text-xs">
                        {t("admin.plugins.action.regex")}
                      </Label>
                      <Input
                        value={action.pattern ?? ""}
                        placeholder="^sv_gravity .*$"
                        onChange={(e) => update(index, { pattern: e.target.value })}
                      />
                    </div>
                    <div>
                      <Label className="text-xs">
                        {t("admin.plugins.action.replacement")}
                      </Label>
                      <Input
                        value={action.replacement ?? ""}
                        placeholder="sv_gravity 800"
                        onChange={(e) => update(index, { replacement: e.target.value })}
                      />
                    </div>
                  </div>
                ) : (
                  <div className="mt-3">
                    <Label className="text-xs">
                      {t("admin.plugins.action.lines")}
                    </Label>
                    <Textarea
                      rows={3}
                      value={(action.lines ?? []).join("\n")}
                      placeholder={"sv_gravity 800\nmp_timelimit 30"}
                      onChange={(e) =>
                        update(index, { lines: e.target.value.split("\n") })
                      }
                    />
                    <p className="mt-1 text-[11px] text-muted-foreground">
                      {t("admin.plugins.action.lines_hint")}
                    </p>
                  </div>
                )}

                <div className="mt-3 flex items-center justify-between">
                  <label className="flex items-center gap-2 text-xs">
                    <Switch
                      checked={action.create_if_missing ?? false}
                      onCheckedChange={(v) => update(index, { create_if_missing: v })}
                    />
                    {t("admin.plugins.action.create_if_missing")}
                  </label>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => onChange(actions.filter((_, i) => i !== index))}
                  >
                    {t("admin.news.remove_image")}
                  </Button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
