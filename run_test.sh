#!/bin/bash

# Download .so files
export LIBSCROLL_ZSTD_VERSION=v0.1.0-rc0-ubuntu20.04
export SCROLL_LIB_PATH=/scroll/lib

sudo mkdir -p $SCROLL_LIB_PATH

sudo wget -O $SCROLL_LIB_PATH/libscroll_zstd.so https://github.com/scroll-tech/da-codec/releases/download/$LIBSCROLL_ZSTD_VERSION/libscroll_zstd.so

# Set the environment variable
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$SCROLL_LIB_PATH
export CGO_LDFLAGS="-L$SCROLL_LIB_PATH -Wl,-rpath,$SCROLL_LIB_PATH"

# Download and install the project dependencies
go run build/ci.go install
go get ./...

# Save the root directory of the project
ROOT_DIR=$(pwd)

# Run genesis test
cd $ROOT_DIR/cmd/geth
go test -test.run TestCustomGenesis

# Run module tests
cd $ROOT_DIR
go run build/ci.go test ./consensus ./core ./eth ./miner ./node ./trie ./rollup/...
