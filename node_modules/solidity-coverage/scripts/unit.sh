#!/usr/bin/env bash

# Toggles optimizer on/off
VIAR_IR=$VIA_IR

node --max-old-space-size=4096 \
  ./node_modules/.bin/nyc \
    --exclude '**/sc_temp/**' \
    --exclude '**/test/**/' \
  -- \
  mocha test/units/* \
    --require "test/util/mochaRootHook.js" \
    --timeout 100000 \
    --no-warnings \
    --exit
