"use client";

import { useId, useState, type ComponentProps, type ReactNode } from "react";
import { AnimatePresence, m } from "motion/react";
import { AlertCircle, ArrowRight, Check, Eye, EyeOff, Loader2 } from "lucide-react";
import { EASE_OUT } from "@/components/landing/motion";
import { TelegramLoginButton } from "@/components/auth/telegram-login-button";
import { useT } from "@/hooks/use-translations";
import type { TelegramAuthUser } from "@/lib/api";
import { cn } from "@/lib/utils";

export const SOCIAL_BUTTONS = [
  { key: "vk", label: "VK", icon: "ri-vk-line" },
  { key: "discord", label: "Discord", icon: "ri-discord-line" },
  { key: "google", label: "Google", icon: "ri-google-line" },
] as const;

export function AuthHeading({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div className="mb-8">
      <h1 className="vx-display text-[1.85rem] leading-[1.1] font-semibold tracking-[-0.03em] text-balance sm:text-[2.1rem]">
        {title}
      </h1>
      {subtitle && <p className="mt-3 text-[15px] leading-[1.55] text-muted-foreground">{subtitle}</p>}
      {children}
    </div>
  );
}

export function AuthError({ message }: { message: string }) {
  return (
    <AnimatePresence initial={false}>
      {message && (
        <m.div
          key="auth-error"
          initial={{ opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: "auto" }}
          exit={{ opacity: 0, height: 0 }}
          transition={{ duration: 0.35, ease: EASE_OUT }}
          className="overflow-hidden"
        >
          <m.div
            key={message}
            role="alert"
            initial={{ x: 0 }}
            animate={{ x: [0, -7, 6, -4, 3, 0] }}
            transition={{ duration: 0.45 }}
            className="mb-6 flex items-start gap-3 rounded-2xl border border-rose-500/25 bg-rose-500/[0.07] px-4 py-3.5 text-[14px] leading-[1.5] text-rose-600 dark:text-rose-300"
          >
            <AlertCircle className="mt-0.5 size-4 shrink-0" />
            <span className="min-w-0 break-words">{message}</span>
          </m.div>
        </m.div>
      )}
    </AnimatePresence>
  );
}

function FieldMessage({ id, error, hint }: { id: string; error?: string; hint?: ReactNode }) {
  return (
    <AnimatePresence initial={false} mode="popLayout">
      {error ? (
        <m.p
          key="error"
          id={id}
          initial={{ opacity: 0, y: -4 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: -4 }}
          transition={{ duration: 0.25 }}
          className="mt-2 text-[13px] text-rose-600 dark:text-rose-400"
        >
          {error}
        </m.p>
      ) : hint ? (
        <m.div
          key="hint"
          id={id}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.25 }}
          className="mt-2 text-[13px] text-muted-foreground"
        >
          {hint}
        </m.div>
      ) : null}
    </AnimatePresence>
  );
}

function inputClass(error?: string) {
  return cn(
    "h-12 w-full rounded-xl border bg-background px-4 text-[15px] text-foreground outline-none transition-[border-color,box-shadow] duration-200 placeholder:text-muted-foreground/60 disabled:opacity-60",
    error
      ? "border-rose-500/70 focus:border-rose-500 focus:shadow-[0_0_0_4px_rgba(244,63,94,0.12)]"
      : "border-border hover:border-foreground/30 focus:border-foreground focus:shadow-[0_0_0_4px_color-mix(in_oklab,var(--foreground)_10%,transparent)]"
  );
}

type FieldProps = Omit<ComponentProps<"input">, "id"> & {
  id: string;
  label: ReactNode;
  error?: string;
  hint?: ReactNode;
  aside?: ReactNode;
};

export function AuthField({ id, label, error, hint, aside, className, ...props }: FieldProps) {
  const messageId = `${id}-message`;
  return (
    <div className={className}>
      <div className="mb-2 flex items-baseline justify-between gap-3">
        <label htmlFor={id} className="text-[13.5px] font-medium">
          {label}
        </label>
        {aside}
      </div>
      <input
        id={id}
        aria-invalid={Boolean(error)}
        aria-describedby={error || hint ? messageId : undefined}
        className={inputClass(error)}
        {...props}
      />
      <FieldMessage id={messageId} error={error} hint={hint} />
    </div>
  );
}

