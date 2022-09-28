#!/bin/bash

# Note: this script is meant to be run in a Debian/Ubuntu docker container, as user 'root'.

set -e -x

go run build/ci.go install -dlgo
go run build/ci.go archive -type tar -signer BUILD_LINUX_SIGNING_KEY

go run build/ci.go install -dlgo -arch 386
go run build/ci.go archive -arch 386 -type tar -signer BUILD_LINUX_SIGNING_KEY
