import type { DashboardServer } from "@/lib/api";
import { t, type TranslateFn } from "@/lib/i18n";

export type ServerStatusInfo = {
  label: string;
  cls: string;
  icon: string;
  category: ServerStatusCategory;
};

export type ServerStatusCategory =
  | "starting"
  | "stopping"
  | "installing"
  | "reinstalling"
  | "updating"
  | "failed"
  | "running"
  | "stopped"
  | "missing"
  | "suspended"
  | "active"
  | "unknown";

// Списки собираются функцией, а не константой: константа считается один раз при
// импорте модуля и подписи в ней застыли бы на языке, который был при загрузке.
export function serverStatusFilterOptions(t: TranslateFn): {
  value: ServerStatusCategory | "all";
  label: string;
}[] {
  return [
    { value: "all", label: t("servers.filter.status_all") },
    { value: "running", label: t("servers.status.running") },
    { value: "stopped", label: t("servers.status.stopped") },
    { value: "installing", label: t("servers.status.installing") },
    { value: "reinstalling", label: t("servers.status.reinstalling") },
    { value: "updating", label: t("servers.status.updating") },
    { value: "failed", label: t("servers.status.failed") },
    { value: "missing", label: t("servers.status.missing") },
    { value: "suspended", label: t("servers.status.suspended") },
    { value: "active", label: t("servers.status.active") },
  ];
}

export type ServerSortOption =
  | "name-asc"
  | "name-desc"
  | "expires-asc"
  | "expires-desc"
  | "status-asc"
  | "status-desc";

export function serverSortOptions(
  t: TranslateFn
): { value: ServerSortOption; label: string }[] {
  return [
    { value: "name-asc", label: t("servers.sort.name_asc") },
    { value: "name-desc", label: t("servers.sort.name_desc") },
    { value: "expires-asc", label: t("servers.sort.expires_asc") },
    { value: "expires-desc", label: t("servers.sort.expires_desc") },
    { value: "status-asc", label: t("servers.sort.status_asc") },
    { value: "status-desc", label: t("servers.sort.status_desc") },
  ];
}

export function getServerStatus(server: {
  status?: string;
  runtime_status?: string;
  provisioning_status?: string;
}): ServerStatusInfo {
  const prov = (server.provisioning_status || "").toLowerCase();
  const runtime = (server.runtime_status || "").toLowerCase();
  const status = (server.status || "").toLowerCase();

  if (["pending", "installing", "provisioning"].includes(prov) && status !== "reinstalling") {
    return {
      label: t("servers.status.installing"),
      cls: "bg-amber-500/10 text-amber-500",
      icon: "ri-loader-4-line animate-spin",
      category: "installing",
    };
  }
  // Переустановка помечается в status, а не в provisioning_status: последний
  // ограничен схемой набором ('pending','provisioning','ready','failed',
  // 'deprovisioning','migrating'), и значения 'reinstalling' там быть не может.
  // Проверка только по provisioning_status делала эту метку недостижимой, и во
  // время переустановки панель показывала обычную «Установку».
  if (prov === "reinstalling" || status === "reinstalling") {
    return {
      label: t("servers.status.reinstalling"),
      cls: "bg-amber-500/10 text-amber-500",
      icon: "ri-loader-4-line animate-spin",
      category: "reinstalling",
    };
  }
  if (prov === "updating" || status === "updating") {
    return {
      label: t("servers.status.updating"),
      cls: "bg-amber-500/10 text-amber-500",
      icon: "ri-loader-4-line animate-spin",
      category: "updating",
    };
  }
  if (prov === "failed") {
    return {
      label: t("servers.status.failed"),
      cls: "bg-rose-500/10 text-rose-500",
      icon: "ri-error-warning-line",
      category: "failed",
    };
  }
  // Переходные состояния объявлены до «работает»: без них starting и stopping
  // проваливались в самый конец и показывались как «Неизвестно» — то есть при
  // обычном выключении сервер на несколько секунд становился непонятно чем.
  if (runtime === "starting" || status === "starting") {
    return {
      label: t("servers.status.starting"),
      cls: "bg-amber-500/10 text-amber-500",
      icon: "ri-loader-4-line animate-spin",
      category: "starting",
    };
  }
  if (runtime === "stopping" || status === "stopping") {
    return {
      label: t("servers.status.stopping"),
      cls: "bg-amber-500/10 text-amber-500",
      icon: "ri-loader-4-line animate-spin",
      category: "stopping",
    };
  }
  if (runtime === "running" || status === "running") {
    return {
      label: t("servers.status.running"),
      cls: "bg-emerald-500/10 text-emerald-500",
      icon: "ri-checkbox-circle-line",
      category: "running",
    };
  }
  if (["offline", "stopped"].includes(runtime) || status === "stopped") {
    return {
      label: t("servers.status.stopped"),
      cls: "bg-muted text-muted-foreground",
      icon: "ri-stop-circle-line",
      category: "stopped",
    };
  }
  if (runtime === "missing") {
    return {
      label: t("servers.status.missing"),
      cls: "bg-rose-500/10 text-rose-500",
      icon: "ri-close-circle-line",
      category: "missing",
    };
  }
  if (status === "suspended") {
    return {
      label: t("servers.status.suspended"),
      cls: "bg-amber-500/10 text-amber-500",
      icon: "ri-stop-circle-line",
      category: "suspended",
    };
  }
  if (status === "active") {
    return {
      label: t("servers.status.active"),
      cls: "bg-emerald-500/10 text-emerald-500",
      icon: "ri-checkbox-circle-line",
      category: "active",
    };
  }
  return {
    label: t("servers.status.unknown"),
    cls: "bg-muted text-muted-foreground",
    icon: "ri-question-line",
    category: "unknown",
  };
}

