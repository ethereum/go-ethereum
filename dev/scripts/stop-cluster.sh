#!/bin/bash
#
# A script to shutdown a dev swarm cluster.

set -e

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
source "${ROOT}/swarm/dev/scripts/util.sh"

DEFAULT_BASE_DIR="${ROOT}/swarm/dev/cluster"

usage() {
  cat >&2 <<USAGE
usage: $0 [options]

Shutdown a dev swarm cluster.

OPTIONS:
  -d, --dir DIR     Base directory [default: ${DEFAULT_BASE_DIR}]
  -h, --help        Show this message
USAGE
}

main() {
  local base_dir="${DEFAULT_BASE_DIR}"

  parse_args "$@"

  local pid_dir="${base_dir}/pids"

  stop_swarm_nodes
  stop_node "geth"
  stop_node "bootnode"
  delete_network
}

parse_args() {
  while true; do
    case "$1" in
      -h | --help)
        usage
        exit 0
        ;;
      -d | --dir)
        if [[ -z "$2" ]]; then
          fail "--dir flag requires an argument"
        fi
        base_dir="$2"
        shift 2
        ;;
      *)
        break
        ;;
    esac
  done

  if [[ $# -ne 0 ]]; then
    usage
    fail "ERROR: invalid arguments: $@"
  fi
}

stop_swarm_nodes() {
  for name in $(ls "${pid_dir}" | grep -oP 'swarm\d+'); do
    stop_node "${name}"
  done
}

stop_node() {
  local name=$1
  local pid_file="${pid_dir}/${name}.pid"

  if [[ -e "${pid_file}" ]]; then
    info "stopping ${name}"
    start-stop-daemon \
      --stop \
      --pidfile "${pid_file}" \
      --remove-pidfile \
      --oknodo \
      --retry 15
  fi

  if ip netns list | grep -qF "${name}"; then
    ip netns delete "${name}"
  fi

  if ip link show "veth${name}0" &>/dev/null; then
    ip link delete dev "veth${name}0"
  fi
}

delete_network() {
  if ip link show "swarmbr0" &>/dev/null; then
    ip link delete dev "swarmbr0"
  fi
}

main "$@"
