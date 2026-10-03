export type RegistryGroup =
  | "security"
  | "billing"
  | "servers"
  | "notifications"
  | "retention"
  | "nodes"
  | "uploads"
  | "interface"
  | "accounting";

export type SettingsTab =
  | "main"
  | "mail"
  | "social"
  | "dockerhub"
  | "telegram"
  | "files"
  | RegistryGroup;

export type RegistryItem = {
  key: string;
  group: RegistryGroup;
  section: string;
  kind: "int" | "bool" | "string" | "enum" | "intlist" | "multi";
  default: string;
  min: number;
  max: number;
  unit: string;
  options?: string[];
  value: string;
  custom: boolean;
};

export type RegistryResponse = { items: RegistryItem[] };

export type AdminSettingsData = {
  values: Record<string, string>;
  panelVersion?: string;
  panel_version?: string;
};
