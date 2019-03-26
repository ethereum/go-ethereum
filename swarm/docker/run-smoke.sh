#!/bin/sh

set -o errexit
set -o pipefail
set -o nounset

$@ || true
