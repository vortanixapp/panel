package sshclient

import (
	"testing"
	"time"
)

func TestExecTimeoutDefaults(t *testing.T) {
	if got := (Config{}).execTimeout(); got != defaultExecTimeout {
		t.Errorf("execTimeout() = %s, ожидалось %s", got, defaultExecTimeout)
	}
	if got := (Config{ExecTimeout: 42 * time.Second}).execTimeout(); got != 42*time.Second {
		t.Errorf("явное значение проигнорировано: %s", got)
	}
	if defaultExecTimeout <= 0 {
		t.Fatal("предел по умолчанию должен быть конечным")
	}
}

func TestDialTimeoutIsSeparateFromExec(t *testing.T) {
	cfg := Config{Timeout: time.Second, ExecTimeout: time.Minute}
	if cfg.execTimeout() == cfg.Timeout {
		t.Error("предел выполнения не должен совпадать с пределом соединения")
	}
}
