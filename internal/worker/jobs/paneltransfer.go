package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/vortanixapp/panel/pkg/sshclient"
)

const (
	panelTransferLogLimit      = 64 * 1024
	panelTransferFlushInterval = 1500 * time.Millisecond
	panelTransferTimeout       = 6 * time.Hour
)

type panelTransferRecord struct {
	ID           string
	Mode         string
	TargetHost   string
	TargetPort   int
	TargetUser   string
	TargetKey    string
	SecretKind   string
	Secret       string
	NewAddress   string
	SameAddress  bool
	FreezeWrites bool
}

func (r *Runner) PanelTransferLoop(ctx context.Context, wake <-chan struct{}) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-wake:
			r.heartbeat.Beat(LoopPanelTransfer)
			r.drainPanelTransfer(ctx)
		case <-ticker.C:
			r.heartbeat.Beat(LoopPanelTransfer)
			r.drainPanelTransfer(ctx)
		}
	}
}

func (r *Runner) drainPanelTransfer(ctx context.Context) {
	for r.processPanelTransfer(ctx) {
	}
}

func (r *Runner) claimPanelTransfer(ctx context.Context) (string, []byte, bool) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", nil, false
	}
	defer tx.Rollback(ctx)

	var jobID string
	var payload []byte
	err = tx.QueryRow(ctx, `
		SELECT id::text, payload
		FROM core.jobs
		WHERE type = 'panel_transfer' AND status = 'pending' AND attempts < 2
		ORDER BY created_at ASC LIMIT 1
		FOR UPDATE SKIP LOCKED
	`).Scan(&jobID, &payload)
	if err != nil {
		return "", nil, false
	}
	if _, err := tx.Exec(ctx, `UPDATE core.jobs SET status = 'running', attempts = attempts + 1 WHERE id = $1`, jobID); err != nil {
		return "", nil, false
	}
	if err := tx.Commit(ctx); err != nil {
		return "", nil, false
	}
	return jobID, payload, true
}

func (r *Runner) processPanelTransfer(ctx context.Context) bool {
	jobID, payload, ok := r.claimPanelTransfer(ctx)
	if !ok {
		return false
	}
	var pl struct {
		TransferID string `json:"transfer_id"`
	}
	_ = json.Unmarshal(payload, &pl)
	if pl.TransferID == "" {
		r.failJobGeneric(ctx, r.db, jobID, "в задаче переноса нет transfer_id")
		return true
	}

	rec, err := r.loadPanelTransfer(ctx, pl.TransferID)
	if err != nil {
		r.failJobGeneric(ctx, r.db, jobID, err.Error())
		return true
	}

	beat, stopBeat := context.WithCancel(ctx)
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-beat.Done():
				return
			case <-t.C:
				r.heartbeat.Beat(LoopPanelTransfer)
			}
		}
	}()
	defer stopBeat()

	runCtx, cancel := context.WithTimeout(ctx, panelTransferTimeout)
	defer cancel()

	if err := r.runPanelTransfer(runCtx, rec); err != nil {
		r.failPanelTransfer(ctx, rec, err.Error())
		r.failJobGeneric(ctx, r.db, jobID, err.Error())
		return true
	}
	_, _ = r.db.Exec(ctx, `UPDATE core.jobs SET status = 'completed', result = '{"ok":true}'::jsonb WHERE id = $1`, jobID)
	return true
}

func (r *Runner) loadPanelTransfer(ctx context.Context, id string) (*panelTransferRecord, error) {
	var rec panelTransferRecord
	var secretEnc *string
	err := r.db.QueryRow(ctx, `
		SELECT id::text, mode, target_host, target_port, target_user, target_host_key,
		       target_secret_kind, target_secret_enc, new_address, same_address, freeze_writes
		FROM core.panel_transfers WHERE id = $1
	`, id).Scan(&rec.ID, &rec.Mode, &rec.TargetHost, &rec.TargetPort, &rec.TargetUser, &rec.TargetKey,
		&rec.SecretKind, &secretEnc, &rec.NewAddress, &rec.SameAddress, &rec.FreezeWrites)
	if err != nil {
		return nil, fmt.Errorf("запись переноса не найдена")
	}
	if secretEnc != nil && *secretEnc != "" {
		rec.Secret = r.secrets.MustDecrypt(*secretEnc)
	}
	_, _ = r.db.Exec(ctx, `UPDATE core.panel_transfers SET target_secret_enc = NULL WHERE id = $1`, id)
	return &rec, nil
}

func (rec *panelTransferRecord) sshConfig() sshclient.Config {
	cfg := sshclient.Config{
		Host:         rec.TargetHost,
		Port:         rec.TargetPort,
		User:         rec.TargetUser,
		KnownHostKey: rec.TargetKey,
		Timeout:      30 * time.Second,
		ExecTimeout:  panelTransferTimeout,
	}
	if rec.SecretKind == "key" {
		cfg.PrivateKey = rec.Secret
	} else {
		cfg.Password = rec.Secret
	}
	return cfg
}

