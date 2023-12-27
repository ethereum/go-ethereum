ARG AGENT_BASE_IMAGE
ARG SETTLEMENT_RPC_URL
FROM ${AGENT_BASE_IMAGE}

# Validator config, see: 
# - https://docs.hyperlane.xyz/docs/operators/validators/setup
# - https://docs.hyperlane.xyz/docs/operators/agent-configuration#config-layers

ENV CONFIG_FILES="/deploy-artifacts/agent-config.json"

ENV HYP_BASE_REORGPERIOD=20
ENV HYP_BASE_ORIGINCHAINNAME="mevcommitsettlement"
ENV HYP_BASE_CHAINS_MEVCOMMITSETTLEMENT_CONNECTION_URL=${SETTLEMENT_RPC_URL}
ENV HYP_BASE_DB="/val-db"
ENV HYP_CHECKPOINTSYNCER_TYPE="localStorage"
ENV HYP_ORIGINCHAINNAME="mevcommitsettlement"
