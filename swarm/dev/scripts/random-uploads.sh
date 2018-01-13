#!/bin/bash
#
# A script to upload random data to a swarm cluster.
#
# Example:
#
#   random-uploads.sh --addr 192.168.33.101:8500 --size 40k --count 1000

set -e

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
source "${ROOT}/swarm/dev/scripts/util.sh"

DEFAULT_ADDR="localhost:8500"
DEFAULT_UPLOAD_SIZE="40k"
DEFAULT_UPLOAD_COUNT="1000"

usage() {
  cat >&2 <<USAGE
usage: $0 [options]

Upload random data to a Swarm cluster.

OPTIONS:
  -a, --addr ADDR     Swarm API address      [default: ${DEFAULT_ADDR}]
  -s, --size SIZE     Individual upload size [default: ${DEFAULT_UPLOAD_SIZE}]
  -c, --count COUNT   Number of uploads      [default: ${DEFAULT_UPLOAD_COUNT}]
  -h, --help          Show this message
USAGE
}

main() {
  local addr="${DEFAULT_ADDR}"
  local upload_size="${DEFAULT_UPLOAD_SIZE}"
  local upload_count="${DEFAULT_UPLOAD_COUNT}"

  parse_args "$@"

  info "uploading ${upload_count} ${upload_size} random files to ${addr}"

  for i in $(seq 1 ${upload_count}); do
    info "upload ${i} / ${upload_count}:"
    do_random_upload
    echo
  done
}

do_random_upload() {
  curl -fsSL -X POST --data-binary "$(random_data)" "http://${addr}/bzz-raw:/"
}

random_data() {
  dd if=/dev/urandom of=/dev/stdout bs="${upload_size}" count=1 2>/dev/null
}

parse_args() {
  while true; do
    case "$1" in
      -h | --help)
        usage
        exit 0
        ;;
      -a | --addr)
        if [[ -z "$2" ]]; then
          fail "--addr flag requires an argument"
        fi
        addr="$2"
        shift 2
        ;;
      -s | --size)
        if [[ -z "$2" ]]; then
          fail "--size flag requires an argument"
        fi
        upload_size="$2"
        shift 2
        ;;
      -c | --count)
        if [[ -z "$2" ]]; then
          fail "--count flag requires an argument"
        fi
        upload_count="$2"
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

main "$@"
