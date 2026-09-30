#!/usr/bin/env bash
# A throwaway local PostgreSQL for the integration tests. It needs the server
# binaries (Debian/Ubuntu: apt install postgresql) but neither Docker nor a
# system-wide database service.
#
#   tools/pg.sh start    initialise (first time) and start on 127.0.0.1:55432
#   tools/pg.sh url      print the connection URL
#   tools/pg.sh stop
#   tools/pg.sh reset    stop and delete the data directory
#
# Data lives in $BANDIT_PG_DIR (default /tmp/banditlab-pg). PostgreSQL refuses
# to run as root, so when this script is run as root it switches to the
# "postgres" user that the package creates.
set -eu
DIR=${BANDIT_PG_DIR:-/tmp/banditlab-pg}
PORT=${BANDIT_PG_PORT:-55432}

BIN=$(dirname "$(ls -d /usr/lib/postgresql/*/bin/pg_ctl 2>/dev/null | sort -V | tail -1 || true)")
if [ ! -x "$BIN/pg_ctl" ]; then BIN=$(dirname "$(command -v pg_ctl || true)"); fi
if [ ! -x "$BIN/pg_ctl" ]; then echo "PostgreSQL server binaries not found" >&2; exit 1; fi

as_pg() {  # run a command as the user that owns the data directory
  if [ "$(id -u)" -eq 0 ]; then su postgres -c "$*"; else sh -c "$*"; fi
}

case "${1:-}" in
  start)
    if [ "$(id -u)" -eq 0 ]; then mkdir -p "$DIR" && chown postgres "$DIR"; else mkdir -p "$DIR"; fi
    if [ ! -f "$DIR/data/PG_VERSION" ]; then
      as_pg "'$BIN/initdb' -D '$DIR/data' -A trust -U postgres >'$DIR/initdb.log' 2>&1"
    fi
    if ! as_pg "'$BIN/pg_ctl' -D '$DIR/data' status" >/dev/null 2>&1; then
      as_pg "'$BIN/pg_ctl' -D '$DIR/data' -o '-p $PORT -k $DIR -c listen_addresses=127.0.0.1' -l '$DIR/server.log' -w start" >/dev/null
    fi
    "$0" url
    ;;
  url)  echo "postgres://postgres@127.0.0.1:$PORT/postgres?sslmode=disable" ;;
  stop) as_pg "'$BIN/pg_ctl' -D '$DIR/data' -m fast stop" 2>/dev/null || true ;;
  reset) "$0" stop; rm -rf "$DIR" ;;
  *) echo "usage: tools/pg.sh start|stop|url|reset" >&2; exit 2 ;;
esac
