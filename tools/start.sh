#!/usr/bin/env bash
# Copies the reference project as it stands at the end of a chapter, so you can
# start (or resume) from a known-good state.
#
#   tools/start.sh ch03 work/banditlab
#
# The target directory must not exist or must be empty. Nothing is overwritten.
set -eu
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
if [ $# -ne 2 ]; then echo "usage: tools/start.sh chNN <new-directory>" >&2; exit 2; fi
src="$ROOT/solutions/$1"
dst="$2"
if [ ! -d "$src" ]; then echo "no such chapter: $1 (see solutions/)" >&2; exit 2; fi
if [ -e "$dst" ] && [ -n "$(ls -A "$dst" 2>/dev/null)" ]; then
  echo "$dst exists and is not empty; pick a new directory" >&2; exit 1
fi
mkdir -p "$dst"
# Copy everything except build artefacts.
(cd "$src" && tar cf - --exclude='*.test' --exclude='*.out' .) | (cd "$dst" && tar xf -)
echo "copied solutions/$1 to $dst"
echo "next: cd $dst && go test ./...   then continue with the next chapter's page"