export function getServerStatusDotClass(st: ServerStatusInfo): string {
  if (st.cls.includes("emerald")) return "bg-emerald-500";
  if (st.cls.includes("amber")) return "bg-amber-500";
  if (st.cls.includes("rose")) return "bg-rose-500";
  return "bg-muted-foreground";
}

export function filterAndSortServers<
  T extends {
    name: string;
    expires_at: string | null;
    game?: { name: string } | null;
    status?: string;
    runtime_status?: string;
    provisioning_status?: string;
    ip_address?: string;
    port?: number;
  },
>(
  servers: T[],
  opts: {
    search: string;
    statusFilter: ServerStatusCategory | "all";
    gameFilter: string;
    sort: ServerSortOption;
  }
): T[] {
  const q = opts.search.trim().toLowerCase();
  let list = servers.filter((server) => {
    if (q) {
      const haystack = [
        server.name,
        server.ip_address ?? "",
        server.port != null ? String(server.port) : "",
      ]
        .join(" ")
        .toLowerCase();
      if (!haystack.includes(q)) return false;
    }
    if (opts.statusFilter !== "all") {
      if (getServerStatus(server).category !== opts.statusFilter) return false;
    }
    if (opts.gameFilter !== "all") {
      const gameName = server.game?.name || "";
      if (gameName !== opts.gameFilter) return false;
    }
    return true;
  });

  const dir = opts.sort.endsWith("-desc") ? -1 : 1;
  list = [...list].sort((a, b) => {
    switch (opts.sort) {
      case "name-asc":
      case "name-desc":
        return a.name.localeCompare(b.name, "ru") * dir;
      case "expires-asc":
      case "expires-desc": {
        const aTime = a.expires_at ? new Date(a.expires_at).getTime() : Infinity;
        const bTime = b.expires_at ? new Date(b.expires_at).getTime() : Infinity;
        return (aTime - bTime) * dir;
      }
      case "status-asc":
      case "status-desc": {
        const aLabel = getServerStatus(a).label;
        const bLabel = getServerStatus(b).label;
        return aLabel.localeCompare(bLabel, "ru") * dir;
      }
      default:
        return 0;
    }
  });

  return list;
}

export function canStartServer(server: DashboardServer) {
  const prov = (server.provisioning_status || "").toLowerCase();
  const runtime = (server.runtime_status || "").toLowerCase();
  const status = (server.status || "").toLowerCase();
  return (
    !["pending", "installing", "provisioning", "reinstalling", "updating"].includes(prov) &&
    prov !== "failed" &&
    (["offline", "stopped"].includes(runtime) || status === "stopped")
  );
}

export function canStopServer(server: DashboardServer) {
  const prov = (server.provisioning_status || "").toLowerCase();
  const runtime = (server.runtime_status || "").toLowerCase();
  const status = (server.status || "").toLowerCase();
  return (
    !["pending", "installing", "provisioning", "reinstalling", "updating"].includes(prov) &&
    prov !== "failed" &&
    (runtime === "running" || status === "running")
  );
}

/**
 * Состояние переходное — сервер вот-вот станет чем-то другим.
 *
 * Пока оно длится, действия питания недоступны, но ряд кнопок должен оставаться
 * на месте: раньше все они исчезали разом и появлялись заново через несколько
 * секунд, из-за чего раскладка прыгала дважды за одну операцию.
 */
export function isTransitionalStatus(category: string): boolean {
  return ["starting", "stopping", "installing", "reinstalling", "updating"].includes(category);
}
