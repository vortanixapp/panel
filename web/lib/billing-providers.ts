import { getLocale, t } from "@/lib/i18n";

export type BillingProviderDef = {
  key: string;
  name: string;
  /** Ключ описания: сам текст берётся при отрисовке, иначе застынет на языке загрузки. */
  descKey: string;
};

export const BILLING_PROVIDERS: BillingProviderDef[] = [
  { key: "stripe", name: "Stripe", descKey: "billing.provider.desc.cards_apple_pay" },
  { key: "paypal", name: "PayPal", descKey: "billing.provider.desc.paypal_balance" },
  { key: "yookassa", name: "ЮKassa", descKey: "billing.provider.desc.cards_ru_sbp" },
  { key: "robokassa", name: "RoboKassa", descKey: "billing.provider.desc.cards_wallets" },
  { key: "freekassa", name: "FreeKassa", descKey: "billing.provider.desc.crypto" },
  { key: "nowpayments", name: "NOWPayments", descKey: "billing.provider.desc.crypto" },
  { key: "coinbase", name: "Coinbase", descKey: "billing.provider.desc.crypto" },
  { key: "cryptocloud", name: "CryptoCloud", descKey: "billing.provider.desc.crypto" },
  { key: "yoomoney", name: "ЮMoney", descKey: "billing.provider.desc.yoomoney_wallet" },
  { key: "cloudpayments", name: "CloudPayments", descKey: "billing.provider.desc.cards_ru" },
  { key: "lava", name: "Lava", descKey: "billing.provider.desc.cards_wallets" },
  { key: "unitpay", name: "Unitpay", descKey: "billing.provider.desc.cards_wallets" },
  { key: "razorpay", name: "Razorpay", descKey: "billing.provider.desc.cards_india" },
  { key: "paystack", name: "Paystack", descKey: "billing.provider.desc.cards_africa" },
  { key: "flutterwave", name: "Flutterwave", descKey: "billing.provider.desc.cards_africa" },
  { key: "square", name: "Square", descKey: "billing.provider.desc.cards_usa" },
  { key: "webpay", name: "WebPay", descKey: "billing.provider.desc.cards_payments" },
  { key: "webmoney", name: "WebMoney", descKey: "billing.provider.desc.webmoney_wallets" },
  { key: "paymaster", name: "PayMaster", descKey: "billing.provider.desc.cards_wallets" },
  { key: "advcash", name: "AdvCash", descKey: "billing.provider.desc.wallets_cards" },
  { key: "payeer", name: "Payeer", descKey: "billing.provider.desc.payeer_wallet" },
  { key: "perfectmoney", name: "Perfect Money", descKey: "billing.provider.desc.pm_wallets" },
  { key: "enot", name: "Enot", descKey: "billing.provider.desc.cards_wallets" },
  { key: "t_kassa", name: "Т‑Касса", descKey: "billing.provider.desc.cards_ru" },
  { key: "mollie", name: "Mollie", descKey: "billing.provider.desc.eu_payments" },
  { key: "payfast", name: "PayFast", descKey: "billing.provider.desc.zar_payments" },
  { key: "authorizenet", name: "Authorize.Net", descKey: "billing.provider.desc.cards" },
  { key: "paddle", name: "Paddle", descKey: "billing.provider.desc.subscriptions" },
  { key: "cryptocom", name: "Crypto.com", descKey: "billing.provider.desc.crypto" },
  { key: "bank", name: "Bank Transfer", descKey: "billing.provider.desc.manual" },
];

/** Описание провайдера на текущем языке. */
export function providerDescription(code: string): string {
  const def = BILLING_PROVIDERS.find((p) => p.key === code);
  return def ? t(def.descKey) : "";
}

export const AVAILABLE_WALLET_CURRENCIES = ["RUB", "USD", "EUR", "UAH"] as const;

export function providerDisplayName(code: string): string {
  return BILLING_PROVIDERS.find((p) => p.key === code)?.name ?? code;
}

export function paymentStatusBadgeCls(status: string): string {
  if (status === "succeeded" || status === "completed" || status === "success") {
    return "bg-emerald-500/10 text-emerald-500";
  }
  if (status === "failed" || status === "cancelled") {
    return "bg-rose-500/10 text-rose-500";
  }
  return "bg-amber-500/10 text-amber-500";
}

export function formatBillingDate(value: string | null | undefined): string {
  if (!value) return "—";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return "—";
  return d
    .toLocaleString(getLocale() === "en" ? "en-GB" : "ru", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    })
    .replace(",", "");
}
