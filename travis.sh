#!/bin/bash

set -e
REPO_DIR=$PWD
GITHUB_REPO=$(basename $PWD)
GITHUB_USER=$(basename $(cd .. && pwd))
export GOPATH=/tmp/$GITHUB_USER/$GITHUB_REPO.$PPID

mkdir -p $GOPATH/src/github.com/$GITHUB_USER
cp -r $REPO_DIR $GOPATH/src/github.com/$GITHUB_USER/$GITHUB_REPO
echo Fetching package dependicies
go get -race github.com/$GITHUB_USER/$GITHUB_REPO/...
echo Fetching test dependicies
TEST_DEPS=$(go list -f '{{.TestImports}} {{.XTestImports}}' github.com/$GITHUB_USER/$GITHUB_REPO/... | sed -e 's/\[//g' | sed -e 's/\]//g')
if [ "$TEST_DEPS" ]; then
  go get -race $TEST_DEPS
fi
# echo Building test dependicies
# go test -race -i github.com/$GITHUB_USER/$GITHUB_REPO/...
# echo Running tests
# go test -race -cpu=1,2,4 -v github.com/$GITHUB_USER/$GITHUB_REPO/...