export function PasswordField({
  id,
  label,
  error,
  hint,
  aside,
  className,
  strengthOf,
  ...props
}: FieldProps & { strengthOf?: string }) {
  const t = useT();
  const [visible, setVisible] = useState(false);
  const messageId = `${id}-message`;

  return (
    <div className={className}>
      <div className="mb-2 flex items-baseline justify-between gap-3">
        <label htmlFor={id} className="text-[13.5px] font-medium">
          {label}
        </label>
        {aside}
      </div>
      <div className="relative">
        <input
          id={id}
          type={visible ? "text" : "password"}
          aria-invalid={Boolean(error)}
          aria-describedby={error || hint ? messageId : undefined}
          className={cn(inputClass(error), "pr-12")}
          {...props}
        />
        <button
          type="button"
          onClick={() => setVisible((value) => !value)}
          aria-label={visible ? t("auth.kit.hide_password") : t("auth.kit.show_password")}
          aria-pressed={visible}
          className="absolute inset-y-0 right-1.5 my-auto flex size-9 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:text-foreground"
        >
          <AnimatePresence mode="popLayout" initial={false}>
            <m.span
              key={visible ? "hide" : "show"}
              initial={{ opacity: 0, scale: 0.6, rotate: -20 }}
              animate={{ opacity: 1, scale: 1, rotate: 0 }}
              exit={{ opacity: 0, scale: 0.6, rotate: 20 }}
              transition={{ duration: 0.2 }}
              className="flex"
            >
              {visible ? <EyeOff className="size-[18px]" /> : <Eye className="size-[18px]" />}
            </m.span>
          </AnimatePresence>
        </button>
      </div>
      {strengthOf !== undefined && <PasswordStrength value={strengthOf} />}
      <FieldMessage id={messageId} error={error} hint={hint} />
    </div>
  );
}

function passwordScore(value: string) {
  if (!value) return 0;
  let score = 0;
  if (value.length >= 8) score++;
  if (value.length >= 12) score++;
  if (/[a-zа-яё]/.test(value) && /[A-ZА-ЯЁ]/.test(value)) score++;
  if (/\d/.test(value)) score++;
  if (/[^A-Za-zА-Яа-яЁё0-9]/.test(value)) score++;
  if (value.length < 8) return 1;
  return Math.min(4, Math.max(1, score));
}

const STRENGTH_TONE = ["bg-muted", "bg-rose-500", "bg-amber-500", "bg-lime-500", "bg-emerald-500"];

function PasswordStrength({ value }: { value: string }) {
  const t = useT();
  const score = passwordScore(value);
  const labels = [
    "",
    t("auth.kit.strength_weak"),
    t("auth.kit.strength_fair"),
    t("auth.kit.strength_good"),
    t("auth.kit.strength_strong"),
  ];

  return (
    <AnimatePresence initial={false}>
      {value && (
        <m.div
          initial={{ opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: "auto" }}
          exit={{ opacity: 0, height: 0 }}
          transition={{ duration: 0.3, ease: EASE_OUT }}
          className="overflow-hidden"
        >
          <div className="flex items-center gap-3 pt-2.5">
            <div className="grid flex-1 grid-cols-4 gap-1">
              {[1, 2, 3, 4].map((step) => (
                <div key={step} className="h-1 overflow-hidden rounded-full bg-muted">
                  <m.div
                    className={cn("h-full w-full origin-left rounded-full", STRENGTH_TONE[score])}
                    initial={false}
                    animate={{ scaleX: score >= step ? 1 : 0 }}
                    transition={{ duration: 0.35, ease: EASE_OUT, delay: score >= step ? step * 0.04 : 0 }}
                  />
                </div>
              ))}
            </div>
            <span className="w-[76px] text-right text-[12px] text-muted-foreground">{labels[score]}</span>
          </div>
        </m.div>
      )}
    </AnimatePresence>
  );
}

