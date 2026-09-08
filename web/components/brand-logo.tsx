"use client";

import Image from "next/image";
import { useBrand } from "@/context/brand-provider";
import {
  DEFAULT_BRAND,
  DEFAULT_BRAND_LOGO_URL,
  DEFAULT_BRAND_MARK_URL,
} from "@/lib/brand";
import { cn } from "@/lib/utils";

export function BrandLogo({
  className,
  accentClassName,
  size = "md",
  priority = false,
}: {
  className?: string;
  accentClassName?: string;
  size?: "sm" | "md" | "lg";
  priority?: boolean;
}) {
  const { name, logoUrl } = useBrand();
  const resolvedLogoUrl =
    logoUrl || (name === DEFAULT_BRAND ? DEFAULT_BRAND_LOGO_URL : "");
  const textSize =
    size === "sm"
      ? "text-base tracking-[0.2em]"
      : size === "lg"
        ? "text-2xl tracking-[0.22em]"
        : "text-xl tracking-[0.2em]";

  const imageSize =
    size === "sm"
      ? { width: 216, height: 45, className: "h-8" }
      : size === "lg"
        ? { width: 336, height: 70, className: "h-14" }
        : { width: 264, height: 55, className: "h-10" };

  const isDefaultBrandLogo =
    resolvedLogoUrl === DEFAULT_BRAND_LOGO_URL && !logoUrl;

  if (resolvedLogoUrl && !isDefaultBrandLogo) {
    return (
      <Image
        src={resolvedLogoUrl}
        alt={name}
        width={imageSize.width}
        height={imageSize.height}
        className={cn(imageSize.className, "w-auto object-contain", className)}
        priority={priority}
        unoptimized
      />
    );
  }

  const accent = accentClassName ?? "text-current";

  if (isDefaultBrandLogo) {
    const markSize = size === "sm" ? 28 : size === "lg" ? 48 : 36;
    return (
      <div className={cn("flex items-center gap-2", className)} aria-label={name}>
        <Image
          src={DEFAULT_BRAND_MARK_URL}
          alt=""
          width={markSize * 2}
          height={markSize * 2}
          className="w-auto shrink-0 object-contain"
          style={{ height: markSize }}
          priority={priority}
          unoptimized
        />
        <span
          className={cn(
            "font-bold whitespace-nowrap text-foreground uppercase",
            textSize
          )}
        >
          {name.slice(0, -2)}
          <span className={accent}>{name.slice(-2)}</span>
        </span>
      </div>
    );
  }

  return (
    <div
      className={cn("font-bold uppercase text-foreground", textSize, className)}
      aria-label={name}
    >
      {name.endsWith("IX") && name.length > 2 ? (
        <>
          {name.slice(0, -2)}
          <span className={accent}>IX</span>
        </>
      ) : (
        name
      )}
    </div>
  );
}

export function BrandMark({
  className,
  priority = false,
}: {
  className?: string;
  priority?: boolean;
}) {
  const { name, logoUrl } = useBrand();
  const resolvedMarkUrl =
    logoUrl || (name === DEFAULT_BRAND ? DEFAULT_BRAND_MARK_URL : "");

  if (!resolvedMarkUrl) {
    return (
      <span
        className={cn(
          "inline-flex size-8 items-center justify-center rounded-lg bg-primary font-bold text-primary-foreground",
          className
        )}
        aria-label={name}
      >
        {name.slice(0, 1)}
      </span>
    );
  }

  return (
    <Image
      src={resolvedMarkUrl}
      alt={`${name} logo`}
      width={48}
      height={48}
      className={cn("size-8 object-contain", className)}
      priority={priority}
      unoptimized
    />
  );
}
