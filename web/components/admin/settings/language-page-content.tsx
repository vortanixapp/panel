"use client";

import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { MoreHorizontal, Plus } from "lucide-react";
import { toast } from "sonner";
import { PageShell } from "@/components/layout/page-shell";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { useLocale } from "@/context/locale-provider";
import { useT } from "@/hooks/use-translations";
import {
  createAdminLanguage,
  deleteAdminLanguage,
  fetchAdminLanguages,
  updateAdminLanguage,
  type AdminLanguage,
  type AdminLanguagePatch,
  type AdminLanguages,
} from "@/lib/api";
import {
  BASE_LOCALES,
  builtinLanguageName,
  CATALOG_KEYS,
  isLocaleCode,
  normalizeLocaleCode,
  type BaseLocale,
} from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { LanguageEditor } from "./language-editor";

const ADMIN_LANGUAGES_KEY = ["admin-languages"] as const;

type DialogState =
  | { mode: "create" }
  | { mode: "edit"; language: AdminLanguage }
  | null;

function errorMessage(err: unknown, fallback: string): string {
  return err instanceof Error && err.message ? err.message : fallback;
}

export function LanguagePageContent() {
  const t = useT();
  const qc = useQueryClient();
  const { locale, reload } = useLocale();
  const [selected, setSelected] = useState("");
  const [dialog, setDialog] = useState<DialogState>(null);
  const [removing, setRemoving] = useState<AdminLanguage | null>(null);
  const [editorDirty, setEditorDirty] = useState(false);

  const query = useQuery({
    queryKey: ADMIN_LANGUAGES_KEY,
    queryFn: fetchAdminLanguages,
  });
  const languages = query.data?.languages;
  const defaultLocale = query.data?.default_locale ?? "";

  useEffect(() => {
    if (!languages?.length) return;
    if (languages.some((l) => l.code === selected)) return;
    setSelected(
      languages.some((l) => l.code === locale) ? locale : languages[0].code
    );
  }, [languages, selected, locale]);

  function applyLanguages(data: AdminLanguages) {
    qc.setQueryData(ADMIN_LANGUAGES_KEY, data);
    void reload();
  }

  const update = useMutation({
    mutationFn: ({ code, patch }: { code: string; patch: AdminLanguagePatch }) =>
      updateAdminLanguage(code, patch),
    onSuccess: (data) => {
      applyLanguages(data);
      toast.success(t("admin.language.updated"));
    },
    onError: (err) => toast.error(errorMessage(err, t("common.save_failed"))),
  });

  const remove = useMutation({
    mutationFn: (code: string) => deleteAdminLanguage(code),
    onSuccess: (data) => {
      applyLanguages(data);
      setRemoving(null);
      toast.success(t("admin.language.deleted"));
    },
    onError: (err) => toast.error(errorMessage(err, t("common.delete_failed"))),
  });

  function select(code: string) {
    if (code === selected) return;
    if (editorDirty && !window.confirm(t("admin.language.discard_confirm"))) {
      return;
    }
    setSelected(code);
  }

  function onDialogSaved(data: AdminLanguages, code: string, created: boolean) {
    applyLanguages(data);
    setDialog(null);
    toast.success(
      t(created ? "admin.language.created" : "admin.language.updated")
    );
    if (created && !editorDirty) setSelected(code);
  }

  function onMessagesSaved() {
    void qc.invalidateQueries({ queryKey: ADMIN_LANGUAGES_KEY });
    void reload();
  }

  const current = languages?.find((l) => l.code === selected);

  return (
    <PageShell variant="admin">
      <div className="w-full space-y-5">
        <div className="flex flex-wrap items-end justify-between gap-6">
          <div className="space-y-1.5">
            <h1 className="text-[26px] leading-none font-bold tracking-tight">
              {t("admin.language.title")}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t("admin.language.subtitle")}
            </p>
          </div>
          <Button
            className="h-[38px] text-[13px]"
            onClick={() => setDialog({ mode: "create" })}
          >
            <Plus className="size-4" />
            {t("admin.language.add")}
          </Button>
        </div>

        {query.isLoading ? (
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            <Skeleton className="h-32 rounded-2xl" />
            <Skeleton className="h-32 rounded-2xl" />
          </div>
        ) : !languages ? (
          <div className="rounded-2xl border bg-card py-12 text-center text-[13px] text-muted-foreground">
            {errorMessage(query.error, t("common.load_failed"))}
          </div>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {languages.map((language) => (
              <LanguageCard
                key={language.code}
                language={language}
                active={language.code === selected}
                isDefault={language.code === defaultLocale}
                busy={update.isPending}
                onSelect={() => select(language.code)}
                onToggle={(enabled) =>
                  update.mutate({ code: language.code, patch: { enabled } })
                }
                onMakeDefault={() =>
                  update.mutate({
                    code: language.code,
                    patch: { default: true, enabled: true },
                  })
                }
                onEdit={() => setDialog({ mode: "edit", language })}
                onDelete={() => setRemoving(language)}
              />
            ))}
          </div>
        )}

        {current && (
          <LanguageEditor
            key={current.code}
            language={current}
            onDirtyChange={setEditorDirty}
            onSaved={onMessagesSaved}
          />
        )}
      </div>

      <LanguageDialog
        state={dialog}
        onClose={() => setDialog(null)}
        onSaved={onDialogSaved}
      />

      <AlertDialog
        open={removing !== null}
        onOpenChange={(open) => {
          if (!open && !remove.isPending) setRemoving(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t("admin.language.delete_title", { name: removing?.name ?? "" })}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("admin.language.delete_hint")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={remove.isPending}>
              {t("common.cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-white hover:bg-destructive/90"
              disabled={remove.isPending}
              onClick={(event) => {
                event.preventDefault();
                if (removing) remove.mutate(removing.code);
              }}
            >
              {remove.isPending ? t("common.deleting") : t("common.delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </PageShell>
  );
}

function LanguageCard({
  language,
  active,
  isDefault,
  busy,
  onSelect,
  onToggle,
  onMakeDefault,
  onEdit,
  onDelete,
}: {
  language: AdminLanguage;
  active: boolean;
  isDefault: boolean;
  busy: boolean;
  onSelect: () => void;
  onToggle: (enabled: boolean) => void;
  onMakeDefault: () => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const t = useT();
  const total = CATALOG_KEYS.length;
  const translated = Math.min(language.translated, total);
  const percent = total ? Math.round((translated / total) * 100) : 0;

  return (
    <div
      className={cn(
        "flex gap-3 rounded-2xl border bg-card p-4 transition-colors",
        active
          ? "border-primary/60 ring-1 ring-primary/25"
          : "hover:border-foreground/20"
      )}
    >
      <button
        type="button"
        onClick={onSelect}
        aria-pressed={active}
        className={cn(
          "min-w-0 flex-1 space-y-3 rounded-lg text-start outline-none focus-visible:ring-2 focus-visible:ring-ring/50",
          !language.enabled && "opacity-70"
        )}
      >
        <div className="space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="truncate text-[15px] font-semibold">
              {language.name}
            </span>
            <span className="rounded-md border px-1.5 py-px font-mono text-[11px] text-muted-foreground">
              {language.code}
            </span>
            {isDefault && (
              <Badge variant="secondary">{t("admin.language.default")}</Badge>
            )}
            {!language.enabled && (
              <Badge variant="outline">{t("admin.language.disabled")}</Badge>
            )}
          </div>
          <div className="text-xs text-muted-foreground">
            {language.builtin
              ? t("admin.language.builtin")
              : t("admin.language.base_hint", {
                  name: builtinLanguageName(language.base),
                })}
          </div>
        </div>
        {language.builtin ? (
          <div className="text-xs text-muted-foreground">
            {translated
              ? t("admin.language.changed_count", { count: translated })
              : t("admin.language.no_changes")}
          </div>
        ) : (
          <div className="space-y-1.5">
            <div className="flex justify-between gap-3 text-xs text-muted-foreground">
              <span>
                {t("admin.language.translated_count", {
                  count: translated,
                  total,
                })}
              </span>
              <span className="font-mono">{percent}%</span>
            </div>
            <div className="h-1.5 overflow-hidden rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-primary transition-[width]"
                style={{ width: `${percent}%` }}
              />
            </div>
          </div>
        )}
      </button>
      <div className="flex shrink-0 flex-col items-end justify-between gap-3">
        <DropdownMenu modal={false}>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="size-8"
              aria-label={t("common.actions")}
            >
              <MoreHorizontal className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={onSelect}>
              {t("admin.language.edit_phrases")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={onEdit}>
              {t("admin.language.edit_language")}
            </DropdownMenuItem>
            <DropdownMenuItem
              disabled={isDefault || busy}
              onClick={onMakeDefault}
            >
              {t("admin.language.make_default")}
            </DropdownMenuItem>
            {!language.builtin && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  variant="destructive"
                  disabled={isDefault}
                  onClick={onDelete}
                >
                  {t("admin.language.delete")}
                </DropdownMenuItem>
              </>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
        <Switch
          checked={language.enabled}
          disabled={isDefault || busy}
          onCheckedChange={onToggle}
          aria-label={t("admin.language.toggle_aria")}
        />
      </div>
    </div>
  );
}

function LanguageDialog({
  state,
  onClose,
  onSaved,
}: {
  state: DialogState;
  onClose: () => void;
  onSaved: (data: AdminLanguages, code: string, created: boolean) => void;
}) {
  const t = useT();
  const editing = state?.mode === "edit" ? state.language : null;
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [base, setBase] = useState<BaseLocale>("en");

  useEffect(() => {
    if (!state) return;
    const language = state.mode === "edit" ? state.language : null;
    setCode(language?.code ?? "");
    setName(language?.name ?? "");
    setBase(language?.base ?? "en");
  }, [state]);

  const normalizedCode = normalizeLocaleCode(code);
  const codeValid = isLocaleCode(normalizedCode);
  const canSave = name.trim() !== "" && (editing !== null || codeValid);

  const save = useMutation({
    mutationFn: () =>
      editing
        ? updateAdminLanguage(
            editing.code,
            editing.builtin
              ? { name: name.trim() }
              : { name: name.trim(), base }
          )
        : createAdminLanguage({ code: normalizedCode, name: name.trim(), base }),
    onSuccess: (data) =>
      onSaved(data, editing?.code ?? normalizedCode, editing === null),
    onError: (err) => toast.error(errorMessage(err, t("common.save_failed"))),
  });

  return (
    <Dialog
      open={state !== null}
      onOpenChange={(open) => {
        if (!open && !save.isPending) onClose();
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>
            {editing
              ? t("admin.language.edit_title", { name: editing.name })
              : t("admin.language.create_title")}
          </DialogTitle>
          <DialogDescription>
            {editing
              ? t("admin.language.edit_hint")
              : t("admin.language.create_hint")}
          </DialogDescription>
        </DialogHeader>
        <form
          className="space-y-4"
          onSubmit={(event) => {
            event.preventDefault();
            if (canSave && !save.isPending) save.mutate();
          }}
        >
          {!editing && (
            <div className="space-y-2">
              <Label htmlFor="language-code">{t("admin.language.code")}</Label>
              <Input
                id="language-code"
                value={code}
                autoFocus
                autoComplete="off"
                spellCheck={false}
                placeholder="uk"
                className="font-mono"
                aria-invalid={code !== "" && !codeValid}
                onChange={(e) => setCode(e.target.value)}
              />
              <p
                className={cn(
                  "text-xs",
                  code !== "" && !codeValid
                    ? "text-destructive"
                    : "text-muted-foreground"
                )}
              >
                {t("admin.language.code_hint")}
              </p>
            </div>
          )}
          <div className="space-y-2">
            <Label htmlFor="language-name">{t("admin.language.name")}</Label>
            <Input
              id="language-name"
              value={name}
              maxLength={64}
              autoComplete="off"
              placeholder={t("admin.language.name_placeholder")}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          {!editing?.builtin && (
            <div className="space-y-2">
              <Label htmlFor="language-base">{t("admin.language.base")}</Label>
              <Select
                value={base}
                onValueChange={(value) => setBase(value as BaseLocale)}
              >
                <SelectTrigger id="language-base" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {BASE_LOCALES.map((item) => (
                    <SelectItem key={item} value={item}>
                      {builtinLanguageName(item)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                {t("admin.language.base_help")}
              </p>
            </div>
          )}
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={save.isPending}
              onClick={onClose}
            >
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={!canSave || save.isPending}>
              {save.isPending
                ? t("common.saving")
                : editing
                  ? t("common.save")
                  : t("common.add")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
