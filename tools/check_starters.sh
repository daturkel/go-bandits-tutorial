#!/usr/bin/env bash
# Checks every starter against its chapter's reference solution:
#   - the starter's tests must not pass (the tasks are not done yet);
#   - the reference solution, with the starter's test files, must pass under -race;
#   - files the starter did not blank out must match the solution exactly.
#
#   tools/check_starters.sh [chNN ...]
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
status=0
chapters=("$@")
if [ ${#chapters[@]} -eq 0 ]; then
  for d in "$ROOT"/starters/ch*; do chapters+=("$(basename "$d")"); done
fi
for ch in "${chapters[@]}"; do
  start="$ROOT/starters/$ch"; sol="$ROOT/solutions/$ch"
  if (cd "$start" && go test ./... > /dev/null 2>&1); then echo "FAIL $ch: the starter's tests already pass"; status=1
  else echo "ok   $ch: starter does not pass yet"; fi

  tmp=$(mktemp -d)
  (cd "$sol" && tar cf - .) | (cd "$tmp" && tar xf -)
  # the starter's tests replace the solution's, so they are the ones checked
  find "$tmp" -name '*_test.go' -delete
  (cd "$start" && find . -name '*_test.go' -print0) | while IFS= read -r -d '' f; do
    mkdir -p "$tmp/$(dirname "$f")"; cp "$start/$f" "$tmp/$f"
  done
  if out=$(cd "$tmp" && go vet ./... && go test -race -count=1 ./... 2>&1); then echo "ok   $ch: solution passes the starter's tests"
  else echo "FAIL $ch: solution fails the starter's tests:"; echo "$out" | head -10; status=1; fi
  rm -rf "$tmp"

  # Files without TASK markers are copied through unchanged.
  drift=$(cd "$start" && grep -rL 'TASK [0-9E]' --include='*.go' . | grep -v '_test.go' | while read -r f; do
    cmp -s "$start/$f" "$sol/$f" || echo "$f"; done)
  if [ -n "$drift" ]; then echo "FAIL $ch: starter files differ from the solution though they have no tasks: $drift"; status=1
  else echo "ok   $ch: files without tasks match the solution"; fi
done
exit $status
