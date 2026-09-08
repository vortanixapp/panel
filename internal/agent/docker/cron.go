package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type CronJob struct {
	ID       string `json:"id"`
	Schedule string `json:"schedule"`
	Command  string `json:"command"`
	Enabled  bool   `json:"enabled"`
}

var cronScheduleRe = regexp.MustCompile(`^[0-9*/,-]+$`)

func SyncCron(ctx context.Context, serverID string, jobs []CronJob) error {
	for i, j := range jobs {
		sched, err := validateCronSchedule(j.Schedule)
		if err != nil {
			return fmt.Errorf("job %s: %w", j.ID, err)
		}
		cmd, err := validateCronCommand(j.Command)
		if err != nil {
			return fmt.Errorf("job %s: %w", j.ID, err)
		}
		jobs[i].Schedule = sched
		jobs[i].Command = cmd
	}

	statePath := filepath.Join(serverDataDir(serverID), ".vortanix_cron.json")
	if err := writeJSONFile(statePath, map[string]any{"jobs": jobs}); err != nil {
		return err
	}

	cronPath := cronFilePath(serverID)
	enabled := make([]CronJob, 0, len(jobs))
	for _, j := range jobs {
		if j.Enabled && strings.TrimSpace(j.Schedule) != "" && strings.TrimSpace(j.Command) != "" {
			enabled = append(enabled, j)
		}
	}
	if len(enabled) == 0 {
		_ = os.Remove(cronPath)
		return nil
	}

	cname := ContainerName(serverID)
	lines := make([]string, 0, len(enabled))
	for _, j := range enabled {
		line := fmt.Sprintf(
			"%s root docker exec %s sh -lc %s >/dev/null 2>&1 # %s",
			j.Schedule,
			shellQuote(cname),
			shellQuote(j.Command),
			cronComment(j.ID),
		)
		lines = append(lines, line)
	}
	content := strings.Join(lines, "\n") + "\n"
	tmp := cronPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, cronPath)
}

func cronFilePath(serverID string) string {
	safe := strings.ReplaceAll(serverID, "-", "")
	if len(safe) > 32 {
		safe = safe[:32]
	}
	return filepath.Join("/etc/cron.d", "vortanix-"+safe)
}

func validateCronSchedule(expr string) (string, error) {
	expr = strings.TrimSpace(expr)
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return "", fmt.Errorf("schedule must have 5 fields (min hour dom mon dow)")
	}
	for _, p := range parts {
		if p == "*" {
			continue
		}
		if !cronScheduleRe.MatchString(p) {
			return "", fmt.Errorf("schedule contains invalid characters")
		}
	}
	return strings.Join(parts, " "), nil
}

func validateCronCommand(cmd string) (string, error) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "", fmt.Errorf("command is required")
	}
	if len(cmd) > 2000 {
		return "", fmt.Errorf("command is too long")
	}
	// Перевод строки внутри команды разрывает строку файла /etc/cron.d, а всё
	// после разрыва cron читает как СВОЮ запись — с полем пользователя, то есть
	// исполняет от root на самой ноде, мимо контейнера. Кавычки от этого не
	// спасают: shellQuote делает из команды корректный аргумент shell, но
	// формат cron.d построчный, и кавычка внутри строки его не склеивает.
	if strings.ContainsAny(cmd, "\n\r") {
		return "", fmt.Errorf("command must be a single line")
	}
	return cmd, nil
}

// cronComment оставляет от идентификатора только безопасные символы: он уходит
// в хвост строки /etc/cron.d, и перевод строки в нём — та же инъекция, что и в
// самой команде.
func cronComment(id string) string {
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		}
		if b.Len() >= 64 {
			break
		}
	}
	return b.String()
}

func writeJSONFile(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func DecodeCronJobs(raw any) ([]CronJob, error) {
	if raw == nil {
		return []CronJob{}, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var jobs []CronJob
	if err := json.Unmarshal(b, &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func CronFileForServer(serverID string) string {
	return cronFilePath(serverID)
}

func ReadCronFile(serverID string) (string, error) {
	b, err := os.ReadFile(cronFilePath(serverID))
	if os.IsNotExist(err) {
		return "", nil
	}
	return string(b), err
}

func EnsureCronInstalled(ctx context.Context) error {
	return exec.CommandContext(ctx, "sh", "-c", "command -v crontab >/dev/null 2>&1 || test -d /etc/cron.d").Run()
}
