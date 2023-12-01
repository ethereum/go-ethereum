FROM alpine:latest

RUN apk add --no-cache git yarn jq

RUN git clone https://github.com/primevprotocol/hyperlane-ui.git
WORKDIR /hyperlane-ui
RUN git checkout ff1586ed9cad6eeddba2cebc5f9b7828c74f25d1

RUN yarn

ARG NEXT_PUBLIC_WALLET_CONNECT_ID

RUN yarn build

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]
