# TODO: unverified or deferred

## Environment (checked at project start)

| Capability | Status |
| --- | --- |
| Go 1.27.1 | available |
| Module proxy (`google.golang.org/grpc` fetched) | works, so gRPC chapters can be compile-verified |
| `protoc` | **not installed**; chapter 13 will need `protoc` (or `buf`) to be installed or fetched via the Go module proxy |
| Docker CLI and Compose plugin | installed, but **the Docker daemon is not running**, so images cannot be built or containers run here |
| PostgreSQL 16 server binaries (`/usr/lib/postgresql/16/bin`) | present and used: chapter 10's store, integration tests and demos ran against a real server started by `tools/pg.sh` |
| `redis-server` | present, but Redis is only described (a labelled design note in chapter 10), not implemented or tested |
| Python 3.11 | available (chapter 14 client) |

## Not yet verified

- Chapter 12: the three-replica demo runs on one machine (processes on different ports, one local PostgreSQL), not behind a real load balancer; its regret and request counts vary from run to run. The `sweep` simulation assumes instant rewards and strict round-robin routing. The `Sync` race (a reward applied to the policy between the store read and the restore is counted twice for one interval) is documented, and there is no test that provokes it.

- Chapter 11: no Prometheus server or Grafana was run. The `/metrics` output on the page is real, the PromQL queries shown are not executed. `go tool pprof`'s browser UI (flame graphs, needs Graphviz for graph views) was not used; the text views were. OpenTelemetry tracing is only mentioned.

- Chapter 10: the `docker run ... postgres:16` command shown as an alternative to `tools/pg.sh` was not run (no Docker daemon). The Redis design note has no code behind it.

- Chapter 14: Dockerfiles, Compose files, and health checks cannot be built or run without a Docker daemon. They will be written and reviewed by hand, and the chapter and its commit message will say so.
- Multi-container load balancing (chapter 14) has the same limitation; a local process-level substitute will be used and labelled.

## Deferred

- Chapters 13, 14 and the capstone (see `site/src/chapters.json` for status).
- Timing figures in transcripts (`go test`, benchmarks) come from the machine that ran `tools/transcripts.sh` and will differ on yours. Seeded simulation output is exact.
