# Docker container spec for building the swarm branch of go-ethereum.
#
# The build process it potentially longer running but every effort was made to
# produce a very minimalistic container that can be reused many times without
# needing to constantly rebuild.
FROM alpine:3.3

# Build go-ethereum on the fly and delete all build tools afterwards
RUN \
  apk add --update go git make gcc musl-dev            && \
  git clone https://github.com/ethersphere/go-ethereum && \
  (cd go-ethereum && git checkout s/sworm-rc3-update)  && \
  (cd go-ethereum && make geth)                        && \
  cp go-ethereum/build/bin/geth /geth                  && \
  apk del go git make gcc musl-dev                     && \
  rm -rf /go-ethereum && rm -rf /var/cache/apk/*

# Make sure bash and jq is available for easier wrapper implementation
RUN apk add --update bash jq

# Inject the startup script
ADD swarm.sh /swarm.sh
RUN chmod +x /swarm.sh

# Export the usual networking ports to allow outside access to the node
EXPOSE 8545 8546 30303

ENTRYPOINT ["/swarm.sh"]
