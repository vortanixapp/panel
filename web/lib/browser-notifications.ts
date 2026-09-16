"use client";

import { useMemo, useSyncExternalStore } from "react";

const STORAGE_KEY = "vx-browser-notifications";
const listeners = new Set<() => void>();

export type BrowserNotificationsPermission = NotificationPermission | "unsupported";

export type BrowserNotificationsState = {
  supported: boolean;
  permission: BrowserNotificationsPermission;
  enabled: boolean;
};

export function browserNotificationsSupported(): boolean {
  return typeof window !== "undefined" && "Notification" in window && window.isSecureContext;
}

function storedEnabled(): boolean {
  try {
    return window.localStorage.getItem(STORAGE_KEY) === "1";
  } catch {
    return false;
  }
}

export function browserNotificationsActive(): boolean {
  return (
    browserNotificationsSupported() &&
    window.Notification.permission === "granted" &&
    storedEnabled()
  );
}

function emit() {
  listeners.forEach((listener) => listener());
}

export function setBrowserNotificationsEnabled(on: boolean) {
  try {
    if (on) window.localStorage.setItem(STORAGE_KEY, "1");
    else window.localStorage.removeItem(STORAGE_KEY);
  } catch {}
  emit();
}

export async function enableBrowserNotifications(): Promise<BrowserNotificationsPermission> {
  if (!browserNotificationsSupported()) return "unsupported";
  let permission = window.Notification.permission;
  if (permission === "default") {
    try {
      permission = await window.Notification.requestPermission();
    } catch {
      permission = window.Notification.permission;
    }
  }
  setBrowserNotificationsEnabled(permission === "granted");
  return permission;
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  window.addEventListener("storage", listener);
  document.addEventListener("visibilitychange", listener);
  return () => {
    listeners.delete(listener);
    window.removeEventListener("storage", listener);
    document.removeEventListener("visibilitychange", listener);
  };
}

function snapshot(): string {
  if (!browserNotificationsSupported()) return "unsupported|0";
  return `${window.Notification.permission}|${storedEnabled() ? 1 : 0}`;
}

function serverSnapshot(): string {
  return "unsupported|0";
}

export function useBrowserNotifications(): BrowserNotificationsState {
  const value = useSyncExternalStore(subscribe, snapshot, serverSnapshot);
  return useMemo(() => {
    const [permission, enabled] = value.split("|") as [BrowserNotificationsPermission, string];
    return {
      supported: permission !== "unsupported",
      permission,
      enabled: permission === "granted" && enabled === "1",
    };
  }, [value]);
}
