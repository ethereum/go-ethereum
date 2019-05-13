#!/usr/bin/env bash
#
# A script to build and run the Swarm development environment using Docker.

set -e

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

# DEFAULT_NAME is the default name for the Docker image and container
DEFAULT_NAME="swarm-dev"

usage() {
  cat >&2 <<USAGE
usage: $0 [options]

Build and run the Swarm development environment.

Depends on Docker being installed locally.

OPTIONS:
  -n, --name NAME          Docker image and container name [default: ${DEFAULT_NAME}]
  -d, --docker-args ARGS   Custom args to pass to 'docker run' (e.g. '-p 8000:8000' to expose a port)
  -h, --help               Show this message
USAGE
}

main() {
  local name="${DEFAULT_NAME}"
  local docker_args=""
  parse_args "$@"
  build_image
  run_image
}

parse_args() {
  while true; do
    case "$1" in
      -h | --help)
        usage
        exit 0
        ;;
      -n | --name)
        if [[ -z "$2" ]]; then
          echo "ERROR: --name flag requires an argument" >&2
          exit 1
        fi
        name="$2"
        shift 2
        ;;
      -d | --docker-args)
        if [[ -z "$2" ]]; then
          echo "ERROR: --docker-args flag requires an argument" >&2
          exit 1
        fi
        docker_args="$2"
        shift 2
        ;;
      *)
        break
        ;;
    esac
  done

  if [[ $# -ne 0 ]]; then
    usage
    echo "ERROR: invalid arguments" >&2
    exit 1
  fi
}

build_image() {
  docker build --tag "${name}" "${ROOT}/swarm/dev"
}

run_image() {
  exec docker run \
    --privileged \
    --interactive \
    --tty \
    --rm \
    --hostname "${name}" \
    --name     "${name}" \
    --volume   "${ROOT}:/go/src/github.com/ethereum/go-ethereum" \
    --volume   "/var/run/docker.sock:/var/run/docker.sock" \
    ${docker_args} \
    "${name}" \
    /bin/bash
}

main "$@"
