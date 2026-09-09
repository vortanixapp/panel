package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/vortanixapp/panel/internal/worker/jobs"
	"github.com/vortanixapp/panel/internal/worker/mail"
	"github.com/vortanixapp/panel/internal/worker/relay"
	"github.com/vortanixapp/panel/pkg/secretbox"
	"github.com/vortanixapp/panel/pkg/tenantpools"
)

const tenantRescanInterval = time.Minute

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

func superviseTenants(
	ctx context.Context,
	tenants *tenantpools.Pools,
	relayClient *relay.Client,
	mailCfg mail.Config,
	secrets *secretbox.Box,
	telegramBotToken string,
	panelURL string,
	wake *waker,
) {
	started := map[string]bool{}

	scan := func() {
		names, err := tenants.Names(ctx)
		if err != nil {
			log.Printf("worker tenants: реестр недоступен: %v", err)
			return
		}
		for _, name := range names {
			if started[name] {
				continue
			}
			pool, err := tenants.ByName(ctx, name)
			if err != nil {
				log.Printf("worker tenants: база %s недоступна: %v", name, err)
				continue
			}
			startLoops(ctx, jobs.New(pool, relayClient, mailCfg, secrets).WithNotify(telegramBotToken, panelURL), wake.subscribe())
			started[name] = true
			log.Printf("worker tenants: обработка задач базы %s запущена", name)
		}
	}

	scan()
	ticker := time.NewTicker(tenantRescanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			scan()
		}
	}
}
