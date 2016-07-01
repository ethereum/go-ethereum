#!/bin/sh

set -e

root="$PWD"

if [ ! -f "$root/build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create build/bin if it doesn't exist yet.
if [ ! -L "$root/build/bin" ]; then
    mkdir -p "$root/build/bin"
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$root/../_go_build/_workspace"
ethdir="$workspace/src/github.com/ethereum"
if [ ! -L "$ethdir/go-ethereum" ]; then
    mkdir -p "$ethdir"
    cd "$ethdir"
    ln -s "$root" go-ethereum
    cd "$root"
fi

# Set up the environment to use the workspace.
# Also add Godeps workspace so we build using canned dependencies.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$ethdir/go-ethereum"
PWD="$ethdir/go-ethereum"

# Launch the arguments with the configured environment.
exec "$@"
