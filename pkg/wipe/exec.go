package wipe

import (
	"context"
	"fmt"
	"path"
	"strings"
)

type Entry struct {
	Name  string
	IsDir bool
}

type FS interface {
	List(ctx context.Context, dir string) ([]Entry, error)
	Delete(ctx context.Context, p string) error
}

type MissingError struct{ Path string }

func (e *MissingError) Error() string { return "не найдено: " + e.Path }

type Result struct {
	Deleted int
	Missing int
	Sample  []string
}

const sampleLimit = 20

func Execute(ctx context.Context, fs FS, targets []Target) (Result, error) {
	var res Result
	for _, t := range targets {
		dirs, err := expandDir(ctx, fs, t.Dir)
		if err != nil {
			return res, err
		}
		if len(dirs) == 0 {
			res.Missing++
			continue
		}
		for _, dir := range dirs {
			if err := sweep(ctx, fs, dir, t, t.Depth, &res); err != nil {
				return res, err
			}
		}
	}
	return res, nil
}

func hasGlob(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

func expandDir(ctx context.Context, fs FS, dir string) ([]string, error) {
	current := []string{""}
	for _, seg := range strings.Split(strings.Trim(dir, "/"), "/") {
		if seg == "" {
			continue
		}
		if seg == ".." || seg == "." {
			return nil, fmt.Errorf("недопустимый путь %q", dir)
		}
		if !hasGlob(seg) {
			for i := range current {
				current[i] += "/" + seg
			}
			continue
		}
		var next []string
		for _, base := range current {
			entries, err := fs.List(ctx, orRoot(base))
			if err != nil {
				if isMissing(err) {
					continue
				}
				return nil, err
			}
			for _, e := range entries {
				if !e.IsDir || !safeName(e.Name) {
					continue
				}
				if ok, _ := path.Match(seg, e.Name); ok {
					next = append(next, base+"/"+e.Name)
				}
			}
		}
		current = next
	}
	out := make([]string, 0, len(current))
	for _, p := range current {
		if p != "" {
			out = append(out, p)
		}
	}
	return out, nil
}

func orRoot(p string) string {
	if p == "" {
		return "/"
	}
	return p
}

func sweep(ctx context.Context, fs FS, dir string, t Target, depth int, res *Result) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	entries, err := fs.List(ctx, dir)
	if err != nil {
		if isMissing(err) {
			res.Missing++
			return nil
		}
		return err
	}
	for _, e := range entries {
		if !safeName(e.Name) {
			continue
		}
		full := dir + "/" + e.Name
		if matches(t.Patterns, e.Name) && (!e.IsDir || t.MatchDirs) {
			if err := fs.Delete(ctx, full); err != nil {
				return fmt.Errorf("удаление %s: %w", full, err)
			}
			res.Deleted++
			if len(res.Sample) < sampleLimit {
				res.Sample = append(res.Sample, full)
			}
			continue
		}
		if e.IsDir && depth > 0 {
			if err := sweep(ctx, fs, full, t, depth-1, res); err != nil {
				return err
			}
		}
	}
	return nil
}

func matches(patterns []string, name string) bool {
	for _, p := range patterns {
		if ok, _ := path.Match(p, name); ok {
			return true
		}
	}
	return false
}

func safeName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\\x00")
}

func isMissing(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := err.(*MissingError); ok {
		return true
	}
	low := strings.ToLower(err.Error())
	return strings.Contains(low, "no such file") ||
		strings.Contains(low, "not exist") ||
		strings.Contains(low, "not found") ||
		strings.Contains(low, "не найден") ||
		strings.Contains(low, "не существует")
}
