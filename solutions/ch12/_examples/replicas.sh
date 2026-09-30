#!/usr/bin/env bash
# Three banditd replicas share one PostgreSQL database. A load generator
# spreads simulated users over them. The run is repeated with syncing off and
# on, and each time we print what every replica believes afterwards next to
# what the database holds.
# Usage: DATABASE_URL=postgres://.../postgres BIN=banditd LOAD=banditload _examples/replicas.sh [off|on]
# With no argument both runs happen, one after the other.
set -u
BIN=${BIN:-banditd}
LOAD=${LOAD:-banditload}
: "${DATABASE_URL:?set DATABASE_URL to a PostgreSQL server (a database named replicas is created)}"
ADMIN=$DATABASE_URL
DB="${ADMIN%/*}/replicas?sslmode=disable"
PORTS="8091 8092 8093"
PIDS=""

pulls() {  # pulls <port>: pull counts per arm as that replica sees them
  curl -s "localhost:$1/stats" | grep -o '"pulls":[0-9]*' | cut -d: -f2 | paste -sd' ' -
}

run() {  # run <label> <sync interval>
  psql "$ADMIN" -qc 'DROP DATABASE IF EXISTS replicas' -c 'CREATE DATABASE replicas' 2> /dev/null
  PIDS=""
  n=0
  for p in $PORTS; do
    n=$((n + 1))
    BANDIT_ADDR=localhost:$p BANDIT_DATABASE_URL="$DB" BANDIT_POLICY=ucb1 BANDIT_SEED=$n \
      BANDIT_REPLICA_ID=replica-$n BANDIT_SYNC_INTERVAL=$2 "$BIN" 2>> "${REPLICA_LOG:-/dev/null}" &
    PIDS="$PIDS $!"
  done
  for p in $PORTS; do until curl -sf "localhost:$p/healthz" > /dev/null; do sleep 0.05; done; done

  echo "== $1 =="
  "$LOAD" -url http://localhost:8091,http://localhost:8092,http://localhost:8093 \
    -c 12 -d 4s -scenario easy -feedback-delay 5ms | sed -n '/^arm choices/,$p'
  sleep 1   # let one more sync interval pass
  for p in $PORTS; do printf 'replica on :%s sees pulls per arm: %s\n' "$p" "$(pulls "$p")"; done
  printf 'the database holds:             %s\n' "$(psql "$DB" -Atc 'SELECT string_agg(pulls::text, '"' '"' ORDER BY arm) FROM arm_totals')"
  kill -TERM $PIDS; wait $PIDS 2> /dev/null
  echo
}

case "${1:-both}" in
  off)  run "sync off (each replica learns only from its own rewards)" 0 ;;
  on)   run "sync every 200ms" 200ms ;;
  both) run "sync off (each replica learns only from its own rewards)" 0
        run "sync every 200ms" 200ms ;;
  *)    echo "usage: replicas.sh [off|on]" >&2; exit 2 ;;
esac
