#!/usr/bin/env bash
set -euo pipefail

# Helper script to check whether `go mod tidy` introduces changes.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

if [ ! -f go.mod ]; then
  echo "go.mod not found in repository root."
  exit 1
fi

echo "Running go mod tidy (dry check)..."

# Save a snapshot of go.mod and go.sum
cp go.mod /tmp/go.mod.before
cp go.sum /tmp/go.sum.before

go mod tidy

changed=
if ! diff -q go.mod /tmp/go.mod.before >/dev/null 2>&1; then
  changed=1
fi

if ! diff -q go.sum /tmp/go.sum.before >/dev/null 2>&1; then
  changed=1
fi

if [ -n "${changed}" ]; then
  echo "⚠️  go mod tidy introduced changes."
  echo "Please review and commit them if appropriate."
else
  echo "✅ go mod tidy did not change go.mod or go.sum."
fi

# Restore original files to avoid modifying the working tree.
mv /tmp/go.mod.before go.mod
mv /tmp/go.sum.before go.sum
