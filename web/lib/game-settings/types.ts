/**
 * Схема настроек игрового сервера — зеркало пакета gamesettings на бэкенде.
 *
 * Раньше набор полей был описан на фронте отдельным списком (layouts.ts), и он
 * разошёлся с правилами валидации: у Minecraft бэкенд принимал 28 полей, панель
 * показывала 8, а поле, которого не было в правилах, при сохранении молча
 * выбрасывалось — с сообщением «Настройки сохранены».
 *
 * Здесь описания полей нет вовсе: панель получает схему от API и рисует по ней.
 * Добавление игры или поля больше не требует правки фронта.
 */

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
  /** Значение подхватывается без перезапуска. По умолчанию false. */
  applies_live?: boolean;
  /** Показывается, но не редактируется: порт назначает панель, слоты — тариф. */
  read_only?: boolean;
  /** Значение наружу не отдаётся, приходит вместо него SECRET_SENTINEL. */
  secret?: boolean;
  /** Поле числа игроков — при тарификации по слотам его задаёт тариф. */
  slots?: boolean;
  /** Поле разрешено очищать; для остальных пустая строка означает «не трогать». */
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
  /** Содержимое больше предела редактора — правится только по SFTP. */
  truncated?: boolean;
  sha256?: string;
  error?: string | null;
};

export type ServerSettingsSchema = {
  ok: boolean;
  game: string;
  profile?: string;
  /** Для игры описаны настройки. Иначе показываем причину из reason. */
  supported: boolean;
  reason?: string | null;
  /** Нода на связи. Если нет — значения показываем из базы и запрещаем запись. */
  node_online: boolean;
  sections: SettingsSection[];
  values: Record<string, string>;
  /** Откуда взято значение: файл, база или значение по умолчанию. Для подсказок. */
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

/**
 * Заглушка вместо значения секретного поля.
 *
 * Пароли RCON и администратора отдавались открытым текстом любому, у кого есть
 * право на просмотр настроек, хотя право на правку в системе отдельное. Теперь
 * наружу уходит эта строка, а обратно она означает «оставить как было».
 */
export const SECRET_SENTINEL = "__vtx_secret__";

/** Значение поля не задано и его нельзя показывать как настоящее. */
export function isSecretPlaceholder(value: string | undefined): boolean {
  return value === SECRET_SENTINEL;
}

/**
 * Раздел «Конфиг» и «Параметры запуска» панель добавляет сама — профиль игры их
 * не объявляет. Держим идентификаторы рядом со схемой, чтобы они не разъехались
 * с бэкендом.
 */
export const SECTION_RAW = "raw";
export const SECTION_STARTUP = "startup";
