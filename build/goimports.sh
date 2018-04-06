#!/usr/bin/env bash

find_files() {
  find . -not \( \
      \( \
        -wholename '.github' \
        -o -wholename './build/_workspace' \
        -o -wholename './build/bin' \
        -o -wholename './crypto/bn256' \
        -o -wholename '*/vendor/*' \
      \) -prune \
    \) -name '*.go'
}

GOFMT="gofmt -s -w";
GOIMPORTS="goimports -w";
find_files | xargs $GOFMT;
find_files | xargs $GOIMPORTS;