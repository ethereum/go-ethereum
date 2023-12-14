FROM alpine:latest

RUN apk update && apk add --no-cache git yarn expect

WORKDIR /

# Hyperlane 3.3 release
# https://github.com/hyperlane-xyz/hyperlane-monorepo/commit/8b16adee486a6b1a1f0d4fbd1b419369241813b2
RUN git clone https://github.com/hyperlane-xyz/hyperlane-monorepo.git 
WORKDIR /hyperlane-monorepo
RUN git checkout 8b16adee486a6b1a1f0d4fbd1b419369241813b2

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

