FROM alpine:latest

RUN apk update && apk add --no-cache git yarn expect

WORKDIR /

RUN git clone https://github.com/primevprotocol/hyperlane-monorepo.git 
WORKDIR /hyperlane-monorepo
# Hyperlane 3.5 release w/ merged primev changes
# https://github.com/primevprotocol/hyperlane-monorepo/commit/ce1fe3f05e915823ecc355e8d764f010b4b38fa9
RUN git checkout ce1fe3f05e915823ecc355e8d764f010b4b38fa9

RUN yarn install
RUN yarn build

RUN ln -s /hyperlane-monorepo/typescript/cli/dist/cli.js /usr/local/bin/hyperlane
RUN chmod +x /usr/local/bin/hyperlane

COPY chain-config.yml /chain-config.yml
COPY multisig-ism.yml /multisig-ism.yml
COPY warp-tokens.yml /warp-tokens.yml

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]

