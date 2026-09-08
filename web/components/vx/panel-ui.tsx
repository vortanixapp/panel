"use client";

import { cn } from "@/lib/utils";

export const VX_PAGE_BG = "var(--vx-bg)";
export const VX_CARD = "border border-[var(--vx-border)] bg-[var(--vx-card)]";
export const VX_INSET = "border border-[var(--vx-border)] bg-[var(--vx-elevated)]";
export const VX_CODE = "border border-[var(--vx-inset)] bg-[var(--vx-code)]";

export const VX_MUTED = "text-[var(--vx-muted)]";
export const VX_FAINT = "text-[var(--vx-faint)]";
export const VX_MONO_LABEL =
  "font-mono text-[10.5px] tracking-[0.08em] uppercase text-[var(--vx-faint)]";

export const VX_ROW_LINE = "border-b border-[var(--vx-elevated)]";

export const VX_INPUT =
  "h-[34px] w-full rounded-[9px] border border-[var(--vx-border-2)] bg-[var(--vx-bg)] px-2.5 text-[12.5px] text-[var(--vx-fg)] outline-none transition-colors placeholder:text-[var(--vx-ghost)] focus:border-[var(--vx-border-hover)] disabled:cursor-not-allowed disabled:opacity-50";
export const VX_INPUT_MONO = cn(VX_INPUT, "font-mono");
export const VX_SELECT =
  "h-[34px] rounded-[9px] border border-[var(--vx-border-2)] bg-[var(--vx-bg)] px-2.5 text-[12.5px] text-[var(--vx-fg)] outline-none transition-colors focus:border-[var(--vx-border-hover)] disabled:cursor-not-allowed disabled:opacity-50";
export const VX_TEXTAREA =
  "w-full rounded-[10px] border border-[var(--vx-border-2)] bg-[var(--vx-code)] p-3 font-mono text-[12px] leading-[1.6] text-[var(--vx-dim)] outline-none transition-colors focus:border-[var(--vx-border-hover)] disabled:cursor-not-allowed disabled:opacity-50";

export type BtnTone = "default" | "primary" | "danger" | "ghost";
export type BtnSize = "sm" | "md";

const TONE: Record<BtnTone, string> = {
  default:
    "border-[var(--vx-border-2)] bg-[var(--vx-inset)] text-[var(--vx-fg)] hover:border-[var(--vx-border-strong)] disabled:hover:border-[var(--vx-border-2)]",
  primary:
    "border-[var(--vx-fg-strong)] bg-[var(--vx-fg-strong)] text-[var(--vx-on-fill)] hover:bg-white hover:border-white disabled:hover:bg-[var(--vx-fg-strong)] disabled:hover:border-[var(--vx-fg-strong)]",
  danger:
    "border-[rgba(224,122,122,0.35)] bg-[rgba(224,122,122,0.08)] text-[var(--vx-danger)] hover:border-[rgba(224,122,122,0.6)] disabled:hover:border-[rgba(224,122,122,0.35)]",
  ghost:
    "border-transparent bg-transparent text-[var(--vx-muted)] hover:text-[var(--vx-fg)] disabled:hover:text-[var(--vx-muted)]",
};

const SIZE: Record<BtnSize, string> = {
  sm: "h-[30px] px-3 rounded-[8px] text-[12px]",
  md: "h-[34px] px-3.5 rounded-[9px] text-[12.5px]",
};

export function btnClass(tone: BtnTone = "default", size: BtnSize = "md") {
  return cn(
    "inline-flex shrink-0 items-center justify-center gap-2 border font-medium whitespace-nowrap transition-colors disabled:cursor-not-allowed disabled:opacity-55",
    TONE[tone],
    SIZE[size]
  );
}

export function Btn({
  tone = "default",
  size = "md",
  className,
  type = "button",
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & {
  tone?: BtnTone;
  size?: BtnSize;
}) {
  return <button type={type} className={cn(btnClass(tone, size), className)} {...props} />;
}

export function Panel({
  title,
  aside,
  className,
  bodyClassName,
  children,
  flush = false,
}: {
  title?: React.ReactNode;
  aside?: React.ReactNode;
  className?: string;
  bodyClassName?: string;
  children?: React.ReactNode;
  flush?: boolean;
}) {
  return (
    <div className={cn("overflow-hidden rounded-[14px]", VX_CARD, className)}>
      {(title || aside) && (
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--vx-border)] px-[18px] py-[13px]">
          <span className="text-[12.5px] font-medium">{title}</span>
          {aside}
        </div>
      )}
      {children != null && (
        <div className={cn(flush ? undefined : "p-[18px]", bodyClassName)}>{children}</div>
      )}
    </div>
  );
}

export type SubTabItem = {
  id: string;
  title: string;
  /** Сколько полей раздела изменено и не сохранено. Ноль не показывается. */
  badge?: number;
};

/**
 * Ряд под-вкладок внутри одной вкладки сервера.
 *
 * Намеренно легче основного ряда из server-tab-shell: там кнопки с рамкой,
 * здесь только заливка у активной. Два одинаковых по весу ряда друг под другом
 * читались бы как один сломанный, и было бы неясно, что чему подчинено.
 */
