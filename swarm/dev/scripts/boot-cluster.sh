#!/bin/bash
#
# A script to boot a dev swarm cluster on a Linux host (typically in a Docker
# container started with swarm/dev/run.sh).
#
# The cluster contains a bootnode, a geth node and multiple swarm nodes, with
# each node having its own data directory in a base directory passed with the
# --dir flag (default is swarm/dev/cluster).
#
# To avoid using different ports for each node and to make networking more
# realistic, each node gets its own network namespace with IPs assigned from
# the 192.168.33.0/24 subnet:
#
# bootnode: 192.168.33.2
# geth:     192.168.33.3
# swarm:    192.168.33.10{1,2,...,n}

set -e

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
source "${ROOT}/swarm/dev/scripts/util.sh"

# DEFAULT_BASE_DIR is the default base directory to store node data
DEFAULT_BASE_DIR="${ROOT}/swarm/dev/cluster"

# DEFAULT_CLUSTER_SIZE is the default swarm cluster size
DEFAULT_CLUSTER_SIZE=3

# Linux bridge configuration for connecting the node network namespaces
BRIDGE_NAME="swarmbr0"
BRIDGE_IP="192.168.33.1"

# static bootnode configuration
BOOTNODE_IP="192.168.33.2"
BOOTNODE_PORT="30301"
BOOTNODE_KEY="32078f313bea771848db70745225c52c00981589ad6b5b49163f0f5ee852617d"
BOOTNODE_PUBKEY="760c4460e5336ac9bbd87952a3c7ec4363fc0a97bd31c86430806e287b437fd1b01abc6e1db640cf3106b520344af1d58b00b57823db3e1407cbc433e1b6d04d"
BOOTNODE_URL="enode://${BOOTNODE_PUBKEY}@${BOOTNODE_IP}:${BOOTNODE_PORT}"

# static geth configuration
GETH_IP="192.168.33.3"
GETH_RPC_PORT="8545"
GETH_RPC_URL="http://${GETH_IP}:${GETH_RPC_PORT}"

usage() {
  cat >&2 <<USAGE
usage: $0 [options]

Boot a dev swarm cluster.

OPTIONS:
  -d, --dir DIR     Base directory to store node data [default: ${DEFAULT_BASE_DIR}]
  -s, --size SIZE   Size of swarm cluster [default: ${DEFAULT_CLUSTER_SIZE}]
  -h, --help        Show this message
USAGE
}

main() {
  local base_dir="${DEFAULT_BASE_DIR}"
  local cluster_size="${DEFAULT_CLUSTER_SIZE}"

  parse_args "$@"

  local pid_dir="${base_dir}/pids"
  local log_dir="${base_dir}/logs"
  mkdir -p "${base_dir}" "${pid_dir}" "${log_dir}"

  stop_cluster
  create_network
  start_bootnode
  start_geth_node
  start_swarm_nodes
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
      -s | --size)
        if [[ -z "$2" ]]; then
          fail "--size flag requires an argument"
        fi
        cluster_size="$2"
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

stop_cluster() {
  info "stopping existing cluster"
  "${ROOT}/swarm/dev/scripts/stop-cluster.sh" --dir "${base_dir}"
}

# create_network creates a Linux bridge which is used to connect the node
# network namespaces together
create_network() {
  local subnet="${BRIDGE_IP}/24"

  info "creating ${subnet} network on ${BRIDGE_NAME}"
  ip link add name "${BRIDGE_NAME}" type bridge
  ip link set dev "${BRIDGE_NAME}" up
  ip address add "${subnet}" dev "${BRIDGE_NAME}"
}

# start_bootnode starts a bootnode which is used to bootstrap the geth and
# swarm nodes
start_bootnode() {
  local key_file="${base_dir}/bootnode.key"
  echo -n "${BOOTNODE_KEY}" > "${key_file}"

  local args=(
    --addr      "${BOOTNODE_IP}:${BOOTNODE_PORT}"
    --nodekey   "${key_file}"
    --verbosity "6"
  )

  start_node "bootnode" "${BOOTNODE_IP}" "$(which bootnode)" ${args[@]}
}

