#!/bin/bash

function version_gt() { test "$(printf '%s\n' "$@" | sort -V | head -n 1)" != "$1"; }
function go_version {
    version=$(go version)
    echo $version
        regex="([0-9].[0-9]+.[0-9])"
        if [[ $version =~ $regex ]]; then 
            echo ${BASH_REMATCH[1]}
    fi
}

golang_version=$(go_version)

# Clean go build cache when go version is greater than or equal to 1.10
if !(version_gt 1.10 $golang_version); then
    go clean -cache
fi
