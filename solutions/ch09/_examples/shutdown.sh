#!/usr/bin/env bash
# Shows what happens to a request that is in flight when banditd is told to
# stop, first with a generous grace period and then with a short one.
# Usage: BIN=/path/to/banditd _examples/shutdown.sh
set -u
BIN=${BIN:-banditd}
URL=http://localhost:8089

start() {  # start <env...>: run banditd in the background, wait until it is up
  env "$@" BANDIT_ADDR=localhost:8089 BANDIT_LOG_FORMAT=text "$BIN" 2> "$LOG" &
  PID=$!
  until curl -sf "$URL/healthz" > /dev/null; do sleep 0.05; done
}
code() { curl -s -o /dev/null -w '%{http_code}' "$@"; }

LOG=$(mktemp)
echo '=== 1. grace period long enough: the slow request finishes ==='
start BANDIT_SELECT_DELAY=1s BANDIT_DRAIN_DELAY=500ms BANDIT_SHUTDOWN_GRACE=5s
( echo "in-flight /select finished with status $(code -X POST $URL/select)" ) &
CLIENT=$!
sleep 0.3
echo "health before the signal:      $(code $URL/healthz)"
kill -TERM $PID
sleep 0.2
echo "health during the drain delay: $(code $URL/healthz)"
wait $CLIENT
wait $PID; echo "banditd exit status: $?"
cut -c1-150 "$LOG" | sed -E 's/^time=[^ ]+ //'

echo
echo '=== 2. grace period too short: the slow request is cut off ==='
start BANDIT_SELECT_DELAY=3s BANDIT_SHUTDOWN_GRACE=300ms
( curl -s -o /dev/null -X POST $URL/select; echo "in-flight /select: curl exit code $? (52 = empty reply)" ) &
CLIENT=$!
sleep 0.3
kill -TERM $PID
wait $CLIENT
wait $PID; echo "banditd exit status: $?"
cut -c1-150 "$LOG" | sed -E 's/^time=[^ ]+ //'
