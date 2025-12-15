```bash
#!/usr/bin/env bash
set -euo pipefail
```

# Prints basic network information by attaching to a running geth instance.
#
# Usage:
#   ./scripts/geth-net-info.sh
#
# By default it attaches to http://127.0.0.1:8545. Override via:
#   RPC_URL=http://host:port ./scripts/geth-net-info.sh

RPC_URL="${RPC_URL:-http://127.0.0.1:8545}"

echo "Using RPC URL: ${RPC_URL}"
echo

payload='{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'
chain_id=$(curl -s -X POST "${RPC_URL}" -H "Content-Type: application/json" -d "${payload}" | jq -r '.result // empty')

if [ -z "${chain_id}" ]; then
  echo "Failed to fetch chainId. Is geth running and RPC enabled?"
  exit 1
fi

payload_block='{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
block_number=$(curl -s -X POST "${RPC_URL}" -H "Content-Type: application/json" -d "${payload_block}" | jq -r '.result // empty')

echo "chainId: ${chain_id}"
echo "latest block: ${block_number}"
