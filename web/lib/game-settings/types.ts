export type FieldKind = "string" | "text" | "int" | "float" | "bool" | "enum";

export type SettingOption = {
  value: string;
  label: string;
};

export type SettingField = {
  key: string;
  label: string;
  hint?: string;
  section: string;
  kind: FieldKind;
  options?: SettingOption[];
  default?: string;
  min?: number;
  max?: number;
  max_len?: number;
  applies_live?: boolean;
  read_only?: boolean;
  secret?: boolean;
  slots?: boolean;
  clearable?: boolean;
};

export type SettingsSection = {
  id: string;
  title: string;
  hint?: string;
  fields: SettingField[];
};

export type SettingsFile = {
  id: string;
  path: string;
  title: string;
  format: string;
  exists: boolean;
  size?: number;
  editable: boolean;
  truncated?: boolean;
  sha256?: string;
  error?: string | null;
};

export type ServerSettingsSchema = {
  ok: boolean;
  game: string;
  profile?: string;
  supported: boolean;
  reason?: string | null;
  node_online: boolean;
  sections: SettingsSection[];
  values: Record<string, string>;
  value_sources?: Record<string, "file" | "stored" | "default">;
  files: SettingsFile[];
  note?: string;
  warnings?: string[];
};

export type SaveSettingsResponse = {
  ok: boolean;
  saved: string[];
  files_written: string[];
  restart_required: boolean;
  restart_fields: string[];
  server_running: boolean;
  message?: string;
};

export type SettingsFileContent = {
  ok: boolean;
  id: string;
  path: string;
  content: string;
  exists: boolean;
  size: number;
  sha256: string;
  truncated: boolean;
  editable: boolean;
};

export const SECRET_SENTINEL = "__vtx_secret__";

export function isSecretPlaceholder(value: string | undefined): boolean {
  return value === SECRET_SENTINEL;
}

export const SECTION_RAW = "raw";
export const SECTION_STARTUP = "startup";
