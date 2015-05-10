#!/bin/bash

# This script runs all package tests and merges the resulting coverage
# profiles. Coverage is accounted per package under test.

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

echo "mode: count" > profile.cov

for pkg in $(go list ./...); do
    # drop the namespace prefix.
    dir=${pkg##github.com/ethereum/go-ethereum/}
    
    if [[ $dir != "tests/vm" ]]; then
        go test -covermode=count -coverprofile=$dir/profile.tmp $pkg
    fi
    if [[ -f $dir/profile.tmp ]]; then
        tail -n +2 $dir/profile.tmp >> profile.cov
        rm $dir/profile.tmp
    fi
done
