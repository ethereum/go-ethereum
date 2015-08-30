#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

GO_ETHEREUM_FLAGS=""

# set gitCommit when running from a Git checkout.
if [ -f ".git/HEAD" ]; then
    GO_ETHEREUM_FLAGS=$GO_ETHEREUM_FLAGS"-ldflags '-X main.gitCommit $(git rev-parse HEAD)'"
fi

: ${GO_OPENCL:="false"}
if [ "$GO_OPENCL" != "false" ]; then
    GO_ETHEREUM_FLAGS=$GO_ETHEREUM_FLAGS" -tags 'opencl'"
fi

echo $GO_ETHEREUM_FLAGS

