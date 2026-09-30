# Project: Learn Go by building a multi-armed bandit service

A multi-page HTML tutorial plus a runnable companion codebase. The reader is an experienced Python/ML developer who wants to learn Go for backend work in an ML-engineering context. They have solid general programming experience but no Go. They learn the language and the backend/distributed-systems patterns together by building one project that grows from a CLI simulation into replicated microservices.

Everything lives in this repo: the site, the per-chapter solution code, and the tooling. See `README.md` for layout and commands.

## Ground rules

1. **Nothing ships unverified.** Every code sample compiles and passes its tests before the chapter is committed.
2. **Pages are built from real source.** Code in HTML comes from compiled files in `solutions/` through include markers; never retype it.
3. **Show real output.** Terminal output comes from actually running the code (seeded, reproducible), via `tools/transcripts.sh`. Charts are generated from harness output.
4. **Be honest about what could not be verified** (Docker, Postgres, ...): say so in the chapter's commit message and in `TODO.md`. Never fabricate output.
5. **Write for the reader described above.** Pages never refer to how the project was produced. Do not open sections with "since you know Python". Short "Go vs. Python" callouts are welcome where the contrast teaches something, not on every page.
6. Use current stable Go idioms: `math/rand/v2`, `log/slog`, generics, `net/http` ServeMux method+path patterns, range-over-int. Pin the toolchain in `go.mod`. Prefer the standard library until a dependency is clearly worth it (gRPC, protobuf, Prometheus client, a DB driver).

## Repo layout

```
README.md  CLAUDE.md  TODO.md
site/        index.html, chNN.html (built, committed), style.css, site.js, src/ (page sources), generated/ (transcripts, charts)
tools/       build_site/ (Go), transcripts.sh, check_exercises.sh
solutions/   chNN/ complete snapshot per chapter, each with its own go.mod
exercises/   chNN/ starters + tests; exM/reference/ holds the answer
```

- `solutions/chNN` are full snapshots, not diffs: `cd solutions/ch05 && go test -race ./...` works alone.
- One commit per chapter: solution snapshot, page, exercise files. (Tags are not used.)
- A chapter page ends with the exact commands to check work and the output expected.
- The primary path is the reader building their own project one chapter at a time (`work/banditlab`, git-ignored). `tools/check.sh chNN [dir]` runs the chapter's reference tests against their code; `tools/start.sh chNN dir` gives skip-ahead and recovery. Each page states its starting point. The site builder enforces that every non-test line a chapter adds is shown via `include` or covered by `copy`. Reference tests may only use names the page defines.

## Chapter page format

What you will build and why (2-3 sentences); concepts introduced; build steps with code; a checkpoint (commands plus expected output); 2-4 exercises with collapsible reference answers; links to previous and next chapters. Introduce each language feature when the project needs it, and explain the why behind Go's design choices where it helps. Site design: code-first, phone-friendly, light and dark modes, visible keyboard focus, copy buttons, filename headers, highlighted added lines, inline SVG charts, look drawn from the subject (bandit arms, regret curves).

## Domain model (keep consistent across chapters)

- Bernoulli reward environments with per-arm probabilities.
- `Policy` interface: choose an arm, update with a reward. Epsilon-greedy, UCB1, Thompson sampling (Beta-Bernoulli).
- Seeded RNG (`rand.New(rand.NewPCG(s1, s2))`) so simulations and tests are deterministic.
- Metrics: cumulative reward and cumulative regret against the best arm.
- Server: a select request returns an arm plus a request ID; a reward request references that ID, possibly late or never (delayed feedback).
- Mergeable state: per-arm success/failure counts, which replicas can sum. Key to the distributed chapters.

## Chapter plan

Part I: 1 modules/packages/first simulation; 2 interfaces and methods; 3 errors, tests, benchmarks; 4 generics and the comparison harness.
Part II: 5 goroutines and channels; 6 shared state (mutex, atomic, race detector); 7 context, select, worker pools.
Part III: 8 net/http; 9 structure and operations (cmd/, internal/, config, slog, graceful shutdown); 10 persistence and delayed feedback; 11 observability and performance.
Part IV: 12 why naive replicas break the bandit (mergeable counts, staleness); 13 splitting into services (protobuf, gRPC); 14 running the system (Docker, Compose, Python client).
Capstone: end-to-end load test comparing policies across replicas with staleness, plus where to go next.

## Definition of done for a chapter

- `gofmt -l .` prints nothing; `go vet ./...`, `go build ./...`, `go test -race ./...` pass in `solutions/chNN`.
- Server code was started and exercised with real requests; the page shows that output.
- The page builds, includes only real source, and all internal links resolve.
- Exercise tests fail against the starter and pass against the reference (`tools/check_exercises.sh`).
- Committed.

## Working style

Build in batches (1-4, 5-7, ...) and pause for feedback after each with a short summary: what was built, what could not be verified, decisions worth reviewing. If a chapter is too big, split it. If the plan turns out wrong, fix the plan and note the change in the README.
