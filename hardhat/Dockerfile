FROM node:22-alpine3.19

RUN apk add --no-cache bash

WORKDIR /usr/src/app

COPY . /usr/src/app

RUN npm uninstall && npm install --save-dev hardhat

RUN npm install wait-on

COPY .hardhat/entrypoint.sh /usr/local/bin

EXPOSE 8545

RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/bin/sh", "/usr/local/bin/entrypoint.sh"]