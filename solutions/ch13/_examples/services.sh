#!/usr/bin/env bash
# Starts the aggregator, the feedback service and two policy services against
# one PostgreSQL database, then uses them from the command line.
# Usage: DATABASE_URL=postgres://.../postgres BIN=/dir/with/binaries _examples/services.sh
set -u
BIN=${BIN:-.}
: "${DATABASE_URL:?set DATABASE_URL to a PostgreSQL server (a database named grpcdemo is created)}"
ADMIN=$DATABASE_URL
DB="${ADMIN%/*}/grpcdemo?sslmode=disable"
LOG=$(mktemp)
export BANDIT_DATABASE_URL="$DB" BANDIT_LOG_FORMAT=text BANDIT_ARMS=3 BANDIT_POLICY=ucb1

psql "$ADMIN" -qc 'DROP DATABASE IF EXISTS grpcdemo' -c 'CREATE DATABASE grpcdemo' 2> /dev/null

start() {  # start <name> <addr> [flags...]: run a daemon in the background
  local name=$1 addr=$2; shift 2
  "$BIN/${name%%-*}" -addr "$addr" "$@" 2>> "$LOG" &
  PIDS[$name]=$!
  until "$BIN/banditrpc" -addr "$addr" health > /dev/null 2>&1; do sleep 0.05; done
}
declare -A PIDS
rpc() { "$BIN/banditrpc" "$@" 2>&1; }

start aggregatord :9192 -reconcile-interval 500ms
start feedbackd :9191 -aggregator localhost:9192 -source feedback-1
start policyd-a :9190 -aggregator localhost:9192 -retry 200ms
start policyd-b :9193 -aggregator localhost:9192 -retry 200ms

echo '$ banditrpc -addr :9190 health'
rpc -addr localhost:9190 health

echo; echo '== three selections on policy A, none rewarded yet =='
for i in 1 2 3; do rpc -addr localhost:9190 select; done | tee /tmp/grpc-selects
IDS=($(sed -E 's/request_id=([^ ]+) .*/\1/' /tmp/grpc-selects))

echo; echo '== reward the first two through the feedback service =='
rpc -addr localhost:9191 reward "${IDS[0]}" 1
rpc -addr localhost:9191 reward "${IDS[1]}" 0
sleep 0.5
echo; echo '== policy B never saw those selections, but knows their outcome =='
rpc -addr localhost:9193 stats
echo; echo '== what the aggregator holds =='
rpc -addr localhost:9192 watch 1

echo; echo '== status codes =='
echo "reward again:      $(rpc -addr localhost:9191 reward "${IDS[0]}" 1)"
echo "unknown id:        $(rpc -addr localhost:9191 reward no-such-id 1)"
echo "reward out of range: $(rpc -addr localhost:9191 reward "${IDS[2]}" 5)"
echo "deadline of 1ns:   $(rpc -addr localhost:9190 -timeout 1ns select)"

echo; echo '== a batch of rewards on one stream =='
for i in 1 2 3 4; do rpc -addr localhost:9190 select; done | sed -E 's/request_id=([^ ]+) .*/\1 1/' > /tmp/grpc-batch
echo "no-such-id 1" >> /tmp/grpc-batch
rpc -addr localhost:9191 batch < /tmp/grpc-batch

echo; echo '== the aggregator goes away; policy A keeps answering =='
kill -TERM "${PIDS[aggregatord]}"; wait "${PIDS[aggregatord]}"
rpc -addr localhost:9190 select
sleep 0.5
start aggregatord :9192 -reconcile-interval 500ms
sleep 1
echo "after it comes back, policy B still knows every reward:"
rpc -addr localhost:9193 stats

echo; echo '== shutdown: SIGTERM to every daemon =='
for n in aggregatord feedbackd policyd-a policyd-b; do kill -TERM "${PIDS[$n]}"; done
for n in aggregatord feedbackd policyd-a policyd-b; do wait "${PIDS[$n]}"; echo "$n exited with status $?"; done

echo; echo '== selected log lines =='
grep -E 'lost the aggregator|msg=stopping|msg=rpc.*code=(NotFound|AlreadyExists|DeadlineExceeded|InvalidArgument)' "$LOG" |
  sed -E 's/^time=[^ ]+ //; s/ err=.*//; s/ (retry_in|grace|duration)=[^ ]+//g' | sort -u
