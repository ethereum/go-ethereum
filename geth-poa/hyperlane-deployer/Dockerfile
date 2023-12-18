FROM alpine:latest

RUN apk update && apk add --no-cache git yarn expect

WORKDIR /

# Hyperlane 3.3 release w/ https://github.com/hyperlane-xyz/hyperlane-monorepo/pull/3063 cherry-picked in.
# See https://github.com/hyperlane-xyz/hyperlane-monorepo/compare/%40hyperlane-xyz/sdk%403.3.0...primevprotocol:hyperlane-monorepo:v3.3
RUN git clone https://github.com/primevprotocol/hyperlane-monorepo.git 
WORKDIR /hyperlane-monorepo
RUN git checkout b1c4064272af7f3382391e9f7c4ce67ac1b84976

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

