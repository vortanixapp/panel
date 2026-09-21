package cronexpr

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type field struct {
	min, max int
	any      bool
	star     bool
	values   map[int]bool
}

type Expr struct {
	minute, hour, dom, month, dow field
}

func Parse(spec string) (*Expr, error) {
	parts := strings.Fields(spec)
	if len(parts) != 5 {
		return nil, fmt.Errorf("в расписании должно быть 5 полей: минута час день месяц день_недели")
	}
	var e Expr
	var err error
	if e.minute, err = parseField(parts[0], 0, 59); err != nil {
		return nil, fmt.Errorf("минуты: %w", err)
	}
	if e.hour, err = parseField(parts[1], 0, 23); err != nil {
		return nil, fmt.Errorf("часы: %w", err)
	}
	if e.dom, err = parseField(parts[2], 1, 31); err != nil {
		return nil, fmt.Errorf("день месяца: %w", err)
	}
	if e.month, err = parseField(parts[3], 1, 12); err != nil {
		return nil, fmt.Errorf("месяц: %w", err)
	}
	if e.dow, err = parseField(parts[4], 0, 7); err != nil {
		return nil, fmt.Errorf("день недели: %w", err)
	}
	if e.dow.values[7] {
		e.dow.values[0] = true
	}
	return &e, nil
}

func parseField(spec string, min, max int) (field, error) {
	f := field{min: min, max: max, values: map[int]bool{}}
	if spec == "*" {
		f.any, f.star = true, true
		for v := min; v <= max; v++ {
			f.values[v] = true
		}
		return f, nil
	}
	for _, item := range strings.Split(spec, ",") {
		if item == "" {
			return f, fmt.Errorf("пустой элемент списка")
		}
		rangePart, stepPart, hasStep := strings.Cut(item, "/")
		step := 1
		if hasStep {
			n, err := strconv.Atoi(stepPart)
			if err != nil || n <= 0 {
				return f, fmt.Errorf("неверный шаг %q", stepPart)
			}
			step = n
		}
		lo, hi := min, max
		switch {
		case rangePart == "*":
			f.star = true
		case strings.Contains(rangePart, "-"):
			a, b, _ := strings.Cut(rangePart, "-")
			x, err1 := strconv.Atoi(a)
			y, err2 := strconv.Atoi(b)
			if err1 != nil || err2 != nil {
				return f, fmt.Errorf("неверный диапазон %q", rangePart)
			}
			lo, hi = x, y
		default:
			x, err := strconv.Atoi(rangePart)
			if err != nil {
				return f, fmt.Errorf("неверное значение %q", rangePart)
			}
			lo, hi = x, x
			if hasStep {
				hi = max
			}
		}
		if lo < min || hi > max || lo > hi {
			return f, fmt.Errorf("значение %q вне диапазона %d–%d", item, min, max)
		}
		for v := lo; v <= hi; v += step {
			f.values[v] = true
		}
	}
	return f, nil
}

func (e *Expr) Match(t time.Time) bool {
	if !e.minute.values[t.Minute()] || !e.hour.values[t.Hour()] || !e.month.values[int(t.Month())] {
		return false
	}
	domMatch := e.dom.values[t.Day()]
	dowMatch := e.dow.values[int(t.Weekday())]
	switch {
	case e.dom.any && e.dow.any:
		return true
	case e.dom.any:
		return dowMatch
	case e.dow.any:
		return domMatch
	default:
		return domMatch || dowMatch
	}
}

func (e *Expr) FixedTime() bool {
	return !e.minute.star && !e.hour.star
}

func (e *Expr) Next(after time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	t := after.In(loc).Truncate(time.Minute).Add(time.Minute)
	limit := t.AddDate(5, 0, 0)
	for t.Before(limit) {
		if e.Match(t) {
			return t
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}
}
