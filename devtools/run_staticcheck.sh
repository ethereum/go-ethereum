#!/usr/bin/env bash
set -euo pipefail

# Wrapper script to run staticcheck against the go-ethereum codebase,
# if staticcheck is installed.

if ! command -v staticcheck >/dev/null 2>&1; then
  echo "staticcheck is not installed. You can install it via:"
  echo "  go install honnef.co/go/tools/cmd/staticcheck@latest"
  exit 0
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

echo "Running staticcheck on ./..."
staticcheck ./...

echo
echo "staticcheck finished."
