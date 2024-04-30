#!/bin/sh -e

../../build/bin/geth --datadir=../gethdata --password=../pwd account import ../secret.txt

../../build/bin/geth --datadir=../gethdata init ../genesis.json

echo "STARTING GETH"

../../build/bin/geth --http --http.port 8545 --http.api eth,net,web3 --ws --ws.api eth,net,web3 --authrpc.jwtsecret ../jwt.hex --datadir ../gethdata --nodiscover --syncmode full --allow-insecure-unlock --unlock 0x123463a4b065722e99115d6c222f267d9cabb524 --password ../pwd
