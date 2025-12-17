#!/usr/bin/env bash
set -euo pipefail

# Dumps the current list of peers from a running geth node.

RPC_URL="${RPC_URL:-http://127.0.0.1:8545}"

echo "Using RPC URL: ${RPC_URL}"
echo

payload='{"jsonrpc":"2.0","method":"admin_peers","params":[],"id":1}'

response="$(curl -s -X POST "${RPC_URL}" -H "Content-Type: application/json" -d "${payload}")"

if [ -z "${response}" ]; then
  echo "No response received. Is geth running with admin RPC enabled?"
  exit 1
fi

echo "Peers response:"
echo "${response}"
