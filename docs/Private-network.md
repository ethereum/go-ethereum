An Ethereum network is a private network if the nodes are not connected to the main
network nodes. In this context private only means reserved or isolated, rather than
protected or secure.

## Choosing A Network ID

Since connections between nodes are valid only if peers have identical protocol version
and network ID, you can effectively isolate your network by setting either of these to a
non default value. We recommend using the `--networkid` command line option for this. Its
argument is an integer, the main network has id 1 (the default). So if you supply your own
custom network ID which is different than the main network your nodes will not connect to
other nodes and form a private network.

## Creating The Genesis Block

Every blockchain starts with the genesis block. When you run geth with default settings
for the first time, the main net genesis block is committed to the database. For a private
network, you usually want a different genesis block.

Here's an example of a custom genesis.json file. The `config` section ensures that certain
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

To create a database that uses this genesis block, run the following command. This will
import and set the canonical genesis block for your chain.

```text
geth --datadir path/to/custom/data/folder init genesis.json
```

Future runs of geth on this data directory will use the genesis block you have defined.

```text
geth --datadir path/to/custom/data/folder --networkid 15
```

## Network Connectivity

With all nodes that you want to run initialized to the desired genesis state, you'll need
to start a bootstrap node that others can use to find each other in your network and/or
over the internet. The clean way is to configure and run a dedicated bootnode:

```text
bootnode --genkey=boot.key
bootnode --nodekey=boot.key
```

With the bootnode online, it will display an enode URL that other nodes can use to connect
to it and exchange peer information. Make sure to replace the displayed IP address
information (most probably [::]) with your externally accessible IP to get the actual
enode URL.

Note: You can also use a full fledged Geth node as a bootstrap node.

### Starting Up Your Member Nodes

With the bootnode operational and externally reachable (you can try `telnet <ip> <port>`
to ensure it's indeed reachable), start every subsequent Geth node pointed to the bootnode
for peer discovery via the --bootnodes flag. It will probably also be desirable to keep
the data directory of your private network separated, so do also specify a custom
`--datadir` flag.

```text
geth --datadir path/to/custom/data/folder --networkid 15 --bootnodes <bootnode-enode-url-from-above>
```

Since your network will be completely cut off from the main and test networks, you'll also
need to configure a miner to process transactions and create new blocks for you.

## Running A Private Miner

Mining on the public Ethereum network is a complex task as it's only feasible using GPUs,
requiring an OpenCL or CUDA enabled ethminer instance. For information on such a setup,
please consult the EtherMining subreddit and the Genoil miner repository.

In a private network setting however, a single CPU miner instance is more than enough for
practical purposes as it can produce a stable stream of blocks at the correct intervals
without needing heavy resources (consider running on a single thread, no need for multiple
ones either). To start a Geth instance for mining, run it with all your usual flags,
extended by:

```text
$ geth <usual-flags> --mine --minerthreads=1 --etherbase=0x0000000000000000000000000000000000000000
```

Which will start mining bocks and transactions on a single CPU thread, crediting all
proceedings to the account specified by --etherbase. You can further tune the mining by
changing the default gas limit blocks converge to (`--targetgaslimit`) and the price
transactions are accepted at (`--gasprice`).