var panelTransferSecretLine = regexp.MustCompile(`(?i)([A-Z0-9_]*(SECRET|PASSWORD|TOKEN|KEY))=\S+`)

func maskSecrets(s string) string {
	return panelTransferSecretLine.ReplaceAllString(s, "$1=***")
}

type panelTransferLog struct {
	ctx context.Context
	r   *Runner
	id  string

	mu        sync.Mutex
	buf       bytes.Buffer
	dirty     bool
	lastFlush time.Time
	pending   []byte
}

func (r *Runner) newPanelTransferLog(ctx context.Context, id string) *panelTransferLog {
	w := &panelTransferLog{ctx: ctx, r: r, id: id}
	if prev := r.panelTransferLogText(ctx, id); prev != "" {
		w.buf.WriteString(prev)
		if !strings.HasSuffix(prev, "\n") {
			w.buf.WriteString("\n")
		}
	}
	return w
}

func (w *panelTransferLog) Write(p []byte) (int, error) {
	w.mu.Lock()
	w.pending = append(w.pending, p...)
	for {
		idx := bytes.IndexByte(w.pending, '\n')
		if idx < 0 {
			break
		}
		line := strings.TrimRight(string(w.pending[:idx]), "\r")
		w.pending = w.pending[idx+1:]
		w.buf.WriteString(maskSecrets(line))
		w.buf.WriteByte('\n')
	}
	w.trimLocked()
	w.dirty = true
	due := time.Since(w.lastFlush) >= panelTransferFlushInterval
	w.mu.Unlock()
	if due {
		w.flush()
	}
	return len(p), nil
}

func (w *panelTransferLog) say(format string, args ...any) {
	_, _ = fmt.Fprintf(w, format+"\n", args...)
	w.flush()
}

func (w *panelTransferLog) trimLocked() {
	if w.buf.Len() <= panelTransferLogLimit {
		return
	}
	s := w.buf.String()
	s = s[len(s)-panelTransferLogLimit:]
	if i := strings.IndexByte(s, '\n'); i >= 0 && i < len(s)-1 {
		s = s[i+1:]
	}
	w.buf.Reset()
	w.buf.WriteString("… лог обрезан …\n")
	w.buf.WriteString(s)
}

func (w *panelTransferLog) flush() {
	w.mu.Lock()
	if !w.dirty {
		w.mu.Unlock()
		return
	}
	snapshot := w.buf.String()
	w.dirty = false
	w.lastFlush = time.Now()
	w.mu.Unlock()
	_, _ = w.r.db.Exec(w.ctx, `
		UPDATE core.panel_transfers SET log = $2, updated_at = now() WHERE id = $1
	`, w.id, snapshot)
}

func (r *Runner) panelTransferLogText(ctx context.Context, id string) string {
	var text string
	if err := r.db.QueryRow(ctx, `SELECT log FROM core.panel_transfers WHERE id = $1`, id).Scan(&text); err != nil {
		return ""
	}
	return text
}

func (r *Runner) setPanelTransferStage(ctx context.Context, id, stage string) {
	_, _ = r.db.Exec(ctx, `
		UPDATE core.panel_transfers SET stage = $2, status = 'running',
		       started_at = COALESCE(started_at, now()), updated_at = now()
		WHERE id = $1
	`, id, stage)
}

func (r *Runner) setPanelTransferBytes(ctx context.Context, id string, done, total int64) {
	_, _ = r.db.Exec(ctx, `
		UPDATE core.panel_transfers SET bytes_done = $2, bytes_total = GREATEST(bytes_total, $3), updated_at = now()
		WHERE id = $1
	`, id, done, total)
}

func (r *Runner) failPanelTransfer(ctx context.Context, rec *panelTransferRecord, message string) {
	_, _ = r.db.Exec(ctx, `
		UPDATE core.panel_transfers
		SET status = 'failed', error = $2, finished_at = now(), updated_at = now()
		WHERE id = $1
	`, rec.ID, maskSecrets(message))
	if rec.FreezeWrites {
		r.setPanelFreeze(ctx, false)
	}
}

func (r *Runner) finishPanelTransfer(ctx context.Context, id, stage string) {
	_, _ = r.db.Exec(ctx, `
		UPDATE core.panel_transfers
		SET status = 'completed', stage = $2, finished_at = now(), updated_at = now()
		WHERE id = $1
	`, id, stage)
}

func (r *Runner) cancelRequested(ctx context.Context, id string) bool {
	var status string
	if err := r.db.QueryRow(ctx, `SELECT status FROM core.panel_transfers WHERE id = $1`, id).Scan(&status); err != nil {
		return false
	}
	return status == "cancelled"
}
