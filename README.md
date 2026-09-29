# Learn Go by building a multi-armed bandit service

A multi-page tutorial and a runnable codebase for developers who know Python and machine learning and want to learn Go for backend work. One project runs through the whole course: it starts as a command-line simulation of a multi-armed bandit and grows into replicated gRPC microservices.

## Reading the course

Open `site/index.html` in a browser. The pages are committed, so no build step or server is needed:

```sh
xdg-open site/index.html        # Linux
open site/index.html            # macOS
```

Each chapter page explains what it builds and why, introduces language features at the moment the project needs them, and ends with a checkpoint (commands plus the output you should see) and exercises with collapsible reference answers.

## Running any chapter

You need **Go 1.24 or newer** (`go version`). `go.mod` pins the toolchain, so an older Go 1.21+ installation downloads the right one on demand.

`solutions/chNN` is a complete, self-contained snapshot of the project at the end of chapter NN, not a diff:

```sh
cd solutions/ch04
go run . run                                   # one seed of each policy
go run . compare -scenario needle -svg out.svg # many seeds, with a chart
go test -race ./...
```

Compare two chapters with `diff -ru solutions/ch03 solutions/ch04`, or check out a tag (`git checkout ch03`) to see the repository as it stood.

## Doing the exercises

`exercises/chNN/exM` holds a starter file with a `// TODO` and a test. From `exercises/chNN`:

```sh
go test ./ex1        # fails until you implement it
```

Reference answers are on the chapter page and in `exM/reference/`.

## Repository layout

| Path | Contents |
| --- | --- |
| `site/` | The course website. `index.html` and `chNN.html` are built output; `src/` holds page sources with include markers. |
| `solutions/` | Per-chapter snapshots. Every page listing is pulled from here. |
| `exercises/` | Exercise starters, tests, and reference answers. |
| `tools/` | The site builder (`build_site`), the transcript generator, and the exercise checker. |
| `TODO.md` | Anything that could not be verified or was deferred. |

## Maintaining the site

Pages are built from real source, never retyped. A page source in `site/src/` contains markers such as

```html
<!-- include: ch02/bandit/policy.go#Policy title="policy.go" diff -->
<!-- transcript: ch02/run -->
<!-- svg: ch04/regret-needle.svg caption="..." -->
```

`include` embeds a declaration (`#Name`, `#Type.Method`, a comma-separated list) or a `region: NAME` block from a file under `solutions/`, escaped and syntax-highlighted; `diff` marks the lines that differ from the same file in the previous chapter. `transcript` embeds real command output and `svg` inlines a generated chart.

```sh
tools/transcripts.sh ch04                   # re-run commands, refresh site/generated/
(cd tools/build_site && go run . ../..)     # rebuild every page and check all internal links
tools/check_exercises.sh                    # starters must fail, references must pass
```

## Chapter plan

Part I, language core (CLI simulation): modules and packages; interfaces and methods; errors, tests, benchmarks; generics and the comparison harness.
Part II, concurrency: goroutines and channels; shared state; context, select and worker pools.
Part III, serving: `net/http`; project structure and operations; persistence and delayed feedback; observability and performance.
Part IV, distributed: why naive replicas break the bandit; splitting into gRPC services; running the system with Docker and a Python client.
Capstone: an end-to-end load test comparing policies across replicas with staleness.

See `site/index.html` for which chapters are written.
