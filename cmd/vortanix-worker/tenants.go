package main

import (
	"context"
	"sync"

	"github.com/vortanixapp/panel/internal/worker/jobs"
)

type waker struct {
	mu  sync.Mutex
	out []chan struct{}
}

func fanout(ctx context.Context, src <-chan struct{}) *waker {
	w := &waker{}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-src:
				w.broadcast()
			}
		}
	}()
	return w
}

func (w *waker) subscribe() <-chan struct{} {
	ch := make(chan struct{}, 16)
	w.mu.Lock()
	w.out = append(w.out, ch)
	w.mu.Unlock()
	return ch
}

func (w *waker) broadcast() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, ch := range w.out {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func startLoops(ctx context.Context, runner *jobs.Runner, wake <-chan struct{}) {
	go runner.Loop(ctx, wake)
	go runner.BackupLoop(ctx, wake)
	go runner.NodeSetupLoop(ctx, wake)
	go runner.DaemonLoop(ctx, wake)
	go runner.MailingLoop(ctx, wake)
	go runner.FTPAccountLoop(ctx, wake)
	go runner.MigrateLoop(ctx, wake)
	go runner.BackupOffsiteLoop(ctx, wake)
	go runner.NodeBulkLoop(ctx, wake)
	go runner.WebhookLoop(ctx)
	go runner.ExpiryLoop(ctx)
	go runner.BackupScheduleLoop(ctx)
	go runner.NotifyDeliveryLoop(ctx)
	go runner.HealthWatchLoop(ctx)
	go runner.StaleJobsLoop(ctx)
}
