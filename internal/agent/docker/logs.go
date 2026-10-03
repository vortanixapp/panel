package docker

import (
	"bytes"
	"context"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const maxLogTail = 1000

type LogEntry struct {
	Time time.Time
	Text string
}

type LogTail struct {
	Lines       []string
	Times       []string
	Incremental bool
}

func TailLogs(ctx context.Context, serverID string, tail int) ([]string, error) {
	out, err := TailLogsSince(ctx, serverID, tail, time.Time{})
	if err != nil {
		return nil, err
	}
	return out.Lines, nil
}

func TailLogsSince(ctx context.Context, serverID string, tail int, since time.Time) (LogTail, error) {
	if tail <= 0 {
		tail = 100
	}
	if tail > maxLogTail {
		tail = maxLogTail
	}
	args := []string{"logs", "--timestamps", "--tail", strconv.Itoa(tail)}
	if !since.IsZero() {
		args = append(args, "--since", since.UTC().Format(time.RFC3339Nano))
	}
	args = append(args, ContainerName(serverID))

	cmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil && stdout.Len() == 0 {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return LogTail{}, err
		}
		if _, ok := parseLogEntry(firstLine(msg)); !ok {
			return LogTail{}, errorWithOutput(err, msg)
		}
	}

	entries := append(parseLogEntries(stdout.String()), parseLogEntries(stderr.String())...)
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Time.Before(entries[j].Time) })
	if !since.IsZero() {
		kept := entries[:0]
		for _, e := range entries {
			if e.Time.After(since) {
				kept = append(kept, e)
			}
		}
		entries = kept
	}
	if len(entries) > tail {
		entries = entries[len(entries)-tail:]
	}

	out := LogTail{
		Lines:       make([]string, 0, len(entries)),
		Times:       make([]string, 0, len(entries)),
		Incremental: !since.IsZero(),
	}
	for _, e := range entries {
		out.Lines = append(out.Lines, e.Text)
		out.Times = append(out.Times, e.Time.UTC().Format(time.RFC3339Nano))
	}
	return out, nil
}

func parseLogEntry(line string) (LogEntry, bool) {
	i := strings.IndexByte(line, ' ')
	if i <= 0 {
		return LogEntry{}, false
	}
	ts, err := time.Parse(time.RFC3339Nano, line[:i])
	if err != nil {
		return LogEntry{}, false
	}
	return LogEntry{Time: ts, Text: cleanLogLine(line[i+1:])}, true
}

func parseLogEntries(raw string) []LogEntry {
	text := strings.TrimRight(raw, "\n\r")
	if text == "" {
		return nil
	}
	var out []LogEntry
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		if entry, ok := parseLogEntry(line); ok {
			out = append(out, entry)
			continue
		}
		if n := len(out); n > 0 {
			out = append(out, LogEntry{Time: out[n-1].Time, Text: cleanLogLine(line)})
		}
	}
	return out
}

func cleanLogLine(line string) string {
	if i := strings.LastIndexByte(line, '\r'); i >= 0 {
		line = line[i+1:]
	}
	line = mcColorCodes.ReplaceAllString(stripANSI(line), "")
	line = strings.TrimRight(line, " \t")
	if !utf8.ValidString(line) {
		line = strings.ToValidUTF8(line, "")
	}
	return line
}

type logCommandError struct {
	err    error
	output string
}

func (e logCommandError) Error() string { return e.err.Error() + ": " + e.output }
func (e logCommandError) Unwrap() error { return e.err }

func errorWithOutput(err error, output string) error {
	return logCommandError{err: err, output: output}
}
