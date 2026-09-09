package jobwake

import (
	"time"

	"github.com/nats-io/nats.go"
)

func natsConnect(url string) (*nats.Conn, error) {
	return nats.Connect(url,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.Timeout(5*time.Second),
	)
}
