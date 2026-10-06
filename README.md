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

## Roadmap and Status

The high-level plan for `warnly`, in order:

|  #  | Step                                                      | Status |
| :-: | --------------------------------------------------------- | :----: |
|  1  | Backend exception monitoring                              |   ⚠️   |
|  2  | Mobile and frontend exception monitoring                  |   ❌   |
|  3  | Swappable storage                                         |   ❌   |
|  4  | SLO and flexible alerting rules                           |   ❌   |
|  N  | Fancy features (to be expanded upon later)                |   ❌   |
