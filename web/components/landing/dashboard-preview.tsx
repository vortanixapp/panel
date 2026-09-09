"use client";

import { cn } from "@/lib/utils";
import {
  Activity,
  HardDrive,
  LayoutDashboard,
  Server,
  Terminal,
  Users,
} from "lucide-react";
import { useT } from "@/hooks/use-translations";

export function DashboardPreview({ className }: { className?: string }) {
  const t = useT();
  return (
    <div
      className={cn(
        "relative overflow-hidden rounded-2xl border border-border/80 bg-muted/30 shadow-2xl shadow-primary/5 ring-1 ring-white/5",
        className
      )}
    >
      <div className="landing-shine pointer-events-none absolute inset-0 z-10" />
      <div className="flex items-center gap-2 border-b border-border/60 bg-card/90 px-4 py-3 backdrop-blur-sm">
        <div className="flex gap-1.5">
          <span className="h-2.5 w-2.5 rounded-full bg-red-500/90 shadow-[0_0_8px_rgba(239,68,68,0.5)]" />
          <span className="h-2.5 w-2.5 rounded-full bg-amber-400/90" />
          <span className="h-2.5 w-2.5 rounded-full bg-emerald-500/90" />
        </div>
        <div className="mx-auto flex h-7 items-center gap-2 rounded-md border border-border/50 bg-muted/50 px-3 text-[10px] text-muted-foreground">
          <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
          panel.vortanix.host
        </div>
      </div>

      <div className="flex min-h-[440px] bg-background/95">
        <aside className="relative hidden w-48 shrink-0 border-r border-border/60 bg-sidebar/95 p-3 sm:block">
          <div className="mb-5 flex items-center gap-2">
            <div className="h-6 w-6 rounded-md bg-primary/25" />
            <div className="h-2.5 w-20 rounded bg-foreground/10" />
          </div>
          <p className="mb-2 px-2 text-[9px] font-medium uppercase tracking-wider text-muted-foreground">
            {t("landing.preview.section_panel")}
          </p>
          <nav className="space-y-0.5">
            <PreviewNavItem icon={LayoutDashboard} active label={t("common.overview")} />
            <PreviewNavItem icon={Server} label={t("common.servers")} />
          </nav>
          <p className="mb-2 mt-4 px-2 text-[9px] font-medium uppercase tracking-wider text-muted-foreground">
            {t("layout.section.admin")}
          </p>
          <nav className="space-y-0.5">
            <PreviewNavItem icon={HardDrive} label={t("layout.nodes")} />
            <PreviewNavItem icon={Users} label={t("common.users")} />
          </nav>
          <div className="absolute bottom-3 left-3 right-3 rounded-lg border border-border/60 bg-card/80 p-2 backdrop-blur">
            <div className="mb-2 flex gap-1 rounded-md bg-muted/60 p-0.5 text-[8px]">
              <span className="flex-1 rounded bg-background py-0.5 text-center shadow-sm">
                User
              </span>
              <span className="flex-1 py-0.5 text-center text-muted-foreground">
                Admin
              </span>
            </div>
            <div className="flex items-center gap-2">
              <div className="h-7 w-7 rounded-full bg-primary/20 ring-1 ring-primary/30" />
              <div className="space-y-1">
                <div className="h-2 w-14 rounded bg-muted" />
                <div className="h-1.5 w-10 rounded bg-muted/60" />
              </div>
            </div>
          </div>
        </aside>

        <main className="flex-1 p-4 sm:p-5">
          <div className="mb-4 flex items-center justify-between gap-3">
            <div className="space-y-1.5">
              <div className="h-4 w-32 rounded-md bg-foreground/12" />
              <div className="h-2.5 w-48 rounded bg-muted" />
            </div>
            <div className="flex gap-2">
              <div className="h-8 w-8 rounded-md border border-border/60 bg-muted/40" />
              <div className="h-8 w-20 rounded-md bg-primary text-center text-[10px] font-medium leading-8 text-primary-foreground">
                {t("landing.preview.add_server")}
              </div>
            </div>
          </div>

          <div className="mb-4 grid gap-2.5 sm:grid-cols-3">
            <PreviewStat
              icon={Server}
              label={t("common.servers")}
              value="24"
              tone="text-primary"
            />
            <PreviewStat
              icon={Activity}
              label={t("landing.preview.online")}
              value="18"
              tone="text-emerald-600 dark:text-emerald-400"
            />
            <PreviewStat
              icon={HardDrive}
              label={t("layout.nodes")}
              value="5"
              tone="text-sky-600 dark:text-sky-400"
            />
          </div>

          <div className="grid gap-3 lg:grid-cols-5">
            <div className="rounded-xl border border-border/60 bg-card/80 p-3 lg:col-span-2">
              <p className="mb-2 text-[10px] font-medium text-muted-foreground">
                {t("admin.load_chart.title")}
              </p>
              <div className="flex h-24 items-end gap-1.5">
                {[40, 65, 45, 80, 55, 70, 50].map((h, i) => (
                  <div
                    key={i}
                    className="flex-1 rounded-t bg-primary/30"
                    style={{ height: `${h}%` }}
                  />
                ))}
              </div>
            </div>
            <div className="rounded-xl border border-border/60 bg-card/80 p-3 lg:col-span-3">
              <div className="mb-2 flex items-center gap-2 text-[10px] text-muted-foreground">
                <Terminal className="h-3.5 w-3.5 text-emerald-500/80" />
                <span>mc-survival-01 · live console</span>
                <span className="ms-auto rounded-full bg-emerald-500/15 px-1.5 py-0.5 text-[9px] text-emerald-600 dark:text-emerald-400">
                  running
                </span>
              </div>
              <div className="space-y-1 rounded-lg bg-[var(--vx-code)] p-3 font-mono text-[10px] leading-relaxed">
                <p className="text-emerald-600 dark:text-emerald-400/90">
                  [connected] agent relay · ws session ok
                </p>
                <p className="text-muted-foreground">
                  [14:22:01] Starting minecraft server version 1.21
                </p>
                <p className="text-muted-foreground">
                  [14:22:03] Done (3.2s)! For help, type &quot;help&quot;
                </p>
                <p className="text-muted-foreground">
                  [14:22:15] Player joined: Steve
                </p>
                <p className="text-primary animate-pulse">&gt; _</p>
              </div>
            </div>
          </div>
        </main>
      </div>

      <div className="pointer-events-none absolute inset-0 bg-gradient-to-t from-background/40 via-transparent to-transparent" />
    </div>
  );
}

function PreviewNavItem({
  icon: Icon,
  label,
  active,
}: {
  icon: React.ElementType;
  label: string;
  active?: boolean;
}) {
  return (
    <div
      className={cn(
        "flex items-center gap-2 rounded-md px-2 py-1.5 text-[11px] transition-colors",
        active
          ? "bg-sidebar-accent text-sidebar-accent-foreground shadow-sm"
          : "text-muted-foreground"
      )}
    >
      <Icon className="h-3.5 w-3.5" />
      <span>{label}</span>
    </div>
  );
}

function PreviewStat({
  icon: Icon,
  label,
  value,
  tone,
}: {
  icon: React.ElementType;
  label: string;
  value: string;
  tone: string;
}) {
  return (
    <div className="rounded-xl border border-border/50 bg-card/70 p-3 backdrop-blur-sm">
      <div className="mb-1 flex items-center justify-between">
        <p className="text-[10px] text-muted-foreground">{label}</p>
        <Icon className={cn("h-3.5 w-3.5 opacity-60", tone)} />
      </div>
      <p className={cn("text-xl font-bold tabular-nums", tone)}>{value}</p>
    </div>
  );
}
