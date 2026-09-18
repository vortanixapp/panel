"use client";

import dynamic from "next/dynamic";
import { useEffect, useState } from "react";
import { isEditorFrame } from "@/lib/site/frame";

const EditorBridge = dynamic(() => import("@/components/site/editor-bridge/bridge"), { ssr: false });

export function EditorBridgeLoader() {
  const [enabled, setEnabled] = useState(false);

  useEffect(() => {
    setEnabled(isEditorFrame());
  }, []);

  return enabled ? <EditorBridge /> : null;
}
