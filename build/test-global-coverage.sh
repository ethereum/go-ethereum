#!/usr/bin/env bash

set -e
echo "" > coverage.txt

for d in $(find ./* -maxdepth 10 -type d -not -path "./build" -not -path "./Godeps/*" ); do
    if ls $d/*.go &> /dev/null; then
        go test  -coverprofile=profile.out -covermode=atomic $d
        if [ -f profile.out ]; then
            cat profile.out >> coverage.txt
            echo '<<<<<< EOF' >> coverage.txt
            rm profile.out
        fi
    fi
done
