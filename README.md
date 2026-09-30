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

Start with the **primer** (`site/primer.html`): a short tour of the Go you need, with runnable examples in `primer/` and small tasks (`ex1` to `ex5`) next to them. Then work through the chapters in order.

**The new format (primer, chapters 1 and 2).** Each chapter has a folder with two projects side by side: `chNN/start`, the previous chapter's finished project with the interesting parts removed, and `chNN/solution`, the finished result. The page explains the tasks, marked `TASK n` in the code, and the tests fail until you write them. Tasks have a hint and an answer behind a click. The start folder also contains code you do not write; the page shows and explains it.

```sh
cd ch01/start && go test ./...      # see what is missing; edit the files in place
```

Your own code is not carried from one chapter to the next: every chapter begins from a start folder built on the reference solution to the last one. Work directly in it; `git restore ch01/start` starts it over. `tools/check_starters.sh` checks that each start folder's tests fail, that its solution passes them, and that untouched files match.

**The older format (chapters 3 to 14 and the capstone)** is still the original: build one project yourself, chapter by chapter. Type or copy each listing (highlighted lines are new; excerpts omit `package` and imports), then check your work against the chapter's reference tests:

```sh
tools/check.sh ch03 work/banditlab
```

The script copies your project aside, swaps in that chapter's reference tests, and runs `gofmt`, `go vet`, `go build` and `go test -race`. It never modifies your files. To skip ahead or recover, `tools/start.sh ch05 work/banditlab` copies the finished reference project of a chapter into a new directory.

The builder fails if a chapter adds source lines that its page neither shows nor tells you to copy, so the pages are enough to reconstruct each chapter.

## Running a reference solution directly

You need **Go 1.27 or newer** (`go version`). `go.mod` pins the toolchain, so any Go 1.21+ installation downloads the right one on demand.

Chapter 13 also needs `buf`, `protoc-gen-go` and `protoc-gen-go-grpc` to regenerate the protobuf code (the chapter shows the `go install` commands); the generated files are committed, so nothing else needs them.

A chapter's solution (`chNN/solution` for chapters 1 and 2, `solutions/chNN` for the rest until they are converted) is a complete, self-contained snapshot of the project at the end of chapter NN, not a diff:

```sh
cd solutions/ch04
go run . run                                   # one seed of each policy
go run . compare -scenario needle -svg out.svg # many seeds, with a chart
go test -race ./...
```

Compare two chapters with `diff -ru solutions/ch03 solutions/ch04`.

## Doing the exercises

In the new format the exercises are tasks inside the chapter's project. For the rest, `exercises/chNN/exM` holds a starter file with a `// TODO` and a test. The primer keeps its five in `primer/ex1` to `ex5`. From `exercises/chNN`:

```sh
go test ./ex1        # fails until you implement it
```

Reference answers are on the chapter page and in `exM/reference/`.

## Repository layout

| Path | Contents |
| --- | --- |
| `site/` | The course website. `index.html` and `chNN.html` are built output; `src/` holds page sources with include markers. |
| `primer/` | The Go primer: runnable examples and five small tasks. |
| `ch01/`, `ch02/` | Chapters in the new format: `start/` to work in, `solution/` with the finished project. |
| `solutions/` | Per-chapter snapshots for the chapters not yet converted (3 to 14 and the capstone). Every page listing is pulled from here or from a `solution/` folder. |
| `exercises/` | Separate exercises for the chapters not yet converted. |
| `tools/` | `check.sh` and `start.sh` for readers; `pg.sh` starts a throwaway local PostgreSQL for chapter 10's integration tests; the site builder (`build_site`), transcript generator, and exercise checker for maintainers. |
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

## Publishing the site

`.github/workflows/pages.yml` rebuilds the site, fails if the committed pages are out of date, and publishes the repository layout (pages plus `solutions/` and `exercises/`, which the pages link to) to GitHub Pages. Enable it once under Settings > Pages > Source: "GitHub Actions". The site is then at `https://<user>.github.io/<repo>/`.

## Chapter plan

Part I, language core (CLI simulation): modules and packages; interfaces and methods; errors, tests, benchmarks; generics and the comparison harness.
Part II, concurrency: goroutines and channels; shared state; context, select and worker pools.
Part III, serving: `net/http`; project structure and operations; persistence and delayed feedback; observability and performance.
Part IV, distributed: why naive replicas break the bandit; splitting into gRPC services; running the system with Docker and a Python client.
Capstone: an end-to-end load test comparing policies across replicas with staleness.

See `site/index.html` for which chapters are written. All fifteen pages (chapters 1 to 14 and the capstone) are.

Changes from the original plan, in case you compare against an older copy:

- The toolchain is Go 1.27.1 throughout (the first chapters were written against 1.24 and moved when it became clear that newer library releases needed it).
- The bandit policies changed once after chapter 8: chapter 12 makes UCB1 count selections that are still waiting for a reward, and keeps the old behaviour as `ucb1:naive`.
- Protobuf code is generated with `buf`, which needs no `protoc`.
- Chapter 14 could not build or run containers where it was written; `TODO.md` lists exactly what was and was not checked.
