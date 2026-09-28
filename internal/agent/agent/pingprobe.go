package agent

import (
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	defaultPingProbePort = 19999
	pingProbeMaxConns    = 64
	pingProbeReadWait    = 3 * time.Second
)

func pingProbePort() int {
	raw := strings.TrimSpace(os.Getenv("PING_PORT"))
	if raw == "" {
		return defaultPingProbePort
	}
	if strings.EqualFold(raw, "off") || raw == "0" {
		return 0
	}
	port, err := strconv.Atoi(raw)
	if err != nil || port < 1 || port > 65535 {
		log.Printf("проба задержки: неверное значение PING_PORT %q", raw)
		return defaultPingProbePort
	}
	return port
}

func (a *Agent) startPingProbe() {
	port := pingProbePort()
	if port == 0 {
		return
	}
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		log.Printf("проба задержки не запущена на порту %d: %v", port, err)
		return
	}
	a.pingPort.Store(int64(port))
	safeGo("ping-probe", func() { servePingProbe(ln) })
}

func servePingProbe(ln net.Listener) {
	defer ln.Close()
	var active atomic.Int64
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return
		}
		if active.Load() >= pingProbeMaxConns {
			conn.Close()
			continue
		}
		active.Add(1)
		safeGo("ping-probe-conn", func() {
			defer active.Add(-1)
			defer conn.Close()
			_ = conn.SetReadDeadline(time.Now().Add(pingProbeReadWait))
			one := make([]byte, 1)
			_, _ = conn.Read(one)
		})
	}
}
