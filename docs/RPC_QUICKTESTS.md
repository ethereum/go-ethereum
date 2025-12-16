# RPC Quick Tests

This document lists a few simple JSON-RPC calls that can be used to
verify that a geth node is responding correctly.

All examples use `curl` against a local HTTP endpoint:

    http://127.0.0.1:8545

## Get the current block number

    curl -X POST http://127.0.0.1:8545 \
      -H "Content-Type: application/json" \
      -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

## Get the node's client version

    curl -X POST http://127.0.0.1:8545 \
      -H "Content-Type: application/json" \
      -d '{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}'

## Get the peer count

    curl -X POST http://127.0.0.1:8545 \
      -H "Content-Type: application/json" \
      -d '{"jsonrpc":"2.0","method":"net_peerCount","params":[],"id":1}'

If these calls return valid JSON responses, the node is up and the
HTTP JSON-RPC endpoint is reachable. If they fail, check:

- that `--http` is enabled,
- that the correct `--http.addr` and `--http.port` are in use,
- any firewall rules or container port mappings.
