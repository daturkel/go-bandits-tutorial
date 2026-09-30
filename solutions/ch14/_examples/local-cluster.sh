#!/usr/bin/env bash
# The topology in compose.yaml, run as plain processes: the same binaries, the
# same environment variables, three policy replicas, one PostgreSQL. It is
# what you can run without Docker; compose.yaml adds the images, the network
# and the restart policy around the same programs.
# Usage: DATABASE_URL=postgres://.../postgres BIN=/dir/with/binaries PYTHON=/path/to/python _examples/local-cluster.sh
set -u
BIN=${BIN:-.}
PYTHON=${PYTHON:-python3}
: "${DATABASE_URL:?set DATABASE_URL to a PostgreSQL server (a database named cluster is created)}"
ADMIN=$DATABASE_URL
DB="${ADMIN%/*}/cluster?sslmode=disable"
# gRPC sends connections through an HTTP proxy when the environment names one.
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY
LOG=$(mktemp)
declare -A PIDS
AGG=127.0.0.1:9292 FEEDBACK=127.0.0.1:9291
REPLICAS=127.0.0.1:9281,127.0.0.1:9282,127.0.0.1:9283

psql "$ADMIN" -qc 'DROP DATABASE IF EXISTS cluster' -c 'CREATE DATABASE cluster' 2> /dev/null

# The environment compose.yaml gives every service.
export BANDIT_ARMS=3 BANDIT_LOG_FORMAT=json BANDIT_DATABASE_URL="$DB"

start() {  # start <name> <host:port> [VAR=value ...]: run a service and wait until it reports SERVING
  local name=$1 addr=$2; shift 2
  env "$@" BANDIT_ADDR=$addr "$BIN/${name%%-*}" 2>> "$LOG" &
  PIDS[$name]=$!
  # The same command compose.yaml uses as its health check.
  until "$BIN/banditrpc" -addr "$addr" health > /dev/null 2>&1; do sleep 0.05; done
}

start aggregatord $AGG
start feedbackd $FEEDBACK BANDIT_AGGREGATOR=$AGG
for i in 1 2 3; do
  start policyd-$i 127.0.0.1:928$i BANDIT_AGGREGATOR=$AGG BANDIT_POLICY=ucb1 BANDIT_INSTANCE=policyd-$i
done
echo "== up: aggregator, feedback, and three policy replicas =="

echo; echo "== twelve selections, default client (pick_first): who answered? =="
"$BIN/banditrpc" -addr $REPLICAS select 12 | sed -E 's/.*instance=//' | sort | uniq -c

echo; echo "== twelve selections, -lb round_robin =="
"$BIN/banditrpc" -addr $REPLICAS -lb round_robin select 12 | sed -E 's/.*instance=//' | sort | uniq -c

echo; echo "== the Python client, 60 selections and rewards through the same replicas =="
(cd "$(dirname "$0")/../clients/python" &&
  BANDIT_POLICY_TARGET=ipv4:$REPLICAS BANDIT_FEEDBACK_TARGET=$FEEDBACK "$PYTHON" demo.py)

echo; echo "== replica policyd-2 is killed; round_robin routes around it =="
kill -KILL "${PIDS[policyd-2]}"; wait "${PIDS[policyd-2]}" 2> /dev/null
sleep 0.3
"$BIN/banditrpc" -addr $REPLICAS -lb round_robin select 12 | sed -E 's/.*instance=//' | sort | uniq -c
echo "health of the dead replica: $("$BIN/banditrpc" -addr 127.0.0.1:9282 -timeout 500ms health 2>&1 | head -1 | cut -c1-70)"
echo "exit status of that health check: $("$BIN/banditrpc" -addr 127.0.0.1:9282 -timeout 500ms health > /dev/null 2>&1; echo $?)"

echo; echo "== SIGTERM to the rest =="
for n in aggregatord feedbackd policyd-1 policyd-3; do kill -TERM "${PIDS[$n]}"; done
for n in aggregatord feedbackd policyd-1 policyd-3; do wait "${PIDS[$n]}"; echo "$n exited with status $?"; done
