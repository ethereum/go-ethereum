FROM node:lts-alpine

RUN apk add --no-cache bash

WORKDIR /usr/app

COPY package*.json ./

RUN npm install

COPY . .

EXPOSE 8545

CMD ["npx", "hardhat", "ignition", "deploy", "/usr/app/ignition/modules/Apollo.js"]