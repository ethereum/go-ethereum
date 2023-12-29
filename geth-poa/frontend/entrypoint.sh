#!/bin/sh
set -exu

# Update src tokens.ts file with hypNativeAddress value from deploy config
ARTIFACT_PATH="/deploy-artifacts/warp-ui-token-config.json"
SRC_TOKENS_PATH="/hyperlane-ui/src/consts/tokens.ts"
HYP_NATIVE_ADDR=$(jq -r '.hypNativeAddress' $ARTIFACT_PATH)
sed -i "/hypNativeAddress/c\    hypNativeAddress: \"$HYP_NATIVE_ADDR\"," $SRC_TOKENS_PATH

# Check if PUBLIC_SETTLEMENT_RPC_URL is set and non-empty, otherwise use SETTLEMENT_RPC_URL
if [ -n "${PUBLIC_SETTLEMENT_RPC_URL+x}" ] && [ -n "$PUBLIC_SETTLEMENT_RPC_URL" ]; then
    RPC_URL="$PUBLIC_SETTLEMENT_RPC_URL"
else
    RPC_URL="$SETTLEMENT_RPC_URL"
fi
# Update src chains.ts file with L2 node url depending on dev vs prod 
SRC_CHAINS_PATH="/hyperlane-ui/src/consts/chains.ts"
sed -i "s|http: 'http://[^']*'|http: '$RPC_URL'|g" "$SRC_CHAINS_PATH"

exec yarn dev
