package sshclient

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

func StreamBetween(src Config, srcCmd string, dst Config, dstCmd string, progress func(int64)) (int64, error) {
	if src.Host == "" || src.User == "" || !src.hasAuth() {
		return 0, fmt.Errorf("ssh источника: не заданы адрес, пользователь или пароль")
	}
	if dst.Host == "" || dst.User == "" || !dst.hasAuth() {
		return 0, fmt.Errorf("ssh приёмника: не заданы адрес, пользователь или пароль")
	}

	srcClient, err := src.dial()
	if err != nil {
		return 0, fmt.Errorf("ssh источника: %w", err)
	}
	defer srcClient.Close()

	dstClient, err := dst.dial()
	if err != nil {
		return 0, fmt.Errorf("ssh приёмника: %w", err)
	}
	defer dstClient.Close()

	srcSession, err := srcClient.NewSession()
	if err != nil {
		return 0, fmt.Errorf("ssh источника: сессия: %w", err)
	}
	defer srcSession.Close()

	dstSession, err := dstClient.NewSession()
	if err != nil {
		return 0, fmt.Errorf("ssh приёмника: сессия: %w", err)
	}
	defer dstSession.Close()

	srcOut, err := srcSession.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("ssh источника: stdout: %w", err)
	}
	dstIn, err := dstSession.StdinPipe()
	if err != nil {
		return 0, fmt.Errorf("ssh приёмника: stdin: %w", err)
	}

	var srcErr, dstErr limitedBuffer
	srcSession.Stderr = &srcErr
	dstSession.Stderr = &dstErr

	if err := dstSession.Start("/bin/bash -lc " + shellQuote(dstCmd)); err != nil {
		return 0, fmt.Errorf("ssh приёмника: запуск: %w", err)
	}
	if err := srcSession.Start("/bin/bash -lc " + shellQuote(srcCmd)); err != nil {
		return 0, fmt.Errorf("ssh источника: запуск: %w", err)
	}

	counter := &countingWriter{onProgress: progress}
	copied, copyErr := io.Copy(io.MultiWriter(dstIn, counter), srcOut)
	closeErr := dstIn.Close()

	srcWait := waitWithTimeout(srcSession, src.execTimeout())
	dstWait := waitWithTimeout(dstSession, dst.execTimeout())

	if copyErr != nil {
		return copied, fmt.Errorf("передача: %w%s", copyErr, sideDetails(&srcErr, &dstErr))
	}
	if srcWait != nil {
		return copied, fmt.Errorf("источник: %w%s", srcWait, sideDetails(&srcErr, &dstErr))
	}
	if dstWait != nil {
		return copied, fmt.Errorf("приёмник: %w%s", dstWait, sideDetails(&srcErr, &dstErr))
	}
	if closeErr != nil && copied == 0 {
		return copied, fmt.Errorf("передача: пустой поток: %w", closeErr)
	}
	return copied, nil
}

type sessionWaiter interface {
	Wait() error
	Close() error
}

func waitWithTimeout(s sessionWaiter, limit time.Duration) error {
	done := make(chan error, 1)
	go func() { done <- s.Wait() }()
	timer := time.NewTimer(limit)
	defer timer.Stop()
	select {
	case err := <-done:
		return err
	case <-timer.C:
		_ = s.Close()
		<-done
		return fmt.Errorf("команда не завершилась за %s", limit)
	}
}

func sideDetails(src, dst *limitedBuffer) string {
	parts := []string{}
	if s := strings.TrimSpace(src.String()); s != "" {
		parts = append(parts, "источник: "+s)
	}
	if s := strings.TrimSpace(dst.String()); s != "" {
		parts = append(parts, "приёмник: "+s)
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, "; ") + ")"
}

type countingWriter struct {
	total      int64
	reported   int64
	onProgress func(int64)
}

const progressStep = 64 << 20

func (w *countingWriter) Write(p []byte) (int, error) {
	w.total += int64(len(p))
	if w.onProgress != nil && w.total-w.reported >= progressStep {
		w.reported = w.total
		w.onProgress(w.total)
	}
	return len(p), nil
}

type limitedBuffer struct {
	mu  sync.Mutex
	buf []byte
}

const stderrLimit = 8 << 10

func (b *limitedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	if len(b.buf) > stderrLimit {
		b.buf = b.buf[len(b.buf)-stderrLimit:]
	}
	return len(p), nil
}

func (b *limitedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

func StreamFrom(cfg Config, cmd string, w io.Writer) (int64, error) {
	if cfg.Host == "" || cfg.User == "" || !cfg.hasAuth() {
		return 0, fmt.Errorf("ssh: не заданы адрес, пользователь или пароль")
	}
	client, err := cfg.dial()
	if err != nil {
		return 0, fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return 0, fmt.Errorf("ssh session: %w", err)
	}
	defer session.Close()

	out, err := session.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("ssh stdout: %w", err)
	}
	var stderr limitedBuffer
	session.Stderr = &stderr

	if err := session.Start("/bin/bash -lc " + shellQuote(cmd)); err != nil {
		return 0, fmt.Errorf("ssh start: %w", err)
	}
	copied, copyErr := io.Copy(w, out)
	waitErr := waitWithTimeout(session, cfg.execTimeout())
	if copyErr != nil {
		return copied, fmt.Errorf("чтение с ноды: %w%s", copyErr, sideDetails(&stderr, &limitedBuffer{}))
	}
	if waitErr != nil {
		return copied, fmt.Errorf("команда на ноде: %w%s", waitErr, sideDetails(&stderr, &limitedBuffer{}))
	}
	return copied, nil
}

func StreamTo(cfg Config, cmd string, r io.Reader) (int64, error) {
	if cfg.Host == "" || cfg.User == "" || !cfg.hasAuth() {
		return 0, fmt.Errorf("ssh: не заданы адрес, пользователь или пароль")
	}
	client, err := cfg.dial()
	if err != nil {
		return 0, fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return 0, fmt.Errorf("ssh session: %w", err)
	}
	defer session.Close()

	in, err := session.StdinPipe()
	if err != nil {
		return 0, fmt.Errorf("ssh stdin: %w", err)
	}
	var stderr limitedBuffer
	session.Stderr = &stderr

	if err := session.Start("/bin/bash -lc " + shellQuote(cmd)); err != nil {
		return 0, fmt.Errorf("ssh start: %w", err)
	}
	copied, copyErr := io.Copy(in, r)
	_ = in.Close()
	waitErr := waitWithTimeout(session, cfg.execTimeout())
	if copyErr != nil {
		return copied, fmt.Errorf("запись на ноду: %w%s", copyErr, sideDetails(&limitedBuffer{}, &stderr))
	}
	if waitErr != nil {
		return copied, fmt.Errorf("команда на ноде: %w%s", waitErr, sideDetails(&limitedBuffer{}, &stderr))
	}
	return copied, nil
}
