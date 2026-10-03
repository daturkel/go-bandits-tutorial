# TODO: unverified or deferred

## Environment (checked at project start)

| Capability | Status |
| --- | --- |
| Go 1.27.1 | available |
| Module proxy (`google.golang.org/grpc` fetched) | works, so gRPC chapters can be compile-verified |
| `protoc` | **not installed**; chapter 13 uses `buf` (a Go program installed with `go install`), which compiles the schema itself, so `protoc` is not needed |
| Docker CLI and Compose plugin | installed, but **the Docker daemon is not running**, so images cannot be built or containers run here |
| PostgreSQL 16 server binaries (`/usr/lib/postgresql/16/bin`) | present and used: chapter 10's store, integration tests and demos ran against a real server started by `tools/pg.sh` |
| `redis-server` | present, but Redis is only described (a labelled design note in chapter 10), not implemented or tested |
| Python 3.11 | available (chapter 14 client) |

## Not yet verified

- Capstone: the benchmark's numbers depend on timing (goroutine scheduling, the machine's speed), so they are not exactly reproducible; the tests assert only an extreme contrast. The lag is injected on the client side of the aggregator stream and is a model of replication delay, not a measurement of a real network. Run on one 4-core machine with the services and the load generator sharing the cores.

- Chapter 13: services talk plaintext gRPC; TLS and mutual TLS are described, not run. The generated code was produced by `buf` v1.73.0 with `protoc-gen-go` and `protoc-gen-go-grpc` v1.6.2 built with a Go 1.26 toolchain (the plugins require it); a reader's newer versions will produce slightly different generated files. The aggregator is a single instance; its restart was exercised (policy instances kept answering and resubscribed), but a second aggregator, or a partition between it and the store, was not.

- Chapter 12: the three-replica demo runs on one machine (processes on different ports, one local PostgreSQL), not behind a real load balancer; its regret and request counts vary from run to run. The `sweep` simulation assumes instant rewards and strict round-robin routing. The `Sync` race (a reward applied to the policy between the store read and the restore is counted twice for one interval) is documented, and there is no test that provokes it.

- Chapter 11: no Prometheus server or Grafana was run. The `/metrics` output on the page is real, the PromQL queries shown are not executed. `go tool pprof`'s browser UI (flame graphs, needs Graphviz for graph views) was not used; the text views were. OpenTelemetry tracing is only mentioned.

- Chapter 10: the `docker run ... postgres:16` command shown as an alternative to `tools/pg.sh` was not run (no Docker daemon). The Redis design note has no code behind it.

- Chapter 14: verified in CI (`.github/workflows/ci.yml`, job `docker compose`): the images build from the Dockerfile, `docker compose up --wait` brings up PostgreSQL, the aggregator, the feedback service and three policy replicas until every health check passes, and `docker compose run --rm client` runs the Python client against them, with Compose DNS spreading its calls across the three replicas. The same job runs for the capstone. Not verified: running behind a real load balancer or on more than one host, and the local machine where the course was written still has no Docker daemon, so transcripts of `docker` commands other than the client run come from CI logs.

## Deferred

- **Convert chapters 8 to 14 and the capstone to the own-project format** (primer and chapters 1 to 7 are done; see "How the course works" in `CLAUDE.md`). Each needs a `chNN/files` and a cumulative `chNN/solution` (moved from `solutions/chNN`, with the stronger chapter 1 to 4 tests carried forward), tasks for every file the reader edits, a rewritten page, and its old `exercises/chNN` folded in as `_extras/chNN`. Scaffolding should fade: contracts and HTTP/gRPC tests in Parts III and IV, infrastructure given. Consider splitting chapter 13 (ten steps, three services). When done, remove `solutions/`, `exercises/`, `tools/check.sh`, `tools/start.sh` and `tools/check_exercises.sh`.

- **CI** (`.github/workflows/ci.yml`) runs on every push to `main` and every pull request: gofmt, vet and `go test -race` for every solution with a PostgreSQL 16 service (so the database tests run), `tools/check_chapters.sh`, `tools/check_exercises.sh`, the site staleness and link check, the Python client tests, `buf lint` plus a check that `buf generate` (with the pinned plugin versions) leaves the committed code unchanged, and a Docker Compose build and run. Still to decide: whether to add `tools/transcripts.sh` runs, which would catch transcripts that no longer match the code but depend on timing.

- Timing figures in transcripts (`go test`, benchmarks) come from the machine that ran `tools/transcripts.sh` and will differ on yours. Seeded simulation output is exact.
