package notify

import (
	"time"
	_ "time/tzdata"
)

type Quiet struct {
	Enabled  bool
	From     int
	To       int
	Critical bool
	TimeZone string
}

func (q Quiet) HoldUntil(now time.Time, def Def) (time.Time, bool) {
	if !q.Enabled || def.Required || q.From == q.To {
		return time.Time{}, false
	}
	if q.Critical && def.Severity == SeverityCritical {
		return time.Time{}, false
	}
	if q.From < 0 || q.From >= 1440 || q.To < 0 || q.To >= 1440 {
		return time.Time{}, false
	}
	loc := time.UTC
	if q.TimeZone != "" {
		if l, err := time.LoadLocation(q.TimeZone); err == nil {
			loc = l
		}
	}
	local := now.In(loc)
	minute := local.Hour()*60 + local.Minute()
	inside := minute >= q.From && minute < q.To
	if q.From > q.To {
		inside = minute >= q.From || minute < q.To
	}
	if !inside {
		return time.Time{}, false
	}
	end := time.Date(local.Year(), local.Month(), local.Day(), q.To/60, q.To%60, 0, 0, loc)
	if !end.After(local) {
		end = time.Date(local.Year(), local.Month(), local.Day()+1, q.To/60, q.To%60, 0, 0, loc)
	}
	return end, true
}
