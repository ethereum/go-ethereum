#!/usr/bin/env bash

cleanup() {
  # Kill the testrpc instance that we started (if we started one and if it's still running).
  if [ -n "$testrpc_pid" ] && ps -p $testrpc_pid > /dev/null; then
    kill -9 $testrpc_pid
  fi
}

testrpc_port=8545

testrpc_running() {
  nc -z localhost "$testrpc_port"
}

start_testrpc() {
  npx ganache-cli --gasLimit 8000000 "${accounts[@]}" > /dev/null &
  testrpc_pid=$!
}
