#!/usr/bin/env bash
# Regenerates the terminal transcripts and charts that pages embed.
#
#   tools/transcripts.sh ch01 [ch02 ...]     (no arguments: every chapter)
#
# Each transcript is written to site/generated/<chapter>/<name>.txt as
# "$ command" lines followed by that command's real output. Seeded simulation
# output is reproducible; timings (go test, benchmarks) vary between machines.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# t <chapter> <name> <commands...>: run each command in solutions/<chapter>.
t() {
  local id=$1 name=$2; shift 2
  local out="$ROOT/site/generated/$id/$name.txt"
  mkdir -p "$(dirname "$out")"
  : > "$out"
  for cmd in "$@"; do
    echo "\$ $cmd" >> "$out"
    (cd "$ROOT/solutions/$id" && eval "$cmd") >> "$out" 2>&1
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

ch01() {
  tfresh ch01 mod-init 'mkdir banditlab && cd banditlab && go mod init banditlab && cat go.mod'
  t ch01 run 'go run .'
  t ch01 run-variants 'go run . -scenario needle -eps 0.05 -steps 100000' 'go run . -scenario nope'
  t ch01 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . -seed 7 -steps 1000'
}

ch02() {
  t ch02 run 'go run .'
  t ch02 run-one 'go run . -scenario spread -policy ucb1 -steps 20000'
  t ch02 value-receiver 'go run ./_examples'
  t ch02 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . -seed 7 -steps 1000'
}

ch03() {
  t ch03 errors 'go run . -scenario nope' 'go run . -policy epsgreedy:lots' 'go run . -policy epsgreedy:2' 'go run . -policy thompson'
  t ch03 test-v 'go test -v -run "TestNewEnvValidation|TestNewPolicy$" ./bandit'
  t ch03 vet 'go vet ./_examples; echo "exit status: $?"'
  t ch03 bench 'go test -run "^$" -bench . -benchmem ./bandit'
  t ch03 cover 'go test -cover ./...'
  t ch03 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . -seed 7 -steps 1000'
}

ch04() {
  t ch04 compare 'go run . compare -scenario needle -steps 5000 -seeds 100'
  t ch04 compare-easy 'go run . compare -scenario easy -steps 5000 -seeds 100'
  t ch04 compare-close 'go run . compare -scenario close -steps 5000 -seeds 100'
  mkdir -p "$ROOT/site/generated/ch04"
  (cd "$ROOT/solutions/ch04" && go run . compare -scenario needle -steps 5000 -seeds 100 \
     -csv "$ROOT/site/generated/ch04/regret-needle.csv" -svg "$ROOT/site/generated/ch04/regret-needle.svg" >/dev/null)
  (cd "$ROOT/solutions/ch04" && go run . compare -scenario close -steps 5000 -seeds 100 \
     -svg "$ROOT/site/generated/ch04/regret-close.svg" >/dev/null)
  t ch04 csv-head 'go run . compare -scenario needle -steps 5000 -seeds 100 -csv /tmp/regret.csv > /dev/null' 'head -n 6 /tmp/regret.csv' 'wc -l /tmp/regret.csv'
  t ch04 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . run -seed 7 -steps 1000'
}

if [ $# -eq 0 ]; then set -- ch01 ch02 ch03 ch04; fi
for c in "$@"; do "$c"; done
