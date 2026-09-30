#!/usr/bin/env bash
# Copies a chapter's starter project: the previous chapter's finished project
# with the parts you write left out.
#
#   tools/starter.sh ch01 work/banditlab
#
# The target directory must not exist or must be empty. Nothing is overwritten.
set -eu
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
if [ $# -ne 2 ]; then echo "usage: tools/starter.sh chNN <new-directory>" >&2; exit 2; fi
src="$ROOT/starters/$1"
dst="$2"
if [ ! -d "$src" ]; then echo "no starter for $1 (see starters/)" >&2; exit 2; fi
if [ -e "$dst" ] && [ -n "$(ls -A "$dst" 2>/dev/null)" ]; then
  echo "$dst exists and is not empty; pick a new directory" >&2; exit 1
fi
mkdir -p "$dst"
(cd "$src" && tar cf - --exclude='*.test' --exclude='*.out' .) | (cd "$dst" && tar xf -)
echo "copied starters/$1 to $dst"
echo "next: cd $dst && go test ./...   then work through the tasks on the chapter's page"
