package settingsreg

const (
	GroupInterface = "interface"
)

var (
	UIPollScale = def(Setting{
		Key: "ui.poll_scale_percent", Group: GroupInterface, Section: "refresh",
		Kind: KindInt, Default: "100", Min: 25, Max: 1000, Unit: UnitPercent, Public: true,
	})
	UIStale = def(Setting{
		Key: "ui.stale_ms", Group: GroupInterface, Section: "refresh",
		Kind: KindInt, Default: "5000", Min: 1000, Max: 120000, Unit: UnitMs, Public: true,
	})
	UIConsoleBuffer = def(Setting{
		Key: "ui.console_buffer_lines", Group: GroupInterface, Section: "refresh",
		Kind: KindInt, Default: "4000", Min: 500, Max: 20000, Unit: UnitCount, Public: true,
	})

	UIToast = def(Setting{
		Key: "ui.toast_ms", Group: GroupInterface, Section: "lists",
		Kind: KindInt, Default: "4000", Min: 1000, Max: 20000, Unit: UnitMs, Public: true,
	})
	UIPageSize = def(Setting{
		Key: "ui.page_size", Group: GroupInterface, Section: "lists",
		Kind: KindInt, Default: "10", Min: 5, Max: 100, Unit: UnitCount, Public: true,
	})
	UIFeedPageSize = def(Setting{
		Key: "ui.feed_page_size", Group: GroupInterface, Section: "lists",
		Kind: KindInt, Default: "30", Min: 10, Max: 200, Unit: UnitCount, Public: true,
	})
	UIActivityPageSize = def(Setting{
		Key: "ui.activity_page_size", Group: GroupInterface, Section: "lists",
		Kind: KindInt, Default: "100", Min: 20, Max: 500, Unit: UnitCount, Public: true,
	})
	UISearchDebounce = def(Setting{
		Key: "ui.search_debounce_ms", Group: GroupInterface, Section: "lists",
		Kind: KindInt, Default: "300", Min: 100, Max: 1500, Unit: UnitMs, Public: true,
	})

	UICookieNoticeDays = def(Setting{
		Key: "ui.cookie_notice_days", Group: GroupInterface, Section: "notices",
		Kind: KindInt, Default: "0", Min: 0, Max: 3650, Unit: UnitDay, Public: true,
	})
)
