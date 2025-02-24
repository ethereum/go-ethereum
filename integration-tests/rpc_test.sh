#!/bin/bash
set -e

signersDump=$(jq . $signersFile)
privKey=$(echo "$signersDump" | jq -r ".[0].priv_key")
rpc_url="http://localhost:8545"

cd matic-cli/tests/rpc-tests

go mod tidy
go run main.go --priv-key $privKey --rpc-url $rpc_url

cd -