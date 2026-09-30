#!/usr/bin/env bash
# Checks your own project against a chapter's reference tests.
#
#   tools/check.sh ch03 [dir]        (dir defaults to work/banditlab)
#
# It never touches your directory. It copies it to a scratch location,
# replaces the test files there with the chapter's reference tests and
# testdata, then runs gofmt, go vet, go build and go test -race. A compile
# error almost always means a name differs from the one on the page.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
if [ $# -lt 1 ] || [ $# -gt 2 ]; then echo "usage: tools/check.sh chNN [dir]" >&2; exit 2; fi
ch="$1"; dir="${2:-work/banditlab}"
ref="$ROOT/solutions/$ch"
[ -d "$ref" ] || { echo "no such chapter: $ch" >&2; exit 2; }
[ -f "$dir/go.mod" ] || { echo "$dir/go.mod not found: is this your project directory?" >&2; exit 2; }

tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
(cd "$dir" && tar cf - --exclude='*.test' .) | (cd "$tmp" && tar xf -)

# Use the reader's module path in the reference tests' imports.
mod=$(awk '$1=="module"{print $2}' "$tmp/go.mod")
refmod=$(awk '$1=="module"{print $2}' "$ref/go.mod")

# Drop the reader's tests and testdata, then bring in the reference ones.
find "$tmp" -name '*_test.go' -delete
find "$tmp" -type d -name testdata -prune -exec rm -rf {} +
(cd "$ref" && find . \( -name '*_test.go' -o -path '*/testdata/*' \) -not -path './_*' -print0) |
  while IFS= read -r -d '' f; do
    mkdir -p "$tmp/$(dirname "$f")"
    sed "s|\"$refmod/|\"$mod/|g" "$ref/$f" > "$tmp/$f"
  done

fail=0
step() { echo "\$ $*"; "$@" || fail=1; }
cd "$tmp"
out=$(gofmt -l .); echo '$ gofmt -l .'; [ -z "$out" ] || { echo "$out"; fail=1; }
step go vet ./...
step go build ./...
step go test -race -count=1 ./...
echo
if [ $fail -eq 0 ]; then
  echo "PASS: your project satisfies the $ch tests"
  if grep -rqs "BANDIT_TEST_DATABASE_URL" --include='*_test.go' --include='postgres.go' . && [ -z "${BANDIT_TEST_DATABASE_URL:-}" ]; then
    echo "note: the PostgreSQL tests were skipped (BANDIT_TEST_DATABASE_URL is not set); tools/pg.sh start gives you a local database to run them against"
  fi
else
  echo "FAIL: see above. Compare with solutions/$ch (diff -ru $dir $ref) or restart from it with tools/start.sh"
fi
exit $fail
