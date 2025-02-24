#!/bin/bash
set -e

signersDump=$(jq . $signersFile)
privKey=$(echo "$signersDump" | jq -r ".[0].priv_key")
rpc_url="http://localhost:8545"


go run matic-cli/tests/rpc-tests/main.go --priv-key $privKey --rpc-url $rpc_url