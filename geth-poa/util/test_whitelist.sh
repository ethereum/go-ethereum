#!/bin/sh

if [ -z "$1" ]; then
    echo "Usage: $0 <HYP_ERC20_ADDR>"
    exit 1
fi
HYP_ERC20_ADDR="$1"  

WHITELIST_ADDR=0x5D1415C0973034d162F5FEcF19B50dA057057e29
EXPECTED_OWNER=0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266
RPC_URL=http://localhost:8545

# Checks that contract deployed to expected address
DATA='{"jsonrpc":"2.0","method":"eth_getCode","params":["'$WHITELIST_ADDR'", "latest"],"id":1}'
RESPONSE=$(curl -s -X POST --data "$DATA" -H "Content-Type: application/json" $RPC_URL)
CODE=$(echo $RESPONSE | jq -r '.result')
if [ -z "$RESPONSE" ] || [ "$RESPONSE" == "null" ]; then
    echo "Error: No response from JSON RPC at $RPC_URL"
    exit 1
fi
if [ "$CODE" != "0x" ]; then
    echo "Contract deployed at $WHITELIST_ADDR"
else
    echo "No contract deployed at $WHITELIST_ADDR! geth and hyperlane hardcodes will not work"
    exit 0
fi

IS_WHITELISTED=$(cast call $WHITELIST_ADDR \
    "isWhitelisted(address)(bool)" $HYP_ERC20_ADDR \
    --rpc-url $RPC_URL)
if [ "$IS_WHITELISTED" == "false" ]; then
    echo "Error: HYP_ERC20_ADDR $HYP_ERC20_ADDR not whitelisted"
    exit 1
fi
echo "HYP_ERC20_ADDR $HYP_ERC20_ADDR is whitelisted"

OWNER=$(cast call $WHITELIST_ADDR \
    "owner()(address)" \
    --rpc-url $RPC_URL)
if [ "$OWNER" != "$EXPECTED_OWNER" ]; then
    echo "Error: Whitelist owner is not $EXPECTED_OWNER"
    exit 1
fi
echo "Whitelist owner is $EXPECTED_OWNER"
