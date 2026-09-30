#!/usr/bin/env bash
# Runs banditd under load and reads its CPU profile.
# Usage: BANDITD=... BANDITLOAD=... _examples/profile.sh <log-level>
set -u
BANDITD=${BANDITD:-banditd}
BANDITLOAD=${BANDITLOAD:-banditload}
LEVEL=${1:-info}
DIR=$(mktemp -d)

"$BANDITD" -addr localhost:8083 -debug-addr localhost:6063 -policy thompson -arms 10 -log-level "$LEVEL" 2> "$DIR/log" &
PID=$!
trap 'kill -INT $PID 2>/dev/null' EXIT
until curl -sf localhost:8083/healthz > /dev/null; do sleep 0.05; done

# Collect 6 seconds of CPU profile while 32 simulated users hammer the service.
curl -s "localhost:6063/debug/pprof/profile?seconds=6" -o "$DIR/cpu.pprof" &
"$BANDITLOAD" -url http://localhost:8083 -c 32 -d 8s -scenario needle -reward-fraction 0.5 | sed -n 1p
wait %2 2>/dev/null
curl -s localhost:6063/debug/pprof/mutex -o "$DIR/mutex.pprof"

echo "--- where the CPU time went (log level $LEVEL) ---"
go tool pprof -top -nodecount=6 "$BANDITD" "$DIR/cpu.pprof" 2>/dev/null | sed -n '5,6p;8,14p'
echo "--- cumulative share of chosen functions ---"
go tool pprof -top -cum -nodecount=400 "$BANDITD" "$DIR/cpu.pprof" 2>/dev/null |
  grep -E 'slog\.\(\*Logger\)\.LogAttrs|bandit\.\(\*Locked\)\.Select$|store\.\(\*Memory\)\.AddPending|runtime\.gcBgMarkWorker$|\(\*conn\)\.serve$' |
  awk '{printf "%7s %7s  %s\n", $4, $5, $6}'
echo "--- time goroutines spent waiting for locks (mutex profile, share of all lock waiting) ---"
go tool pprof -top -cum -nodecount=400 "$BANDITD" "$DIR/mutex.pprof" 2>/dev/null |
  grep -E 'store\.\(\*Memory\)\.(AddPending|Reward)$|bandit\.\(\*Locked\)\.(Select|Update)$' |
  awk '{printf "%7s %7s  %s\n", $4, $5, $6}'
