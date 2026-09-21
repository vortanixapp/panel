package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/vortanixapp/panel/pkg/cronexpr"
)

type CronJob struct {
	ID       string `json:"id"`
	Schedule string `json:"schedule"`
	Command  string `json:"command"`
	Enabled  bool   `json:"enabled"`
}

type CronState struct {
	Jobs []CronJob `json:"jobs"`
	TZ   string    `json:"tz,omitempty"`
}

type CronRun struct {
	JobID      string    `json:"job_id"`
	StartedAt  time.Time `json:"started_at"`
	DurationMs int64     `json:"duration_ms"`
	ExitCode   int       `json:"exit_code"`
	Error      string    `json:"error,omitempty"`
	Output     string    `json:"output,omitempty"`
}

const (
	stateCronRunsFile = "cron_runs.json"
	cronRunsKeep      = 50
	cronOutputMax     = 4 << 10
)

var cronRunsMu sync.Mutex

func SyncCron(serverID string, jobs []CronJob, tz string) ([]string, error) {
	invalid := []string{}
	for i, j := range jobs {
		cmd, err := validateCronCommand(j.Command)
		if err != nil {
			return nil, fmt.Errorf("задание %s: %w", j.ID, err)
		}
		jobs[i].Command = cmd
		jobs[i].Schedule = strings.Join(strings.Fields(j.Schedule), " ")
		if _, err := cronexpr.Parse(jobs[i].Schedule); err != nil {
			invalid = append(invalid, j.ID)
		}
	}
	if err := writeStateFile(serverID, stateCronFile, CronState{Jobs: jobs, TZ: CronLocationName(tz)}); err != nil {
		return nil, err
	}
	return invalid, nil
}

func CronLocationName(tz string) string {
	tz = strings.TrimSpace(tz)
	if tz == "" || tz == "Local" {
		return ""
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return ""
	}
	return tz
}

func ReadCronState(serverID string) (CronState, bool) {
	var state CronState
	if !readStateFile(serverID, stateCronFile, &state) {
		return CronState{}, false
	}
	return state, true
}

func validateCronCommand(cmd string) (string, error) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "", errors.New("команда не указана")
	}
	if len(cmd) > 2000 {
		return "", errors.New("команда длиннее 2000 символов")
	}
	if strings.ContainsAny(cmd, "\n\r") {
		return "", errors.New("команда должна быть одной строкой")
	}
	return cmd, nil
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

func RunCronJob(ctx context.Context, serverID string, job CronJob) CronRun {
	run := CronRun{JobID: job.ID, StartedAt: time.Now().UTC()}
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", "exec", ContainerName(serverID), "sh", "-lc", job.Command)
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	run.DurationMs = time.Since(run.StartedAt).Milliseconds()
	text := out.Bytes()
	if len(text) > cronOutputMax {
		text = text[len(text)-cronOutputMax:]
	}
	run.Output = strings.TrimSpace(string(text))
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case ctx.Err() != nil:
		run.ExitCode = -1
		run.Error = "задание не уложилось в отведённое время"
	case errors.As(err, &exitErr):
		run.ExitCode = exitErr.ExitCode()
		run.Error = fmt.Sprintf("команда завершилась с кодом %d", run.ExitCode)
	default:
		run.ExitCode = -1
		run.Error = err.Error()
	}
	return run
}

func RecordCronRun(serverID string, run CronRun) {
	cronRunsMu.Lock()
	defer cronRunsMu.Unlock()
	var runs []CronRun
	readStateFile(serverID, stateCronRunsFile, &runs)
	runs = append(runs, run)
	if len(runs) > cronRunsKeep {
		runs = runs[len(runs)-cronRunsKeep:]
	}
	_ = writeStateFile(serverID, stateCronRunsFile, runs)
}

func StateServers(ctx context.Context) []string {
	seen := map[string]bool{}
	var ids []string
	add := func(id string) {
		if safeServerID(id) && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	base := envOr("VORTANIX_STATE_DIR", "/opt/vortanix/state")
	if entries, err := os.ReadDir(base); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				add(e.Name())
			}
		}
	}
	if managed, err := ListManagedServerIDs(ctx); err == nil {
		for _, id := range managed {
			add(id)
		}
	}
	return ids
}
