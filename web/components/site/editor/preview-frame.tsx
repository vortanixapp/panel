"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Loader2 } from "lucide-react";
import { editorPreviewUrl, isMessage, type FrameMessage, type ParentMessage } from "@/lib/site/bridge";
import { cn } from "@/lib/utils";

export type Device = "phone" | "tablet" | "desktop";

const WIDTH: Record<Device, number | null> = { phone: 390, tablet: 820, desktop: null };

export type PreviewHandle = {
  post: (message: ParentMessage) => void;
  navigate: (path: string) => void;
};

export function PreviewFrame({
  initialPath,
  device,
  state,
  onMessage,
  onReady,
  handleRef,
}: {
  initialPath: string;
  device: Device;
  state: ParentMessage;
  onMessage: (message: FrameMessage) => void;
  onReady: () => void;
  handleRef: React.RefObject<PreviewHandle | null>;
}) {
  const frameRef = useRef<HTMLIFrameElement>(null);
  const readyRef = useRef(false);
  const [src, setSrc] = useState(() => editorPreviewUrl(initialPath));
  const [loading, setLoading] = useState(true);
  const stateRef = useRef(state);
  const onMessageRef = useRef(onMessage);
  const onReadyRef = useRef(onReady);

  useEffect(() => {
    onMessageRef.current = onMessage;
    onReadyRef.current = onReady;
  }, [onMessage, onReady]);

  const post = useCallback((message: ParentMessage) => {
    const target = frameRef.current?.contentWindow;
    if (!target || !readyRef.current) return;
    target.postMessage(message, window.location.origin);
  }, []);

  useEffect(() => {
    stateRef.current = state;
    post(state);
  }, [state, post]);

  useEffect(() => {
    handleRef.current = {
      post,
      navigate: (path: string) => {
        if (readyRef.current) {
          post({ type: "vx:navigate", path });
        } else {
          setLoading(true);
          setSrc(editorPreviewUrl(path));
        }
      },
    };
    return () => {
      handleRef.current = null;
    };
  }, [handleRef, post]);

  useEffect(() => {
    const onWindowMessage = (event: MessageEvent) => {
      if (event.origin !== window.location.origin) return;
      if (event.source !== frameRef.current?.contentWindow) return;
      if (!isMessage<FrameMessage>(event.data)) return;
      const message = event.data;
      if (message.type === "vx:ready") {
        readyRef.current = true;
        setLoading(false);
        post(stateRef.current);
        onReadyRef.current();
      }
      onMessageRef.current(message);
    };
    window.addEventListener("message", onWindowMessage);
    return () => window.removeEventListener("message", onWindowMessage);
  }, [post]);

  const width = WIDTH[device];

  return (
    <div className="relative flex min-h-0 flex-1 justify-center overflow-auto bg-muted/50 p-3">
      <div
        className={cn(
          "relative h-full shrink-0 overflow-hidden bg-background transition-[width] duration-300",
          width ? "rounded-[18px] border shadow-xl" : "w-full rounded-lg border"
        )}
        style={width ? { width } : undefined}
      >
        <iframe
          ref={frameRef}
          src={src}
          title="preview"
          className="size-full border-0"
          onLoad={() => {
            if (!readyRef.current) setLoading(false);
          }}
        />
        {loading ? (
          <div className="absolute inset-0 flex items-center justify-center bg-background/70">
            <Loader2 className="size-6 animate-spin text-muted-foreground" />
          </div>
        ) : null}
      </div>
    </div>
  );
}
