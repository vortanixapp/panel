package docker

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"unicode/utf8"
)

func TailLogs(ctx context.Context, serverID string, tail int) ([]string, error) {
	if tail <= 0 {
		tail = 100
	}
	if tail > 1000 {
		tail = 1000
	}
	cname := ContainerName(serverID)
	cmd := exec.CommandContext(
		ctx,
		"docker",
		"logs",
		"--tail",
		strconv.Itoa(tail),
		cname,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil && stdout.Len() == 0 {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return nil, err
		}
		return nil, errorWithOutput(err, msg)
	}

	lines := splitLogLines(stdout.String())
	if extra := splitLogLines(stderr.String()); len(extra) > 0 && err == nil {
		lines = append(lines, extra...)
	}
	return lines, nil
}

func splitLogLines(raw string) []string {
	text := strings.TrimRight(raw, "\n\r")
	if text == "" {
		return nil
	}
	out := []string{}
	for _, line := range strings.Split(text, "\n") {
		if i := strings.LastIndexByte(line, '\r'); i >= 0 {
			line = line[i+1:]
		}
		line = mcColorCodes.ReplaceAllString(stripANSI(line), "")
		line = strings.TrimRight(line, " \t")
		if !utf8.ValidString(line) {
			line = strings.ToValidUTF8(line, "")
		}
		out = append(out, line)
	}
	return out
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
