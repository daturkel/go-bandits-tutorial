# TODO: unverified or deferred

## Environment (checked at project start)

| Capability | Status |
| --- | --- |
| Go 1.24.7 | available |
| Module proxy (`google.golang.org/grpc` fetched) | works, so gRPC chapters can be compile-verified |
| `protoc` | **not installed**; chapter 13 will need `protoc` (or `buf`) to be installed or fetched via the Go module proxy |
| Docker CLI and Compose plugin | installed, but **the Docker daemon is not running**, so images cannot be built or containers run here |
| PostgreSQL 16 server binaries (`/usr/lib/postgresql/16/bin`) | present and used: chapter 10's store, integration tests and demos ran against a real server started by `tools/pg.sh` |
| `redis-server` | present, but Redis is only described (a labelled design note in chapter 10), not implemented or tested |
| Python 3.11 | available (chapter 14 client) |

## Not yet verified

- Chapter 10: the `docker run ... postgres:16` command shown as an alternative to `tools/pg.sh` was not run (no Docker daemon). The Redis design note has no code behind it.

- Chapter 14: Dockerfiles, Compose files, and health checks cannot be built or run without a Docker daemon. They will be written and reviewed by hand, and the chapter and its commit message will say so.
- Multi-container load balancing (chapter 14) has the same limitation; a local process-level substitute will be used and labelled.

## Deferred

- Chapters 5 to 14 and the capstone (see `site/src/chapters.json` for status).
- Timing figures in transcripts (`go test`, benchmarks) come from the machine that ran `tools/transcripts.sh` and will differ on yours. Seeded simulation output is exact.
