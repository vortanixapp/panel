package netaddr

import (
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"
)

var ErrPrivateAddress = errors.New("адрес находится во внутренней сети: исходящие запросы туда запрещены (разрешить можно переменной OUTBOUND_ALLOW_PRIVATE=1)")

func allowPrivateOutbound() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("OUTBOUND_ALLOW_PRIVATE"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func privateAddress(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1]&0xc0 == 64 {
		return true
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast()
}

func publicOnly(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	if privateAddress(net.ParseIP(host)) {
		return ErrPrivateAddress
	}
	return nil
}

func OutboundClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: timeout,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
	}
	if !allowPrivateOutbound() {
		dialer.Control = publicOnly
	} else {
		transport.Proxy = http.ProxyFromEnvironment
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("слишком много перенаправлений")
			}
			return nil
		},
	}
}
