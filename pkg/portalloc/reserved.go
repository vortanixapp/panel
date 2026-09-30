package portalloc

import (
	"os"
	"strconv"
	"strings"
)

// Порты, которые клиент не должен занимать своим контейнером. Docker
// запускается от root и займёт любой свободный порт хоста, а после этого служба
// ноды или панели при перезапуске уже не поднимется.
var reservedPorts = map[int]bool{
	22:    true, // sshd
	25:    true, // smtp
	80:    true, // caddy
	443:   true, // caddy
	2222:  true, // отдельный sshd для SFTP игровых серверов
	3306:  true, // MySQL 8.0 на ноде
	3307:  true, // MySQL 5.7 на ноде
	3308:  true, // MariaDB на ноде
	4222:  true, // NATS
	5432:  true, // PostgreSQL
	6379:  true, // Redis
	8082:  true, // relay
	8083:  true, // console
	8084:  true, // metrics
	8222:  true, // NATS monitoring
	8443:  true, // relay TLS для агентов
	8444:  true, // phpMyAdmin на ноде
	9090:  true, // Prometheus
	19999: true,
}

// MinClientPort — ниже 1024 порты привилегированные, там живут службы системы.
const MinClientPort = 1024

// Reserved сообщает, занят ли порт инфраструктурой. Список расширяется
// переменной VORTANIX_RESERVED_PORTS со значениями через запятую.
func Reserved(port int) bool {
	if port < MinClientPort || port > 65535 {
		return true
	}
	if reservedPorts[port] {
		return true
	}
	for _, raw := range strings.Split(os.Getenv("VORTANIX_RESERVED_PORTS"), ",") {
		if raw = strings.TrimSpace(raw); raw == "" {
			continue
		}
		if extra, err := strconv.Atoi(raw); err == nil && extra == port {
			return true
		}
	}
	return false
}
