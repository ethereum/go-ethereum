rm -rf ../newcoin_data
mkdir ../newcoin_data
./build/bin/geth init --datadir ../newcoin_data/ genesis.json
./build/bin/geth  --datadir ../newcoin_data --rpc  --rpcport 8545 --mine --miner.threads=1 --etherbase=0x4d058e24aEC4d7Ce5341641931c73936F313862D console

