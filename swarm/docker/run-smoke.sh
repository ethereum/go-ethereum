#!/bin/sh

set -o errexit
set -o pipefail
set -o nounset

/swarm-smoke $@ 2>&1 || true
