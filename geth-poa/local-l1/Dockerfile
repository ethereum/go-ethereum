# Local L1 nodes are NOT built from source
FROM ethereum/client-go:v1.13.4 as builder

RUN apk add --no-cache jq

# Reuse entrypoint for mev-commit chain
COPY ./entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY ./local-l1/genesis.json /genesis.json

ENTRYPOINT ["/bin/sh", "/entrypoint.sh"]
