ARG AGENT_BASE_IMAGE
ARG SETTLEMENT_RPC_URL
FROM ${AGENT_BASE_IMAGE}

# Relayer config, see: 
# - https://docs.hyperlane.xyz/docs/operators/relayers/setup
# - https://docs.hyperlane.xyz/docs/operators/agent-configuration#config-layers

ENV CONFIG_FILES="/deploy-artifacts/agent-config.json"

# TODO: enforce minimum gas payment in prod

CMD ./relayer \
    --relayChains mevcommitsettlement,sepolia \
    --chains.mevcommitsettlement.connection.url ${SETTLEMENT_RPC_URL} \
    --chains.sepolia.connection.url "https://eth-sepolia.g.alchemy.com/v2/a0wg_g1X-Wz4IeVA-0SS3PsQhbyQNjc_" \
    --db /relayer-db \
    --defaultSigner.key $RELAYER_PRIVATE_KEY \
    --allowLocalCheckpointSyncers true \
    --gasPaymentEnforcement '[{"type": "none"}]'
