#!/bin/sh

set -ex

# Deploys create2 proxy according to https://github.com/primev/deterministic-deployment-proxy

PROXY_ADDRESS="0x4e59b44847b379578588920ca78fbf26c0b4956c"
SIGNER_ADDRESS="0x3fab184622dc19b6109349b94811493bf2a45362"
# The following transaction string contains fixed from address corresponding to the signer address: 0x3fab184622dc19b6109349b94811493bf2a45362
TRANSACTION="0xf8a58085174876e800830186a08080b853604580600e600039806000f350fe7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156039578182fd5b8082525050506014600cf31ba02222222222222222222222222222222222222222222222222222222222222222a02222222222222222222222222222222222222222222222222222222222222222"

help() {
    echo "Usage: $0 <RPC_URL>"
    echo "  RPC_URL: URL of the JSON RPC endpoint"
    exit 1
}

RPC_URL="${1:-$RPC_URL}"
if [ -z "${RPC_URL}" ]; then
    help
fi

RESPONSE=$(
  curl \
    --silent \
    --request POST \
    --header "Content-Type: application/json" \
    --data '{
      "jsonrpc": "2.0",
      "method": "eth_getCode",
      "params": ["'"${PROXY_ADDRESS}"'", "latest"],
      "id": 1
    }' \
  "${RPC_URL}")
if [ -z "${RESPONSE}" ] || [ "${RESPONSE}" = "null" ]; then
    echo "Error: No response from JSON RPC at ${RPC_URL}"
    exit 1
fi
if [ "$(echo "${RESPONSE}" | jq -r '.result')" != "0x" ]; then
    echo "Contract already deployed at ${PROXY_ADDRESS}"
    exit 0
fi

echo "No contract deployed at ${PROXY_ADDRESS}, deploying..."
curl \
  --silent "${RPC_URL}" \
  --request 'POST' \
  --header 'Content-Type: application/json' \
  --data '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "eth_sendRawTransaction",
    "params": ["'"${TRANSACTION}"'"]
  }'
sleep 5

# For prod we'll have to set gas params s.t. no ether is leftover here. For now we warn
RESPONSE=$(
  curl \
    --silent \
    --request POST \
    --header "Content-Type: application/json" \
    --data '{
      "jsonrpc": "2.0",
      "method": "eth_getBalance",
      "params": ["'"${SIGNER_ADDRESS}"'", "latest"],
      "id": 1
    }' \
  "${RPC_URL}")
if [ -z "${RESPONSE}" ] || [ "${RESPONSE}" = "null" ]; then
    echo "Error: No response from JSON RPC at ${RPC_URL}"
    exit 1
fi
RESULT="$(echo "${RESPONSE}" | jq -r '.result')"
if [ "${RESULT}" != "0x0" ]; then
    echo "WARNING: Deployment signer (${SIGNER_ADDRESS}) has leftover balance of ${RESULT} wei."
fi
