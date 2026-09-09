package sshclient

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	Host        string
	Port        int
	User        string
	Password    string
	Timeout     time.Duration
	ExecTimeout time.Duration
}

const defaultExecTimeout = 10 * time.Minute

func (c Config) execTimeout() time.Duration {
	if c.ExecTimeout > 0 {
		return c.ExecTimeout
	}
	return defaultExecTimeout
}

func runSession(session *ssh.Session, cmd string, limit time.Duration) error {
	done := make(chan error, 1)
	go func() { done <- session.Run(cmd) }()
	timer := time.NewTimer(limit)
	defer timer.Stop()
	select {
	case err := <-done:
		return err
	case <-timer.C:
		_ = session.Close()
		<-done
		return fmt.Errorf("команда не завершилась за %s", limit)
	}
}

func (c Config) dial() (*ssh.Client, error) {
	if c.Port <= 0 {
		c.Port = 22
	}
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}
	cfg := &ssh.ClientConfig{
		User:            c.User,
		Auth:            []ssh.AuthMethod{ssh.Password(c.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         c.Timeout,
	}
	return ssh.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port), cfg)
}

func Run(cfg Config, commands []string, log io.Writer) error {
	if cfg.Host == "" || cfg.User == "" || cfg.Password == "" {
		return fmt.Errorf("ssh host, user and password are required")
	}
	client, err := cfg.dial()
	if err != nil {
		return fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("ssh session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = io.MultiWriter(log, &stdout)
	session.Stderr = io.MultiWriter(log, &stderr)

	script := strings.Join(commands, "\n")
	wrapped := "set -e\nexport DEBIAN_FRONTEND=noninteractive\n" + script + "\n"
	if err := runSession(session, "/bin/bash -lc "+shellQuote(wrapped), cfg.execTimeout()); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func RunCapture(cfg Config, command string) (string, error) {
	if cfg.Host == "" || cfg.User == "" || cfg.Password == "" {
		return "", fmt.Errorf("ssh host, user and password are required")
	}
	client, err := cfg.dial()
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var buf bytes.Buffer
	session.Stdout = &buf
	session.Stderr = &buf
	if err := runSession(session, "/bin/bash -lc "+shellQuote(command), cfg.execTimeout()); err != nil {
		return buf.String(), err
	}
	return buf.String(), nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
