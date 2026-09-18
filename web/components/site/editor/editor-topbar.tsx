"use client";

import { useState } from "react";
import Link from "next/link";
import {
  AlertTriangle,
  ArrowLeft,
  Check,
  CloudOff,
  Eye,
  History,
  Loader2,
  Monitor,
  MoreHorizontal,
  MousePointerClick,
  Redo2,
  Smartphone,
  Tablet,
  Trash2,
  Undo2,
  Upload,
} from "lucide-react";
import { toast } from "sonner";
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
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useEditor } from "@/components/site/editor/editor-context";
import type { Device } from "@/components/site/editor/preview-frame";
import type { EditorMode } from "@/lib/site/store";
import { cn } from "@/lib/utils";

export type ViewerMode = "auto" | "guest" | "client";

function Segment<T extends string>({
  value,
  items,
  onChange,
}: {
  value: T;
  items: { id: T; label: string; icon?: typeof Monitor; iconOnly?: boolean }[];
  onChange: (next: T) => void;
}) {
  return (
    <div className="flex items-center gap-0.5 rounded-lg bg-muted/70 p-0.5">
      {items.map((item) => (
        <button
          key={item.id}
          type="button"
          title={item.label}
          aria-label={item.label}
          aria-pressed={value === item.id}
          onClick={() => onChange(item.id)}
          className={cn(
            "inline-flex h-7 items-center gap-1.5 rounded-md px-2 text-xs transition-colors",
            value === item.id ? "bg-background font-medium text-foreground shadow-xs" : "text-muted-foreground hover:text-foreground"
          )}
        >
          {item.icon ? <item.icon className="size-3.5" /> : null}
          {item.iconOnly ? null : <span className="hidden xl:inline">{item.label}</span>}
        </button>
      ))}
    </div>
  );
}

export function EditorTopbar({
  device,
  onDevice,
  mode,
  onMode,
  viewerMode,
  onViewerMode,
  showViewer,
  locale,
  onLocale,
  onVersions,
  onPublish,
}: {
  device: Device;
  onDevice: (device: Device) => void;
  mode: EditorMode;
  onMode: (mode: EditorMode) => void;
  viewerMode: ViewerMode;
  onViewerMode: (mode: ViewerMode) => void;
  showViewer: boolean;
  locale: string;
  onLocale: (locale: string) => void;
  onVersions: () => void;
  onPublish: () => void;
}) {
  const { editor, t, languages, path } = useEditor();
  const [confirmDiscard, setConfirmDiscard] = useState(false);
  const pending = editor.dirty || Boolean(editor.status?.changed);

  const status = (() => {
    switch (editor.save) {
      case "saving":
        return (
          <span className="inline-flex items-center gap-1.5 text-xs text-muted-foreground">
            <Loader2 className="size-3.5 animate-spin" />
            {t("template.status.saving")}
          </span>
        );
      case "error":
        return (
          <span className="inline-flex max-w-[260px] items-center gap-1.5 truncate text-xs text-destructive" title={editor.saveError ?? ""}>
            <CloudOff className="size-3.5 shrink-0" />
            <span className="truncate">{editor.saveError ?? t("template.status.error")}</span>
          </span>
        );
      case "conflict":
        return (
          <span className="inline-flex items-center gap-1.5 text-xs text-amber-600 dark:text-amber-500">
            <AlertTriangle className="size-3.5" />
            {t("template.status.conflict")}
          </span>
        );
      default:
        return editor.dirty ? (
          <span className="text-xs text-muted-foreground">{t("template.status.unsaved")}</span>
        ) : (
          <span className="inline-flex items-center gap-1.5 text-xs text-muted-foreground">
            <Check className="size-3.5" />
            {pending ? t("template.status.draft_saved") : t("template.status.published")}
          </span>
        );
    }
  })();

  return (
    <header className="flex h-14 shrink-0 items-center gap-3 border-b bg-background px-3">
      <Button asChild variant="ghost" size="sm" className="gap-1.5 px-2">
        <Link href="/admin/dashboard">
          <ArrowLeft className="size-4" />
          <span className="hidden lg:inline">{t("template.topbar.back")}</span>
        </Link>
      </Button>
      <div className="min-w-0">
        <div className="text-sm leading-tight font-semibold">{t("template.title")}</div>
        <div className="truncate font-mono text-[11px] leading-tight text-muted-foreground">{path}</div>
      </div>

      <div className="mx-auto flex items-center gap-2">
        <Segment
          value={mode}
          onChange={onMode}
          items={[
            { id: "edit", label: t("template.topbar.mode_edit"), icon: MousePointerClick },
            { id: "preview", label: t("template.topbar.mode_preview"), icon: Eye },
          ]}
        />
        <Segment
          value={device}
          onChange={onDevice}
          items={[
            { id: "desktop", label: t("template.topbar.desktop"), icon: Monitor, iconOnly: true },
            { id: "tablet", label: t("template.topbar.tablet"), icon: Tablet, iconOnly: true },
            { id: "phone", label: t("template.topbar.phone"), icon: Smartphone, iconOnly: true },
          ]}
        />
        {showViewer ? (
          <Select value={viewerMode} onValueChange={(value) => onViewerMode(value as ViewerMode)}>
            <SelectTrigger size="sm" className="h-8 w-[150px] text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="auto">{t("template.viewer.auto")}</SelectItem>
              <SelectItem value="guest">{t("template.viewer.guest")}</SelectItem>
              <SelectItem value="client">{t("template.viewer.client")}</SelectItem>
            </SelectContent>
          </Select>
        ) : null}
        {languages.length > 1 ? (
          <Select value={locale} onValueChange={onLocale}>
            <SelectTrigger size="sm" className="h-8 w-[130px] text-xs" aria-label={t("template.topbar.language")}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {languages.map((lang) => (
                <SelectItem key={lang.code} value={lang.code}>
                  {lang.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : null}
      </div>

      <div className="flex items-center gap-1.5">
        <div className="mr-1 hidden md:block">{status}</div>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-8"
          title={`${t("template.topbar.undo")} (Ctrl+Z)`}
          aria-label={t("template.topbar.undo")}
          disabled={!editor.canUndo}
          onClick={editor.undo}
        >
          <Undo2 className="size-4" />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-8"
          title={`${t("template.topbar.redo")} (Ctrl+Shift+Z)`}
          aria-label={t("template.topbar.redo")}
          disabled={!editor.canRedo}
          onClick={editor.redo}
        >
          <Redo2 className="size-4" />
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={onVersions}>
          <History className="size-3.5" />
          <span className="hidden lg:inline">{t("template.topbar.versions")}</span>
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button type="button" variant="ghost" size="icon" className="size-8" aria-label={t("template.topbar.more")}>
              <MoreHorizontal className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem disabled={!pending} onSelect={() => setConfirmDiscard(true)} className="text-destructive focus:text-destructive">
              <Trash2 className="size-4" />
              {t("template.topbar.discard")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button type="button" size="sm" disabled={!pending || editor.busy || editor.save === "conflict"} onClick={onPublish}>
          <Upload className="size-3.5" />
          {t("template.topbar.publish")}
        </Button>
      </div>

      <AlertDialog open={confirmDiscard} onOpenChange={setConfirmDiscard}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("template.discard.title")}</AlertDialogTitle>
            <AlertDialogDescription>{t("template.discard.hint")}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                void editor
                  .discard()
                  .then(() => toast.success(t("template.discard.done")))
                  .catch((error: unknown) => toast.error(error instanceof Error ? error.message : t("template.discard.failed")));
              }}
            >
              {t("template.discard.confirm")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </header>
  );
}
