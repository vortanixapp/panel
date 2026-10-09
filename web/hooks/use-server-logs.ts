"use client";

import { useEffect, useRef } from "react";
import { keepPreviousData, useQuery, useQueryClient } from "@tanstack/react-query";

import { useConsoleStream } from "@/components/servers/console/use-console-stream";
import { fetchServerLogs } from "@/lib/api";
import { pollMs } from "@/lib/public-settings";
import {
  appendStreamLines,
  dropReplayedLines,
  mergeLogs,
  normalizeLogLine,
  type ServerLogStore,
} from "@/features/servers/log-utils";

const POLL_STEPS_MS = [5_000, 5_000, 10_000, 15_000, 30_000];
const REPLAY_WINDOW_MS = 2_000;

export function useServerLogs(id: string, tail: number, live: boolean) {
  const queryClient = useQueryClient();
  const queryKey = ["server-logs", id, tail] as const;
  const quietPolls = useRef(0);
  const stream = useConsoleStream(id, undefined, live);
  const streaming = live && stream.status === "connected";

  const query = useQuery({
    queryKey,
    queryFn: async () => {
      const prev = queryClient.getQueryData<ServerLogStore>(queryKey);
      const response = await fetchServerLogs(id, tail, prev?.lastTime || undefined);
      const next = mergeLogs(prev, response, tail);
      quietPolls.current = prev && next === prev ? quietPolls.current + 1 : 0;
      return next;
    },
    enabled: !!id,
    placeholderData: keepPreviousData,
    refetchInterval: live && !streaming
      ? () => pollMs(POLL_STEPS_MS[Math.min(quietPolls.current, POLL_STEPS_MS.length - 1)])
      : false,
  });

  const lastId = useRef(0);
  const connectedAt = useRef(0);
  const hasSnapshot = !!query.data;

  useEffect(() => {
    if (stream.status === "connected") connectedAt.current = Date.now();
  }, [stream.status]);

  useEffect(() => {
    if (!streaming || !hasSnapshot) return;
    const fresh = stream.lines.filter(
      (line) => line.id > lastId.current && line.kind !== "input" && line.kind !== "reply" && line.kind !== "notice"
    );
    if (fresh.length === 0) return;
    lastId.current = fresh[fresh.length - 1].id;

    queryClient.setQueryData<ServerLogStore>(queryKey, (prev) => {
      if (!prev) return prev;
      const texts = fresh.map((line) => normalizeLogLine(line.raw));
      const replayCount = fresh.filter((line) => line.at - connectedAt.current < REPLAY_WINDOW_MS).length;
      const skip = replayCount > 0
        ? dropReplayedLines(prev.lines.map(normalizeLogLine), texts.slice(0, replayCount))
        : 0;
      const keep = fresh.slice(skip);
      if (keep.length === 0) return prev;
      return appendStreamLines(
        prev,
        keep.map((line) => normalizeLogLine(line.raw)),
        keep.map((line) => new Date(line.at).toISOString()),
        tail
      );
    });
  }, [stream.lines, streaming, hasSnapshot, queryClient, queryKey, tail]);

  return query;
}
