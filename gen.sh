#!/usr/bin/env sh

set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$repo_root"

python_bin=${PYTHON_BIN:-python}

if ! command -v "$python_bin" >/dev/null 2>&1; then
    echo "Python executable not found: $python_bin" >&2
    exit 1
fi

if ! "$python_bin" -c "import grpc_tools.protoc" >/dev/null 2>&1; then
    echo "grpcio-tools is required; install worker/requirements-dev.txt" >&2
    exit 1
fi

for plugin in protoc-gen-go protoc-gen-go-grpc; do
    if ! command -v "$plugin" >/dev/null 2>&1; then
        echo "Required protobuf plugin not found: $plugin" >&2
        exit 1
    fi
done

"$python_bin" -m grpc_tools.protoc \
    -I. \
    --go_out=gateway \
    --go_opt=paths=source_relative \
    --go-grpc_out=gateway \
    --go-grpc_opt=paths=source_relative \
    proto/analysis.proto

"$python_bin" -m grpc_tools.protoc \
    -Iproto \
    --python_out=worker \
    --grpc_python_out=worker \
    proto/analysis.proto
