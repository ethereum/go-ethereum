#!/usr/bin/env bash

build/bin/geth --rpc --rpcaddr "localhost" --rpcport "8080" --maxpeers "2048" --etherbase "0" console
