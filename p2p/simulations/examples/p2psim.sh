#!/bin/bash
#
# Start a network simulation using the API started by connectivity.go

set -e

main() {
  if ! which p2psim &>/dev/null; then
    fail "missing p2psim binary (you need to build p2p/simulations/cmd/p2psim)"
  fi

  info "creating the example network"
  p2psim network create --config '{"id": "example", "default_service": "ping-pong"}'

  info "creating 10 nodes"
  export P2PSIM_NETWORK="example"
  for i in $(seq 1 10); do
    p2psim node create
    p2psim node start "$(node_name $i)"
  done

  info "connecting node01 to all other nodes"
  for i in $(seq 2 10); do
    p2psim node connect "node01" "$(node_name $i)"
  done

  info "done"
}

node_name() {
  local num=$1
  echo "node$(printf '%02d' $num)"
}

info() {
  echo -e "\033[1;32m---> $(date +%H:%M:%S) ${@}\033[0m"
}

fail() {
  echo -e "\033[1;31mERROR: ${@}\033[0m" >&2
  exit 1
}

main "$@"
