---
title: Private Networks
sort_key: B
---

This guide explains how to set up a private network of multiple Geth nodes. An Ethereum
network is a private network if the nodes are not connected to the main network. In this
context private only means reserved or isolated, rather than protected or secure.

### Choosing A Network ID

The network ID is an integer number which isolates Ethereum peer-to-peer networks.
Connections between blockchain nodes will occur only if both peers use the same genesis
block and network ID. Use the `--networkid` command line option to set the network ID used
by geth.

The main network has ID 1. If you supply your own custom network ID which is different
than the main network, your nodes will not connect to other nodes and form a private
network. If you're planning to connect to your private chain on the Internet, it's best to
choose a network ID that isn't already used. You can find a community-run registry of
Ethereum networks at <https://chainid.network>.

### Choosing A Consensus Algorithm

While the main network uses proof-of-work to secure the blockchain, Geth also supports the
'clique' proof-of-authority consensus algorithm as an alternative for private
networks. We strongly recommend 'clique' for new private network deployments because it is
much less resource intensive than proof-of-work. The clique system is also used for
several public Ethereum testnets such as [Rinkeby](https://www.rinkeby.io) and
[Görli](https://goerli.net).

Here are the key differences between the two consensus algorithms available in Geth:

Ethash consensus, being a proof-of-work algorithm, is a system that allows open
participation by anyone willing to dedicate resources to mining. While this is a great
property to have for a public network, the overall security of the blockchain strictly
depends on the total amount of resources used to secure it. As such, proof-of-work is a
poor choice for private networks with few miners. The Ethash mining 'difficulty' is
adjusted automatically so that new blocks are created approximately 12 seconds apart. As
more mining resources are deployed on the network, creating a new block becomes harder so
that the average block time matches the target block time.

Clique consensus is a proof-of-authority system where new blocks can be created by
authorized 'signers' only. The clique consenus protocol is specified in
[EIP-225][clique-eip]. The initial set of authorized signers is configured in the genesis
block. Signers can be authorized and de-authorized using a voting mechanism, thus allowing
the set of signers to change while the blockchain operates. Clique can be configured to
target any block time (within reasonable limits) since it isn't tied to the difficulty
adjustment.

[clique-eip]: https://eips.ethereum.org/EIPS/eip-225

### Creating The Genesis Block

Every blockchain starts with the genesis block. When you run Geth with default settings
for the first time, it commits the main net genesis to the database. For a private
network, you usually want a different genesis block.

The genesis block is configured using the _genesis.json_ file. When creating a genesis
block, you need to decide on a few initial parameters for your blockchain:

- Ethereum platform features enabled at launch (`config`). Enabling protocol features
  while the blockchain is running requires scheduling a hard fork.
- Initial block gas limit (`gasLimit`). Your choice here impacts how much EVM computation
  can happen within a single block. We recommend using the main Ethereum network as a
  [guideline to find a good amount][gaslimit-chart]. The block gas limit can be adjusted
  after launch using the `--miner.gastarget` command-line flag.
- Initial allocation of ether (`alloc`). This determines how much ether is available to
  the addresses you list in the genesis block. Additional ether can be created through
  mining as the chain progresses.

[gaslimit-chart]: https://etherscan.io/chart/gaslimit

#### Clique Example

This is an example of a genesis.json file for a proof-of-authority network. The `config`
section ensures that all known protocol changes are available and configures the 'clique'
engine to be used for consensus.

Note that the initial signer set must be configured through the `extradata` field. This
field is required for clique to work.

First create the signer account keys using the [geth account](./managing-your-accounts)
command (run this command multiple times to create more than one signer key).

```shell
geth account new --datadir data
```

Take note of the Ethereum address printed by this command.

To create the initial extradata for your network, collect the signer addresses and encode
`extradata` as the concatenation of 32 zero bytes, all signer addresses, and 65 further
zero bytes. In the example below, `extradata` contains a single initial signer address,
`0x7df9a875a174b3bc565e6424a0050ebc1b2d1d82`.

You can use the `period` configuration option to set the target block time of the chain.

```json
{
  "config": {
    "chainId": 15,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
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

#### Ethash Example

Since ethash is the default consensus algorithm, no additional parameters need to be
configured in order to use it. You can influence the initial mining difficulty using the
`difficulty` parameter, but note that the difficulty adjustment algorithm will quickly
adapt to the amount of mining resources you deploy on the chain.

```json
{
  "config": {
    "chainId": 15,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
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

### Initializing the Geth Database

To create a blockchain node that uses this genesis block, run the following command. This
imports and sets the canonical genesis block for your chain.

```shell
geth init --datadir data genesis.json
```

Future runs of geth using this data directory will use the genesis block you have defined.

```shell
geth --datadir data --networkid 15
```

### Scheduling Hard Forks

As Ethereum protocol development progresses, new Ethereum features become available. To
enable these features on your private network, you must schedule a hard fork.

First, choose any future block number where the hard fork will activate. Continuing from
the genesis.json example above, let's assume your network is running and its current block
number is 35421. To schedule the 'Istanbul' fork, we pick block 40000 as the activation
block number and modify our genesis.json file to set it:

```json
{
  "config": {
    ...
    "istanbulBlock": 40000,
    ...
  },
  ...
}
```

In order to update to the new fork, first ensure that all Geth instances on your private
network actually support the Istanbul fork (i.e. ensure you have the latest version of
Geth installed). Now shut down all nodes and re-run the `init` command to enable the new
chain configuration:

```shell
geth init --datadir data genesis.json
```

### Setting Up Networking

Once your node is initialized to the desired genesis state, it is time to set up the
peer-to-peer network. Any node can be used as an entry point. We recommend dedicating a
single node as the rendezvous point which all other nodes use to join. This node is called
the 'bootstrap node'.

First, determine the IP address of the machine your bootstrap node will run on. If you are
using a cloud service such as Amazon EC2, you'll find the IP of the virtual machine in the
management console. Please also ensure that your firewall configuration allows both UDP
and TCP traffic on port 30303.

The bootstrap node needs to know about its own IP address in order to be able to relay it
others. The IP is set using the `--nat` flag (insert your own IP instead of the example
address below).

```shell
geth --datadir data --networkid 15 --nat extip:172.16.254.4
```

Now extract the 'node record' of the bootnode using the JS console.

```shell
geth attach data/geth.ipc --exec admin.nodeInfo.enr
```

This command should print a base64 string such as the following example. Other nodes will
use the information contained in the bootstrap node record to connect to your peer-to-peer
network.

```text
"enr:-Je4QEiMeOxy_h0aweL2DtZmxnUMy-XPQcZllrMt_2V1lzynOwSx7GnjCf1k8BAsZD5dvHOBLuldzLYxpoD5UcqISiwDg2V0aMfGhGlQhqmAgmlkgnY0gmlwhKwQ_gSJc2VjcDI1NmsxoQKX_WLWgDKONsGvxtp9OeSIv2fRoGwu5vMtxfNGdut4cIN0Y3CCdl-DdWRwgnZf"
```

Setting up peer-to-peer networking depends on your requirements. If you connect nodes
across the Internet, please ensure that your bootnode and all other nodes have public IP
addresses assigned, and both TCP and UDP traffic can pass the firewall.

If Internet connectivity is not required or all member nodes connect using well-known IPs,
we strongly recommend setting up Geth to restrict peer-to-peer connectivity to an IP
subnet. Doing so will further isolate your network and prevents cross-connecting with
other blockchain networks in case your nodes are reachable from the Internet. Use the
`--netrestrict` flag to configure a whitelist of IP networks:

```shell
geth <other-flags> --netrestrict 172.16.254.0/24
```

With the above setting, Geth will only allow connections from the 172.16.254.0/24 subnet,
and will not attempt to connect to other nodes outside of the set IP range.

### Running Member Nodes

Before running a member node, you have to initialize it with the same genesis file as
used for the bootstrap node.

With the bootnode operational and externally reachable (you can try `telnet <ip> <port>`
to ensure it's indeed reachable), you can start more Geth nodes and connect them via the
bootstrap node using the `--bootnodes` flag.

To create a member node running on the same machine as the bootstrap node, choose a
separate data directory (example: `data-2`) and listening port (example: `30305`):

```shell
geth --datadir data-2 --networkid 15 --port 30305 --bootnodes <bootstrap-node-record>
```

With the member node running, you can check whether it is connected to the bootstrap node
or any other node in your network by attaching a console and running `admin.peers`. It may
take up to a few seconds for the nodes to get connected.

```shell
geth attach data-2/geth.ipc --exec admin.peers
```

### Clique: Running A Signer

To set up Geth for signing blocks in proof-of-authority mode, a signer account must be
available. The account must be unlocked to mine blocks. The following command will prompt
for the account password, then start signing blocks:

```shell
geth <other-flags> --unlock 0x7df9a875a174b3bc565e6424a0050ebc1b2d1d82 --mine
```

You can further configure mining by changing the default gas limit blocks converge to
(with `--miner.gastarget`) and the price transactions are accepted at (with `--miner.gasprice`).

### Ethash: Running A Miner

For proof-of-work in a simple private network, a single CPU miner instance is enough to
create a stable stream of blocks at regular intervals. To start a Geth instance for
mining, run it with all the usual flags and add the following to configure mining:

```shell
geth <other-flags> --mine --miner.threads=1 --miner.etherbase=0x0000000000000000000000000000000000000000
```

This will start mining bocks and transactions on a single CPU thread, crediting all block
rewards to the account specified by `--miner.etherbase`.