# start_geth_node starts a geth node with --datadir pointing at <base-dir>/geth
# and a single, unlocked account with password "geth"
start_geth_node() {
  local dir="${base_dir}/geth"
  mkdir -p "${dir}"

  local password="geth"
  echo "${password}" > "${dir}/password"

  # create an account if necessary
  if [[ ! -e "${dir}/keystore" ]]; then
    info "creating geth account"
    create_account "${dir}" "${password}"
  fi

  # get the account address
  local address="$(jq --raw-output '.address' ${dir}/keystore/*)"
  if [[ -z "${address}" ]]; then
    fail "failed to get geth account address"
  fi

  local args=(
    --datadir   "${dir}"
    --networkid "321"
    --bootnodes "${BOOTNODE_URL}"
    --unlock    "${address}"
    --password  "${dir}/password"
    --rpc
    --rpcaddr   "${GETH_IP}"
    --rpcport   "${GETH_RPC_PORT}"
    --verbosity "6"
  )

  start_node "geth" "${GETH_IP}" "$(which geth)" ${args[@]}
}

start_swarm_nodes() {
  for i in $(seq 1 ${cluster_size}); do
    start_swarm_node "${i}"
  done
}

# start_swarm_node starts a swarm node with a name like "swarmNN" (where NN is
# a zero-padded integer like "07"), --datadir pointing at <base-dir>/<name>
# (e.g. <base-dir>/swarm07) and a single account with <name> as the password
start_swarm_node() {
  local num=$1
  local name="swarm$(printf '%02d' ${num})"
  local ip="192.168.33.1$(printf '%02d' ${num})"

  local dir="${base_dir}/${name}"
  mkdir -p "${dir}"

  local password="${name}"
  echo "${password}" > "${dir}/password"

  # create an account if necessary
  if [[ ! -e "${dir}/keystore" ]]; then
    info "creating account for ${name}"
    create_account "${dir}" "${password}"
  fi

  # get the account address
  local address="$(jq --raw-output '.address' ${dir}/keystore/*)"
  if [[ -z "${address}" ]]; then
    fail "failed to get swarm account address"
  fi

  local args=(
    --bootnodes    "${BOOTNODE_URL}"
    --datadir      "${dir}"
    --identity     "${name}"
    --ens-api      "${GETH_RPC_URL}"
    --bzznetworkid "321"
    --bzzaccount   "${address}"
    --password     "${dir}/password"
    --verbosity    "6"
  )

  start_node "${name}" "${ip}" "$(which swarm)" ${args[@]}
}

# start_node runs the node command as a daemon in a network namespace
start_node() {
  local name="$1"
  local ip="$2"
  local path="$3"
  local cmd_args=${@:4}

  info "starting ${name} with IP ${ip}"

  create_node_network "${name}" "${ip}"

  # add a marker to the log file
  cat >> "${log_dir}/${name}.log" <<EOF

>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
Starting ${name} node - $(date)
>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

EOF

  # run the command in the network namespace using start-stop-daemon to
  # daemonise the process, sending all output to the log file
  local daemon_args=(
    --start
    --background
    --no-close
    --make-pidfile
    --pidfile "${pid_dir}/${name}.pid"
    --exec "${path}"
  )
  if ! ip netns exec "${name}" start-stop-daemon ${daemon_args[@]} -- $cmd_args &>> "${log_dir}/${name}.log"; then
    fail "could not start ${name}, check ${log_dir}/${name}.log"
  fi
}

# create_node_network creates a network namespace and connects it to the Linux
# bridge using a veth pair
create_node_network() {
  local name="$1"
  local ip="$2"

  # create the namespace
  ip netns add "${name}"

  # create the veth pair
  local veth0="veth${name}0"
  local veth1="veth${name}1"
  ip link add name "${veth0}" type veth peer name "${veth1}"

  # add one end to the bridge
  ip link set dev "${veth0}" master "${BRIDGE_NAME}"
  ip link set dev "${veth0}" up

  # add the other end to the namespace, rename it eth0 and give it the ip
  ip link set dev "${veth1}" netns "${name}"
  ip netns exec "${name}" ip link set dev "${veth1}" name "eth0"
  ip netns exec "${name}" ip link set dev "eth0" up
  ip netns exec "${name}" ip address add "${ip}/24" dev "eth0"
}

create_account() {
  local dir=$1
  local password=$2

  geth --datadir "${dir}" --password /dev/stdin account new <<< "${password}"
}

main "$@"
