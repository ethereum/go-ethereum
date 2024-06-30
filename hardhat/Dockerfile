FROM node:22-alpine3.19

RUN apk add --no-cache bash

WORKDIR /usr/src/app

COPY . /usr/src/app

RUN npm install
RUN npm install wait-on

COPY go-ethereum/hardhat/entrypoint.sh /usr/local/bin

EXPOSE 8545

RUN chmod +x go-ethereum/hardhat/entrypoint.sh/entrypoint.sh

ENTRYPOINT ["/bin/sh", "/usr/local/bin/entrypoint.sh"]