#!/usr/bin/env bash
# Verifies every exercise: the starter must FAIL its tests (a real test failure,
# not a compile error) and the reference answer must pass them under -race.
#
#   tools/check_exercises.sh [chNN ...]
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
status=0
chapters=("$@")
if [ ${#chapters[@]} -eq 0 ]; then
  for d in "$ROOT"/exercises/ch*; do chapters+=("$(basename "$d")"); done
fi
for ch in "${chapters[@]}"; do
  for ex in "$ROOT/exercises/$ch"/ex*/; do
    name="$ch/$(basename "$ex")"
    tmp=$(mktemp -d)
    cp "$ROOT/exercises/$ch/go.mod" "$tmp/"
    [ -f "$ROOT/exercises/$ch/go.sum" ] && cp "$ROOT/exercises/$ch/go.sum" "$tmp/"
    mkdir "$tmp/ex"
    find "$ex" -maxdepth 1 -type f -exec cp {} "$tmp/ex/" \;
    # starter must fail with a test failure, not a build error
    out=$(cd "$tmp" && go test ./ex/ 2>&1); code=$?
    if [ $code -eq 0 ]; then echo "FAIL $name: starter passes its tests"; status=1
    elif echo "$out" | grep -q "^--- FAIL\|^    --- FAIL\|panic:" ; then echo "ok   $name: starter fails as expected"
    else echo "FAIL $name: starter does not compile or fails oddly:"; echo "$out" | head -5; status=1; fi
    # reference must pass
    cp "$ex"reference/*.go "$tmp/ex/"
    if out=$(cd "$tmp" && go vet ./ex/ && go test -race -count=1 ./ex/ 2>&1); then echo "ok   $name: reference passes"
    else echo "FAIL $name: reference fails:"; echo "$out" | head -10; status=1; fi
    rm -rf "$tmp"
  done
done
exit $status
