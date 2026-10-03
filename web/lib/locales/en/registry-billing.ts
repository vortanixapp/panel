export const registryBilling = {
  "admin.settings.reg.group.billing":
    "Payment reminders, what happens to overdue servers, rental and top-up limits. Values apply immediately, no restart needed.",

  "admin.settings.reg.section.billing.reminders": "Reminders and renewal",
  "admin.settings.reg.section.billing.expiry": "Overdue servers",
  "admin.settings.reg.section.billing.expiry.description":
    "By default a server stops right after the paid period ends, and paid servers are never deleted.",
  "admin.settings.reg.section.billing.rental": "Rental",
  "admin.settings.reg.section.billing.payments": "Top-ups and payment providers",
  "admin.settings.reg.section.billing.bonuses": "Bonuses and data retention",

  "admin.settings.reg.billing.dunning_stages_days": "Remind before expiry",
  "admin.settings.reg.billing.dunning_stages_days.hint":
    "Days before expiry, comma-separated, for example 1, 3, 7",
  "admin.settings.reg.billing.autorenew_horizon_hours": "Auto-renew ahead of expiry",
  "admin.settings.reg.billing.autorenew_horizon_hours.hint":
    "A server with auto-renew on is renewed this long before it ends",
  "admin.settings.reg.billing.expiring_soon_days": "“Expiring soon” threshold",
  "admin.settings.reg.billing.expiring_soon_days.hint":
    "Used by counters and highlights on the dashboard and server list",
  "admin.settings.reg.billing.balance_low_days": "Low balance warning horizon",
  "admin.settings.reg.billing.balance_low_days.hint":
    "Warn when the balance will not cover a renewal due within this period",
  "admin.settings.reg.billing.hourly_low_balance_hours": "Hourly server: warn when balance lasts less than",

  "admin.settings.reg.billing.server_grace_hours": "Grace before stopping",
  "admin.settings.reg.billing.server_grace_hours.hint": "0 stops the server right after expiry",
  "admin.settings.reg.billing.hosting_grace_hours": "Grace before suspending hosting",
  "admin.settings.reg.billing.hosting_grace_hours.hint": "0 suspends right after expiry",
  "admin.settings.reg.billing.server_delete_days": "Delete a paid server after",
  "admin.settings.reg.billing.server_delete_days.hint":
    "0 never deletes. Counted from the stop; the server is deleted with its files and cannot be restored",
  "admin.settings.reg.billing.trial_keep_hours": "Keep a trial server after it stops",

  "admin.settings.reg.billing.rent_max_batch": "Servers per order",
  "admin.settings.reg.billing.rent_max_days": "Maximum rental and renewal term",
  "admin.settings.reg.billing.hourly_prepaid_hours": "Hourly rental: minimum balance for",
  "admin.settings.reg.billing.hourly_catchup_hours": "Hourly charging: catch up at most",
  "admin.settings.reg.billing.hourly_catchup_hours.hint":
    "How many hours are charged at once after the panel was down",

  "admin.settings.reg.billing.topup_min": "Minimum top-up",
  "admin.settings.reg.billing.topup_min.hint": "In the payment currency, 0 means no limit",
  "admin.settings.reg.billing.topup_max": "Maximum top-up",
  "admin.settings.reg.billing.topup_max.hint": "In the payment currency, 0 means no limit",
  "admin.settings.reg.billing.topup_default": "Default top-up amount",
  "admin.settings.reg.billing.topup_presets": "Quick top-up amounts",
  "admin.settings.reg.billing.topup_presets.hint": "Buttons on the balance page, comma-separated",
  "admin.settings.reg.billing.invoice_expire_min": "Invoice lifetime",
  "admin.settings.reg.billing.invoice_expire_min.hint": "For providers that accept a lifetime (Lava, Enot)",
  "admin.settings.reg.billing.fx_fresh_hours": "Exchange rates stay fresh for",
  "admin.settings.reg.billing.fx_stale_hours": "Exchange rates usable if the source is down for",

  "admin.settings.reg.billing.bonus_cooldown_hours": "Bonus wheel interval",
  "admin.settings.reg.billing.retain_years": "Keep anonymized data of a deleted account",
  "admin.settings.reg.billing.retain_years.hint":
    "Check legal requirements: accounting data has statutory minimum retention periods",
};
