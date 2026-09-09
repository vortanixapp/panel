export type PanelBasePath = "" | "/admin";
export type PanelVariant = "user" | "admin";

export function panelPath(base: PanelBasePath, path: string): string {
  if (!path.startsWith("/")) {
    return `${base}/${path}`;
  }
  return `${base}${path}`;
}

export function serversListPath(base: PanelBasePath): string {
  return panelPath(base, "/servers");
}

export function serverPath(
  base: PanelBasePath,
  id: string,
  suffix = ""
): string {
  return panelPath(base, `/servers/${id}${suffix}`);
}

export function dashboardPath(variant: PanelVariant): string {
  return variant === "admin" ? "/admin/dashboard" : "/dashboard";
}

export function variantToBasePath(variant: PanelVariant): PanelBasePath {
  return variant === "admin" ? "/admin" : "";
}

export function isAdminSection(pathname: string): boolean {
  return pathname.startsWith("/admin");
}
