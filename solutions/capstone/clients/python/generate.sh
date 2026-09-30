#!/bin/sh
# Generates the Python stubs from the schema the Go services use.
#   sh generate.sh [PROTO_DIR]     (default: ../../proto)
# Needs grpcio-tools (pip install -r requirements.txt), which bundles protoc.
set -eu
PROTO=${1:-../../proto}
OUT=$(dirname "$0")/gen
mkdir -p "$OUT"
python -m grpc_tools.protoc -I "$PROTO" --python_out="$OUT" --grpc_python_out="$OUT" bandit/v1/bandit.proto
# The generated code imports its sibling as a top-level module; make the
# directories packages so `from bandit.v1 import bandit_pb2` works.
touch "$OUT/bandit/__init__.py" "$OUT/bandit/v1/__init__.py"
