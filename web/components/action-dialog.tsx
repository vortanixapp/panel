"use client";

import { useEffect, useState, useSyncExternalStore } from "react";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useT } from "@/hooks/use-translations";

type ActionOptions = {
  title?: string;
  confirmText?: string;
  destructive?: boolean;
  placeholder?: string;
  defaultValue?: string;
};

type Request =
  | (ActionOptions & { kind: "confirm"; message: string; resolve: (v: boolean) => void })
  | (ActionOptions & { kind: "prompt"; message: string; resolve: (v: string | null) => void });

let current: Request | null = null;
const listeners = new Set<() => void>();

function publish(next: Request | null) {
  current = next;
  listeners.forEach((l) => l());
}

function cancelCurrent() {
  if (!current) return;
  if (current.kind === "confirm") current.resolve(false);
  else current.resolve(null);
}

export function confirmAction(message: string, options: ActionOptions = {}): Promise<boolean> {
  return new Promise((resolve) => {
    cancelCurrent();
    publish({ ...options, kind: "confirm", message, resolve });
  });
}

export function promptAction(message: string, options: ActionOptions = {}): Promise<string | null> {
  return new Promise((resolve) => {
    cancelCurrent();
    publish({ ...options, kind: "prompt", message, resolve });
  });
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function ActionDialogHost() {
  const t = useT();
  const request = useSyncExternalStore(subscribe, () => current, () => null);
  const [value, setValue] = useState("");

  useEffect(() => {
    setValue(request?.kind === "prompt" ? request.defaultValue ?? "" : "");
  }, [request]);

  const close = (ok: boolean) => {
    if (!request) return;
    publish(null);
    if (request.kind === "confirm") request.resolve(ok);
    else request.resolve(ok ? value : null);
  };

  return (
    <AlertDialog open={request !== null} onOpenChange={(open) => !open && close(false)}>
      <AlertDialogContent>
        <form
          className="grid gap-4"
          onSubmit={(e) => {
            e.preventDefault();
            close(true);
          }}
        >
          <AlertDialogHeader className="text-start">
            <AlertDialogTitle>{request?.title ?? t("common.action_confirm_title")}</AlertDialogTitle>
            <AlertDialogDescription className="whitespace-pre-line">{request?.message}</AlertDialogDescription>
          </AlertDialogHeader>
          {request?.kind === "prompt" && (
            <Input
              autoFocus
              value={value}
              placeholder={request.placeholder}
              onChange={(e) => setValue(e.target.value)}
            />
          )}
          <AlertDialogFooter>
            <AlertDialogCancel type="button">{t("common.cancel")}</AlertDialogCancel>
            <Button type="submit" variant={request?.destructive ? "destructive" : "default"} autoFocus={request?.kind !== "prompt"}>
              {request?.confirmText ?? t("common.confirm")}
            </Button>
          </AlertDialogFooter>
        </form>
      </AlertDialogContent>
    </AlertDialog>
  );
}
