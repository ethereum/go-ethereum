# Tomochain

[![Build Status](https://travis-ci.org/tomochain/tomochain.svg?branch=master)](https://travis-ci.org/tomochain/tomochain) [![Join the chat at https://gitter.im/tomochain/tomochain](https://badges.gitter.im/tomochain/tomochain.svg)](https://gitter.im/tomochain/tomochain?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

## About Tomochain

TomoChain is an innovative solution to the scalability problem with the Ethereum blockchain.
Our mission is to be a leading force in building the Internet of Value, and its infrastructure.
We are working to create an alternative, scalable financial system which is more secure, transparent, efficient, inclusive and equitable for everyone.

TomoChain relies on a system of 150 Masternodes with Proof of Stake Voting consensus that can support near-zero fee, and 2-second transaction confirmation time.
Security, stability and chain finality are guaranteed via novel techniques such as double validation, staking via smart-contracts and "true" randomization processes.

Tomochain supports all EVM-compatible smart-contracts, protocols, and atomic cross-chain token transfers.
New scaling techniques such as sharding, private-chain generation, hardware integration will be continuously researched and incorporated into Tomochain's masternode architecture which will be an ideal scalable smart-contract public blockchain for decentralized apps, token issuances and token integrations for small and big businesses.

More details can be found at our [technical white paper](https://tomochain.com/docs/technical-whitepaper---1.0.pdf)

Reading more about us on:

- our website: http://tomochain.com
- our blogs and announcements: https://medium.com/tomochain
- our documentation site: https://docs.tomochain.com

## Tomochain vs Giants

Tomochain is built by the mindset of standing on the giants shoulder.
We have learned from all advanced technical design concept of many well-known public blockchains on the market and shaped up the platform with our own ingredients.
See below the overall technical comparison table that we try to make clear the position of Tomochain comparing to some popular blockchains at the top-tier.

![Tomochain](https://cdn-images-1.medium.com/max/1600/1*LkiIWFHPXh-0Whv3Hm1yMQ.png)

**We just updated the number of masternodes accepted in the network to a maximum of 150*

## Building the source

Tomochain provides client binary called `tomo` for both running a masternode and running a full-node.
Building `tomo` requires both a Go (1.7+) and a C compiler.
Install them by your own way. Once the dependencies are installed, just run below commands:

```bash
$ git clone https://github.com/tomochain/tomochain tomochain
$ cd tomochain
$ make tomo
```

Alternatively, you could quickly download pre-complied binary on our [github release page](https://github.com/tomochain/tomochain/releases)

## Running tomo

This section explains how to run the tomo binary.
We also offer an official Docker image and a quick startup cli if your goal is to run a masternode.
Please refer the [official documentation](https://docs.tomochain.com/get-started/masternode) on how to become a masternode for more information.

### Attaching to the Tomochain test network

We published our test network 2.0 with full implementation of PoSV consensus at https://stats.testnet.tomochain.com.
If you'd like to experiment with smart contracts creation and DApps, you might be interested in giving it a try on our test network.

In order to connect to one of the masternodes on the test network, just run this below command:

```bash
$ tomo attach https://testnet.tomochain.com
```

### Running a full node

If you would like to run your own full node, you can try it on the test network by running the commands below:

```bash
// 1. create a folder to store tomochain data on your machine
$ export DATA_DIR=/path/to/your/data/folder
$ mkdir -p $DATA_DIR/tomo

// 2. download our genesis file
$ export GENESIS_PATH=$DATA_DIR/genesis.json
$ curl -L https://raw.githubusercontent.com/tomochain/tomochain/master/genesis/testnet.json -o $GENESIS_PATH

// 3. init the chain from genesis
$ tomo init $GENESIS_PATH --datadir $DATA_DIR

// 4. get a test account. Create a new one if you don't have any:
$ export KEYSTORE_DIR=keystore
$ touch $DATA_DIR/password && echo 'your-password' > $DATA_DIR/password
$ tomo account new \
      --datadir $DATA_DIR \
      --keystore $KEYSTORE_DIR \
      --password $DATA_DIR/password

// if you already have a test account, import it now
$ tomo  account import ./private_key \
      --datadir $DATA_DIR \
      --keystore $KEYSTORE_DIR \
      --password $DATA_DIR/password

// get the account
$ account=$(
  tomo account list --datadir $DATA_DIR  --keystore $KEYSTORE_DIR \
  2> /dev/null \
  | head -n 1 \
  | cut -d"{" -f 2 | cut -d"}" -f 1
)

// 5. prepare the bootnode and stats information
$ export BOOTNODES="enode://4d3c2cc0ce7135c1778c6f1cfda623ab44b4b6db55289543d48ecfde7d7111fd420c42174a9f2fea511a04cf6eac4ec69b4456bfaaae0e5bd236107d3172b013@52.221.28.223:30301,enode://298780104303fcdb37a84c5702ebd9ec660971629f68a933fd91f7350c54eea0e294b0857f1fd2e8dba2869fcc36b83e6de553c386cf4ff26f19672955d9f312@13.251.101.216:30301,enode://46dba3a8721c589bede3c134d755eb1a38ae7c5a4c69249b8317c55adc8d46a369f98b06514ecec4b4ff150712085176818d18f59a9e6311a52dbe68cff5b2ae@13.250.94.232:30301"
$ export param="--unlock $account --bootnodes $BOOTNODES --ethstats sun:anna-coal-flee-carrie-zip-hhhh-tarry-laue-felon-rhine@stats.testnet.tomochain.com:443"

// 6. Start up your full node now
$ export NAME=YOUR_FULLNODE_NAME
$ tomo $params \
  --verbosity 4 \
  --datadir $DATA_DIR \
  --keystore $KEYSTORE_DIR \
  --identity $NAME \
  --password $DATA_DIR \
  --networkid 89 \
  --port 30303 \
  --rpc \
  --rpccorsdomain "*" \
  --rpcaddr 0.0.0.0 \
  --rpcport 8545 \
  --rpcvhosts "*" \
  --ws \
  --wsaddr 0.0.0.0 \
  --wsport 8546 \
  --wsorigins "*" \
  --mine \
  --gasprice "1" \
  --targetgaslimit "420000000"
```

*Some explanations on the flags*

```
--verbosity: log level from 1 to 5. Here we're using 4 for debug messages
--datadir: path to your data directory created above.
--keystore: path to your account's keystore created above.
--identity: your full-node's name.
--password: your account's password.
--networkid: our testnet network ID.
--port: your full-node's listening port (default to 30303)
--rpc, --rpccorsdomain, --rpcaddr, --rpcport, --rpcvhosts: your full-node will accept RPC requests at 8545 TCP.
--ws, --wsaddr, --wsport, --wsorigins: your full-node will accept Websocket requests at 8546 TCP.
--mine: your full-node wants to register to be a candidate for masternode selection.
--gasprice: Minimal gas price to accept for mining a transaction.
--targetgaslimit: Target gas limit sets the artificial target gas floor for the blocks to mine (default: 4712388)
```

## Road map

The implementation of the following features is being studied by our research team:

- Layer 2 scalability with state sharding
- Asynchronize EVM execution
- Multi-chains interoperabilty
- Spam filtering
- DEX integration

## Contribution and technical discuss

Thank you for considering to try out our network and/or help out with the source code.
We would love to get your help, feel free to lend a hand.
Even the smallest bit of code, bug reporting or just discussing ideas are highly appreciated.

If you would like to contribute to the tomochain source code, please refer to our Developer Guide for details on configuring development environment, managing dependencies, compiling, testing and submitting your code changes to our repo.

Please also make sure your contributions adhere to the base coding guidelines:

- Code must adhere the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e uses [gofmt](https://golang.org/cmd/gofmt/)).
- Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
- Pull requests need to be based on and opened against the `master` branch.
- Problem you are trying to contribute must be well-explained as an issue on our [github issue page](https://github.com/tomochain/tomochain/issues)
- Commit messages should be short but clear enough and should refer to the corresponding pre-logged issue mentioned above.

For technical discussion, feel free to join our chat at [Gitter](https://gitter.im/tomochain/tomochain).
