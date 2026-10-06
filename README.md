<h1>
<p align="center">
<br>Warnly
</h1>
  <p align="center">
    Exception monitoring system, designed specifically for self-hosting
    <br />
    <a href="#about">About</a>
    ·
    <a href="#demo">Demo</a>
    ·
    <a href="#documentation">Documentation</a>
    ·
    <a href="internal#project-structure">Developing</a>
  </p>
</p>

## About

Error logs should be categorized into issues, with each issue assigned to the appropriate team member. In an ideal scenario, a well-functioning application should operate silently. Warnly, in line with Sentry's best practices, address this effectively.

Enterprise-focused solutions tend to prioritize complex features that create unnecessary overhead for self-hosting scenarios. There's an opportunity to take the core monitoring functionality and package it into a single binary that eliminates operational complexity while maintaining essential features. That's how Warnly was born: an open-source project designed specifically for self-hosting.

For more details, see [About Warnly](https://docs.warnly.io/).

## Demo

Try the demo application at [https://demo.warnly.io](https://demo.warnly.io). Use username `admin` and password `admin` to sign in. The infrastructure is graciously provided by [VPSDime](https://vpsdime.com/).

## Documentation

See the [documentation](https://docs.warnly.io/) on the Warnly website.

### Analytics storage

ClickHouse remains the default. Alternatively, Warnly supports
[MySQL with the DuckDB storage engine](https://github.com/EvgeniyPatlan/ducksdb-mysql-engine)
using `evgeniypatlan/test-images:mysql-9.7-duckdb-v0.2.0`.

To run the development stack with this backend (Docker Compose 2.24.4+):

```sh
docker compose -f docker-compose.yml -f docker-compose.duckdb.yml up --build
```

For an existing server, create a dedicated analytics database, grant the application
user access to it, and configure:

```sh
ANALYTICS_BACKEND=mysql-duckdb
ANALYTICS_DSN='warnly:password@tcp(localhost:3306)/warnly_analytics?parseTime=true&interpolateParams=true'
KAFKA_BROKERS=
```

The transactional database still uses `MYSQL_DSN`. It can be on the same MySQL/DuckDB
server, with a separate database. `CLICKHOUSE_DSN` is unnecessary in this mode.
Initial analytics migrations create `event` and `event_tag` with `ENGINE=DuckDB`;
the server must provide that engine. Existing ClickHouse data is not migrated.

The driver enables parameter interpolation for DuckDB query pushdown and uses UTC.
Events and their tags are written atomically; retrying an event ID replaces it within
its project. Expired events are excluded from reads immediately and physically removed
at startup and hourly. Kafka ingestion currently relies on ClickHouse's Kafka engine,
so `mysql-duckdb` requires direct ingestion with `KAFKA_BROKERS` empty.

The system pages read MySQL `information_schema` and `performance_schema`; grant
`SELECT` on `performance_schema.*` to the application user for error and query statistics.
Query statistics are cumulative since startup/reset, and per-query read bytes are
unavailable. Table sizes and row counts are the estimates reported by the engine.
Text search and tag-value comparisons use MySQL's fallback execution path to work
around v0.2.0 engine limitations; supported aggregate queries execute in DuckDB.

Run the container integration tests (Docker required):

```sh
make test-duckdb
```

The tests also run in the existing `INTEGRATION=1 go test ./...` CI job. They verify
the actual table engines, migrations, event round trips, filters, metrics, tags,
pagination, retention, and the engine's query-pushdown counter.

## Development and builds (Linux / macOS)

The UI uses Svelte 5 and SvelteKit with `@sveltejs/adapter-static` in SPA mode.
Go serves the compiled HTML, CSS and JavaScript embedded in the binary, including
fallbacks for direct links to projects and issues. Node.js is only a build tool;
there is no Node server in production. The authenticated JSON API lives under
`/api`; Sentry ingestion and OIDC callback URLs are unchanged.

For the complete development stack, install Docker Engine with Compose v2 on
Linux or Docker Desktop on macOS, then run:

```sh
make dev
```

This creates `.env` from `.env.sample` if absent, installs the build dependencies
in Docker, starts MySQL and ClickHouse, builds the frontend and runs Go with Air.
Open <http://localhost:8080>. The sample login is `admin` / `admin`.
Air rebuilds the static frontend and Go binary when source files change; refresh
the browser after rebuilding. `make dev-down` stops the stack without deleting data.
Use `COMPOSE="docker-compose"` if your installation uses that command.
Existing `.env` files are preserved. For the default direct-ingestion setup,
leave `KAFKA_BROKERS` empty; Kafka requires the optional `queue` Compose profile.

For a native build (amd64 / arm64):

```sh
make install
make build
```

`make install` requires `make`, `curl` and `tar`. It installs the Go version from
`go.mod` and Node 22 locally in `.tools` when suitable tools are missing, then
installs the locked npm dependencies and Go modules. It requires no sudo and does
not modify shell startup files. Make targets automatically use these local tools.
`make setup` is an alias; pre-commit installation is available as `make hooks`.

The result is `bin/warnly`, which includes the static UI. `make run` builds and
starts it, reading `.env` (environment variables take precedence). For native
execution, set database DSNs to reachable addresses: Compose exposes MySQL at
`localhost:3326` and ClickHouse at `localhost:9030`. The sample DSNs use Docker
service names and are intended for `make dev`.

For Svelte hot reload, run `make frontend-dev` alongside the Go server. Open the
Vite URL printed in the terminal; `/api`, `/oidc` and `/ingest` are proxied to
`http://127.0.0.1:8080`. Override this with `WARNLY_API_URL` if needed. Production
and `make dev` serve the UI directly from Go at port 8080.

```sh
make frontend-check # Svelte diagnostics
make frontend-build # Install and compile static assets
make frontend-test  # Browser scenarios (installs Chromium)
make test-unit      # Go tests without container integration
make test           # Go integration tests; requires Docker
```

Always build the frontend before a direct `go build` or release. Generated files
in `internal/server/frontend/dist` are ignored by Git. Docker and GoReleaser do
this automatically. A Go binary built without these files returns a clear 503
for UI requests; API and unit tests can still run without Node.

## Roadmap and Status

The high-level plan for `warnly`, in order:

|  #  | Step                                                      | Status |
| :-: | --------------------------------------------------------- | :----: |
|  1  | Backend exception monitoring                              |   ⚠️   |
|  2  | Mobile and frontend exception monitoring                  |   ❌   |
|  3  | Swappable storage                                         |   ❌   |
|  4  | SLO and flexible alerting rules                           |   ❌   |
|  N  | Fancy features (to be expanded upon later)                |   ❌   |
