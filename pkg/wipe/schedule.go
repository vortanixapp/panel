package wipe

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/cronexpr"
)

const (
	ScheduleWeekly  = "weekly"
	ScheduleMonthly = "monthly"
	ScheduleCron    = "cron"
	ScheduleOnce    = "once"
)

type Schedule struct {
	Type      string
	Weekday   int
	Nth       int
	TimeOfDay string
	Cron      string
	RunAt     time.Time
	TZ        string
}

func (s Schedule) location() (*time.Location, error) {
	name := strings.TrimSpace(s.TZ)
	if name == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("неизвестный часовой пояс %q", name)
	}
	return loc, nil
}

func parseClock(v string) (int, int, error) {
	h, m, ok := strings.Cut(strings.TrimSpace(v), ":")
	if !ok {
		return 0, 0, errors.New("время указывается как ЧЧ:ММ")
	}
	hour, err1 := strconv.Atoi(h)
	minute, err2 := strconv.Atoi(m)
	if err1 != nil || err2 != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, errors.New("время указывается как ЧЧ:ММ")
	}
	return hour, minute, nil
}

func (s Schedule) Validate() error {
	if _, err := s.location(); err != nil {
		return err
	}
	switch s.Type {
	case ScheduleWeekly:
		if s.Weekday < 0 || s.Weekday > 6 {
			return errors.New("день недели должен быть от 0 до 6")
		}
		_, _, err := parseClock(s.TimeOfDay)
		return err
	case ScheduleMonthly:
		if s.Weekday < 0 || s.Weekday > 6 {
			return errors.New("день недели должен быть от 0 до 6")
		}
		if s.Nth < 1 || s.Nth > 5 {
			return errors.New("номер недели должен быть от 1 до 5")
		}
		_, _, err := parseClock(s.TimeOfDay)
		return err
	case ScheduleCron:
		expr, err := cronexpr.Parse(s.Cron)
		if err != nil {
			return err
		}
		if !expr.FixedTime() {
			return errors.New("для вайпа нужно точное время: минута и час не могут быть «*»")
		}
		return nil
	case ScheduleOnce:
		if s.RunAt.IsZero() {
			return errors.New("не указано время вайпа")
		}
		return nil
	}
	return errors.New("неизвестный тип расписания")
}

func (s Schedule) Next(after time.Time) (time.Time, bool) {
	loc, err := s.location()
	if err != nil {
		return time.Time{}, false
	}
	switch s.Type {
	case ScheduleWeekly:
		hour, minute, err := parseClock(s.TimeOfDay)
		if err != nil {
			return time.Time{}, false
		}
		base := after.In(loc)
		for i := 0; i < 15; i++ {
			day := base.AddDate(0, 0, i)
			if int(day.Weekday()) != s.Weekday {
				continue
			}
			candidate := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, loc)
			if candidate.After(after) {
				return candidate, true
			}
		}
	case ScheduleMonthly:
		hour, minute, err := parseClock(s.TimeOfDay)
		if err != nil {
			return time.Time{}, false
		}
		base := after.In(loc)
		for i := 0; i < 14; i++ {
			first := time.Date(base.Year(), base.Month()+time.Month(i), 1, 0, 0, 0, 0, loc)
			day := nthWeekday(first, s.Weekday, s.Nth)
			candidate := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, loc)
			if candidate.After(after) {
				return candidate, true
			}
		}
	case ScheduleCron:
		expr, err := cronexpr.Parse(s.Cron)
		if err != nil {
			return time.Time{}, false
		}
		next := expr.Next(after, loc)
		return next, !next.IsZero()
	case ScheduleOnce:
		if s.RunAt.After(after) {
			return s.RunAt, true
		}
	}
	return time.Time{}, false
}

func nthWeekday(firstOfMonth time.Time, weekday, nth int) time.Time {
	offset := (weekday - int(firstOfMonth.Weekday()) + 7) % 7
	day := firstOfMonth.AddDate(0, 0, offset)
	if nth >= 5 {
		for day.AddDate(0, 0, 7).Month() == firstOfMonth.Month() {
			day = day.AddDate(0, 0, 7)
		}
		return day
	}
	return day.AddDate(0, 0, (nth-1)*7)
}

func DefaultAnnounceMinutes() []int {
	return []int{60, 30, 10, 5, 1}
}

func NormalizeMinutes(in []int) []int {
	seen := map[int]bool{}
	out := make([]int, 0, len(in))
	for _, m := range in {
		if m < 1 || m > 1440 || seen[m] {
			continue
		}
		seen[m] = true
		out = append(out, m)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] > out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}

func RenderAnnouncement(template string, minutes int, fallback string) string {
	text := strings.TrimSpace(template)
	if text == "" {
		text = fallback
	}
	text = strings.ReplaceAll(text, "{minutes}", strconv.Itoa(minutes))
	text = strings.NewReplacer("\r", " ", "\n", " ", "\"", "'").Replace(text)
	runes := []rune(text)
	if len(runes) > 200 {
		text = string(runes[:200])
	}
	return text
}
