# Monitoring

[English](monitoring.md) · [Русский](monitoring.ru.md)

Prometheus and Grafana next to the panel. They show how the API, the
database and the relay behave under load, so slow spots are found by numbers
instead of guesses. It works out of the box.

## It runs out of the box

Prometheus and Grafana are part of the panel stack: they start with the panel and
update together with it, nothing is enabled by hand. On an existing panel they
appear after the update that brings them (the updater starts the new services
itself). If their images cannot be downloaded, the panel is still updated and
monitoring is left as it is.

They use up to about 0.9 GB of memory together (limits 512 MB and 384 MB) and a
couple of GB of disk for metrics (`PROMETHEUS_RETENTION_SIZE`).

## Where to find the access details

**Administration → Settings → Monitoring** shows everything in one place: the
Grafana address (built from the address you opened the panel on, so it works on a
domain and on an IP), the login, the password with show and copy buttons, the
state of Grafana and Prometheus, the retention and which services the metrics
are collected from.

Grafana listens on port `3001` of the machine (`GRAFANA_PORT`), so no domain is
needed. Open `https://<machine IP>:3001`. The certificate is generated on the
first start and is self-signed, so the browser warns once: continue, the
connection is encrypted anyway. Open port 3001 in the firewall if there is one.
To keep Grafana off the internet set `GRAFANA_BIND=127.0.0.1` in `deploy/.env`
and use an SSH tunnel (`ssh -L 3001:127.0.0.1:3001 user@host`).

The login is `admin`. The password is `GRAFANA_ADMIN_PASSWORD` from `deploy/.env`
(`scripts/init-env.sh` generates it); if it is empty, a random one is created. It
is applied on the very first start of Grafana, later it is changed in the Grafana
profile page and the panel keeps showing the initial one. The dashboard
**Vortanix: обзор** is installed automatically.

Prometheus is not published at all, it is reachable only inside the Docker
network, because it has no login of its own. Any query can be run in Grafana:
**Explore**, data source Prometheus. Retention is `PROMETHEUS_RETENTION` (15 days)
and `PROMETHEUS_RETENTION_SIZE` (2 GB), whichever is reached first.

The same key charts are also in the panel itself: **Administration →
Performance**. They need no extra port or password and work on a domain and on an
IP alike.

## Updates and moving the panel

- **Updates.** Monitoring is updated together with the panel. The configuration
  (`deploy/monitoring`) comes with the release files, as the rest of `deploy`.
  Metrics and Grafana settings are kept in Docker volumes and survive updates.
- **Moving the panel.** The database, uploads and secrets are moved, monitoring
  data is not: on the new machine the charts start from zero and the Grafana
  password is generated again (see [panel-transfer.md](panel-transfer.md)).

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
- Postgres also logs queries slower than `PG_SLOW_QUERY_MS` (500 ms): `docker compose logs postgres`.
  The `pg_stat_statements` extension collects statistics for all queries; the heaviest ones:
  `docker compose exec postgres psql -U vortanix -c "SELECT calls, round(mean_exec_time) AS ms, left(query, 100) FROM pg_stat_statements ORDER BY total_exec_time DESC LIMIT 15"`.
  Postgres restarts once after the update to pick up these settings.

## Using the numbers

To act on what the charts show, see [performance-tuning.md](performance-tuning.md).

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
