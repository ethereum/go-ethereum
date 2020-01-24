Running the Optimism Execution Manager Node:

1. Build geth

       $ go build

2. Initialize the node

       $ ./build/bin/geth init etc/optimism.json

3. Import the signers private key (ask in Slack for the private key file):

        $ ./build/bin/geth account import private_key.hex

4. Deploy the Execution Manager (in [packages/ovm](https://github.com/op-optimism/optimistic-rollup/tree/master/packages/ovm))

        $ yarn run deploy:execution-manager local

5. Run geth

        $ EXECUTION_MANAGER_ADDRESS=0xEB1Be3E5Ff32bd47D9589f3f1E73B1788F36639c ./build/bin/geth --datadir data  --syncmode 'full' --rpc --rpcaddr 'localhost'  --rpcapi 'eth,net' --networkid 12 --allow-insecure-unlock --gasprice '1' -unlock '0xAb521188aA30ccc4a88Ec9ea6BC55541b72eD1d3' --mine
