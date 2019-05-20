#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# set gitCommit when running from a Git checkout.
if [ -f ".git/HEAD" ]; then
    echo "-ldflags '-X main.gitCommit $(git rev-parse HEAD)'"
fi
