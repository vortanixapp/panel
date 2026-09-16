package sshclient

import (
	"bytes"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	Host         string
	Port         int
	User         string
	Password     string
	PrivateKey   string
	Timeout      time.Duration
	ExecTimeout  time.Duration
	KnownHostKey string
	OnHostKey    func(fingerprint string)
}

type HostKeyError struct {
	Host     string
	Expected string
	Got      string
}

func (e *HostKeyError) Error() string {
	return fmt.Sprintf(
		"ключ SSH у %s изменился: панель помнит %s, а нода представилась %s. "+
			"Так выглядит и переустановка ноды, и подмена в сети. Если ноду переставляли, "+
			"сбросьте отпечаток на её странице и повторите",
		e.Host, e.Expected, e.Got)
}

func IsHostKeyError(err error) bool {
	var hk *HostKeyError
	return errors.As(err, &hk)
}

func (c Config) hostKeyCallback() ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fingerprint := ssh.FingerprintSHA256(key)
		known := strings.TrimSpace(c.KnownHostKey)
		if known != "" {
			if subtle.ConstantTimeCompare([]byte(known), []byte(fingerprint)) != 1 {
				return &HostKeyError{Host: c.Host, Expected: known, Got: fingerprint}
			}
			return nil
		}
		if c.OnHostKey != nil {
			c.OnHostKey(fingerprint)
		}
		return nil
	}
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

func (c Config) authMethods() ([]ssh.AuthMethod, error) {
	var auth []ssh.AuthMethod
	if strings.TrimSpace(c.PrivateKey) != "" {
		signer, err := ssh.ParsePrivateKey([]byte(c.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("ssh: не разобран приватный ключ: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}
	if c.Password != "" {
		auth = append(auth, ssh.Password(c.Password))
	}
	if len(auth) == 0 {
		return nil, fmt.Errorf("ssh: не задан ни пароль, ни приватный ключ")
	}
	return auth, nil
}

func (c Config) hasAuth() bool {
	return c.Password != "" || strings.TrimSpace(c.PrivateKey) != ""
}

func (c Config) dial() (*ssh.Client, error) {
	if c.Port <= 0 {
		c.Port = 22
	}
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}
	auth, err := c.authMethods()
	if err != nil {
		return nil, err
	}
	cfg := &ssh.ClientConfig{
		User:            c.User,
		Auth:            auth,
		HostKeyCallback: c.hostKeyCallback(),
		Timeout:         c.Timeout,
	}
	return ssh.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port), cfg)
}

func Run(cfg Config, commands []string, log io.Writer) error {
	if cfg.Host == "" || cfg.User == "" || !cfg.hasAuth() {
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

	var tail limitedBuffer
	session.Stdout = io.MultiWriter(log, &tail)
	session.Stderr = io.MultiWriter(log, &tail)

	script := strings.Join(commands, "\n")
	wrapped := "set -e\nexport DEBIAN_FRONTEND=noninteractive\n" + script + "\n"
	if err := runSession(session, "/bin/bash -lc "+shellQuote(wrapped), cfg.execTimeout()); err != nil {
		if msg := lastLines(tail.String(), 10); msg != "" {
			return fmt.Errorf("%s", msg)
		}
		return err
	}
	return nil
}

func lastLines(s string, n int) string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimRight(line, "\r "); strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

func Probe(cfg Config, command string) (string, error) {
	if cfg.Host == "" || cfg.User == "" || !cfg.hasAuth() {
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
	if err := runSession(session, command, cfg.execTimeout()); err != nil {
		return buf.String(), err
	}
	return buf.String(), nil
}

func RunCapture(cfg Config, command string) (string, error) {
	if cfg.Host == "" || cfg.User == "" || !cfg.hasAuth() {
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
	if err := session.Run("/bin/bash -lc " + shellQuote(command)); err != nil {
		return buf.String(), err
	}
	return buf.String(), nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
