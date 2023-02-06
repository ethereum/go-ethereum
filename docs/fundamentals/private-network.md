---
title: Private Networks
description: Tutorial on setting up private Ethereum networks
---

This guide explains how to set up a private network of multiple Geth nodes. An Ethereum network is private if the nodes are not connected to the main network. In this context private only means reserved or isolated, rather than protected or secure. A fully controlled, private Ethereum network is useful as a backend for core developers working on issues relating to networking/blockchain syncing etc. Private networks are also useful for Dapp developers testing multi-block and multi-user scenarios.

## Prerequisites {#prerequisites}

To follow the tutorial on this page it is necessary to have a working Geth installation (instructions [here](/docs/getting-started/installing-geth)). It is also helpful to understand Geth fundamentals (see [Getting Started](/docs/getting-started)).

## Private Networks {#private-networks}

A private network is composed of multiple Ethereum nodes that can only connect to each other. In order to run multiple nodes locally, each one requires a separate data directory (`--datadir`). The nodes must also know about each other and be able to exchange information, share an initial state and a common consensus algorithm. The remainder of this page will explain how to configure Geth so that these basic requirements are met, enabling a private network to be started.

### Choosing A Network ID {#choosing-network-id}

Ethereum Mainnet has Network ID = 1. There are also many other networks that Geth can connect to by providing alternative Chain IDs, some are testnets and others are alternative networks built from forks of the Geth source code. Providing a network ID that is not already being used by an existing network or testnet means the nodes using that network ID can only connect to each other, creating a private network. A list of current network IDs is available at [Chainlist.org](https://chainlist.org/). The network ID is controlled using the `networkid` flag, e.g.

```sh
geth --networkid 12345
```

### Choosing A Consensus Algorithm {#choosing-a-consensus-mechanism}

While the main network uses proof-of-stake (PoS) to secure the blockchain, Geth also supports the the 'Clique' proof-of-authority (PoA) consensus algorithm and the Ethash proof-of-work algorithm as alternatives for private networks. Clique is strongly recommended for private testnets because PoA is far less resource-intensive than PoW. The key differences between the consensus algorithms available in Geth are:

#### Ethash {#ethash}

Geth's PoW algorithm, [Ethash](https://ethereum.org/en/developers/docs/consensus-mechanisms/pow/mining-algorithms/ethash), is a system that allows open participation by anyone willing to dedicate resources to mining. While this is a critical property for a public network, the overall security of the blockchain strictly depends on the total amount of resources used to secure it. As such, PoW is a poor choice for private networks with few miners. The Ethash mining 'difficulty' is adjusted automatically so that new blocks are created approximately 12 seconds apart. As more mining resources are deployed on the network, creating a new block becomes harder so that the average block time matches the target block time.

#### Clique {#clique}

Clique consensus is a PoA system where new blocks can be created by authorized 'signers' only. The clique consenus protocol is specified in [EIP-225](https://eips.ethereum.org/EIPS/eip-225). The initial set of authorized signers is configured in the genesis block. Signers can be authorized and de-authorized using a voting mechanism, thus allowing the set of signers to change while the blockchain operates. Clique can be configured to target any block time (within reasonable limits) since it isn't tied to the difficulty adjustment.

### Creating The Genesis Block {#creating-genesis-block}

Every blockchain starts with a genesis block. When Geth is run with default settings for the first time, it commits the Mainnet genesis to the database. For a private network, it is generally preferable to use a different genesis block. The genesis block is configured using a _genesis.json_ file whose path must be provided to Geth on start-up. When creating a genesis block, a few initial parameters for the private blockchain must be defined:

- Ethereum platform features enabled at launch (`config`). Enabling and disabling features once the blockchain is running requires scheduling a [hard fork](https://ethereum.org/en/glossary/#hard-fork).

- Initial block gas limit (`gasLimit`). This impacts how much EVM computation can happen within a single block. Mirroring the main Ethereum network is generally a [good choice](https://etherscan.io/chart/gaslimit). The block gas limit can be adjusted after launch using the `--miner.gastarget` command-line flag.

- Initial allocation of ether (`alloc`). This determines how much ether is available to the addresses listed in the genesis block. Additional ether can be created through mining as the chain progresses.

#### Clique Example {#clique-example}

Below is an example of a `genesis.json` file for a PoA network. The `config` section ensures that all known protocol changes are available and configures the 'clique' engine to be used for consensus. Note that the initial signer set must be configured through the `extradata` field. This field is required for Clique to work.

The signer account keys can be generated using the [geth account](/docs/fundamentals/account-management) command (this command can be run multiple times to create more than one signer key).

```sh
geth account new --datadir data
```

The Ethereum address printed by this command should be recorded. To encode the signer addresses in `extradata`, concatenate 32 zero bytes, all signer addresses and 65 further zero bytes. The result of this concatenation is then used as the value accompanying the `extradata` key in `genesis.json`. In the example below, `extradata` contains a single initial signer address, `0x7df9a875a174b3bc565e6424a0050ebc1b2d1d82`.

The `period` configuration option sets the target block time of the chain.

```json
{
  "config": {
    "chainId": 12345,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
    "istanbulBlock": 0,
    "berlinBlock": 0,
    "clique": {
      "period": 5,
      "epoch": 30000
    }
  },
  "difficulty": "1",
  "gasLimit": "8000000",
  "extradata": "0x00000000000000000000000000000000000000000000000000000000000000007df9a875a174b3bc565e6424a0050ebc1b2d1d820000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
  "alloc": {
    "7df9a875a174b3bc565e6424a0050ebc1b2d1d82": { "balance": "300000" },
    "f41c74c9ae680c1aa78f42e5647a62f353b7bdde": { "balance": "400000" }
  }
}
```

#### Ethash Example {#ethash-example}

Since Ethash is the default consensus algorithm, no additional parameters need to be configured in order to use it. The initial mining difficulty is influenced using the `difficulty` parameter, but note that the difficulty adjustment algorithm will quickly adapt to the amount of mining resources deployed on the chain.

```json
{
  "config": {
    "chainId": 12345,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
    "istanbulBlock": 0,
    "berlinBlock": 0,
    "ethash": {}
  },
  "difficulty": "1",
  "gasLimit": "8000000",
  "alloc": {
    "7df9a875a174b3bc565e6424a0050ebc1b2d1d82": { "balance": "300000" },
    "f41c74c9ae680c1aa78f42e5647a62f353b7bdde": { "balance": "400000" }
  }
}
```

### Initializing the Geth Database {#initializing-geth-database}

To create a blockchain node that uses this genesis block, first use `geth init` to import and sets the canonical genesis block for the new chain. This requires the path to `genesis.json` to be passed as an argument.

```sh
geth init --datadir data genesis.json
```

When Geth is started using `--datadir data` the genesis block defined in `genesis.json` will be used. For example:

```sh
geth --datadir data --networkid 12345
```

### Scheduling Hard Forks {#scheduling-hard-forks}

As Ethereum protocol development progresses, new features become available. To enable these features on an existing private network, a hard fork must be scheduled. To do this, a future block number must be chosen which determines precisely when the hard fork will activate. Continuing the `genesis.json` example above and assuming the current block number is 35421, a hard fork might be scheduled for block 40000. This hard fork might upgrade the network to conform to the 'London' specs. First, all the Geth instances on the private network must be recent enough to support the specific hard fork. If so, `genesis.json` can be updated so that the `londonBlock` key gets the value 40000. The Geth instances are then shut down and `geth init` is run to update their configuration. When the nodes are restarted they will pick up where they left off and run normally until block 40000, at which point they will automatically upgrade.

The modification to `genesis.json` is as follows:

```json
{
  "config": {
    "londonBlock": 40000
  }
}
```

The upgrade command is:

```sh
geth init --datadir data genesis.json
```

### Setting Up Networking {#setting-up-networking}

With the node configured and initialized, the next step is to set up a peer-to-peer network. This requires a bootstrap node. The bootstrap node is a normal node that is designated to be the entry point that other nodes use to join the network. Any node can be chosen to be the bootstrap node.

To configure a bootstrap node, the IP address of the machine the bootstrap node will run on must be known. The bootsrap node needs to know its own IP address so that it can broadcast it to other nodes. On a local machine this can be found using tools such as `ifconfig` and on cloud instances such as Amazon EC2 the IP address of the virtual machine can be found in the management console. Any firewalls must allow UDP and TCP traffic on port 30303.

The bootstrap node IP is set using the `--nat` flag (the command below contains an example address - replace it with the correct one).

```sh
geth --datadir data --networkid 15 --nat extip:172.16.254.4
```

The 'node record' of the bootnode can be extracted using the JS console:

```sh
geth attach --exec admin.nodeInfo.enr data/geth.ipc
```

This command should print a base64 string such as the following example. Other nodes will use the information contained in the bootstrap node record to connect to the peer-to-peer network.

```text
"enr:-Je4QEiMeOxy_h0aweL2DtZmxnUMy-XPQcZllrMt_2V1lzynOwSx7GnjCf1k8BAsZD5dvHOBLuldzLYxpoD5UcqISiwDg2V0aMfGhGlQhqmAgmlkgnY0gmlwhKwQ_gSJc2VjcDI1NmsxoQKX_WLWgDKONsGvxtp9OeSIv2fRoGwu5vMtxfNGdut4cIN0Y3CCdl-DdWRwgnZf"
```

If the nodes are intended to connect across the Internet, the bootnode and all other nodes must have public IP addresses assigned, and both TCP and UDP traffic can pass their firewalls. If Internet connectivity is not required or all member nodes connect using well-known IPs, Geth should be set up to restrict peer-to-peer connectivity to an IP subnet. Doing so will further isolate the network and prevents cross-connecting with other blockchain networks in case the nodes are reachable from the Internet. Use the
`--netrestrict` flag to configure a whitelist of IP networks:

```sh
geth <other-flags> --netrestrict 172.16.254.0/24
```

With the above setting, Geth will only allow connections from the 172.16.254.0/24 subnet, and will not attempt to connect to other nodes outside of the set IP range.

### Running Member Nodes {#running-member-nodes}

Before running a member node, it must be initialized with the same genesis file as used for the bootstrap node. With the bootnode operational and externally reachable (`telnet <ip> <port>` will confirm that it is indeed reachable), more Geth nodes can be started and connected to them via the bootstrap node using the `--bootnodes` flag. The process is to start Geth on the same machine as the bootnode, with a separate data directory and listening port and the bootnode node record provided as an argument:

For example, using data directory (example: `data2`) and listening port (example: `30305`):

```sh
geth --datadir data2 --networkid 12345 --port 30305 --bootnodes <bootstrap-node-record>
```

With the member node running, it is possible to check that it is connected to the bootstrap node or any other node in the network by attaching a console and running `admin.peers`. It may take up to a few seconds for the nodes to get connected.

```sh
geth attach data2/geth.ipc --exec admin.peers
```

### Running A Signer (Clique) {#running-a-signer}

To set up Geth for signing blocks in Clique, a signer account must be available. The account must already be available as a keyfile in the keystore. To use it for signing blocks, it must be unlocked. The following command, for address `0x7df9a875a174b3bc565e6424a0050ebc1b2d1d82` will prompt for the account password, then start signing blocks:

```sh
geth <other-flags> --unlock 0x7df9a875a174b3bc565e6424a0050ebc1b2d1d82 --mine
```

Mining can be further configured by changing the default gas limit blocks converge to (with `--miner.gastarget`) and the price transactions are accepted at (with `--miner.gasprice`).

### Running A Miner (Ethash) {#running-a-miner}

For PoW in a simple private network, a single CPU miner instance is enough to create a stable stream of blocks at regular intervals. To start a Geth instance for mining, it can be run with all the usual flags plus the following to configure mining:

```sh
geth <other-flags> --mine --miner.threads=1 --miner.etherbase=0xf41c74c9ae680c1aa78f42e5647a62f353b7bdde
```

This will start mining bocks and transactions on a single CPU thread, crediting all block rewards to the account specified by `--miner.etherbase`.

## End-to-end example {#end-to-end-example}

This section will run through the commands for setting up a simple private network of two nodes. Both nodes will run on the local machine using the same genesis block and network ID. The data directories for each node will be named `node1` and `node2`.

```sh
`mkdir node1 node2`
```

Each node will have an associated account that will receive some ether at launch. The following command creates an account for Node 1:

```sh
geth --datadir node1 account new
```

This command returns a request for a password. Once a password has been provided the following information is returned to the terminal:

```terminal
Your new account is locked with a password. Please give a password. Do not foget this password.
Password:
Repeat password:

Your new key was generated

Public address of the key: 0xC1B2c0dFD381e6aC08f34816172d6343Decbb12b
Path of the secret key file: node1/keystore/UTC--2022-05-13T14-25-49.229126160Z--c1b2c0dfd381e6ac08f34816172d6343decbb12b

- You can share your public address with anyone. Others need it to interact with you.
- You must NEVER share the secret key with anyone! The key controls access to your funds!
- You must BACKUP your key file! Without the key, it's impossible to access account funds!
- You must remember your password! Without the password, it's impossible to decrypt the key!
```

The keyfile and account password should be backed up securely. These steps can then be repeated for Node 2. These commands create keyfiles that are stored in the `keystore` directory in `node1` and `node2` data directories. In order to unlock the accounts later the passwords for each account should be saved to a text file in each node's data directory.

In each data directory save a copy of the following `genesis.json` to the top level project directory. The account addresses in the `alloc` field should be replaced with those created for each node in the previous step (without the leading `0x`).

```json
{
  "config": {
    "chainId": 12345,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
    "istanbulBlock": 0,
    "muirGlacierBlock": 0,
    "berlinBlock": 0,
    "londonBlock": 0,
    "arrowGlacierBlock": 0,
    "grayGlacierBlock": 0,
    "clique": {
      "period": 5,
      "epoch": 30000
    }
  },
  "difficulty": "1",
  "gasLimit": "800000000",
  "extradata": "0x00000000000000000000000000000000000000000000000000000000000000007df9a875a174b3bc565e6424a0050ebc1b2d1d820000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
  "alloc": {
    "C1B2c0dFD381e6aC08f34816172d6343Decbb12b": { "balance": "500000" },
    "c94d95a5106270775351eecfe43f97e8e75e59e8": { "balance": "500000" }
  }
}
```

The nodes can now be set up using `geth init` as follows:

```sh
geth init --datadir node1 genesis.json
```

This should be repeated for both nodes. The following will be returned to the terminal:

```terminal
INFO [05-13|15:41:47.520] Maximum peer count                       ETH=50 LES=0 total=50
INFO [05-13|15:41:47.520] Smartcard socket not found, disabling    err="stat /run/pcscd/pcscd.comm: no such file or directory"
INFO [05-13|15:41:47.520] Set global gas cap                       cap=50,000,000
INFO [05-13|15:41:47.520] Allocated cache and file handles         database=/home/go-ethereum/node2/geth/chaindata cache=16.00MiB handles=16
INFO [05-13|15:41:47.542] Writing custom genesis block
INFO [05-13|15:41:47.542] Persisted trie from memory database      nodes=3 size=397.00B time="41.246µs" gcnodes=0 gcsize=0.00B gctime=0s livenodes=1 livesize=0.00B
INFO [05-13|15:41:47.543] Successfully wrote genesis state         database=chaindata hash=c9a158..d415a0
INFO [05-13|15:41:47.543] Allocated cache and file handles         database=/home/go-ethereum/node2/geth/chaindata cache=16.00MiB handles=16
INFO [05-13|15:41:47.556] Writing custom genesis block
INFO [05-13|15:41:47.557] Persisted trie from memory database      nodes=3 size=397.00B time="81.801µs" gcnodes=0 gcsize=0.00B gctime=0s livenodes=1 livesize=0.00B
INFO [05-13|15:41:47.558] Successfully wrote genesis state         database=chaindata hash=c9a158..d415a0
```

The next step is to configure a bootnode. This can be any node, but for this tutorial the developer tool `bootnode` will be used to quickly and easily configure a dedicated bootnode. First the bootnode requires a key, which can be created with the following command, which will save a key to `boot.key`:

```sh
bootnode -genkey boot.key
```

This key can then be used to generate a bootnode as follows:

```sh
bootnode -nodekey boot.key -addr :30305
```

The choice of port passed to `-addr` is arbitrary, but public Ethereum networks use 30303, so this is best avoided. The `bootnode` command returns the following logs to the terminal, confirming that it is running:

```terminal
enode://f7aba85ba369923bffd3438b4c8fde6b1f02b1c23ea0aac825ed7eac38e6230e5cadcf868e73b0e28710f4c9f685ca71a86a4911461637ae9ab2bd852939b77f@127.0.0.1:0?discport=30305
Note: you're using cmd/bootnode, a developer tool.
We recommend using a regular node as bootstrap node for production deployments.
INFO [05-13|15:50:03.645] New local node record                    seq=1,652,453,403,645 id=a2d37f4a7d515b3a ip=nil udp=0 tcp=0
```

The two nodes can now be started. Open separate terminals for each node, leaving the bootnode running in the original terminal. In each terminal, run the following command (replacing `node1` with `node2` where appropriate, and giving each node different `--port` and `authrpc.port` IDs. The account address and password file for node 1 must also be provided:

```sh
./geth --datadir node1 --port 30306 --bootnodes enode://f7aba85ba369923bffd3438b4c8fde6b1f02b1c23ea0aac825ed7eac38e6230e5cadcf868e73b0e28710f4c9f685ca71a86a4911461637ae9ab2bd852939b77f@127.0.0.1:0?discport=30305  --networkid 123454321 --unlock 0xC1B2c0dFD381e6aC08f34816172d6343Decbb12b --password node1/password.txt --authrpc.port 8551
```

This will start the node using the bootnode as an entry point. Repeat the same command with the information appropriate to node 2. In each terminal, the following logs indicate success:

```terminal
INFO [05-13|16:17:40.061] Maximum peer count                       ETH=50 LES=0 total=50
INFO [05-13|16:17:40.061] Smartcard socket not found, disabling    err="stat /run/pcscd/pcscd.comm: no such file or directory"
INFO [05-13|16:17:40.061] Set global gas cap                       cap=50,000,000
INFO [05-13|16:17:40.061] Allocated trie memory caches             clean=154.00MiB dirty=256.00MiB
INFO [05-13|16:17:40.061] Allocated cache and file handles         database=/home/go-ethereum/node1/geth/chaindata cache=512.00MiB handles=524,288
INFO [05-13|16:17:40.094] Opened ancient database                  database=/home/go-ethereum/node1/geth/chaindata/ancient readonly=false
INFO [05-13|16:17:40.095] Initialised chain configuration          config="{ChainID: 123454321 Homestead: 0 DAO: nil DAOSupport: false EIP150: 0 EIP155: 0 EIP158: 0 Byzantium: 0 Constantinople: 0 Petersburg: 0 Istanbul: nil, Muir Glacier: nil, Berlin: nil, London: nil, Arrow Glacier: nil, MergeFork: nil, Terminal TD: nil, Engine: clique}"
INFO [05-13|16:17:40.096] Initialising Ethereum protocol           network=123,454,321 dbversion=8
INFO [05-13|16:17:40.098] Loaded most recent local header          number=0 hash=c9a158..d415a0 td=1 age=53y1mo2w
INFO [05-13|16:17:40.098] Loaded most recent local full block      number=0 hash=c9a158..d415a0 td=1 age=53y1mo2w
INFO [05-13|16:17:40.098] Loaded most recent local fast block      number=0 hash=c9a158..d415a0 td=1 age=53y1mo2w
INFO [05-13|16:17:40.099] Loaded local transaction journal         transactions=0 dropped=0
INFO [05-13|16:17:40.100] Regenerated local transaction journal    transactions=0 accounts=0
INFO [05-13|16:17:40.100] Gasprice oracle is ignoring threshold set threshold=2
WARN [05-13|16:17:40.100] Unclean shutdown detected                booted=2022-05-13T16:16:46+0100 age=54s
INFO [05-13|16:17:40.100] Starting peer-to-peer node               instance=Geth/v1.10.18-unstable-8d84a701-20220503/linux-amd64/go1.18.1
INFO [05-13|16:17:40.130] New local node record                    seq=1,652,454,949,228 id=f1364e6d060c4625 ip=127.0.0.1 udp=30306 tcp=30306
INFO [05-13|16:17:40.130] Started P2P networking                   self=enode://87606cd0b27c9c47ca33541d4b68cf553ae6765e22800f0df340e9788912b1e3d2759b3d1933b6f739c720701a56ce26f672823084420746d04c25fc7b8c6824@127.0.0.1:30306
INFO [05-13|16:17:40.133] IPC endpoint opened                      url=/home/go-ethereum/node1/geth.ipc
INFO [05-13|16:17:40.785] Unlocked account                         address=0xC1B2c0dFD381e6aC08f34816172d6343Decbb12b
INFO [05-13|16:17:42.636] New local node record                    seq=1,652,454,949,229 id=f1364e6d060c4625 ip=82.11.59.221 udp=30306 tcp=30306
INFO [05-13|16:17:43.309] Mapped network port                      proto=tcp extport=30306 intport=30306 interface="UPNP IGDv1-IP1"
INFO [05-13|16:17:43.822] Mapped network port                      proto=udp extport=30306 intport=30306 interface="UPNP IGDv1-IP1"
[05-13|16:17:50.150] Looking for peers                        peercount=0 tried=0 static=0
INFO [05-13|16:18:00.164] Looking for peers                        peercount=0 tried=0 static=0
```

In the first terminal that is currently running the logs resembling the following will be displayed, showing the discovery process in action:

```terminal
INFO [05-13|15:50:03.645] New local node record                    seq=1,652,453,403,645 id=a2d37f4a7d515b3a ip=nil udp=0 tcp=0
TRACE[05-13|16:15:49.228]  PING/v4                               id=f1364e6d060c4625 addr=127.0.0.1:30306 err=nil
TRACE[05-13|16:15:49.229]  PONG/v4                               id=f1364e6d060c4625 addr=127.0.0.1:30306 err=nil
TRACE[05-13|16:15:49.229]  PING/v4                               id=f1364e6d060c4625 addr=127.0.0.1:30306 err=nil
TRACE[05-13|16:15:49.230]  PONG/v4                               id=f1364e6d060c4625 addr=127.0.0.1:30306 err=nil
TRACE[05-13|16:15:49.730]  FINDNODE/v4                           id=f1364e6d060c4625 addr=127.0.0.1:30306 err=nil
TRACE[05-13|16:15:49.731]  NEIGHBORS/v4                          id=f1364e6d060c4625 addr=127.0.0.1:30306 err=nil
TRACE[05-13|16:15:50.231]  FINDNODE/v4                           id=f1364e6d060c4625 addr=127.0.0.1:30306 err=nil
TRACE[05-13|16:15:50.231]  NEIGHBORS/v4                          id=f1364e6d060c4625 addr=127.0.0.1:30306 err=nil
TRACE[05-13|16:15:50.561]  FINDNODE/v4                           id=f1364e6d060c4625 addr=127.0.0.1:30306 err=nil
TRACE[05-13|16:15:50.561]  NEIGHBORS/v4                          id=f1364e6d060c4625 addr=127.0.0.1:30306 err=nil
TRACE[05-13|16:15:50.731]  FINDNODE/v4                           id=f1364e6d060c4625 addr=127.0.0.1:30306 err=nil
TRACE[05-13|16:15:50.731]  NEIGHBORS/v4                          id=f1364e6d060c4625 addr=127.0.0.1:30306 err=nil
TRACE[05-13|16:15:51.231]  FINDNODE/v4                           id=f1364e6d060c4625 addr=127.0.0.1:30306 err=nil
TRACE[05-13|16:15:51.232]  NEIGHBORS/v4                          id=f1364e6d060c4625 addr=127.0.0.1:30306 err=nil
TRACE[05-13|16:15:52.591]  FINDNODE/v4                           id=f1364e6d060c4625 addr=127.0.0.1:30306 err=nil
TRACE[05-13|16:15:52.591]  NEIGHBORS/v4                          id=f1364e6d060c4625 addr=127.0.0.1:30306 err=nil
TRACE[05-13|16:15:57.767]  PING/v4                               id=f1364e6d060c4625 addr=127.0.0.1:30306 err=nil
```

It is now possible to attach a Javascript console to either node to query the network properties:

```sh
geth attach node1/geth.ipc
```

Once the Javascript console is running, check that the node is connected to one other peer (node 2):

```sh
net.peerCount
```

The details of this peer can also be queried and used to check that the peer really is Node 2:

```sh
admin.peers
```

This should return the following:

```terminal
[{
    caps: ["eth/66", "snap/1"],
    enode: "enode://6a4576fb12004aa13949dbf25de978102483a6521e6d5d87c5b7ccb1944bbf8995dc730303ae891732410b1dd2e684277e9292fc0a17372a789bb4e87bdf366b@127.0.0.1:30307",
    id: "d300c59ba301abcb5f4a3866aab6f833857c3ddf2f0febb583410b1dc466f175",
    name: "Geth/v1.10.18-unstable-8d84a701-20220503/linux-amd64/go1.18.1",
    network: {
      inbound: false,
      localAddress: "127.0.0.1:56620",
      remoteAddress: "127.0.0.1:30307",
      static: false,
      trusted: false
    },
    protocols: {
      eth: {
        difficulty: 1,
        head: "0xc9a158a687eff8a46128bd5b9aaf6b2f04f10f0683acbd7f031514db9ad415a0",
        version: 66
      },
      snap: {
        version: 1
      }
    }
}]
```

The account associated with Node 1 was supposed to be funded with some ether at the chain genesis. This can be checked easily using `eth.getBalance()`:

```sh
eth.getBalance(eth.accounts[0])
```

This account can then be unlocked and some ether sent to Node 2, using the following commands:

```js
// send some Wei
eth.sendTransaction({
  to: '0xc94d95a5106270775351eecfe43f97e8e75e59e8',
  from: eth.accounts[0],
  value: 25000
});

//check the transaction was successful by querying Node 2's account balance
eth.getBalance('0xc94d95a5106270775351eecfe43f97e8e75e59e8');
```

The same steps can then be repeated to attach a console to Node 2.

## Summary {#summary}

This page explored the various options for configuring a local private network. A step by step guide showed how to set up and launch a private network, unlock the associated accounts, attach a console to check the network status and make some basic interactions.
