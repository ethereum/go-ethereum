#!/bin/bash

set -e -x

# Note: this script is meant to be run in a Debian/Ubuntu docker container, # as user 'root'.

go run build/ci.go debsrc -sftp-user geth-ci -signer "Go Ethereum Linux Builder <geth-ci@ethereum.org>"
