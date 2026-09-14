export type SettingsTab =
  | "main"
  | "mail"
  | "social"
  | "dockerhub"
  | "telegram"
  | "files";

export type AdminSettingsData = {
  values: Record<string, string>;
  panelVersion?: string;
  panel_version?: string;
};
