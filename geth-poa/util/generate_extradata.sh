#!/bin/bash

# See: https://geth.ethereum.org/docs/fundamentals/private-network's extradata section.

# Start with 32 bytes of zeros
extradata="0x$(printf '0%.0s' {1..64})"

for addr in "$@"; do
  # Remove '0x' from the start of the address if present
  addr="${addr#0x}"
  extradata+="$addr"
done

# End with 65 bytes of zeros
extradata+=$(printf '0%.0s' {1..130})

echo "$extradata"
