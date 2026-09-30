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

ch01() {
  verify ch01
  tfresh ch01 mod-init 'mkdir banditlab && cd banditlab && go mod init banditlab && cat go.mod'
  t ch01 run 'go run .'
  t ch01 run-variants 'go run . -scenario needle -eps 0.05 -steps 100000' 'go run . -scenario nope'
  t ch01 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . -seed 7 -steps 1000'
}

ch02() {
  verify ch02
  t ch02 run 'go run .'
  t ch02 run-one 'go run . -scenario spread -policy ucb1 -steps 20000'
  t ch02 value-receiver 'go run ./_examples'
  t ch02 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . -seed 7 -steps 1000'
}

ch03() {
  verify ch03
  t ch03 errors 'go run . -scenario nope' 'go run . -policy epsgreedy:lots' 'go run . -policy epsgreedy:2' 'go run . -policy thompson'
  t ch03 test-v 'go test -v -run "TestNewEnvValidation|TestNewPolicy$" ./bandit'
  t ch03 vet 'go vet ./_examples/vetbug; echo "exit status: $?"'
  t ch03 typednil 'go run ./_examples/typednil'
  t ch03 bench 'go test -run "^$" -bench . -benchmem ./bandit'
  t ch03 cover 'go test -count=1 -cover ./...'
  t ch03 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . -seed 7 -steps 1000'
}

ch04() {
  verify ch04
  t ch04 compare 'go run . compare -scenario needle -steps 5000 -seeds 100'
  t ch04 compare-easy 'go run . compare -scenario easy -steps 5000 -seeds 100'
  t ch04 compare-close 'go run . compare -scenario close -steps 5000 -seeds 100'
  mkdir -p "$ROOT/site/generated/ch04"
  (cd "$ROOT/solutions/ch04" && go run . compare -scenario needle -steps 5000 -seeds 100 \
     -csv "$ROOT/site/generated/ch04/regret-needle.csv" -svg "$ROOT/site/generated/ch04/regret-needle.svg" >/dev/null)
  (cd "$ROOT/solutions/ch04" && go run . compare -scenario close -steps 5000 -seeds 100 \
     -svg "$ROOT/site/generated/ch04/regret-close.svg" >/dev/null)
  t ch04 csv-head 'go run . compare -scenario needle -steps 5000 -seeds 100 -csv /tmp/regret.csv > /dev/null' 'head -n 6 /tmp/regret.csv' 'wc -l /tmp/regret.csv'
  t ch04 svg-head 'go run . compare -scenario needle -steps 5000 -seeds 100 -svg /tmp/regret.svg > /dev/null' 'head -n 4 /tmp/regret.svg | cut -c1-140' 'grep -c "<polyline" /tmp/regret.svg'
  t ch04 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . run -seed 7 -steps 1000'
}

ch05() {
  verify ch05
  t ch05 chanbasics 'go run ./_examples/chanbasics'
  t ch05 deadlock 'go run ./_examples/deadlock 2>&1 | head -4'
  t ch05 bench 'go test -run "^$" -bench Compare ./harness'
  t ch05 timing 'go build -o /tmp/banditsim .' \
    'time /tmp/banditsim compare -sequential -scenario needle -steps 5000 -seeds 100 > /dev/null' \
    'time /tmp/banditsim compare -scenario needle -steps 5000 -seeds 100 > /dev/null' \
    'diff <(/tmp/banditsim compare -sequential -seeds 20) <(/tmp/banditsim compare -seeds 20) && echo "identical output"'
  t ch05 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . run -seed 7 -steps 1000'
}

ch06() {
  verify ch06
  t ch06 lostupdate 'go run ./_examples/lostupdate'
  t ch06 race 'go run -race ./_examples/race 2>&1 | head -30'
  t ch06 copylock 'go vet ./_examples/copylock'
  t ch06 locked-test 'go test -race -count=1 -v -run "TestLockedConcurrentUse|TestLockedMatches" ./bandit'
  t ch06 bench 'go test -run "^$" -bench "LockedSelectUpdate|SnapshotHeavy" -cpu 1,4 ./bandit'
  t ch06 checkpoint 'gofmt -l .' 'go vet ./...' 'go build ./...' 'go test -race -count=1 ./...' 'go run . run -seed 7 -steps 1000'
}

ch07() {
  verify ch07
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

if [ $# -eq 0 ]; then set -- ch01 ch02 ch03 ch04 ch05 ch06 ch07 ch08; fi
for c in "$@"; do "$c"; done
