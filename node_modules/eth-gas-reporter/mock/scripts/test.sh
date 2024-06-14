#!/usr/bin/env bash

# Executes cleanup function at script exit.
trap cleanup EXIT

# Load helpers
cd mock
source ./scripts/integration_tests.sh
source ./scripts/install_reporter.sh
source ./scripts/launch_testrpc.sh

# -----------------------  Conditional TestRPC Launch on 8545 ---------------------------------------

if testrpc_running; then
  echo "Using existing client instance"
else
  echo "Starting our own ganache-cli instance"
  start_testrpc
fi

# Buidler is super fast on launch
sleep 5

# -----------------------  Install Reporter and run tests ------------------------------------------
install_reporter
test_truffle_v5_basic
test_truffle_v5_with_options
test_buildler_v5_plugin
