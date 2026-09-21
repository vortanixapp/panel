package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/vortanixapp/panel/pkg/protocol"
)

type ConsoleEmit func(kind, code, data string)

const (
	consoleLease      = 3 * time.Minute
	consoleSweep      = 30 * time.Second
	consolePoll       = 3 * time.Second
	consoleTailLines  = "100"
	consoleBatchDelay = 60 * time.Millisecond
	consoleBatchLimit = 32 << 10
)

type consoleSession struct {
	cancel  context.CancelFunc
	expires time.Time
}

var (
	consoleMu       sync.Mutex
	consoleSessions = map[string]*consoleSession{}
	consoleJanitor  sync.Once
)

func AttachConsole(serverID, sessionID string, leased bool, emit ConsoleEmit) {
	if sessionID == "" {
		return
	}
	consoleMu.Lock()
	defer consoleMu.Unlock()
	if s, ok := consoleSessions[sessionID]; ok {
		s.expires = time.Now().Add(consoleLease)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &consoleSession{cancel: cancel}
	if leased {
		s.expires = time.Now().Add(consoleLease)
	}
	consoleSessions[sessionID] = s
	consoleJanitor.Do(func() { go sweepConsoleSessions() })
	go func() {
		defer cancel()
		streamConsole(ctx, serverID, emit)
		consoleMu.Lock()
		if consoleSessions[sessionID] == s {
			delete(consoleSessions, sessionID)
		}
		consoleMu.Unlock()
	}()
}

func StopConsole(sessionID string) {
	consoleMu.Lock()
	s, ok := consoleSessions[sessionID]
	if ok {
		delete(consoleSessions, sessionID)
	}
	consoleMu.Unlock()
	if ok {
		s.cancel()
	}
}

func sweepConsoleSessions() {
	ticker := time.NewTicker(consoleSweep)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		var expired []*consoleSession
		consoleMu.Lock()
		for id, s := range consoleSessions {
			if !s.expires.IsZero() && now.After(s.expires) {
				expired = append(expired, s)
				delete(consoleSessions, id)
			}
		}
		consoleMu.Unlock()
		for _, s := range expired {
			s.cancel()
		}
	}
}

func streamConsole(ctx context.Context, serverID string, emit ConsoleEmit) {
	cname := ContainerName(serverID)
	batch := &consoleBatch{emit: emit}
	defer batch.flush()

	since := ""
	waiting := false
	failures := 0
	for ctx.Err() == nil {
		status := containerStatus(ctx, cname)
		if status != "running" {
			if since == "" {
				if status != "" {
					_ = pipeLogs(ctx, cname, []string{"--tail", consoleTailLines}, batch)
				}
				since = logsSince(time.Now())
			}
			if !waiting {
				waiting = true
				batch.flush()
				emit(protocol.ConsoleFrameNotice, "waiting_start", "")
			}
			if !sleepContext(ctx, consolePoll) {
				return
			}
			continue
		}
		waiting = false

		args := []string{"--follow", "--tail", consoleTailLines}
		if since != "" {
			args = []string{"--follow", "--since", since}
		}
		started := time.Now()
		err := pipeLogs(ctx, cname, args, batch)
		since = logsSince(time.Now())

		pause := time.Second
		if err != nil && time.Since(started) < 2*time.Second {
			failures++
			pause = min(time.Duration(failures)*consolePoll, 30*time.Second)
		} else {
			failures = 0
		}
		if !sleepContext(ctx, pause) {
			return
		}
	}
}

func containerStatus(ctx context.Context, cname string) string {
	out, err := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Status}}", cname).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func logsSince(t time.Time) string {
	return fmt.Sprintf("%d.%09d", t.Unix(), t.Nanosecond())
}

func pipeLogs(ctx context.Context, cname string, args []string, batch *consoleBatch) error {
	full := append(append([]string{"logs"}, args...), cname)
	cmd := exec.CommandContext(ctx, "docker", full...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		pumpChunks(stdout, batch.add)
	}()
	go func() {
		defer wg.Done()
		pumpChunks(stderr, batch.add)
	}()
	wg.Wait()
	return cmd.Wait()
}

func pumpChunks(r io.Reader, add func(string)) {
	br := bufio.NewReaderSize(r, 64<<10)
	for {
		chunk, err := br.ReadSlice('\n')
		if len(chunk) > 0 {
			text := string(chunk)
			if err != nil && err != bufio.ErrBufferFull && !strings.HasSuffix(text, "\n") {
				text += "\n"
			}
			add(text)
		}
		if err != nil && err != bufio.ErrBufferFull {
			return
		}
	}
}

func sleepContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

type consoleBatch struct {
	emit  ConsoleEmit
	mu    sync.Mutex
	buf   strings.Builder
	timer *time.Timer
}

func (b *consoleBatch) add(chunk string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.WriteString(chunk)
	if b.buf.Len() >= consoleBatchLimit {
		b.emitLocked()
		return
	}
	if b.timer == nil {
		b.timer = time.AfterFunc(consoleBatchDelay, b.flush)
	}
}

func (b *consoleBatch) flush() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.emitLocked()
}

func (b *consoleBatch) emitLocked() {
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	if b.buf.Len() == 0 {
		return
	}
	data := b.buf.String()
	b.buf.Reset()
	b.emit(protocol.ConsoleFrameOutput, "", data)
}
