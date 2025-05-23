#!/bin/bash
# This script demonstrates the behavior of the prestateTracer
# when accessing account balance with and without disableCode option

# Set up variables
# Replace these values with actual addresses when deploying
CONTRACT_ADDR="0xd343fdd530afc898c23f5d0db2d9849b71303425"
OTHER_CONTRACT_ADDR="0x7d161ee7becca09e22ebf5fc22a17eecceded6b5"

# Generate calldata for getExternalBalance function
DATA=$(cast calldata "getExternalBalance(address)" $OTHER_CONTRACT_ADDR)

echo "Testing prestateTracer without disableCode (default):"
cast rpc debug_traceCall \
  '{"to":"'"$CONTRACT_ADDR"'","data":"'"$DATA"'"}' \
  latest \
  '{"tracer":"prestateTracer", "tracerConfig": { "diffMode": false } }' --rpc-url http://localhost:8546 | jq

echo -e "\nTesting prestateTracer with disableCode=true:"
cast rpc debug_traceCall \
  '{"to":"'"$CONTRACT_ADDR"'","data":"'"$DATA"'"}' \
  latest \
  '{"tracer":"prestateTracer", "tracerConfig": { "diffMode": false, "disableCode": true } }' --rpc-url http://localhost:8546 | jq
