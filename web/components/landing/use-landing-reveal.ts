"use client";

import { useEffect, useRef } from "react";

export function useLandingReveal<T extends HTMLElement>() {
  const ref = useRef<T>(null);

  useEffect(() => {
    const root = ref.current;
    if (!root) return;

    const targets = Array.from(
      root.querySelectorAll<HTMLElement>(
        "[data-reveal],[data-bar],[data-bar-h]"
      )
    );

    const show = (el: HTMLElement) => {
      el.dataset.shown = "true";
      const width = el.getAttribute("data-bar");
      if (width) el.style.width = `${width}%`;
      const height = el.getAttribute("data-bar-h");
      if (height) el.style.height = `${height}%`;
    };

    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (reduced || !("IntersectionObserver" in window)) {
      targets.forEach(show);
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue;
          show(entry.target as HTMLElement);
          observer.unobserve(entry.target);
        }
      },
      { rootMargin: "0px 0px -8% 0px", threshold: 0.12 }
    );

    targets.forEach((el) => observer.observe(el));
    const fallback = setTimeout(() => targets.forEach(show), 4000);

    return () => {
      observer.disconnect();
      clearTimeout(fallback);
    };
  }, []);

  return ref;
}
