import type { TranslateFn } from "@/lib/i18n";

export type SetupStatus = "pending" | "installing" | "installed" | "failed";

export type LocationSetupStep = {
  component: string;
  endpoint: string;
  labelKey: string;
  descriptionKey: string;
  required?: boolean;
};

export const LOCATION_SETUP_STEPS: LocationSetupStep[] = [
  {
    component: "packages",
    endpoint: "packages",
    labelKey: "admin.setup.step.packages",
    descriptionKey: "admin.setup.step.packages_desc",
    required: true,
  },
  {
    component: "docker",
    endpoint: "docker",
    labelKey: "admin.setup.step.docker",
    descriptionKey: "admin.setup.step.docker_desc",
    required: true,
  },
  {
    component: "mysql",
    endpoint: "mysql",
    labelKey: "admin.setup.step.mysql",
    descriptionKey: "admin.setup.step.mysql_desc",
  },
  {
    component: "phpmyadmin",
    endpoint: "phpmyadmin",
    labelKey: "admin.setup.step.phpmyadmin",
    descriptionKey: "admin.setup.step.phpmyadmin_desc",
  },
  {
    component: "ftp",
    endpoint: "ftp",
    labelKey: "admin.setup.step.ftp",
    descriptionKey: "admin.setup.step.ftp_desc",
  },
  {
    component: "quota",
    endpoint: "quota",
    labelKey: "admin.setup.step.quota",
    descriptionKey: "admin.setup.step.quota_desc",
  },
  {
    component: "daemon",
    endpoint: "daemon",
    labelKey: "admin.setup.step.daemon",
    descriptionKey: "admin.setup.step.daemon_desc",
    required: true,
  },
  {
    component: "images",
    endpoint: "images",
    labelKey: "admin.setup.step.images",
    descriptionKey: "admin.setup.step.images_desc",
  },
];

export function setupStatusLabel(t: TranslateFn, status?: string): string {
  if (status === "installed") return t("admin.setup.status.installed");
  if (status === "installing") return t("admin.setup.status.installing");
  if (status === "failed") return t("admin.setup.status.failed");
  return t("admin.setup.status.pending");
}

export function setupStatusVariant(
  status?: string
): "default" | "secondary" | "destructive" | "outline" {
  if (status === "installed") return "secondary";
  if (status === "installing") return "default";
  if (status === "failed") return "destructive";
  return "outline";
}
