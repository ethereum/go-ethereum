To set up local xdpos you need pass env NETWORK=local and inject 2 files when starting the container
1. genesis.json - deploy to path "/work/genesis.json" in the container. 
   - Creating genesis.json using puppeth
     1. "make puppeth" from base repo directory 
     2. run the binary (genesis wizard) "./build/bin/puppeth"
     3. the output genesis.json will be in your ~/.puppeth directory

2. bootnodes.list - deploy to path "/work/bootnodes.list" in the container.
    - check example bootnode format in cicd/devnet or cicd/testnet 
    - REQUIRES newline at the end of the file, or the last line won't read
