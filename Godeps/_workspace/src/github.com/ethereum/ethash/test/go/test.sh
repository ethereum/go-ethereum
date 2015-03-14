#!/bin/bash

# Strict mode
set -e

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do
  DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
done
TEST_DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

export GOPATH=${HOME}/.go
export PATH=$PATH:$GOPATH/bin 
echo "# getting go dependencies (can take some time)..."
cd ${TEST_DIR}/../.. && go get 
cd ${GOPATH}/src/github.com/ethereum/go-ethereum
git checkout poc-9
cd ${TEST_DIR} && go test
