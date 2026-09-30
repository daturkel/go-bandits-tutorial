#!/usr/bin/env bash
# Starts banditd, makes a few requests, and shows what /metrics reports.
# Usage: BIN=banditd _examples/metrics.sh
set -u
BIN=${BIN:-banditd}
URL=http://localhost:8082

"$BIN" -addr localhost:8082 -policy ucb1 -arms 3 -log-level warn 2> /dev/null &
PID=$!
trap 'kill -INT $PID 2>/dev/null' EXIT
until curl -sf "$URL/healthz" > /dev/null; do sleep 0.05; done

# Two selections that are rewarded, one that is not, and two mistakes.
for r in 1 0; do
  ID=$(curl -s -X POST "$URL/select" | sed -E 's/.*"request_id":"([^"]+)".*/\1/')
  curl -s -o /dev/null -X POST -d "{\"request_id\":\"$ID\",\"reward\":$r}" "$URL/reward"
done
curl -s -o /dev/null -X POST "$URL/select"
curl -s -o /dev/null -X POST -d '{"request_id":"nope","reward":1}' "$URL/reward"
curl -s -o /dev/null "$URL/nowhere"

echo '$ curl -s localhost:8082/metrics | grep ...'
curl -s "$URL/metrics" | grep -E '^banditd_(http_requests_total|selections_total|reward_total|arm_pulls|arm_mean|http_request_duration_seconds_count)' | sort
echo
echo '# a histogram is a family of cumulative counters, one per bucket:'
curl -s "$URL/metrics" | grep -E '^banditd_http_request_duration_seconds_bucket\{route="POST /select"' | sed -n '1,3p;14,17p'
echo
echo '# the runtime reports itself too:'
curl -s "$URL/metrics" | grep -E '^(go_goroutines|go_memstats_alloc_bytes|process_open_fds) '
