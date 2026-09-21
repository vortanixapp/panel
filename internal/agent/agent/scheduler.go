package agent

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/vortanixapp/panel/internal/agent/docker"
	"github.com/vortanixapp/panel/pkg/cronexpr"
)

const (
	cronJobTimeout   = 10 * time.Minute
	cronFailInterval = time.Hour
	cronParallel     = 8
)

type cronEntry struct {
	job  docker.CronJob
	expr *cronexpr.Expr
}

type cronServer struct {
	loc     *time.Location
	entries []cronEntry
}

type cronScheduler struct {
	mu        sync.Mutex
	servers   map[string]cronServer
	running   map[string]bool
	lastWall  map[string]string
	failAt    map[string]time.Time
	lastTick  time.Time
	slots     chan struct{}
	wg        sync.WaitGroup
	isRunning func(ctx context.Context, serverID string) bool
	exec      func(ctx context.Context, serverID string, job docker.CronJob) docker.CronRun
	record    func(serverID string, run docker.CronRun)
	onFail    func(serverID string, run docker.CronRun) bool
}

func newCronScheduler(onFail func(serverID string, run docker.CronRun) bool) *cronScheduler {
	return &cronScheduler{
		servers:  map[string]cronServer{},
		running:  map[string]bool{},
		lastWall: map[string]string{},
		failAt:   map[string]time.Time{},
		slots:    make(chan struct{}, cronParallel),
		isRunning: func(ctx context.Context, serverID string) bool {
			return docker.Status(ctx, serverID) == "running"
		},
		exec:   docker.RunCronJob,
		record: docker.RecordCronRun,
		onFail: onFail,
	}
}

func (s *cronScheduler) load(serverID string) {
	state, ok := docker.ReadCronState(serverID)
	if !ok {
		s.drop(serverID)
		return
	}
	s.set(serverID, state)
}

func (s *cronScheduler) set(serverID string, state docker.CronState) {
	loc := time.UTC
	if state.TZ != "" {
		if l, err := time.LoadLocation(state.TZ); err == nil {
			loc = l
		}
	}
	entries := make([]cronEntry, 0, len(state.Jobs))
	keep := map[string]bool{}
	for _, j := range state.Jobs {
		if !j.Enabled {
			continue
		}
		expr, err := cronexpr.Parse(j.Schedule)
		if err != nil {
			continue
		}
		entries = append(entries, cronEntry{job: j, expr: expr})
		keep[serverID+"/"+j.ID] = true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(serverID, keep)
	if len(entries) == 0 {
		delete(s.servers, serverID)
		return
	}
	s.servers[serverID] = cronServer{loc: loc, entries: entries}
}

func (s *cronScheduler) drop(serverID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.servers, serverID)
	s.pruneLocked(serverID, nil)
}

func (s *cronScheduler) pruneLocked(serverID string, keep map[string]bool) {
	prefix := serverID + "/"
	for key := range s.lastWall {
		if strings.HasPrefix(key, prefix) && !keep[key] {
			delete(s.lastWall, key)
		}
	}
	for key := range s.failAt {
		if strings.HasPrefix(key, prefix) && !keep[key] {
			delete(s.failAt, key)
		}
	}
}

func (s *cronScheduler) jobs() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, srv := range s.servers {
		n += len(srv.entries)
	}
	return n
}

func (s *cronScheduler) run() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	for _, id := range docker.StateServers(ctx) {
		s.load(id)
	}
	cancel()
	if n := s.jobs(); n > 0 {
		log.Printf("планировщик cron: заданий %d", n)
	}
	for {
		next := time.Now().Truncate(time.Minute).Add(time.Minute)
		time.Sleep(time.Until(next))
		s.tick(next)
	}
}

func (s *cronScheduler) tick(at time.Time) {
	type due struct {
		serverID string
		job      docker.CronJob
	}
	var list []due
	s.mu.Lock()
	if !at.After(s.lastTick) {
		s.mu.Unlock()
		return
	}
	s.lastTick = at
	for id, srv := range s.servers {
		local := at.In(srv.loc)
		wall := local.Format("2006-01-02 15:04")
		for _, e := range srv.entries {
			if !e.expr.Match(local) {
				continue
			}
			key := id + "/" + e.job.ID
			if e.expr.FixedTime() {
				if s.lastWall[key] == wall {
					continue
				}
				s.lastWall[key] = wall
			}
			if s.running[key] {
				continue
			}
			s.running[key] = true
			list = append(list, due{serverID: id, job: e.job})
		}
	}
	s.mu.Unlock()
	for _, d := range list {
		s.wg.Add(1)
		go s.fire(d.serverID, d.job)
	}
}

func (s *cronScheduler) fire(serverID string, job docker.CronJob) {
	key := serverID + "/" + job.ID
	defer s.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("планировщик cron: сбой задания %s: %v", key, r)
		}
		s.mu.Lock()
		delete(s.running, key)
		s.mu.Unlock()
	}()
	s.slots <- struct{}{}
	defer func() { <-s.slots }()
	ctx, cancel := context.WithTimeout(context.Background(), cronJobTimeout)
	defer cancel()
	if !s.isRunning(ctx, serverID) {
		return
	}
	run := s.exec(ctx, serverID, job)
	s.record(serverID, run)
	if run.Error == "" {
		return
	}
	s.mu.Lock()
	due := time.Since(s.failAt[key]) >= cronFailInterval
	s.mu.Unlock()
	if !due || s.onFail == nil || !s.onFail(serverID, run) {
		return
	}
	s.mu.Lock()
	s.failAt[key] = time.Now()
	s.mu.Unlock()
}
