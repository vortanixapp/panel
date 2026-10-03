# Monitoring

[English](monitoring.md) · [Русский](monitoring.ru.md)

Optional Prometheus and Grafana next to the panel. They show how the API, the
database and the relay behave under load, so slow spots are found by numbers
instead of guesses. Nothing starts unless you ask for it.

## Start

```bash
docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml --profile monitoring up -d
```

Grafana listens on port `3001` of the machine (`GRAFANA_PORT`), so no domain is
needed. Open `https://<machine IP>:3001`. The certificate is generated on the
first start and is self-signed, so the browser warns once: continue, the
connection is encrypted anyway. Open port 3001 in the firewall if there is one.

Log in as `admin`. The password is `GRAFANA_ADMIN_PASSWORD` from `deploy/.env`
(`scripts/init-env.sh` generates it). If it is empty, a random one is created,
show it with:

```bash
docker compose -f deploy/docker-compose.yml exec grafana cat /certs/admin-password
```

Grafana asks for it only on the very first start. To change it later use the
Grafana profile page. The dashboard **Vortanix: обзор** is installed
automatically.

Prometheus is not published at all, it is reachable only inside the Docker
network, because it has no login of its own. Any query can be run in Grafana:
**Explore**, data source Prometheus. Metrics are kept for 15 days, change it with
`PROMETHEUS_RETENTION`.

The same key charts are also in the panel itself: **Administration →
Performance**. They need no extra port or password and work on a domain and on an
IP alike.

## What the dashboard shows

- API requests per second and p95 latency per route, 5xx errors.
- Database: p95 query time for the write and read pools, slow queries, pool
  connections in use and time spent waiting for a free connection.
- Go goroutines and memory of the API.
- Relay requests and latency (agent traffic).

## Reading it

- A route with high p95 and many requests per second is the first thing to
  optimize.
- Panel titles are in Russian. "Соединения пула базы" (pool connections)
  touching the limit together with a growing "Ожидание соединения из пула"
  (waiting for a connection) means the pool is too small or some queries hold
  connections too long.
- Slow queries are counted here, the query text is in the API log as
  `slow query ... sql=...`. The threshold is `DB_SLOW_QUERY_MS` (500 ms by
  default).

## Profiling

For a CPU or memory profile of the API add `PPROF_ADDR=127.0.0.1:6060` to
`deploy/.env` and recreate the `api` service. The address is inside the
container and is not published through Caddy. Take the profile in the container
and analyse it anywhere:

```bash
docker compose -f deploy/docker-compose.yml exec api wget -qO /tmp/cpu.pprof "http://127.0.0.1:6060/debug/pprof/profile?seconds=30"
docker compose -f deploy/docker-compose.yml cp api:/tmp/cpu.pprof .
go tool pprof cpu.pprof
```
