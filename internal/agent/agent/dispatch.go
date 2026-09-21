package agent

import (
	"context"
	"errors"
	"log"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

const (
	dispatchGlobalLimit = 8
	dispatchServerReads = 4
	dispatchNodeReads   = 2
	dispatchLaneQueue   = 64
	dispatchPoolQueue   = 64
	dispatchLaneIdle    = time.Minute

	codeAgentBusy  = "agent_busy"
	codeAgentPanic = "agent_panic"
)

var (
	errAgentBusy  = errors.New("агент перегружен, повторите позже")
	errAgentPanic = errors.New("внутренняя ошибка агента")
)

type job struct {
	id      string
	action  string
	timeout time.Duration
	run     func(ctx context.Context)
}

type lane struct {
	jobs   chan job
	active atomic.Bool
}

type dispatcher struct {
	mu       sync.Mutex
	lanes    map[string]*lane
	global   chan struct{}
	srvRead  chan struct{}
	nodeRead chan struct{}
	running  atomic.Int64
	queued   atomic.Int64
	waiting  atomic.Int64
	fail     func(j job, err error, code string)
}

func newDispatcher(fail func(j job, err error, code string)) *dispatcher {
	return &dispatcher{
		lanes:    map[string]*lane{},
		global:   make(chan struct{}, dispatchGlobalLimit),
		srvRead:  make(chan struct{}, dispatchServerReads),
		nodeRead: make(chan struct{}, dispatchNodeReads),
		fail:     fail,
	}
}

func (d *dispatcher) stats() (running, queued int64) {
	return d.running.Load(), d.queued.Load() + d.waiting.Load()
}

func (d *dispatcher) serial(key string, j job) {
	d.mu.Lock()
	l := d.lanes[key]
	if l == nil {
		l = &lane{jobs: make(chan job, dispatchLaneQueue)}
		d.lanes[key] = l
		go d.runLane(key, l)
	}
	select {
	case l.jobs <- j:
		d.queued.Add(1)
		d.mu.Unlock()
	default:
		d.mu.Unlock()
		d.fail(j, errAgentBusy, codeAgentBusy)
	}
}

func (d *dispatcher) laneBusy(key string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	l := d.lanes[key]
	return l != nil && (len(l.jobs) > 0 || l.active.Load())
}

func (d *dispatcher) serverRead(serverID string, j job) {
	key := "srv:" + serverID
	if d.laneBusy(key) {
		d.serial(key, j)
		return
	}
	d.parallel(d.srvRead, j)
}

func (d *dispatcher) nodeReadJob(j job) {
	d.parallel(d.nodeRead, j)
}

func (d *dispatcher) parallel(pool chan struct{}, j job) {
	if d.waiting.Add(1) > dispatchPoolQueue {
		d.waiting.Add(-1)
		d.fail(j, errAgentBusy, codeAgentBusy)
		return
	}
	go func() {
		pool <- struct{}{}
		d.waiting.Add(-1)
		defer func() { <-pool }()
		d.execute(j)
	}()
}

func (d *dispatcher) runLane(key string, l *lane) {
	idle := time.NewTimer(dispatchLaneIdle)
	defer idle.Stop()
	for {
		select {
		case j := <-l.jobs:
			d.queued.Add(-1)
			l.active.Store(true)
			d.execute(j)
			l.active.Store(false)
			if !idle.Stop() {
				select {
				case <-idle.C:
				default:
				}
			}
			idle.Reset(dispatchLaneIdle)
		case <-idle.C:
			d.mu.Lock()
			if len(l.jobs) == 0 {
				delete(d.lanes, key)
				d.mu.Unlock()
				return
			}
			d.mu.Unlock()
			idle.Reset(dispatchLaneIdle)
		}
	}
}

func (d *dispatcher) execute(j job) {
	d.global <- struct{}{}
	d.running.Add(1)
	defer func() {
		d.running.Add(-1)
		<-d.global
		if r := recover(); r != nil {
			log.Printf("команда %s (%s) завершилась аварийно: %v\n%s", j.action, j.id, r, debug.Stack())
			d.fail(j, errAgentPanic, codeAgentPanic)
		}
	}()
	timeout := j.timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	j.run(ctx)
}

func actionTimeout(action string) time.Duration {
	switch action {
	case "logs", "stats", "game_query", "mysql_list_catalog":
		return time.Minute
	case "files_list", "files_read", "files_write", "files_mkdir", "files_delete", "files_write_binary":
		return 5 * time.Minute
	case "power", "ports_sync":
		return 20 * time.Minute
	case "plugins_apply", "maps_apply", "archive_cache_fetch", "destroy":
		return time.Hour
	case "backup_create", "backup_restore", "mysql_migrate_db":
		return 3 * time.Hour
	default:
		return 2 * time.Minute
	}
}
