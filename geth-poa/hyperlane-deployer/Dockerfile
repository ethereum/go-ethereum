FROM alpine:latest

RUN apk update && apk add --no-cache git yarn expect

WORKDIR /

# Hyperlane 3.5 release w/ current primev changes in main
RUN git clone https://github.com/primevprotocol/hyperlane-monorepo.git 
WORKDIR /hyperlane-monorepo
RUN git checkout 4f73750291e9afa249ffa4d2fe34227cc9f275a8

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

