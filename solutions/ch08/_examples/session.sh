#!/usr/bin/env bash
# Starts the service, walks through the API with curl, then stops it and
# prints what it logged. Usage: BIN=/path/to/banditsim _examples/session.sh
set -eu
BIN=${BIN:-banditsim}
URL=http://localhost:8088
LOG=$(mktemp)

$BIN serve -addr localhost:8088 -policy ucb1 -arms 3 2> "$LOG" &
PID=$!
trap 'kill $PID 2>/dev/null || true' EXIT
until curl -sf "$URL/healthz" > /dev/null; do sleep 0.05; done

# req prints the curl command it is about to run, then the full response.
req() { echo "\$ curl -i $1"; eval "curl -si $1" | tr -d '\r'; echo; }

req "$URL/healthz"

# First selection. The response headers show the request id as well.
RESP=$(curl -si -X POST "$URL/select" | tr -d '\r')
echo "\$ curl -i -X POST $URL/select"; echo "$RESP"; echo
ID=$(echo "$RESP" | sed -nE 's/.*"request_id":"([^"]+)".*/\1/p')

req "-X POST -d '{\"request_id\":\"$ID\",\"reward\":1}' $URL/reward"
req "-X POST -d '{\"request_id\":\"$ID\",\"reward\":1}' $URL/reward"   # again: conflict
req "-X POST -d '{\"request_id\":\"x\",\"reward\":7}' $URL/reward"      # out of range
req "-X POST -d '{\"request_id\":\"x\",\"reward\":1}' $URL/reward"      # never issued
req "$URL/select"                                                         # wrong method

# Three selections with no rewards in between: the policy has learned nothing new.
echo "\$ for i in 1 2 3; do curl -s -X POST $URL/select; done"
for i in 1 2 3; do curl -s -X POST "$URL/select"; done
echo

req "$URL/stats"

kill -INT $PID
wait $PID 2>/dev/null || true
echo "--- server log (stderr) ---"
cut -c1-170 "$LOG"
