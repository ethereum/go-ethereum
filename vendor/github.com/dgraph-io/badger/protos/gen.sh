#!/bin/bash

# You might need to go get -v github.com/gogo/protobuf/...

protos=${GOPATH-$HOME/go}/src/github.com/dgraph-io/badger/protos
pushd $protos > /dev/null
protoc --gofast_out=plugins=grpc:. -I=. *.proto
