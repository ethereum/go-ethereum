#!/bin/bash

set -e

# Add godep workspace to GOPATH. We do it manually instead of using
# 'godep go test' or 'godep restore' so godep doesn't need to be installed.
GOPATH="$PWD/Godeps/_workspace:$GOPATH"

# Install packages before testing. Not doing this would cause
# 'go test' to recompile all package dependencies before testing each package.
go install ./...

# Run test coverage on each subdirectories and merge the coverage profile.
echo "mode: count" > profile.cov

# Standard go tooling behavior is to ignore dirs with leading underscors
for dir in $(find . -maxdepth 10 -not -path './.git*' -not -path '*/_*' -type d);
do
if ls $dir/*.go &> /dev/null; then
    # echo $dir
    if [[ $dir != "./tests/vm" && $dir != "." ]]
    then
        go test -covermode=count -coverprofile=$dir/profile.tmp $dir
    fi
    if [ -f $dir/profile.tmp ]
    then
        cat $dir/profile.tmp | tail -n +2 >> profile.cov
        rm $dir/profile.tmp
    fi
fi
done

go tool cover -func profile.cov

