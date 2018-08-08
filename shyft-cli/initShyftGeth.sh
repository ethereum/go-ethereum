#!/bin/sh
if [[ -z "${DBENV}" ]]; then
  ./build/bin/geth --identity "ShyftTestnetNode" --keystore ./ --datadir "./shyftData" init ShyftNetwork.json
else
  /bin/geth --identity "ShyftTestnetNode" --keystore ./ --datadir "./shyftData" init ShyftNetwork.json
fi
