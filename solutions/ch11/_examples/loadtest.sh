#!/usr/bin/env bash
# Compares banditd's throughput and latency at two log levels. At "debug" every
# request is logged (what the service did before the fix in chapter 11); at
# "info" only requests worth reading are. Runs alternate, so slow drift in the
# machine affects both equally.
# Usage: BANDITD=/path/banditd BANDITLOAD=/path/banditload _examples/loadtest.sh
set -u
BANDITD=${BANDITD:-banditd}
BANDITLOAD=${BANDITLOAD:-banditload}
LOG=$(mktemp)

one() {  # one <log-level>: start the server, load it for 6 seconds, report
  : > "$LOG"
  "$BANDITD" -addr localhost:8081 -policy thompson -arms 10 -log-level "$1" 2> "$LOG" &
  PID=$!
  until curl -sf localhost:8081/healthz > /dev/null; do sleep 0.05; done
  OUT=$("$BANDITLOAD" -url http://localhost:8081 -c 32 -d 6s -scenario needle -reward-fraction 0.5)
  kill -INT $PID; wait $PID 2>/dev/null
  RPS=$(echo "$OUT" | sed -nE '1s/.*, ([0-9]+) requests.*/\1/p')
  read -r P50 P99 <<< "$(echo "$OUT" | awk '$1=="select"{print $4, $6}')"
  printf 'log-level=%-5s  %6s requests/s   select p50 %-7s p99 %-7s   log written: %s\n' \
    "$1" "$RPS" "$P50" "$P99" "$(du -h "$LOG" | cut -f1)"
}

for round in 1 2 3; do
  one debug
  one info
done
