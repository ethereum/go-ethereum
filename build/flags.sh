#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Since Go 1.5, the separator char for link time assignments
# is '=' and using ' ' prints a warning. However, Go < 1.5 does
# not support using '='.
sep=$(go version | awk '{ if ($3 >= "go1.5" || index($3, "devel")) print "="; else print " "; }' -)

# set gitCommit when running from a Git checkout.
if [ -f ".git/HEAD" ]; then
    echo "-ldflags '-X main.gitCommit$sep$(git rev-parse HEAD)'"
fi

if [ ! -z "$GO_OPENCL" ]; then
   echo "-tags opencl"
fi
