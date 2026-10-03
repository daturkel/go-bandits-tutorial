#!/usr/bin/env bash
# Checks the chapters that use the files/solution layout, by playing a student:
#   - start from the previous chapter's solution (chapter 1: an empty module)
#     and copy the chapter's files/ in, as the page tells the reader to;
#   - that project must not pass its tests yet (the tasks are not done);
#   - the solution must pass gofmt, vet and go test -race, extras included;
#   - files without TASK markers must be identical to the solution's copy, and
#     after copying, every test file must match the solution's (the reader has
#     the whole current test suite), unless the reader writes it in a task;
#   - it lists the files the reader edits by hand, which the page must cover.
#
#   tools/check_chapters.sh [chNN ...]
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
status=0
chapters=("$@")
if [ ${#chapters[@]} -eq 0 ]; then
  for d in "$ROOT"/ch*/files; do chapters+=("$(basename "$(dirname "$d")")"); done
fi
fail() { echo "FAIL $ch: $*"; status=1; }
extras() { (cd "$1" && ls -d _extras/*/ 2>/dev/null | sed 's|^|./|; s|/$||' | tr '\n' ' '); }

for ch in "${chapters[@]}"; do
  files="$ROOT/$ch/files"; sol="$ROOT/$ch/solution"
  n=$((10#${ch#ch})); prev="$ROOT/$(printf 'ch%02d' $((n - 1)))/solution"
  work=$(mktemp -d)
  if [ "$n" -eq 1 ]; then cp "$sol/go.mod" "$work/"; else (cd "$prev" && tar cf - .) | (cd "$work" && tar xf -); fi
  (cd "$files" && tar cf - .) | (cd "$work" && tar xf -)

  if (cd "$work" && go test ./... $(extras "$work") > /dev/null 2>&1); then fail "a reader's project passes before the tasks are done"
  else echo "ok   $ch: tests fail until the tasks are done"; fi

  if out=$(cd "$sol" && test -z "$(gofmt -l .)" && go vet ./... $(extras "$sol") && go test -race -count=1 ./... $(extras "$sol") 2>&1); then
    echo "ok   $ch: solution passes gofmt, vet and go test -race"
  else fail "solution does not pass:"; echo "$out" | grep -v '^ok' | head -15; fi

  while IFS= read -r f; do
    [ -f "$sol/$f" ] || { fail "$f is in files/ but not in the solution"; continue; }
    grep -q 'TASK [0-9E]' "$files/$f" && continue
    cmp -s "$files/$f" "$sol/$f" || fail "$f has no TASK marker but differs from the solution"
  done < <(cd "$files" && find . -type f | sed 's|^\./||')

  edited=()
  while IFS= read -r f; do
    if [ ! -f "$work/$f" ]; then fail "$f is in the solution but the reader never gets or writes it (not in files/ or an earlier solution)"; continue; fi
    cmp -s "$work/$f" "$sol/$f" && continue
    if [ -f "$files/$f" ] && grep -q 'TASK [0-9E]' "$files/$f"; then continue; fi
    case "$f" in *_test.go) fail "$f: the reader's copy differs from the solution's";; *) edited+=("$f");; esac
  done < <(cd "$sol" && find . -type f | sed 's|^\./||' | sort)
  [ "$n" -eq 1 ] || echo "info $ch: files the reader edits by hand: ${edited[*]:-none}"
  rm -rf "$work"
done
exit $status