export function SubTabs({
  items,
  active,
  onSelect,
  className,
}: {
  items: SubTabItem[];
  active: string;
  onSelect: (id: string) => void;
  className?: string;
}) {
  if (items.length < 2) return null;
  return (
    <div
      role="tablist"
      className={cn(
        "flex items-center gap-0.5 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden",
        className
      )}
    >
      {items.map((item) => {
        const on = item.id === active;
        return (
          <button
            key={item.id}
            type="button"
            role="tab"
            aria-selected={on}
            onClick={() => onSelect(item.id)}
            className={cn(
              "inline-flex h-[26px] shrink-0 items-center gap-1.5 rounded-[7px] px-2.5 text-[12px] transition-colors",
              on
                ? "bg-[var(--vx-tint)] font-medium text-[var(--vx-fg-strong)]"
                : "text-[var(--vx-muted)] hover:text-[var(--vx-fg)]"
            )}
          >
            {item.title}
            {item.badge ? (
              <span className="inline-flex h-[15px] min-w-[15px] items-center justify-center rounded-full bg-[var(--vx-fg-strong)] px-1 font-mono text-[9.5px] leading-none text-[var(--vx-on-fill)]">
                {item.badge}
              </span>
            ) : null}
          </button>
        );
      })}
    </div>
  );
}

export function InfoRow({ k, v }: { k: React.ReactNode; v: React.ReactNode }) {
  return (
    <div
      className={cn(
        "flex items-center justify-between gap-3 py-[9px] text-[12.5px] last:border-b-0",
        VX_ROW_LINE
      )}
    >
      <span className={VX_MUTED}>{k}</span>
      <span className="truncate text-right font-mono text-[12px] font-medium">{v}</span>
    </div>
  );
}

export function Toggle({
  checked,
  onChange,
  disabled,
  label,
}: {
  checked: boolean;
  onChange: (next: boolean) => void;
  disabled?: boolean;
  label?: string;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        "box-border inline-flex h-[19px] w-[34px] shrink-0 items-center rounded-full p-[2px] transition-colors disabled:cursor-not-allowed disabled:opacity-50",
        checked ? "bg-[var(--vx-fg-strong)]" : "bg-[var(--vx-tint-hover)]"
      )}
    >
      <span
        className={cn(
          "h-[15px] w-[15px] rounded-full transition-[margin] duration-150",
          checked ? "ml-[15px] bg-[var(--vx-bg)]" : "ml-0 bg-[var(--vx-muted)]"
        )}
      />
    </button>
  );
}

export function Field({
  label,
  className,
  children,
}: {
  label: React.ReactNode;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <label className={cn("grid gap-1.5 text-[11.5px]", VX_MUTED, className)}>
      {label}
      {children}
    </label>
  );
}

export function EmptyState({ children }: { children: React.ReactNode }) {
  return (
    <div className={cn("py-7 text-center text-[12.5px]", VX_FAINT)}>{children}</div>
  );
}

export function Notice({
  tone = "danger",
  className,
  children,
}: {
  tone?: "danger" | "warn";
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "rounded-[14px] px-[18px] py-3.5 text-[13px]",
        tone === "warn"
          ? "border border-[rgba(232,160,60,0.28)] bg-[rgba(232,160,60,0.06)] text-[var(--vx-warn)]"
          : "border border-[rgba(224,122,122,0.28)] bg-[rgba(224,122,122,0.07)] text-[var(--vx-danger)]",
        className
      )}
    >
      {children}
    </div>
  );
}

export function Bar({
  pct,
  className,
  barClassName,
}: {
  pct: number;
  className?: string;
  barClassName?: string;
}) {
  const width = Math.max(0, Math.min(100, Number.isFinite(pct) ? pct : 0));
  return (
    <div className={cn("h-[3px] overflow-hidden rounded-full bg-[var(--vx-inset)]", className)}>
      <div
        className={cn("h-full bg-[var(--vx-muted)] transition-[width] duration-500", barClassName)}
        style={{ width: `${width}%` }}
      />
    </div>
  );
}

export function Tile({
  label,
  value,
  sub,
  pct,
}: {
  label: string;
  value: React.ReactNode;
  sub: React.ReactNode;
  pct: number;
}) {
  return (
    <div className={cn("rounded-[14px] px-[18px] py-4", VX_CARD)}>
      <div className={cn(VX_MONO_LABEL, "tracking-[0.09em]")}>{label}</div>
      <div className="mt-2.5 font-mono text-[24px] font-medium tracking-[-0.02em] text-[var(--vx-fg)]">
        {value}
      </div>
      <div className={cn("mt-1 text-[11.5px]", VX_FAINT)}>{sub}</div>
      <Bar pct={pct} className="mt-3" />
    </div>
  );
}

export function TableHead({
  cols,
  className,
}: {
  cols: React.ReactNode[];
  className?: string;
}) {
  return (
    <div
      className={cn(
        "grid gap-3 py-2.5 text-[10.5px] tracking-[0.08em] uppercase",
        VX_ROW_LINE,
        VX_FAINT,
        className
      )}
    >
      {cols.map((c, i) => (
        <span key={i}>{c}</span>
      ))}
    </div>
  );
}
