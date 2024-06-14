#!/usr/bin/env bash

# Installs latest reporter state, including added dependencies

# Copy over the package and install
# Expects to be run from within ./mock
install_reporter() {

  cp ./../package.json ./package.json
  npx yarn

  # Copy over eth-gas-reporter
  if [ ! -e node_modules/eth-gas-reporter ]; then
    mkdir node_modules/eth-gas-reporter
  fi

  cp -r ./../lib node_modules/eth-gas-reporter
  cp ./../index.js node_modules/eth-gas-reporter/index.js
  cp ./../codechecks.js node_modules/eth-gas-reporter/codechecks.js
  cp ./../package.json node_modules/eth-gas-reporter/package.json

}

