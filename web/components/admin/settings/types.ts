export type SettingsTab =
  | "main"
  | "payments"
  | "mail"
  | "social"
  | "dockerhub"
  | "telegram"
  | "files";

export type PaymentProviderField = {
  label: string;
  type?: string;
  default?: string;
  options?: Record<string, string>;
};

export type PaymentProvider = {
  key: string;
  name: string;
  enabled: boolean;
  config: Record<string, string>;
  fields: Record<string, PaymentProviderField>;
};

export type AdminSettingsData = {
  values: Record<string, string>;
  paymentProviders?: Record<string, PaymentProvider> | PaymentProvider[];
  payment_providers?: Record<string, PaymentProvider> | PaymentProvider[];
  panelVersion?: string;
  panel_version?: string;
};
