#!/bin/sh
set -exu

# Update src tokens.ts file with hypNativeAddress value from deploy config
ARTIFACT_PATH="/deploy-artifacts/warp-ui-token-config.json"
SRC_TOKENS_PATH="/hyperlane-ui/src/consts/tokens.ts"
HYP_NATIVE_ADDR=$(jq -r '.hypNativeAddress' $ARTIFACT_PATH)
sed -i "/hypNativeAddress/c\    hypNativeAddress: \"$HYP_NATIVE_ADDR\"," $SRC_TOKENS_PATH

# Update src chains.ts file with L2 node url depending on dev vs prod 
SRC_CHAINS_PATH="/hyperlane-ui/src/consts/chains.ts"
sed -i "s|http: 'http://[^']*'|http: '$SETTLEMENT_RPC_URL'|g" "$SRC_CHAINS_PATH"

exec yarn dev