export function AuthCheckbox({
  children,
  className,
  ...props
}: Omit<ComponentProps<"input">, "type" | "children"> & { children: ReactNode }) {
  const generated = useId();
  const id = props.id ?? generated;
  return (
    <label htmlFor={id} className={cn("group flex cursor-pointer items-start gap-3", className)}>
      <span className="relative mt-[1px] flex size-[18px] shrink-0">
        <input id={id} type="checkbox" className="peer absolute inset-0 size-full cursor-pointer opacity-0" {...props} />
        <span className="pointer-events-none flex size-full items-center justify-center rounded-[6px] border border-border bg-background text-background transition-[background-color,border-color] duration-200 group-hover:border-foreground/40 peer-checked:border-foreground peer-checked:bg-foreground peer-focus-visible:ring-2 peer-focus-visible:ring-foreground/30 [&>svg]:scale-50 [&>svg]:opacity-0 [&>svg]:transition-[opacity,transform] [&>svg]:duration-200 peer-checked:[&>svg]:scale-100 peer-checked:[&>svg]:opacity-100">
          <Check className="size-3" strokeWidth={3} />
        </span>
      </span>
      <span className="text-[14px] leading-[1.45] text-muted-foreground transition-colors group-hover:text-foreground">
        {children}
      </span>
    </label>
  );
}

export function AuthSubmit({
  loading,
  disabled,
  children,
}: {
  loading: boolean;
  disabled?: boolean;
  children: ReactNode;
}) {
  return (
    <m.button
      type="submit"
      disabled={loading || disabled}
      whileTap={{ scale: 0.98 }}
      className="group relative flex h-12 w-full items-center justify-center gap-2.5 overflow-hidden rounded-full bg-primary px-6 text-[15px] font-semibold text-primary-foreground transition-opacity disabled:cursor-not-allowed disabled:opacity-60"
    >
      <AnimatePresence mode="popLayout" initial={false}>
        {loading ? (
          <m.span
            key="loading"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            className="flex items-center gap-2.5"
          >
            <Loader2 className="size-4 animate-spin" />
            {children}
          </m.span>
        ) : (
          <m.span
            key="idle"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            className="flex items-center gap-2.5"
          >
            {children}
            <ArrowRight className="size-4 transition-transform duration-300 group-hover:translate-x-1" />
          </m.span>
        )}
      </AnimatePresence>
    </m.button>
  );
}

export function AuthSocial({
  providers,
  telegramBot,
  onProvider,
  onTelegram,
}: {
  providers: readonly (typeof SOCIAL_BUTTONS)[number][];
  telegramBot: string;
  onProvider: (provider: string) => void;
  onTelegram: (user: TelegramAuthUser) => void;
}) {
  const t = useT();
  if (providers.length === 0 && !telegramBot) return null;

  return (
    <div className="mt-8">
      <div className="flex items-center gap-4 text-[12.5px] text-muted-foreground">
        <span className="h-px flex-1 bg-border" />
        {t("auth.or")}
        <span className="h-px flex-1 bg-border" />
      </div>
      {providers.length > 0 && (
        <div
          className={cn(
            "mt-6 grid gap-2.5",
            providers.length === 1 ? "grid-cols-1" : providers.length === 2 ? "grid-cols-2" : "grid-cols-3"
          )}
        >
          {providers.map((provider) => (
            <m.button
              key={provider.key}
              type="button"
              whileTap={{ scale: 0.97 }}
              onClick={() => onProvider(provider.key)}
              className="flex h-11 items-center justify-center gap-2 rounded-full border border-border text-[14px] font-medium transition-colors hover:border-foreground/40 hover:bg-accent"
            >
              <i className={cn(provider.icon, "text-[17px]")} />
              {provider.label}
            </m.button>
          ))}
        </div>
      )}
      {telegramBot && (
        <div className="mt-3 flex justify-center">
          <TelegramLoginButton botUsername={telegramBot} onAuth={onTelegram} />
        </div>
      )}
    </div>
  );
}

export function AuthSwitch({ children }: { children: ReactNode }) {
  return <p className="mt-10 text-center text-[14px] text-muted-foreground">{children}</p>;
}
