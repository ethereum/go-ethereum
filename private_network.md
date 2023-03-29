# private network

1. Make a new account in Metamask (or whichever method you prefer). Copy paste the address into `genesis.json`'s `alloc` field. This account will be allocated 300 ETH at startup.

2. Replace the etherbase in the following with your account (it doesn't really matter though, since mining doesn't require signing). Then, run:
```bash
make geth
./build/bin/geth --datadir ~/.astriageth/ init genesis.json
./build/bin/geth --datadir ~/.astriageth/ --http --http.port=8545 --ws --ws.port=8545 --networkid=1337 --http.corsdomain='*' --ws.origins='*' --mine --miner.threads 1 --miner.etherbase=0xDB8c0F738639Da247bC67251548a186b2107bf4d
```

4. Open up Metamask and go to the Localhost 8545 network. You should see your account has 300 ETH. You can now transfer this to other accounts.