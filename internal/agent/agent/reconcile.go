package agent

import (
	"bufio"
	"context"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/vortanixapp/panel/internal/agent/docker"
)

const (
	firewallDebounce     = 300 * time.Millisecond
	firewallApplyTimeout = 2 * time.Minute
	firewallFailInterval = time.Hour
	dockerEventsRetry    = 10 * time.Second
)

type firewallKeeper struct {
	mu     sync.Mutex
	timers map[string]*time.Timer
	failAt map[string]time.Time
	apply  func(ctx context.Context, serverID string) error
	onFail func(serverID string, err error) bool
}

func newFirewallKeeper(onFail func(serverID string, err error) bool) *firewallKeeper {
	return &firewallKeeper{
		timers: map[string]*time.Timer{},
		failAt: map[string]time.Time{},
		apply:  docker.ApplyFirewall,
		onFail: onFail,
	}
}

func (k *firewallKeeper) schedule(serverID string) {
	if !docker.HasFirewallRules(serverID) {
		return
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if t, ok := k.timers[serverID]; ok {
		t.Reset(firewallDebounce)
		return
	}
	k.timers[serverID] = time.AfterFunc(firewallDebounce, func() {
		k.mu.Lock()
		delete(k.timers, serverID)
		k.mu.Unlock()
		k.applyNow(serverID)
	})
}

func (k *firewallKeeper) applyNow(serverID string) {
	ctx, cancel := context.WithTimeout(context.Background(), firewallApplyTimeout)
	defer cancel()
	err := k.apply(ctx, serverID)
	if err == nil {
		k.mu.Lock()
		delete(k.failAt, serverID)
		k.mu.Unlock()
		return
	}
	log.Printf("файрвол сервера %s не применён: %v", serverID, err)
	k.report(serverID, err)
}

func (k *firewallKeeper) report(serverID string, err error) {
	k.mu.Lock()
	due := time.Since(k.failAt[serverID]) >= firewallFailInterval
	k.mu.Unlock()
	if !due || k.onFail == nil || !k.onFail(serverID, err) {
		return
	}
	k.mu.Lock()
	k.failAt[serverID] = time.Now()
	k.mu.Unlock()
}

func (k *firewallKeeper) reconcileAll() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	ids := docker.StateServers(ctx)
	cancel()
	for _, id := range ids {
		if docker.HasFirewallRules(id) {
			k.applyNow(id)
		}
	}
}

func (k *firewallKeeper) watch() {
	for {
		if err := k.follow(); err != nil {
			log.Printf("поток событий Docker прервался: %v", err)
		}
		time.Sleep(dockerEventsRetry)
	}
}

func (k *firewallKeeper) follow() error {
	cmd := exec.Command("docker", "events",
		"--filter", "type=container",
		"--filter", "event=start",
		"--filter", "event=die",
		"--format", "{{.Action}} {{.Actor.Attributes.name}}")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	settled := time.AfterFunc(time.Second, k.reconcileAll)
	defer settled.Stop()
	sc := bufio.NewScanner(out)
	for sc.Scan() {
		action, name, _ := strings.Cut(strings.TrimSpace(sc.Text()), " ")
		if action != "start" && action != "die" {
			continue
		}
		if id := docker.ServerIDFromContainer(name); id != "" {
			k.schedule(id)
		}
	}
	return cmd.Wait()
}
