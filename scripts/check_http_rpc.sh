#!/usr/bin/env bash
set -euo pipefail

# Simple helper to verify that an HTTP JSON-RPC endpoint is reachable.
#
# Usage:
#   ./scripts/check_http_rpc.sh
#   RPC_URL=http://10.0.0.5:8545 ./scripts/check_http_rpc.sh

RPC_URL="${RPC_URL:-http://127.0.0.1:8545}"

echo "Checking HTTP JSON-RPC endpoint: ${RPC_URL}"
echo

payload='{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}'

response="$(curl -s -X POST "${RPC_URL}" -H "Content-Type: application/json" -d "${payload}")"

if [ -z "${response}" ]; then
  echo "No response received. Is the node running and HTTP enabled?"
  exit 1
fi

echo "Raw response:"
echo "${response}"
echo
echo "If this includes a valid client version string, the HTTP JSON-RPC endpoint is working."
