# private network

### astria
1. Make a new account in Metamask (or whichever method you prefer). Copy paste the address into `genesis.json`'s `alloc` field. This account will be allocated 300 ETH at startup.

2. To build and initialize Geth:
```bash
make geth
./build/bin/geth --datadir ~/.astriageth/ init genesis.json
```

To run without mining (ie. using the conductor):
```bash
./build/bin/geth --datadir ~/.astriageth/ --http --http.port=8545 --ws --ws.port=8545 --networkid=1337 --http.corsdomain='*' --ws.origins='*' --grpc --grpc.addr=localhost --grpc.port 50051
```

4. Open up Metamask and go to the Localhost 8545 network. You should see your account has 300 ETH. You can now transfer this to other accounts.

### ethash
To run with mining (which you don't want if running Astria):
1. Remove the `"terminalTotalDifficulty": 0,` line in `genesis.json`. Then run steps 1-2 as above.
2. Replace the etherbase in the following with your account (it doesn't really matter though, since mining doesn't require signing). Then,
```bash
./build/bin/geth --datadir ~/.astriageth/ --http --http.port=8545 --ws --ws.port=8545 --networkid=1337 --http.corsdomain='*' --ws.origins='*' --mine --miner.threads 1 --miner.etherbase=0x46B77EFDFB20979E1C29ec98DcE73e3eCbF64102 --grpc --grpc.addr=localhost --grpc.port 50051
```
