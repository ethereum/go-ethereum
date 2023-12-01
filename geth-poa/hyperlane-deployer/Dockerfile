FROM alpine:latest

RUN apk update && apk add --no-cache git yarn expect

WORKDIR /
RUN git clone https://github.com/primevprotocol/hyperlane-monorepo.git 
WORKDIR /hyperlane-monorepo
# RUN git checkout v3-audit-remediations
RUN git checkout ca301c80861701e17b13b9a06454795b317696b2
# TODO: checkout stable version, seemed like v3-audit-remediations changed build process
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

