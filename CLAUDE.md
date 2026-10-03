# Project: Learn Go by building a multi-armed bandit service

A multi-page HTML tutorial plus a runnable companion codebase. The reader is an experienced Python/ML developer who wants to learn Go for backend work in an ML-engineering context. They have solid general programming experience but no Go. They learn the language and the backend/distributed-systems patterns together by building one project that grows from a CLI simulation into replicated microservices.

Everything lives in this repo: the site, the per-chapter solution code, and the tooling. See `README.md` for layout and commands.

## Ground rules

1. **Nothing ships unverified.** Every code sample compiles and passes its tests before the chapter is committed.
2. **Pages are built from real source.** Code in HTML comes from compiled files (`chNN/solution`, `chNN/files`, `solutions/`) through include markers; never retype it.
3. **Show real output.** Terminal output comes from actually running the code (seeded, reproducible), via `tools/transcripts.sh`. Charts are generated from harness output.
4. **Be honest about what could not be verified** (Docker, Postgres, ...): say so in the chapter's commit message and in `TODO.md`. Never fabricate output.
5. **Write for the reader described above.** Pages never refer to how the project was produced. Do not open sections with "since you know Python". Short "Go vs. Python" callouts are welcome where the contrast teaches something, not on every page.
6. Use current stable Go idioms: `math/rand/v2`, `log/slog`, generics, `net/http` ServeMux method+path patterns, range-over-int. Pin the toolchain in `go.mod`. Prefer the standard library until a dependency is clearly worth it (gRPC, protobuf, Prometheus client, a DB driver).

7. **Tasks only use what the reader has been shown.** Before a task, the page must have introduced every language construct its reference answer needs (not only the library calls: `if`, `import`, `%` and function literals count) and show each standard library function it calls, with its signature and a link to its pkg.go.dev entry, the first time it comes up. Explain what non-obvious parameters mean (for example `ParseFloat`'s bit size). Extras may be skipped, so a later chapter cannot rely on something only an extra introduced. `tools/build_site` enforces the library half of this (see `stdlib.go`); the language half is a manual check against the answer, so do it for every task you write.

## How the course works (the format)

The reader builds **one project of their own** through the whole course, in `work/banditlab` inside their clone (git-ignored), as their own git repository. Each chapter adds to the code they wrote in the previous one. Per chapter, the repo provides:

- `chNN/files/`: copied into the reader's project at the start of the chapter (`cp -R ../../chNN/files/. .`). It contains every test file that is new or changed in this chapter, new files with `TASK n` stubs, and complete "given" files that teach little (a `main.go`, generated code, an SVG renderer). It must never contain a file the reader wrote or edited (a file from an earlier chapter's solution that the reader owns), nor a test file the reader wrote in an earlier task.
- `chNN/solution/`: the complete project at the end of the chapter, cumulative (including `_examples/` and all `_extras/chMM/` so far). It is the catch-up point: `cp -R ../../chNN/solution/. .` plus `git diff`.

Changes to code the reader already owns are **tasks described in prose** ("change `Run` to return an error; the compiler lists every caller"), with the solution's diff as the answer. When a chapter's tests cannot compile until such changes are made, the page says so and orders the tasks so the compile-unblocking ones come first. Tests check only the public contract and the names the page defines, so any reasonable implementation passes. Extras live in `_extras/chNN` (their own package, skipped by `./...`, run with `go test ./_extras/chNN`); later chapters must not rely on anything only an extra introduced. Scaffolding fades: stubs with signatures in Part I, specs and signatures in Part II, contracts and HTTP/gRPC tests in Parts III–IV, with infrastructure (proto, Docker, SQL schema) given and explained.

**Ownership.** A file the reader has filled in or edited is theirs for the rest of the course: no later `files/` may contain it, and later changes to it are tasks. Given files that a later chapter may replace start with the header "This file comes with the course, and later chapters may replace it…"; no task may ask the reader to edit one (if a chapter wants the reader to change a given file, it drops the header from then on and the file becomes theirs). `files/` only replaces existing source files that carry the header. `scenarios.go` is given in chapter 1 without the header because the reader takes it over in chapter 3.

`tools/check_chapters.sh` plays a reader through each converted chapter (previous solution + files: tests must fail; solution: gofmt, vet, `test -race` pass; given files and tests match the solution), enforces the ownership rules, and lists the files the reader edits by hand, which the page must cover with tasks.

Chapters 5 to 14 and the capstone are still in the earlier format (listings to copy, `solutions/chNN`, separate `exercises/chNN`, `tools/check.sh`) and are to be converted.

## Repo layout

```
README.md  CLAUDE.md  TODO.md
site/        index.html, chNN.html (built, committed), style.css, site.js, src/ (page sources), generated/ (transcripts, charts)
primer/      runnable examples, ex1..ex5 tasks (exN/reference holds the answers)
chNN/        files/ and solution/ per converted chapter
solutions/   chNN/ reference projects for chapters not yet converted
exercises/   chNN/ separate exercises for chapters not yet converted
tools/       build_site/ (Go), transcripts.sh, check_chapters.sh, check_exercises.sh, check.sh, start.sh, pg.sh
```

- Every solution is a full snapshot with its own `go.mod`: `cd ch04/solution && go test -race ./...` works alone.
- Transcripts that show the reader's own project are made with `tw` in `tools/transcripts.sh`, which builds the reader's project in `work/banditlab` so that the commands shown (`cp -R ../../chNN/files/. .`) are the real ones.
- The site builder enforces that every non-test line a chapter adds is shown via `include` or covered by `copy`, and that each standard library symbol an exercise uses for the first time is linked to pkg.go.dev in that exercise.

## Chapter page format

What you will build and why, opening with real output of the finished program; the Go in this chapter; bringing in the chapter's files (the copy command, what is new and changed, and what the first test run shows); topic sections, each teaching a concept with a small example before the task that needs it; tasks (why, spec, docs links and signatures for new library calls, tiered hints, answer); a checkpoint (commands plus expected output) and a commit; what you learned; things to try; extra tasks. No "Step n" headings. Introduce each language feature before a task needs it, and explain the why behind Go's design choices where it helps. Site design: code-first, phone-friendly, light and dark modes, visible keyboard focus, copy buttons, filename headers, highlighted added lines, inline SVG charts, look drawn from the subject (bandit arms, regret curves).

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

- `tools/check_chapters.sh chNN` passes: the solution is gofmt-clean and passes vet and `go test -race`, extras included; a reader who copies `files/` onto the previous solution sees failing tests; given files and tests match the solution.
- Server code was started and exercised with real requests; the page shows that output.
- The page builds, includes only real source, and all internal links resolve.
- Every file the check lists as edited by the reader is covered by a task on the page, and the page's claims about what fails when (for example "after task 3 the tests compile") were checked by playing the reader.
- Committed.

## Working style

Build in batches (1-4, 5-7, ...) and pause for feedback after each with a short summary: what was built, what could not be verified, decisions worth reviewing. If a chapter is too big, split it. If the plan turns out wrong, fix the plan and note the change in the README.
