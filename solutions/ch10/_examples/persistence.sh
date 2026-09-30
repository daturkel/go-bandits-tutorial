#!/usr/bin/env bash
# Runs banditd against PostgreSQL, stops it, runs it again, and shows that the
# learned state and the waiting selections survived; then shows selections
# expiring. Usage: DATABASE_URL=postgres://... BIN=banditd _examples/persistence.sh
set -u
BIN=${BIN:-banditd}
: "${DATABASE_URL:?set DATABASE_URL to an empty PostgreSQL database}"
URL=http://localhost:8090
LOG=$(mktemp)

start() {  # start <env...>
  env BANDIT_ADDR=localhost:8090 BANDIT_DATABASE_URL="$DATABASE_URL" BANDIT_POLICY=ucb1 "$@" "$BIN" 2>> "$LOG" &
  PID=$!
  until curl -sf "$URL/healthz" > /dev/null; do sleep 0.05; done
}
stop() { kill -TERM $PID; wait $PID; }
select_id() { curl -s -X POST "$URL/select" | sed -E 's/.*"request_id":"([^"]+)".*/\1/'; }
arm_of() { curl -s -X POST "$URL/select" | sed -E 's/.*"arm":([0-9]+).*/\1/'; }
pulls() { curl -s "$URL/stats" | sed -E 's/.*"arms":\[(.*)\]\}/\1/'; }

echo "== first run: six rewarded selections, one left waiting =="
start
for i in 1 2 3 4 5 6; do
  RESP=$(curl -s -X POST "$URL/select")
  ID=$(echo "$RESP" | sed -E 's/.*"request_id":"([^"]+)".*/\1/')
  ARM=$(echo "$RESP" | sed -E 's/.*"arm":([0-9]+).*/\1/')
  REWARD=0; [ "$ARM" = 1 ] && REWARD=1
  curl -s -o /dev/null -X POST -d "{\"request_id\":\"$ID\",\"reward\":$REWARD}" "$URL/reward"
done
WAITING=$(select_id)
echo "arms: $(pulls)"
stop

echo
echo "== second run, same database =="
start
echo "arms: $(pulls)"
echo "reward for the selection made before the restart: HTTP $(curl -s -o /dev/null -w '%{http_code}' -X POST -d "{\"request_id\":\"$WAITING\",\"reward\":1}" "$URL/reward")"
echo "the same reward again:                            HTTP $(curl -s -o /dev/null -w '%{http_code}' -X POST -d "{\"request_id\":\"$WAITING\",\"reward\":1}" "$URL/reward")"
stop

echo
echo "== what is in the database =="
psql "$DATABASE_URL" -c 'SELECT * FROM arm_totals ORDER BY arm'
psql "$DATABASE_URL" -c 'SELECT arm, rewarded, count(*) FROM pending GROUP BY arm, rewarded ORDER BY arm, rewarded'

echo
echo "== third run: 2s time-to-live, silence counts as reward 0 =="
start BANDIT_PENDING_TTL=2s BANDIT_EXPIRE_INTERVAL=500ms BANDIT_EXPIRE_REWARD=0
for i in 1 2 3; do select_id > /dev/null; done
LATE=$(select_id)
echo "arms right after four unanswered selections: $(pulls)"
sleep 3
echo "arms three seconds later:                    $(pulls)"
echo "a reward arriving after the deadline: HTTP $(curl -s -o /dev/null -w '%{http_code}' -X POST -d "{\"request_id\":\"$LATE\",\"reward\":1}" "$URL/reward")"
stop
psql "$DATABASE_URL" -Atc "SELECT 'expired rows still in pending: ' || count(*) FROM pending WHERE expires_at <= now()"

echo
echo "== server log (selected lines) =="
grep -E "database connected|selections expired|no database" "$LOG" | sed -E 's/^time=[^ ]+ //' | cut -c1-140
