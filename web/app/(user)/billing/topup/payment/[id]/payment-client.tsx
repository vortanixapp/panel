"use client";

import { BillingTopupPaymentContent } from "@/components/user/billing-topup-payment-content";

export function PaymentClient({ id }: { id: string }) {
  return <BillingTopupPaymentContent id={id} />;
}
