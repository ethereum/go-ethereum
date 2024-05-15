#!/bin/sh

# Deploys create2 proxy according to https://github.com/primev/deterministic-deployment-proxy

set -e

# Use the first command line argument, if provided. Otherwise, use the environment variable.
JSON_RPC="${1:-$JSON_RPC_URL}"

if [ -z "${JSON_RPC}" ]; then
    echo "Usage: $0 <JSON_RPC_URL> or set the JSON_RPC_URL environment variable."
    exit 1
fi

if ! [ -x "$(command -v curl)" ]; then
    echo "Curl must be installed to deploy the create2 proxy" >&2
    exit 1
fi

# Check if contract already deployed
DATA='{"jsonrpc":"2.0","method":"eth_getCode","params":["0x4e59b44847b379578588920ca78fbf26c0b4956c", "latest"],"id":1}'
RESPONSE=$(curl -s -X POST --data "${DATA}" -H "Content-Type: application/json" "${JSON_RPC}")
CODE=$(echo "${RESPONSE}" | jq -r '.result')
if [ -z "${RESPONSE}" ] || [ "${RESPONSE}" = "null" ]; then
    echo "Error: No response from JSON RPC at ${JSON_RPC}"
    exit 1
fi
if [ "${CODE}" != "0x" ]; then
    echo "Contract already deployed at 0x4e59b44847b379578588920ca78fbf26c0b4956c"
    exit 0
else
    echo "No contract deployed at 0x4e59b44847b379578588920ca78fbf26c0b4956c. Deploying..."
fi

# Note deployement signer account needs at least 10000000000000000 wei allocated on genesis to send tx

# Set presigned transaction 
TRANSACTION="0xf8a58085174876e800830186a08080b853604580600e600039806000f350fe7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156039578182fd5b8082525050506014600cf31ba02222222222222222222222222222222222222222222222222222222222222222a02222222222222222222222222222222222222222222222222222222222222222"

# deploy contract 
curl -s "${JSON_RPC}" -X 'POST' -H 'Content-Type: application/json' --data "{\"jsonrpc\":\"2.0\", \"id\":1, \"method\": \"eth_sendRawTransaction\", \"params\": [\"$TRANSACTION\"]}"

sleep 5

# For prod we'll have to set gas params s.t. no ether is leftover here. For now we warn
RESPONSE=$(curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x3fab184622dc19b6109349b94811493bf2a45362", "latest"],"id":1}' -H "Content-Type: application/json" "${JSON_RPC}")
if [ "$(echo "${RESPONSE}" | jq -r '.result')" != "0x0" ]; then
    echo "WARNING: Deployment signer (0x3fab184622dc19b6109349b94811493bf2a45362) has leftover balance of $(echo "${RESPONSE}" | jq -r '.result') wei"
fi
