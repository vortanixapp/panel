import type { QueryClient, QueryKey } from "@tanstack/react-query";
import { currentAccount } from "@/lib/accounts";

const PREFIX = "vx-cache:";
const MAX_AGE_MS = 6 * 60 * 60 * 1000;
const SAVE_DELAY_MS = 1500;
const KEYS: QueryKey[] = [["dashboard"], ["my-servers"]];

type Stored = Record<string, { data: unknown; at: number }>;

function storageKey(userId: string) {
  return PREFIX + userId;
}

function sameKey(a: QueryKey, b: QueryKey) {
  return a.length === b.length && a.every((part, i) => part === b[i]);
}

function read(userId: string): Stored {
  try {
    const raw = window.localStorage.getItem(storageKey(userId));
    return raw ? (JSON.parse(raw) as Stored) : {};
  } catch {
    return {};
  }
}

export function clearPersistedCache() {
  if (typeof window === "undefined") return;
  try {
    const stale: string[] = [];
    for (let i = 0; i < window.localStorage.length; i++) {
      const key = window.localStorage.key(i);
      if (key?.startsWith(PREFIX)) stale.push(key);
    }
    stale.forEach((key) => window.localStorage.removeItem(key));
  } catch {}
}

export function restorePersistedCache(qc: QueryClient) {
  const userId = currentAccount()?.user_id;
  if (!userId) return;
  const stored = read(userId);
  const now = Date.now();
  for (const key of KEYS) {
    const entry = stored[JSON.stringify(key)];
    if (!entry || now - entry.at > MAX_AGE_MS) continue;
    if (qc.getQueryData(key) !== undefined) continue;
    qc.setQueryData(key, entry.data, { updatedAt: entry.at });
  }
}

export function startCachePersistence(qc: QueryClient): () => void {
  const userId = currentAccount()?.user_id;
  if (!userId) return () => {};
  let timer: number | undefined;

  const save = () => {
    timer = undefined;
    const out: Stored = {};
    for (const key of KEYS) {
      const state = qc.getQueryState(key);
      if (state?.status === "success" && state.data !== undefined) {
        out[JSON.stringify(key)] = { data: state.data, at: state.dataUpdatedAt };
      }
    }
    try {
      window.localStorage.setItem(storageKey(userId), JSON.stringify(out));
    } catch {}
  };

  const unsubscribe = qc.getQueryCache().subscribe((event) => {
    if (event.type !== "updated" || event.action.type !== "success") return;
    if (!KEYS.some((key) => sameKey(key, event.query.queryKey))) return;
    if (timer === undefined) timer = window.setTimeout(save, SAVE_DELAY_MS);
  });

  return () => {
    unsubscribe();
    if (timer !== undefined) window.clearTimeout(timer);
  };
}
