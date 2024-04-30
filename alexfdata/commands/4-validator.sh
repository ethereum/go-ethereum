#!/bin/sh -e

rm -rf ../validatordata

../validator --datadir ../validatordata --accept-terms-of-use --interop-num-validators 64 --chain-config-file ../config.yml
