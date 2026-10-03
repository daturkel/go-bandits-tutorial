#!/usr/bin/env bash
# Regenerates the terminal transcripts and charts that pages embed.
#
#   tools/transcripts.sh ch01 [ch02 ...]     (no arguments: every chapter)
#
# Each transcript is written to site/generated/<chapter>/<name>.txt as
# "$ command" lines followed by that command's real output. Seeded simulation
# output is reproducible; timings (go test, benchmarks) vary between machines.
set -u
export TIMEFORMAT='real %2Rs, cpu %2Us'
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# A reader commits each chapter; transcripts that show git status start from that.
GITPRE='git init -q && git add -A && git -c user.name=reader -c user.email=reader@example.com commit -qm "previous chapter"'

# Reader states in the middle of a chapter, for transcripts that show what the
# compiler says partway through. Each is a sequence of edits a reader makes.
CH02_TASKS_1_2='cp -R ../../ch02/files/. . && cp ../../ch02/solution/bandit/sim.go bandit/ && sed -i "s|^\t// TASK 1: list the three methods every policy has: Select, Update and Name.\$|\tName() string\n\tSelect() int\n\tUpdate(arm int, reward float64)|" bandit/policy.go'
CH03_PART_A='cp -R ../../ch03/files/. . &&
  sed -i "s|^func NewEnv(probs \[\]float64, rng \*rand.Rand) \*Env {|func NewEnv(probs []float64, rng *rand.Rand) (*Env, error) {|; s|^\treturn &Env{probs: slices.Clone(probs), rng: rng}\$|\treturn \&Env{probs: slices.Clone(probs), rng: rng}, nil|" bandit/env.go &&
  sed -i "s|^func Scenario(name string) (\[\]float64, bool) {|func Scenario(name string) ([]float64, error) {|; s|^\tprobs, ok := scenarios\[name\]\$|\tprobs := scenarios[name]|; s|^\treturn slices.Clone(probs), ok\$|\treturn slices.Clone(probs), nil|" bandit/scenarios.go &&
  sed -i "s|) \*EpsilonGreedy {\$|) (*EpsilonGreedy, error) {|; s|Epsilon: epsilon, rng: rng}\$|Epsilon: epsilon, rng: rng}, nil|" bandit/epsgreedy.go &&
  sed -i "s|^func NewUCB1(nArms int) \*UCB1 {|func NewUCB1(nArms int) (*UCB1, error) {|; s|return &UCB1{armStats: newArmStats(nArms)}\$|return \&UCB1{armStats: newArmStats(nArms)}, nil|" bandit/ucb1.go &&
  sed -i "s|^\t\treturn NewEpsilonGreedy(nArms, epsilon, rng), nil|\t\tp, err := NewEpsilonGreedy(nArms, epsilon, rng)\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t\treturn p, nil|; s|^\t\treturn NewUCB1(nArms), nil|\t\tp, err := NewUCB1(nArms)\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t\treturn p, nil|" bandit/policy.go'

# After chapter 5's task 1: CompareSequential exists and Compare only calls it.
CH05_TASK1='cp -R ../../ch05/files/. . && awk "/^func Compare\\(spec Spec\\)/ { print; print \"\\treturn CompareSequential(spec)\"; print \"}\"; print \"\"; print \"func notYet(spec Spec) (*Report, error) {\"; next } { print }" ../../ch05/solution/harness/harness.go > harness/harness.go'
# soldir <chapter>: the chapter's reference solution, in either layout.
soldir() {
  if [ -d "$ROOT/$1/solution" ]; then echo "$ROOT/$1/solution"; else echo "$ROOT/solutions/$1"; fi
}

# t <chapter> <name> <commands...>: run each command in the chapter's solution.
t() {
  local id=$1 name=$2; shift 2
  local out="$ROOT/site/generated/$id/$name.txt"
  mkdir -p "$(dirname "$out")"
  : > "$out"
  for cmd in "$@"; do
    echo "\$ $cmd" >> "$out"
    (cd "$(soldir "$id")" && eval "$cmd") >> "$out" 2>&1
  done
  echo "wrote ${out#$ROOT/}"
}

