FROM alpine:latest

RUN apk add --no-cache git yarn jq

RUN git clone https://github.com/primevprotocol/hyperlane-ui.git
WORKDIR /hyperlane-ui
RUN git checkout 48ae5056012ae579897acca35535f35adf53fdda

RUN yarn

ARG NEXT_PUBLIC_WALLET_CONNECT_ID

RUN yarn build

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]
