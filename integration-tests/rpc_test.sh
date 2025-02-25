#!/bin/bash
set -e

signersFile="matic-cli/devnet/devnet/signer-dump.json"
signersDump=$(jq . "$signersFile")
privKey=$(echo "$signersDump" | jq -r ".[0].priv_key")
rpc_url="http://localhost:8545"

cd matic-cli/tests/rpc-tests

go mod tidy
go run . --priv-key "$privKey" --rpc-url "$rpc_url" --log-req-res true

cd -