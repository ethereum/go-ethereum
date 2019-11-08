---
title: Private network
---

## Private network

<!-- TODO: Bring in -->

See \[[the Private Network Page|Private network]] for more information.

<!-- TODO -->

### Setup bootnode

The first time a node connects to the network it uses one of the predefined [bootnodes](https://github.com/ethereum/go-ethereum/blob/master/params/bootnodes.go). Through these bootnodes a node can join the network and find other nodes. In the case of a private cluster these predefined bootnodes are not of much use. Therefore go-ethereum offers a bootnode implementation that can be configured and run in your private network.

It can be run through the command.

    > bootnode
    Fatal: Use -nodekey or -nodekeyhex to specify a private key

As can be seen the bootnode asks for a key. Each ethereum node, including a bootnode is identified by an enode identifier. These identifiers are derived from a key. Therefore you will need to give the bootnode such key. Since we currently don't have one we can instruct the bootnode to generate a key (and store it in a file) before it starts.

    > bootnode -genkey bootnode.key
    I0216 09:53:08.076155 p2p/discover/udp.go:227] Listening, enode://890b6b5367ef6072455fedbd7a24ebac239d442b18c5ab9d26f58a349dad35ee5783a0dd543e4f454fed22db9772efe28a3ed6f21e75674ef6203e47803da682@[::]:30301

(exit with CTRL-C)

The stored key can be seen with:

    > cat bootnode.key
    dc90f8f7324f1cc7ba52c4077721c939f98a628ed17e51266d01c9cd0294033a

To instruct geth nodes to use our own bootnode(s) use the `--bootnodes` flag. This is a comma separated list of bootnode enode identifiers.

    geth --bootnodes "enode://890b6b5367ef6072455fedbd7a24ebac239d442b18c5ab9d26f58a349dad35ee5783a0dd543e4f454fed22db9772efe28a3ed6f21e75674ef6203e47803da682@[::]:30301"

(what [::] means is explained previously)

Since it is convenient to start the bootnode each time with the same enode we can give the bootnode program the just generated key on the next time it is started.

    bootnode -nodekey bootnode.key
    I0216 10:01:19.125600 p2p/discover/udp.go:227] Listening, enode://890b6b5367ef6072455fedbd7a24ebac239d442b18c5ab9d26f58a349dad35ee5783a0dd543e4f454fed22db9772efe28a3ed6f21e75674ef6203e47803da682@[::]:30301

or

    bootnode -nodekeyhex dc90f8f7324f1cc7ba52c4077721c939f98a628ed17e51266d01c9cd0294033a
    I0216 10:01:40.094089 p2p/discover/udp.go:227] Listening, enode://890b6b5367ef6072455fedbd7a24ebac239d442b18c5ab9d26f58a349dad35ee5783a0dd543e4f454fed22db9772efe28a3ed6f21e75674ef6203e47803da682@[::]:30301

* * *

An Ethereum network is private if the nodes are not connected to a main
network. In this context private means reserved or isolated, rather than
protected or secure.

## Choosing A Network ID

Since connections between nodes are valid only if peers have identical protocol versions
and network IDs, you can effectively isolate your network by setting either of these to a
non default value. We recommend using the `--networkid` command line option for this. Its
argument is an integer, the main network has **id 1** (the default). If you supply your own
custom network ID which is different from the main network your nodes will not connect to
other nodes and form a private network.

## Creating The Genesis Block

Every blockchain starts with the genesis block. When you run geth with default settings
for the first time, it commits the main network genesis block to the database. For a private
network, you usually want to use a different genesis block.

Here's an example of a custom _genesis.json_ file. The `config` section ensures that certain
protocol upgrades are immediately available. The `alloc` section pre-funds accounts.

```json
{
    "config": {
        "chainId": 15,
        "homesteadBlock": 0,
        "eip155Block": 0,
        "eip158Block": 0
    },
    "difficulty": "200000000",
    "gasLimit": "2100000",
    "alloc": {
        "7df9a875a174b3bc565e6424a0050ebc1b2d1d82": { "balance": "300000" },
        "f41c74c9ae680c1aa78f42e5647a62f353b7bdde": { "balance": "400000" }
    }
}
```

To create a database that uses this genesis block, run the following command. This imports and sets the canonical genesis block for your chain.

```shell
geth --datadir path/to/custom/data/folder init genesis.json
```

Future runs of geth using this data directory will use the genesis block you defined.

```text
geth --datadir path/to/custom/data/folder --networkid 15
```

## Network Connectivity

With all nodes initialized to the desired genesis state, you need
to start a bootstrap node that others can use to find each other in your network and/or
over the internet. The clean way is to configure and run a dedicated bootnode:

```shell
bootnode --genkey=boot.key
bootnode --nodekey=boot.key
```

The bootnode shows an enode URL that other nodes can use to connect
to it and exchange peer information. Make sure to replace the displayed IP address
information (most probably `[::]`) with your externally accessible IP to get the actual
enode URL.

<!-- TODO: Then why bother? -->

**Note**: You can also use a full fledged geth node as a bootstrap node.

### Starting Up Your Member Nodes

With the bootnode operational and externally reachable (you can try `telnet <ip> <port>`
to check), start every subsequent geth node pointed to the bootnode
for peer discovery via the `--bootnodes` flag. You should keep
the data directory of your private network separated, so also specify a custom
`--datadir` flag.

```text
geth --datadir path/to/custom/data/folder --networkid 15 --bootnodes {bootnode-enode-url-from-above}
```

Since your network is cut off from the main and test networks, you need to configure a miner to process transactions and create new blocks for you.

## Running A Private Miner

Mining on the public Ethereum network is a complex task as it's only feasible using GPUs,
requiring an OpenCL or CUDA enabled [ethminer](https://github.com/ethereum-mining/ethminer) instance. For information on such a setup,
please consult the [EtherMining subreddit](https://www.reddit.com/r/EtherMining/) and the [Genoil miner repository](https://github.com/Genoil).

In a private network setting a single CPU miner instance is more than enough as it can produce a stable stream of blocks at the correct intervals
without needing heavy resources (consider running on a single thread, no need for multiple
ones either). To start a geth instance for mining, run it with all your usual flags,
extended by:

```shell
geth <usual-flags> --mine --minerthreads 1 --etherbase {ACCOUNT}
```

Which will start mining blocks and transactions on a single CPU thread, crediting all
proceedings to the account specified by `--etherbase`. You can further tune the mining by
changing the default gas limit blocks converge to (`--targetgaslimit`) and the price
transactions are accepted at (`--gasprice`).
