#!/usr/bin/env bash
# Checks the chapters that use the files/solution layout, by playing a reader:
#   - start from the previous chapter's solution (chapter 1: an empty module)
#     and copy the chapter's files/ in, as the page tells the reader to;
#   - that project must not pass its tests yet (the tasks are not done);
#   - the solution must pass gofmt, vet and go test -race, extras included;
#   - files without TASK markers must be identical to the solution's copy, and
#     after copying, every test file must match the solution's (the reader has
#     the whole current test suite), unless the reader writes it in a task;
#   - it lists the files the reader edits by hand, which the page must cover.
#
# It also enforces who owns which file, from chapter 1 on:
#   - a file the reader has filled in or edited is theirs, and no later
#     chapter's files/ may contain it (copying would overwrite their work);
#   - files the course may replace later say so in a header (courseHeader
#     below); the reader is never asked to edit one, and files/ only ever
#     replaces an existing source file if it has that header.
#
#   tools/check_chapters.sh [chNN ...]   (no arguments: every chapter)
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
courseHeader='This file comes with the course'
status=0
all=()
for d in "$ROOT"/ch*/files; do all+=("$(basename "$(dirname "$d")")"); done
declare -A want owned
if [ $# -eq 0 ]; then for ch in "${all[@]}"; do want[$ch]=1; done; else for ch in "$@"; do want[$ch]=1; done; fi
fail() { echo "FAIL $ch: $*"; status=1; }
extras() { (cd "$1" && ls -d _extras/*/ 2>/dev/null | sed 's|^|./|; s|/$||' | tr '\n' ' '); }

# Ownership is cumulative, so every chapter is walked in order; the expensive
# test runs happen only for the chapters asked for.
for ch in "${all[@]}"; do
  files="$ROOT/$ch/files"; sol="$ROOT/$ch/solution"
  n=$((10#${ch#ch})); prev="$ROOT/$(printf 'ch%02d' $((n - 1)))/solution"
  work=$(mktemp -d)
  if [ "$n" -eq 1 ]; then cp "$sol/go.mod" "$work/"; else (cd "$prev" && tar cf - .) | (cd "$work" && tar xf -); fi
  existed=$(cd "$work" && find . -type f | sed 's|^\./||')

  # Ownership rules for what this chapter ships.
  while IFS= read -r f; do
    if [ -n "${owned[$f]:-}" ]; then fail "files/ contains $f, which the reader has owned since ${owned[$f]}; change it with a task instead"; fi
    case "$f" in *_test.go|testdata/*|_examples/*) continue;; esac
    if grep -qxF "$f" <<< "$existed" && ! grep -q "$courseHeader" "$files/$f"; then
      fail "files/ replaces $f, which has no course header, so the reader may have changed it"
    fi
  done < <(cd "$files" && find . -type f | sed 's|^\./||' | sort)

  (cd "$files" && tar cf - .) | (cd "$work" && tar xf -)

  if [ -n "${want[$ch]:-}" ]; then
    if (cd "$work" && go test ./... $(extras "$work") > /dev/null 2>&1); then fail "a reader's project passes before the tasks are done"
    else echo "ok   $ch: tests fail until the tasks are done"; fi

    if out=$(cd "$sol" && test -z "$(gofmt -l .)" && go vet ./... $(extras "$sol") && go test -race -count=1 ./... $(extras "$sol") 2>&1); then
      echo "ok   $ch: solution passes gofmt, vet and go test -race"
    else fail "solution does not pass:"; echo "$out" | grep -v '^ok' | head -15; fi
  fi

  while IFS= read -r f; do
    [ -f "$sol/$f" ] || { fail "$f is in files/ but not in the solution"; continue; }
    if grep -q 'TASK [0-9E]' "$files/$f"; then owned[$f]=${owned[$f]:-$ch}; continue; fi
    cmp -s "$files/$f" "$sol/$f" || fail "$f has no TASK marker but differs from the solution"
  done < <(cd "$files" && find . -type f | sed 's|^\./||')

  edited=()
  while IFS= read -r f; do
    if [ ! -f "$work/$f" ]; then fail "$f is in the solution but the reader never gets or writes it (not in files/ or an earlier solution)"; continue; fi
    cmp -s "$work/$f" "$sol/$f" && continue
    if [ -f "$files/$f" ] && grep -q 'TASK [0-9E]' "$files/$f"; then continue; fi
    # A test file the reader wrote in an earlier task is theirs to change too.
    case "$f" in *_test.go) if [ -z "${owned[$f]:-}" ]; then fail "$f: the reader's copy differs from the solution's"; continue; fi;; esac
    grep -q "$courseHeader" "$work/$f" && fail "the reader edits $f, but its header says the course may replace it"
    edited+=("$f"); owned[$f]=${owned[$f]:-$ch}
  done < <(cd "$sol" && find . -type f | sed 's|^\./||' | sort)
  if [ -n "${want[$ch]:-}" ] && [ "$n" -gt 1 ]; then echo "info $ch: files the reader edits by hand: ${edited[*]:-none}"; fi
  rm -rf "$work"
done
exit $status
