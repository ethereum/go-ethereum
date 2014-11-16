#!/bin/bash

set -e

TEST_DEPS=$(go list -f '{{.TestImports}} {{.XTestImports}}' github.com/ethereum/go-ethereum/... | sed -e 's/\[//g' | sed -e 's/\]//g')
if [ "$TEST_DEPS" ]; then
  go get -race $TEST_DEPS
fi
