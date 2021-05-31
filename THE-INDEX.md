# the-index geth

## Build

1. brew install go (1.15+)

2. git clone into `$GOPATH/src/github.com/orbs-network/the-index-go-ethereum`

3. go into the repo directory 

4. run `make geth`

## Develop

1. go into the repo directory

2. make sure `./the-index` directory for index outputs is created

3. to delete old index data run `rm -rf ./the-index/*.rlp`

4. to reset the chain run `./build/bin/geth --rinkeby removedb`

5. run `export THEINDEX_PATH=./the-index/; ./build/bin/geth --rinkeby --syncmode=full --port 0`

## Run

1. go into the repo directory

2. choose a directory for index output and make sure it is created, for example `./the-index`

3. run `export THEINDEX_PATH=./the-index/; ./build/bin/geth --cache=4096 --maxpeers=50 --syncmode=full`

4. chaindata is in `~/Library/Ethereum/geth/chaindata`