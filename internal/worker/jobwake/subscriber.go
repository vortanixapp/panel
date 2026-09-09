package jobwake

import (
	"context"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

const subject = "vortanix.jobs.wake"

func Subscribe(ctx context.Context, url string) <-chan struct{} {
	ch := make(chan struct{}, 16)
	if url == "" {
		return ch
	}
	nc, err := nats.Connect(url,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		log.Printf("worker jobwake: NATS disabled: %v", err)
		return ch
	}
	_, err = nc.Subscribe(subject, func(_ *nats.Msg) {
		select {
		case ch <- struct{}{}:
		default:
		}
	})
	if err != nil {
		log.Printf("worker jobwake: subscribe failed: %v", err)
		nc.Close()
		return ch
	}
	log.Printf("worker jobwake: subscribed %s", subject)
	go func() {
		<-ctx.Done()
		nc.Close()
	}()
	return ch
}
