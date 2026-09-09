"use client";

import { useEffect, useRef } from "react";
import type { TelegramAuthUser } from "@/lib/api";

declare global {
  interface Window {
    __vortanixTelegramAuth?: (user: TelegramAuthUser) => void;
  }
}

type Props = {
  botUsername: string;
  onAuth: (user: TelegramAuthUser) => void;
  buttonSize?: "large" | "medium" | "small";
  className?: string;
};

export function TelegramLoginButton({
  botUsername,
  onAuth,
  buttonSize = "medium",
  className,
}: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const onAuthRef = useRef(onAuth);
  onAuthRef.current = onAuth;

  useEffect(() => {
    const container = containerRef.current;
    if (!container || !botUsername) return;

    window.__vortanixTelegramAuth = (user: TelegramAuthUser) => {
      onAuthRef.current(user);
    };

    container.innerHTML = "";
    const script = document.createElement("script");
    script.src = "https://telegram.org/js/telegram-widget.js?22";
    script.async = true;
    script.setAttribute("data-telegram-login", botUsername);
    script.setAttribute("data-size", buttonSize);
    script.setAttribute("data-userpic", "false");
    script.setAttribute("data-radius", "8");
    script.setAttribute("data-onauth", "__vortanixTelegramAuth(user)");
    container.appendChild(script);

    return () => {
      container.innerHTML = "";
      delete window.__vortanixTelegramAuth;
    };
  }, [botUsername, buttonSize]);

  return <div ref={containerRef} className={className} />;
}
