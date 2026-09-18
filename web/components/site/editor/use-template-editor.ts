"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError } from "@/lib/api";
import {
  discardTemplate,
  fetchAdminTemplate,
  publishTemplate,
  restoreTemplateVersion,
  saveTemplateDraft,
  type TemplateState,
  type TemplateStatus,
} from "@/lib/site/api";
import type { SiteDocument } from "@/lib/site/types";

export type SaveState = "idle" | "saving" | "saved" | "error" | "conflict";

const HISTORY_LIMIT = 100;
const COALESCE_MS = 1200;
const AUTOSAVE_MS = 900;

type History = {
  doc: SiteDocument;
  past: SiteDocument[];
  future: SiteDocument[];
  coalesce: { key: string; at: number } | null;
};

function statusOf(state: TemplateState): TemplateStatus {
  return {
    revision: state.revision,
    changed: state.changed,
    pending_texts: state.pending_texts,
    draft_updated_at: state.draft_updated_at,
    draft_updated_by: state.draft_updated_by,
    published_at: state.published_at,
    published_by: state.published_by,
  };
}

export function useTemplateEditor() {
  const [loaded, setLoaded] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [history, setHistory] = useState<History>({ doc: {}, past: [], future: [], coalesce: null });
  const [published, setPublished] = useState<SiteDocument>({});
  const [status, setStatus] = useState<TemplateStatus | null>(null);
  const [save, setSave] = useState<SaveState>("idle");
  const [saveError, setSaveError] = useState<string | null>(null);
  const [conflict, setConflict] = useState<TemplateState | null>(null);
  const [busy, setBusy] = useState(false);
  const [savedDoc, setSavedDoc] = useState<SiteDocument>({});

  const revisionRef = useRef(0);
  const docRef = useRef<SiteDocument>({});
  const savedRef = useRef<SiteDocument>({});
  const savingRef = useRef<Promise<void> | null>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const failedRef = useRef<SiteDocument | null>(null);

  const doc = history.doc;
  const dirty = doc !== savedDoc;

  useEffect(() => {
    docRef.current = doc;
  }, [doc]);

  const adopt = useCallback((state: TemplateState) => {
    revisionRef.current = state.revision;
    savedRef.current = state.draft;
    docRef.current = state.draft;
    setSavedDoc(state.draft);
    setHistory({ doc: state.draft, past: [], future: [], coalesce: null });
    setPublished(state.published);
    setStatus(statusOf(state));
    setConflict(null);
    setSaveError(null);
    setSave("idle");
  }, []);

  const reload = useCallback(async () => {
    try {
      adopt(await fetchAdminTemplate());
      setLoadError(null);
    } catch (error) {
      setLoadError(error instanceof Error ? error.message : "Request failed");
    } finally {
      setLoaded(true);
    }
  }, [adopt]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const apply = useCallback((fn: (doc: SiteDocument) => SiteDocument, coalesceKey?: string) => {
    setHistory((current) => {
      const next = fn(current.doc);
      if (next === current.doc) return current;
      const now = Date.now();
      const merge =
        coalesceKey !== undefined &&
        current.coalesce?.key === coalesceKey &&
        now - current.coalesce.at < COALESCE_MS;
      return {
        doc: next,
        past: merge ? current.past : [...current.past, current.doc].slice(-HISTORY_LIMIT),
        future: [],
        coalesce: coalesceKey !== undefined ? { key: coalesceKey, at: now } : null,
      };
    });
  }, []);

  const undo = useCallback(() => {
    setHistory((current) => {
      if (current.past.length === 0) return current;
      const previous = current.past[current.past.length - 1];
      return {
        doc: previous,
        past: current.past.slice(0, -1),
        future: [current.doc, ...current.future].slice(0, HISTORY_LIMIT),
        coalesce: null,
      };
    });
  }, []);

  const redo = useCallback(() => {
    setHistory((current) => {
      if (current.future.length === 0) return current;
      const [next, ...rest] = current.future;
      return {
        doc: next,
        past: [...current.past, current.doc].slice(-HISTORY_LIMIT),
        future: rest,
        coalesce: null,
      };
    });
  }, []);

  const saveNow = useCallback(async (force = false): Promise<void> => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    if (savingRef.current) {
      await savingRef.current;
    }
    const snapshot = docRef.current;
    if (snapshot === savedRef.current && !force) return;
    const run = (async () => {
      setSave("saving");
      try {
        const next = await saveTemplateDraft(snapshot, revisionRef.current, force);
        revisionRef.current = next.revision;
        savedRef.current = snapshot;
        setSavedDoc(snapshot);
        setStatus(next);
        setConflict(null);
        setSaveError(null);
        setSave(docRef.current === snapshot ? "saved" : "idle");
      } catch (error) {
        if (error instanceof ApiError && error.status === 409 && error.data.template) {
          setConflict(error.data.template as TemplateState);
          setSave("conflict");
        } else {
          failedRef.current = snapshot;
          setSaveError(error instanceof Error ? error.message : "Request failed");
          setSave("error");
        }
      }
    })();
    savingRef.current = run;
    try {
      await run;
    } finally {
      savingRef.current = null;
    }
  }, []);

  useEffect(() => {
    if (!loaded || !dirty || save === "conflict") return;
    if (save === "error" && doc === failedRef.current) return;
    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      timerRef.current = null;
      void saveNow();
    }, AUTOSAVE_MS);
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, [doc, dirty, loaded, save, saveNow]);

  useEffect(() => {
    const onBeforeUnload = (event: BeforeUnloadEvent) => {
      if (docRef.current !== savedRef.current || savingRef.current) {
        event.preventDefault();
      }
    };
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => window.removeEventListener("beforeunload", onBeforeUnload);
  }, []);

  const resolveConflict = useCallback(
    async (choice: "theirs" | "mine") => {
      if (!conflict) return;
      if (choice === "theirs") {
        adopt(conflict);
        return;
      }
      revisionRef.current = conflict.revision;
      setConflict(null);
      await saveNow(true);
    },
    [adopt, conflict, saveNow]
  );

  const withBusy = useCallback(async <T,>(run: () => Promise<T>): Promise<T> => {
    setBusy(true);
    try {
      return await run();
    } finally {
      setBusy(false);
    }
  }, []);

  const publish = useCallback(
    (note: string) =>
      withBusy(async () => {
        await saveNow();
        if (docRef.current !== savedRef.current) await saveNow();
        adopt(await publishTemplate(note, revisionRef.current));
      }),
    [adopt, saveNow, withBusy]
  );

  const discard = useCallback(
    () =>
      withBusy(async () => {
        if (timerRef.current) clearTimeout(timerRef.current);
        if (savingRef.current) await savingRef.current;
        adopt(await discardTemplate());
      }),
    [adopt, withBusy]
  );

  const restore = useCallback(
    (id: number) =>
      withBusy(async () => {
        if (timerRef.current) clearTimeout(timerRef.current);
        if (savingRef.current) await savingRef.current;
        adopt(await restoreTemplateVersion(id));
      }),
    [adopt, withBusy]
  );

  return {
    loaded,
    loadError,
    doc,
    published,
    status,
    save,
    saveError,
    conflict,
    busy,
    dirty,
    canUndo: history.past.length > 0,
    canRedo: history.future.length > 0,
    apply,
    undo,
    redo,
    saveNow,
    resolveConflict,
    publish,
    discard,
    restore,
    reload,
  };
}

export type TemplateEditor = ReturnType<typeof useTemplateEditor>;
