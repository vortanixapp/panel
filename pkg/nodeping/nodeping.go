package nodeping

import (
	"context"
	"net"
	"strconv"
	"time"
)

const Port = 19999

const dialTimeout = 1500 * time.Millisecond

func Addr(host string, port int) string {
	if port <= 0 {
		port = Port
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func Available(ctx context.Context, host string, port int) bool {
	if host == "" {
		return false
	}
	d := net.Dialer{Timeout: dialTimeout}
	conn, err := d.DialContext(ctx, "tcp", Addr(host, port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
