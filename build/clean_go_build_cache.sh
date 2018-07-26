#!/bin/sh

# Cleaning the Go cache only makes sense if we actually have Go installed... or
# if Go is actually callable. This does not hold true during deb packaging, so
# we need an explicit check to avoid build failures.
if ! command -v go > /dev/null; then
  exit
fi

version_gt() {
  test "$(printf '%s\n' "$@" | sort -V | head -n 1)" != "$1"
}

golang_version=$(go version |cut -d' ' -f3 |sed 's/go//')

# Clean go build cache when go version is greater than or equal to 1.10
if !(version_gt 1.10 $golang_version); then
    go clean -cache
fi
