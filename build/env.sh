#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
chydir="$workspace/src/github.com/chattynet"
if [ ! -L "$chydir/chatty" ]; then
    mkdir -p "$chydir"
    cd "$chydir"
    ln -s ../../../../../. chatty
    cd "$root"
fi

# Set up the environment to use the workspace.
# Also add Godeps workspace so we build using canned dependencies.
GOPATH="$chydir/chatty/Godeps/_workspace:$workspace"
GOBIN="$PWD/build/bin"
export GOPATH GOBIN

# Run the command inside the workspace.
cd "$chydir/chatty"
PWD="$chydir/chatty"

# Launch the arguments with the configured environment.
exec "$@"
