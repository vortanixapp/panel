import { t } from "@/lib/i18n";

export function isAdminRole(role: string | undefined): boolean {
  return role === "owner" || role === "admin";
}

export function isOwnerRole(role: string | undefined): boolean {
  return role === "owner";
}

export function isStaffRole(role: string | undefined): boolean {
  return isAdminRole(role) || role === "support";
}

export function roleLabel(role: string): string {
  switch (role) {
    case "owner":
      return t("layout.role.owner");
    case "admin":
      return t("layout.role.admin");
    case "support":
      return t("layout.role.support");
    case "user":
      return t("layout.role.user");
    default:
      return role;
  }
}