# tw <chapter> <name> <commands...>: play a reader. The project is
# work/banditlab inside the repository, as the pages suggest, holding the
# previous chapter's solution (chapter 1: nothing). The commands run there in
# order, so a transcript can copy the chapter's files in (cp -R ../../chNN/files/. .)
# and then show what the reader sees.
tw() {
  local id=$1 name=$2; shift 2
  local out="$ROOT/site/generated/$id/$name.txt" dir="$ROOT/work/banditlab"
  local n=$((10#${id#ch}))
  rm -rf "$dir"; mkdir -p "$dir"
  if [ "$n" -gt 1 ]; then cp -R "$ROOT/$(printf 'ch%02d' $((n - 1)))/solution/." "$dir/"; fi
  # TW_PRE: commands run first and not shown (the set-up a transcript assumes).
  if [ -n "${TW_PRE:-}" ]; then (cd "$dir" && eval "$TW_PRE") > /dev/null 2>&1; fi
  mkdir -p "$(dirname "$out")"
  : > "$out"
  for cmd in "$@"; do
    echo "\$ $cmd" >> "$out"
    (cd "$dir" && eval "$cmd") >> "$out" 2>&1
  done
  rm -rf "$ROOT/work"
  echo "wrote ${out#$ROOT/}"
}

# tp <name> <commands...>: run each command in primer/.
tp() {
  local name=$1; shift
  local out="$ROOT/site/generated/primer/$name.txt"
  mkdir -p "$(dirname "$out")"
  : > "$out"
  for cmd in "$@"; do
    echo "\$ $cmd" >> "$out"
    (cd "$ROOT/primer" && eval "$cmd") >> "$out" 2>&1
  done
  echo "wrote ${out#$ROOT/}"
}

# tfresh <chapter> <name> <commands...>: run in an empty scratch directory.
tfresh() {
  local id=$1 name=$2; shift 2
  local out="$ROOT/site/generated/$id/$name.txt" dir
  dir=$(mktemp -d)
  mkdir -p "$(dirname "$out")"
  : > "$out"
  for cmd in "$@"; do
    echo "\$ $cmd" >> "$out"
    (cd "$dir" && eval "$cmd") >> "$out" 2>&1
  done
  rm -rf "$dir"
  echo "wrote ${out#$ROOT/}"
}

# verify <chapter>: what a reader sees when tools/check.sh passes on their project.
verify() {
  local id=$1 out="$ROOT/site/generated/$1/verify.txt"
  mkdir -p "$(dirname "$out")"
  rm -rf "$ROOT/work/banditlab"
  (cd "$ROOT" && tools/start.sh "$id" work/banditlab >/dev/null)
  { echo "\$ tools/check.sh $id work/banditlab"; (cd "$ROOT" && tools/check.sh "$id" work/banditlab 2>&1); } > "$out"
  rm -rf "$ROOT/work"
  echo "wrote ${out#$ROOT/}"
}

primer() {
  tp version 'go version'
  tp hello 'go run ./hello'
  tp values 'go run ./values'
  tp functions 'go run ./functions'
  tp control 'go run ./control'
  tp collections 'go run ./collections'
  tp types 'go run ./types'
  tp test 'go test -v ./stats 2>&1 | grep -v "^=== RUN"'
}
ch01() {
  tw ch01 setup 'git init -q' 'go mod init banditlab' 'go mod edit -go=1.27.0 -toolchain=go1.27.1' 'cat go.mod' 'cp -R ../../ch01/files/. .'
  TW_PRE='go mod init banditlab; go mod edit -go=1.27.0 -toolchain=go1.27.1; cp -R ../../ch01/files/. .' \
    tw ch01 starter-tests "go test ./... 2>&1 | grep -E '^(--- FAIL|FAIL|ok)'"
  t ch01 run 'go run .'
  t ch01 run-variants 'go run . -scenario needle -eps 0.05 -steps 100000' 'go run . -scenario nope'
  t ch01 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . -seed 7 -steps 1000'
}

ch02() {
  TW_PRE="$GITPRE" tw ch02 copy-in 'cp -R ../../ch02/files/. .' 'git status --short'
  TW_PRE='cp -R ../../ch02/files/. .' tw ch02 starter-tests 'go build ./...'
  TW_PRE="$CH02_TASKS_1_2" tw ch02 tests-compile 'go build ./...' 'go test ./... 2>&1 | head -12'
  t ch02 run 'go run .'
  t ch02 run-one 'go run . -scenario spread -policy ucb1 -steps 20000'
  t ch02 value-receiver 'go run ./_examples/valuereceiver'
  t ch02 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . -seed 7 -steps 1000'
}

ch03() {
  TW_PRE="$GITPRE" tw ch03 copy-in 'cp -R ../../ch03/files/. .' 'git status --short'
  TW_PRE='cp -R ../../ch03/files/. .' tw ch03 starter-tests 'go build ./...'
  TW_PRE="$CH03_PART_A" tw ch03 part-a 'go build ./...'
  t ch03 errors 'go run . -scenario nope' 'go run . -policy epsgreedy:lots' 'go run . -policy epsgreedy:2' 'go run . -policy thompson'
  t ch03 test-v 'go test -v -run "TestNewEnvValidation|TestNewPolicy$" ./bandit'
  t ch03 vet 'go vet ./_examples/vetbug; echo "exit status: $?"'
  t ch03 typednil 'go run ./_examples/typednil'
  t ch03 bench 'go test -run "^$" -bench . -benchmem ./bandit'
  t ch03 cover 'go test -count=1 -cover ./...'
  t ch03 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . -seed 7 -steps 1000'
}

ch04() {
  TW_PRE="$GITPRE" tw ch04 copy-in 'cp -R ../../ch04/files/. .' 'git status --short'
  TW_PRE='cp -R ../../ch04/files/. .' tw ch04 starter-tests 'go build ./...' 'go test ./... 2>&1 | grep -v "^\s" | head -16'
  t ch04 compare 'go run . compare -scenario needle -steps 5000 -seeds 100'
  t ch04 compare-easy 'go run . compare -scenario easy -steps 5000 -seeds 100'
  t ch04 compare-close 'go run . compare -scenario close -steps 5000 -seeds 100'
  mkdir -p "$ROOT/site/generated/ch04"
  (cd "$ROOT/ch04/solution" && go run . compare -scenario needle -steps 5000 -seeds 100 \
     -csv "$ROOT/site/generated/ch04/regret-needle.csv" -svg "$ROOT/site/generated/ch04/regret-needle.svg" >/dev/null)
  (cd "$ROOT/ch04/solution" && go run . compare -scenario close -steps 5000 -seeds 100 \
     -svg "$ROOT/site/generated/ch04/regret-close.svg" >/dev/null)
  t ch04 csv-head 'go run . compare -scenario needle -steps 5000 -seeds 100 -csv /tmp/regret.csv > /dev/null' 'head -n 6 /tmp/regret.csv' 'wc -l /tmp/regret.csv'
  t ch04 svg-head 'go run . compare -scenario needle -steps 5000 -seeds 100 -svg /tmp/regret.svg > /dev/null' 'head -n 4 /tmp/regret.svg | cut -c1-140' 'grep -c "<polyline" /tmp/regret.svg'
  t ch04 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . run -seed 7 -steps 1000'
}

ch05() {
  # The problem, in the reader's chapter 4 project: one comparison, one core.
  tw ch05 before 'go build -o /tmp/banditsim .' \
    'time /tmp/banditsim compare -scenario needle -steps 5000 -seeds 1000 > /dev/null'
  TW_PRE="$GITPRE" tw ch05 copy-in 'cp -R ../../ch05/files/. .' 'git status --short'
  TW_PRE='cp -R ../../ch05/files/. .' tw ch05 starter-tests 'go build ./...'
  TW_PRE="$CH05_TASK1" tw ch05 after-task1 'go test ./harness'
  t ch05 chanbasics 'go run ./_examples/chanbasics'
  t ch05 deadlock 'go run ./_examples/deadlock 2>&1 | head -4'
  t ch05 bench 'go test -run "^$" -bench Compare ./harness'
  t ch05 timing 'go build -o /tmp/banditsim .' \
    'time /tmp/banditsim compare -sequential -scenario needle -steps 5000 -seeds 1000 > /dev/null' \
    'time /tmp/banditsim compare -scenario needle -steps 5000 -seeds 1000 > /dev/null' \
    'diff <(/tmp/banditsim compare -sequential -seeds 20) <(/tmp/banditsim compare -seeds 20) && echo "identical output"'
  t ch05 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . run -seed 7 -steps 1000'
}

ch06() {
  TW_PRE="$GITPRE" tw ch06 copy-in 'cp -R ../../ch06/files/. .' 'git status --short'
  # The problem, in the reader's project right after copying the files in.
  TW_PRE='cp -R ../../ch06/files/. .' tw ch06 race 'go run -race ./_examples/race 2>&1 | head -30'
  TW_PRE='cp -R ../../ch06/files/. .' tw ch06 starter-tests 'go test ./bandit 2>&1 | grep -E "^(--- FAIL|FAIL|ok|panic: )"'
  t ch06 lostupdate 'go run ./_examples/lostupdate'
  t ch06 race-fixed 'go run -race ./_examples/race -locked'
  t ch06 copylock 'go vet ./_examples/copylock'
  t ch06 locked-test 'go test -race -count=1 -v -run "TestLockedConcurrentUse|TestLockedMatches" ./bandit'
  t ch06 bench 'go test -run "^$" -bench "LockedSelectUpdate|SnapshotHeavy" -cpu 1,4 ./bandit'
  t ch06 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . run -seed 7 -steps 1000'
}

ch07() {
  # The problem, in the reader's chapter 6 project: a typo that is known at the
  # first job still costs the whole comparison.
  tw ch07 before 'go build -o /tmp/banditsim .' \
    'time /tmp/banditsim compare -policy ucbl,ucb1,thompson -steps 2000 -seeds 10000'
  TW_PRE="$GITPRE" tw ch07 copy-in 'cp -R ../../ch07/files/. .' 'git status --short'
  TW_PRE='cp -R ../../ch07/files/. .' tw ch07 starter-tests 'go build ./...'
  t ch07 after 'go build -o /tmp/banditsim .' \
    'time /tmp/banditsim compare -policy ucbl,ucb1,thompson -steps 2000 -seeds 10000'
  t ch07 selectdemo 'go run ./_examples/selectdemo'
  t ch07 leak 'go run ./_examples/leak 2>&1 | head -9'
  t ch07 pool-test 'go test -race -count=1 -v ./pool'
  t ch07 cancel 'go build -o /tmp/banditsim .' \
    '/tmp/banditsim compare -seeds 100000 -steps 5000 -timeout 200ms; echo "exit status: $?"' \
    'timeout --preserve-status -s INT 0.5 /tmp/banditsim compare -seeds 100000 -steps 5000; echo "exit status: $?"'
  t ch07 progress 'go build -o /tmp/banditsim .' \
    '/tmp/banditsim compare -seeds 300 -steps 5000 -progress 2>&1 >/dev/null | tr "\r" "\n"'
  t ch07 workers 'go build -o /tmp/banditsim .' \
    'time /tmp/banditsim compare -workers 1 > /dev/null' \
    'time /tmp/banditsim compare -workers 2 > /dev/null' \
    'time /tmp/banditsim compare > /dev/null'
  t ch07 bench 'go test -run "^$" -bench Compare ./harness'
  t ch07 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . run -seed 7 -steps 1000'
}

ch08() {
  verify ch08
  t ch08 session 'go build -o /tmp/banditsim .' 'BIN=/tmp/banditsim bash _examples/session.sh'
  t ch08 rejections 'go test -race -count=1 -v -run "TestRewardRejections|TestRoutingErrors|TestChainOrder" ./server'
  t ch08 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . run -seed 7 -steps 1000'
}

ch09() {
  verify ch09
  # The restructuring commands, run on a copy of the previous chapter's project.
  local out="$ROOT/site/generated/ch09/restructure.txt" dir
  mkdir -p "$(dirname "$out")"
  dir=$(mktemp -d)
  "$ROOT/tools/start.sh" ch08 "$dir/banditlab" > /dev/null
  : > "$out"
  for cmd in \
    'mkdir -p internal cmd/banditsim cmd/banditd' \
    'mv bandit harness pool stat server internal/' \
    'mv main.go run.go compare.go main_test.go testdata cmd/banditsim/' \
    'rm serve.go' \
    "grep -rl 'banditlab/' --include='*.go' . | xargs sed -i -E 's|banditlab/(bandit\\|harness\\|pool\\|stat\\|server)\"|banditlab/internal/\\1\"|'" \
    'go list ./...'; do
    echo "\$ $cmd" >> "$out"
    (cd "$dir/banditlab" && eval "$cmd") >> "$out" 2>&1
  done
  rm -rf "$dir"
  echo "wrote ${out#$ROOT/}"

  t ch09 layout "find . -name '*.go' -not -path './_examples/*' -not -name '*_test.go' | sort"
  t ch09 help 'go run ./cmd/banditd -h 2>&1'
  t ch09 slogdemo 'go run ./_examples/slogdemo'
  t ch09 internal 'cd _examples/otherapp && go build ./... 2>&1'
  t ch09 serve-tests 'go test -race -count=1 -v -run "TestServe" ./internal/server'
  t ch09 shutdown 'go build -o /tmp/banditd ./cmd/banditd' 'BIN=/tmp/banditd bash _examples/shutdown.sh'
  t ch09 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run ./cmd/banditsim run -seed 7 -steps 1000'
}

ch10() {
  "$ROOT/tools/pg.sh" start > /dev/null
  # What a reader sees without a database: the integration tests skip.
  ( unset BANDIT_TEST_DATABASE_URL; verify ch10 )
  local pgurl="postgres://postgres@127.0.0.1:55432/postgres?sslmode=disable"

  t ch10 skipped 'go test -count=1 -v -run "Postgres" ./internal/store 2>&1 | grep -E "^(--- |ok)"'
  t ch10 integration "BANDIT_TEST_DATABASE_URL='$pgurl' go test -race -count=1 -v -run 'Conformance|Persists|Refuses|Restart' ./internal/store ./internal/server ./cmd/banditd 2>&1 | grep -E '^(--- |    --- |ok|FAIL)'"

  # Break the claim query on purpose and watch the conformance tests catch it.
  local out="$ROOT/site/generated/ch10/mutation.txt" dir
  dir=$(mktemp -d)
  cp -r "$ROOT/solutions/ch10/." "$dir/"
  : > "$out"
  for cmd in \
    "sed -i 's/ AND NOT rewarded//' internal/store/postgres.go" \
    "go test -count=1 -run 'PostgresConformance' ./internal/store 2>&1 | grep -E '^ +(--- FAIL|storetest)'"; do
    echo "\$ $cmd" >> "$out"
    (cd "$dir" && BANDIT_TEST_DATABASE_URL="$pgurl" eval "$cmd") >> "$out" 2>&1
  done
  rm -rf "$dir"
  echo "wrote ${out#$ROOT/}"

  t ch10 persistence 'go build -o /tmp/banditd ./cmd/banditd' \
    "psql '$pgurl' -qc 'DROP DATABASE IF EXISTS demo' -c 'CREATE DATABASE demo' 2>/dev/null" \
    "DATABASE_URL='postgres://postgres@127.0.0.1:55432/demo?sslmode=disable' BIN=/tmp/banditd bash _examples/persistence.sh"
  t ch10 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run ./cmd/banditsim run -seed 7 -steps 1000'
}

ch11() {
  verify ch11
  t ch11 build 'go build -o /tmp/banditd ./cmd/banditd' 'go build -o /tmp/banditload ./cmd/banditload' 'go run ./cmd/banditload -h 2>&1 | head -24'
  t ch11 metrics 'BIN=/tmp/banditd bash _examples/metrics.sh'
  t ch11 bench 'go test -run "^$" -bench SelectHandler -benchmem -count=3 ./internal/server 2>&1 | grep -E "^(cpu|Benchmark)"'
  t ch11 profile-debug 'BANDITD=/tmp/banditd BANDITLOAD=/tmp/banditload bash _examples/profile.sh debug'
  t ch11 profile-info 'BANDITD=/tmp/banditd BANDITLOAD=/tmp/banditload bash _examples/profile.sh info'
  t ch11 loadtest 'BANDITD=/tmp/banditd BANDITLOAD=/tmp/banditload bash _examples/loadtest.sh'
  t ch11 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run ./cmd/banditsim run -seed 7 -steps 1000'
}

ch12() {
  "$ROOT/tools/pg.sh" start > /dev/null
  ( unset BANDIT_TEST_DATABASE_URL; verify ch12 )
  local pgurl="postgres://postgres@127.0.0.1:55432/postgres?sslmode=disable"
  local gen="$ROOT/site/generated/ch12"
  mkdir -p "$gen"

  t ch12 herding 'go test -count=1 -v -run "SpreadsSelections|BonusShrinks|ReleasePending|ReducesRegretUnderDelay" ./internal/bandit 2>&1 | grep -E "^(--- |ok)"'
  t ch12 delay-easy 'go run ./cmd/banditsim compare -scenario easy -policy ucb1:naive,ucb1,thompson,epsgreedy:0.1 -steps 5000 -seeds 40 -delay 200'
  t ch12 delay-needle 'go run ./cmd/banditsim compare -scenario needle -policy ucb1:naive,ucb1,thompson,epsgreedy:0.1 -steps 5000 -seeds 40 -delay 1000'
  (cd "$ROOT/solutions/ch12" && go run ./cmd/banditsim compare -scenario needle -policy ucb1:naive,ucb1,thompson,epsgreedy:0.1 -steps 5000 -seeds 40 -delay 1000 \
     -csv "$gen/delay-needle.csv" -svg "$gen/delay-needle.svg" >/dev/null)
  t ch12 sweep 'go run ./cmd/banditsim sweep -scenario easy -replicas 4 -steps 5000 -seeds 40'
  (cd "$ROOT/solutions/ch12" && go run ./cmd/banditsim sweep -scenario easy -replicas 4 -steps 5000 -seeds 40 \
     -csv "$gen/sweep-easy.csv" -svg "$gen/sweep-easy.svg" >/dev/null)
  t ch12 integration "BANDIT_TEST_DATABASE_URL='$pgurl' go test -race -count=1 -v -run 'Conformance|ConcurrentFirstStart' ./internal/store 2>&1 | grep -E '^(--- |    --- |ok|FAIL)'"
  t ch12 replicas-off 'go build -o /tmp/banditd ./cmd/banditd' 'go build -o /tmp/banditload ./cmd/banditload' \
    "DATABASE_URL='$pgurl' BIN=/tmp/banditd LOAD=/tmp/banditload bash _examples/replicas.sh off"
  t ch12 replicas-on "DATABASE_URL='$pgurl' BIN=/tmp/banditd LOAD=/tmp/banditload bash _examples/replicas.sh on"
  t ch12 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run ./cmd/banditsim run -seed 7 -steps 1000'
}

ch13() {
  "$ROOT/tools/pg.sh" start > /dev/null
  export PATH="$(go env GOPATH)/bin:$PATH"
  ( unset BANDIT_TEST_DATABASE_URL; verify ch13 )
  local pgurl="postgres://postgres@127.0.0.1:55432/postgres?sslmode=disable"

  t ch13 generate 'buf lint && echo "buf lint: no problems"' 'buf generate' 'wc -l internal/gen/bandit/v1/*.go'
  t ch13 unit 'go test -race -count=1 ./internal/rpcx ./internal/services/... ./internal/daemon ./cmd/banditrpc 2>&1'
  t ch13 integration "BANDIT_TEST_DATABASE_URL='$pgurl' go test -race -count=1 -v -run TestServicesOverTCP ./internal/apps 2>&1 | grep -E '^(--- |ok|FAIL)'"
  t ch13 services 'go build -o /tmp/g13/ ./cmd/aggregatord ./cmd/feedbackd ./cmd/policyd ./cmd/banditrpc' \
    "DATABASE_URL='$pgurl' BIN=/tmp/g13 bash _examples/services.sh"
  t ch13 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run ./cmd/banditsim run -seed 7 -steps 1000'
}

ch14() {
  # site/generated/ch14/compose-client.txt is not made here: it is the output of
  # the "docker compose" job in .github/workflows/ci.yml, copied from its log,
  # because this machine has no Docker daemon.
  "$ROOT/tools/pg.sh" start > /dev/null
  export PATH="$(go env GOPATH)/bin:$PATH"
  ( unset BANDIT_TEST_DATABASE_URL; verify ch14 )
  local pgurl="postgres://postgres@127.0.0.1:55432/postgres?sslmode=disable"
  local venv=/tmp/banditlab-venv
  [ -x "$venv/bin/python" ] || { python3 -m venv "$venv" && "$venv/bin/pip" install -q -r "$ROOT/solutions/ch14/clients/python/requirements.txt"; } > /dev/null 2>&1

  t ch14 compose 'docker compose --profile demo config -q && echo "compose.yaml: valid"' 'docker compose --profile demo config --format json | python3 _examples/compose_summary.py'
  t ch14 python "cd clients/python && PATH=$venv/bin:\$PATH sh generate.sh && ls gen/bandit/v1" "cd clients/python && PATH=$venv/bin:\$PATH python -m unittest -v 2>&1 | grep -E '^(test_|Ran|OK|FAILED)' | sed -E 's/ \(test_client[^)]*\)//'"
  t ch14 cluster 'go build -o /tmp/g14/ ./cmd/aggregatord ./cmd/feedbackd ./cmd/policyd ./cmd/banditrpc' \
    "DATABASE_URL='$pgurl' BIN=/tmp/g14 PYTHON=$venv/bin/python bash _examples/local-cluster.sh"
  t ch14 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run ./cmd/banditsim run -seed 7 -steps 1000'
}

capstone() {
  verify capstone
  t capstone unit 'go test -race -count=1 ./internal/bench ./cmd/banditbench 2>&1'
  t capstone bench-easy 'go build -o /tmp/banditbench ./cmd/banditbench' '/tmp/banditbench'
  t capstone bench-needle '/tmp/banditbench -scenario needle -lags 0s,100ms -n 6000'
  t capstone sim 'go run ./cmd/banditsim sweep -scenario easy -replicas 3 -steps 4000 -seeds 40 -intervals 0,1000,100,10,1'
  t capstone checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run ./cmd/banditsim run -seed 7 -steps 1000'
}

if [ $# -eq 0 ]; then set -- primer ch01 ch02 ch03 ch04 ch05 ch06 ch07 ch08 ch09 ch10 ch11 ch12 ch13 ch14 capstone; fi
for c in "$@"; do "$c"; done
