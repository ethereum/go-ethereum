# Eth Protocol Test Suite

To run the eth protocol test suite, a geth node must be initialized as described below

Geth Node:
1. initialize the geth node with the `genesis.json` file contained in the `testdata` directory
2. import the `halfchain.rlp` file in the `testdata` directory
3. run geth with the following flags: `--datadir <datadir> --nodiscover --nat=none --networkid 19763 --verbosity 5`

Eth-Test:

1. build devp2p: `go build ./cmd/devp2p/`
2. run `./devp2p rlpx eth-test <enode ID> cmd/devp2p/internal/ethtest/testdata/fullchain.rlp cmd/devp2p/internal/ethtest/testdata/genesis.json`


