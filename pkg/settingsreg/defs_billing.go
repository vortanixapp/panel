package settingsreg

import (
	"fmt"
	"strconv"
)

const (
	GroupBilling = "billing"
)

func init() {
	addCheck(func(v map[string]string) error {
		lower, _ := strconv.ParseInt(v[BillingTopupMin.Key], 10, 64)
		upper, _ := strconv.ParseInt(v[BillingTopupMax.Key], 10, 64)
		if upper > 0 && lower > upper {
			return fmt.Errorf("минимальная сумма пополнения больше максимальной")
		}
		return nil
	})
}

var (
	BillingDunningStages = def(Setting{
		Key: "billing.dunning_stages_days", Group: GroupBilling, Section: "reminders",
		Kind: KindIntList, Default: "1,3,7", Min: 1, Max: 60, Unit: UnitDay,
	})
	BillingAutoRenewHorizon = def(Setting{
		Key: "billing.autorenew_horizon_hours", Group: GroupBilling, Section: "reminders",
		Kind: KindInt, Default: "24", Min: 1, Max: 168, Unit: UnitHour,
	})
	BillingExpiringSoonDays = def(Setting{
		Key: "billing.expiring_soon_days", Group: GroupBilling, Section: "reminders",
		Kind: KindInt, Default: "7", Min: 1, Max: 60, Unit: UnitDay,
	})
	BillingBalanceLowDays = def(Setting{
		Key: "billing.balance_low_days", Group: GroupBilling, Section: "reminders",
		Kind: KindInt, Default: "5", Min: 1, Max: 30, Unit: UnitDay,
	})
	BillingHourlyLowBalance = def(Setting{
		Key: "billing.hourly_low_balance_hours", Group: GroupBilling, Section: "reminders",
		Kind: KindInt, Default: "6", Min: 1, Max: 72, Unit: UnitHour,
	})

	BillingServerGrace = def(Setting{
		Key: "billing.server_grace_hours", Group: GroupBilling, Section: "expiry",
		Kind: KindInt, Default: "0", Min: 0, Max: 720, Unit: UnitHour,
	})
	BillingHostingGrace = def(Setting{
		Key: "billing.hosting_grace_hours", Group: GroupBilling, Section: "expiry",
		Kind: KindInt, Default: "0", Min: 0, Max: 720, Unit: UnitHour,
	})
	BillingServerDeleteDays = def(Setting{
		Key: "billing.server_delete_days", Group: GroupBilling, Section: "expiry",
		Kind: KindInt, Default: "0", Min: 0, Max: 365, Unit: UnitDay,
	})
	BillingTrialKeepHours = def(Setting{
		Key: "billing.trial_keep_hours", Group: GroupBilling, Section: "expiry",
		Kind: KindInt, Default: "72", Min: 1, Max: 720, Unit: UnitHour,
	})

	BillingRentMaxBatch = def(Setting{
		Key: "billing.rent_max_batch", Group: GroupBilling, Section: "rental",
		Kind: KindInt, Default: "5", Min: 1, Max: 50, Unit: UnitCount, Public: true,
	})
	BillingRentMaxDays = def(Setting{
		Key: "billing.rent_max_days", Group: GroupBilling, Section: "rental",
		Kind: KindInt, Default: "365", Min: 30, Max: 3650, Unit: UnitDay,
	})
	BillingHourlyPrepaid = def(Setting{
		Key: "billing.hourly_prepaid_hours", Group: GroupBilling, Section: "rental",
		Kind: KindInt, Default: "24", Min: 1, Max: 168, Unit: UnitHour,
	})
	BillingHourlyCatchup = def(Setting{
		Key: "billing.hourly_catchup_hours", Group: GroupBilling, Section: "rental",
		Kind: KindInt, Default: "24", Min: 1, Max: 168, Unit: UnitHour,
	})

	BillingTopupMin = def(Setting{
		Key: "billing.topup_min", Group: GroupBilling, Section: "payments",
		Kind: KindInt, Default: "0", Min: 0, Max: 100000000, Public: true,
	})
	BillingTopupMax = def(Setting{
		Key: "billing.topup_max", Group: GroupBilling, Section: "payments",
		Kind: KindInt, Default: "0", Min: 0, Max: 100000000, Public: true,
	})
	BillingTopupDefault = def(Setting{
		Key: "billing.topup_default", Group: GroupBilling, Section: "payments",
		Kind: KindInt, Default: "1000", Min: 1, Max: 100000000, Public: true,
	})
	BillingTopupPresets = def(Setting{
		Key: "billing.topup_presets", Group: GroupBilling, Section: "payments",
		Kind: KindIntList, Default: "500,1000,2000,5000,10000", Min: 1, Max: 100000000, Public: true,
	})
	BillingInvoiceExpire = def(Setting{
		Key: "billing.invoice_expire_min", Group: GroupBilling, Section: "payments",
		Kind: KindInt, Default: "5", Min: 1, Max: 1440, Unit: UnitMin,
	})
	BillingFXFresh = def(Setting{
		Key: "billing.fx_fresh_hours", Group: GroupBilling, Section: "payments",
		Kind: KindInt, Default: "6", Min: 1, Max: 168, Unit: UnitHour,
	})
	BillingFXStale = def(Setting{
		Key: "billing.fx_stale_hours", Group: GroupBilling, Section: "payments",
		Kind: KindInt, Default: "72", Min: 1, Max: 720, Unit: UnitHour,
	})

	BillingBonusCooldown = def(Setting{
		Key: "billing.bonus_cooldown_hours", Group: GroupBilling, Section: "bonuses",
		Kind: KindInt, Default: "24", Min: 1, Max: 720, Unit: UnitHour,
	})
	BillingRetainYears = def(Setting{
		Key: "billing.retain_years", Group: GroupBilling, Section: "bonuses",
		Kind: KindInt, Default: "5", Min: 1, Max: 50, Unit: UnitCount,
	})
)
