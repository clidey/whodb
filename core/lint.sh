#!/usr/bin/env bash
# Build-on-demand wrapper for golangci-lint with gounslop module plugin.
# The custom binary (golangci-lint + gounslop) is built once into core/tmp/
# and reused. Pass all arguments through, e.g.:
#   ./lint.sh run ./...
#   ./lint.sh fmt ./...
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY="${TMPDIR:-/tmp}/whodb-custom-gcl"

# Build if missing
if [ ! -x "${BINARY}" ]; then
	echo "lint.sh: custom binary not found, building..." >&2
	golangci-lint custom --destination "${TMPDIR:-/tmp}" --name whodb-custom-gcl
	if [ ! -x "${BINARY}" ]; then
		echo "lint.sh: ERROR: golangci-lint custom did not produce the expected binary at ${BINARY}" >&2
		exit 1
	fi
	echo "lint.sh: custom binary built at ${BINARY}" >&2
fi

# Default to ./... if no paths provided
if [ $# -eq 0 ]; then
	exec "${BINARY}" run --config "${SCRIPT_DIR}/.golangci.yml" ./...
else
	exec "${BINARY}" "$@"
fi
