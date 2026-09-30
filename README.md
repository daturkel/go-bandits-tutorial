# Learn Go by building a multi-armed bandit service

A multi-page tutorial and a runnable codebase for developers who know Python and machine learning and want to learn Go for backend work. One project runs through the whole course: it starts as a command-line simulation of a multi-armed bandit and grows into replicated gRPC microservices.

## Reading the course

Open `site/index.html` in a browser. The pages are committed, so no build step or server is needed:

```sh
xdg-open site/index.html        # Linux
open site/index.html            # macOS
```

Each chapter page explains what it builds and why, introduces language features at the moment the project needs them, and ends with a checkpoint (commands plus the output you should see) and exercises with collapsible reference answers.

## Following along

The intended path is to build the project yourself, one chapter at a time, in a directory of your own. `work/banditlab` inside your clone is convenient (git ignores `work/`), and the commands below assume it; anywhere works.

1. Chapter 1 starts from an empty directory. Each later chapter starts from your project as it stood at the end of the previous one.
2. Type or copy each listing from the page. Highlighted lines are new in that chapter; excerpts omit `package` and imports (gopls adds imports on save).
3. At the end of a chapter, check your work against the reference tests:

   ```sh
   tools/check.sh ch03 work/banditlab
   ```

   The script copies your project aside, swaps in that chapter's reference tests, and runs `gofmt`, `go vet`, `go build` and `go test -race`. It never modifies your files. A compile error usually means a name differs from the page.
4. If your project has drifted, or you want to skip ahead, start from the reference version of any chapter:

   ```sh
   tools/start.sh ch05 work/banditlab   # copies solutions/ch05 into a new directory
   ```

The builder fails if a chapter adds source lines that its page neither shows nor tells you to copy, so the pages are enough to reconstruct each chapter.

## Running a reference solution directly

You need **Go 1.24 or newer** (`go version`). `go.mod` pins the toolchain, so any Go 1.21+ installation downloads the right one on demand.

`solutions/chNN` is a complete, self-contained snapshot of the project at the end of chapter NN, not a diff:

```sh
cd solutions/ch04
go run . run                                   # one seed of each policy
go run . compare -scenario needle -svg out.svg # many seeds, with a chart
go test -race ./...
```

Compare two chapters with `diff -ru solutions/ch03 solutions/ch04`.

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
| `tools/` | `check.sh` and `start.sh` for readers; the site builder (`build_site`), transcript generator, and exercise checker for maintainers. |
| `TODO.md` | Anything that could not be verified or was deferred. |

## Maintaining the site

Pages are built from real source, never retyped. A page source in `site/src/` contains markers such as

```html
<!-- include: ch02/bandit/policy.go#Policy title="policy.go" diff -->
<!-- transcript: ch02/run -->
<!-- svg: ch04/regret-needle.svg caption="..." -->
<!-- copy: ch04/harness/svg.go why="..." -->
```

`include` embeds a declaration (`#Name`, `#Type.Method`, a comma-separated list) or a `region: NAME` block from a file under `solutions/`, escaped and syntax-highlighted; `diff` marks the lines that differ from the same file in the previous chapter. `transcript` embeds real command output, `svg` inlines a generated chart, and `copy` tells the reader to copy a file instead of typing it. Every non-test source line a chapter adds must be covered by an `include` or a `copy`, or the build fails.

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
