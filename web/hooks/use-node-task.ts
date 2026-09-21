"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useQueryClient, type QueryKey } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  apiErrorCode,
  fetchAgentTask,
  type NodeTask,
  type StartedTask,
} from "@/lib/api";
import { agentErrorText } from "@/lib/agents";

const POLL_MS = 1500;
const FINAL = new Set(["done", "failed", "expired"]);

export function useNodeTask<R = Record<string, unknown>>(
  nodeId: string,
  opts: {
    invalidate?: QueryKey[];
    successText?: string | ((task: NodeTask<R>) => string);
    failureText?: string;
    silent?: boolean;
    onDone?: (task: NodeTask<R>) => void;
  } = {}
) {
  const qc = useQueryClient();
  const [task, setTask] = useState<NodeTask<R> | null>(null);
  const [starting, setStarting] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const optsRef = useRef(opts);
  optsRef.current = opts;

  useEffect(() => () => {
    if (timer.current) clearTimeout(timer.current);
  }, []);

  const finish = useCallback(
    (done: NodeTask<R>) => {
      const o = optsRef.current;
      for (const key of o.invalidate ?? []) void qc.invalidateQueries({ queryKey: key });
      if (!o.silent) {
        if (done.status === "done") {
          const text = typeof o.successText === "function" ? o.successText(done) : o.successText;
          if (text) toast.success(text);
        } else {
          const base = done.error || o.failureText || "";
          toast.error(agentErrorText(done.error_code ?? "", base));
        }
      }
      o.onDone?.(done);
    },
    [qc]
  );

  const poll = useCallback(
    (taskId: string) => {
      const tick = async () => {
        try {
          const { task: next } = await fetchAgentTask<R>(nodeId, taskId);
          setTask(next);
          if (FINAL.has(next.status)) {
            finish(next);
            return;
          }
        } catch {
          return;
        }
        timer.current = setTimeout(tick, POLL_MS);
      };
      if (timer.current) clearTimeout(timer.current);
      timer.current = setTimeout(tick, 400);
    },
    [finish, nodeId]
  );

  const run = useCallback(
    async (start: () => Promise<StartedTask>) => {
      setStarting(true);
      try {
        const res = await start();
        setTask({
          id: res.task_id,
          action: "",
          method: res.method ?? "relay",
          status: "sent",
          created_at: new Date().toISOString(),
        });
        poll(res.task_id);
        return res;
      } catch (err) {
        const message = err instanceof Error ? err.message : "";
        toast.error(agentErrorText(apiErrorCode(err), message || optsRef.current.failureText || ""));
        return null;
      } finally {
        setStarting(false);
      }
    },
    [poll]
  );

  const attach = useCallback(
    (existing: NodeTask<R> | null | undefined) => {
      if (!existing || FINAL.has(existing.status)) return;
      if (task?.id === existing.id) return;
      setTask(existing);
      poll(existing.id);
    },
    [poll, task?.id]
  );

  const busy = starting || (task != null && !FINAL.has(task.status));
  return { task, busy, run, attach };
}
