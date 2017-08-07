#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
ethdir="$workspace/src/github.com/ethereum"
if [ ! -L "$ethdir/go-ethereum" ]; then
    mkdir -p "$ethdir"
    cd "$ethdir"
    ln -s ../../../../../. go-ethereum
    cd "$workspace/src/github.com"
    ln -s ./ethereum/go-ethereum/vendor/github.com/Azure Azure
    ln -s ./ethereum/go-ethereum/vendor/github.com/dgrijalva dgrijalva
    cd "$workspace/src"
    ln -s ./github.com/ethereum/go-ethereum/vendor/golang.org golang.org
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$ethdir/go-ethereum"
PWD="$ethdir/go-ethereum"

# Launch the arguments with the configured environment.
exec "$@"
