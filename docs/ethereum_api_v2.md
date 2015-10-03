The current design for setting up a node is non-trivial and requires you to
write a lot of custom startup code (e.g. setting up node configuration, gas,
verbosity, port, etc). I suggest that we allow custom configuration but set any
default value to those that aren't set (e.g. data dir, name, networking, keys).

I suggest we also split out the RPC, IPC and Whisper from the `eth.Backend`
interface and move those secondary interface and startup code to a `eth/utility`
package which can handle most of the domain specific code.

The node should also contain a `Com`munication interface that, depending on the
node type (i.e. full, light), returns a type allowing you to communicate and
interface with the ethereum network and allows you to filter for specific
ethereum related events (i.e. block, log).

Currently the `eth.Ethereum` is responsibly for just about anything, managing
keys, managing whisper, starting stopping filtering system, etc. Most of this
logic should instead by moved to the binary implementing the client. For example
the `KeyManager` shouldn't be managed by the `Ethereum` node but instead should
be managed by the client code. There needs to be a clear separation between
**client** and **node**.

### Node

A `Node` is responsible for allowing access to the ethereum state and management
thereof, setting up the P2P stack (arguable) and allowing direct access to the
the higher level `Com` object while keeping its integrity and allow direct
access to the lower level APIs of the node.

It can be argued that the P2P extensions (e.g. Ethereum, Whisper) should not be
integrated directly in to the node but instead should be offered to the node as
service. 

### Client

A `Client` is what wraps up the node, offers several services to the `Node` and
possibly allows some level of interaction with the system. An example of a cient
would be `geth`. While the node offers little flexibility in terms of signing
transactions, allowing user confirmations and accesses to the Ethereum internal
state due to lack of interfacing, the client should fill that gap by tying the
several systems together and providing the user the need and tools. Managing the
user's keys, setting up a REPS and confirmation dialogs are all examples of
tasks the client should be managing.

***

I'll continue to update this issue and in the meantime I recommend anyone to
comment on this issue with ideas and suggestions.

#### Example

```go
package main

import (
    "fmt"
    "math/big"
    "os"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/core/vm"
    "github.com/ethereum/go-ethereum/eth"
)

func main() {
    // setup ethereum. the rest of the defaults will be picked for us
    // (port, host, ipc, etc). Second argument is the type of node; full/light
    node, err := eth.New(eth.Config{
        Name:    "My ethereum node",
        Datadir: "/tmp/001",
    }, eth.Light)
    if err != nil {
        logger.Fatalln(err)
    }

    // communication interface. `com` is an interface and depends on the
    // node type given to `eth.New`. `com` is the basic state accessor
    com := node.Com()
    // state interfacing
    com.GetBalance(common.Address{})
    com.SetBalance(common.Address{}, big.NewInt(1))
    com.SetAccountStorage(common.Address{}, common.Hash{}, common.Hash{})
    // and allow filtering
    id := com.Filters().AddLogFilter(filters.Log{FromBlock: 0, ToBlock: -1}, func(logs vm.Logs) {
        fmt.Println("get log event")
    })
    id = com.Filters().AddBlockFilter(filters.BlockAny|filters.BlockFork, func(typ filter.BlockEvent, block *types.Block) {
        fmt.Println("block event:", typ)
    })
    // send transactions
    tx := types.NewTransaction(common.Address{/* to */, big.NewInt(1), big.NewInt(1), []byte{1,2,3})
    tx = tx.WithSignECDSA(privateKey)
    node.TxPool().SubmitTransaction(tx)

    // setup service
    // HTTP RPC
    http, err := util.StartRPC(com, remote.Http{":0"})
    if err != nil {
        logger.Fatalln(err)
    }
    rpc.EnableAPIs(remote.APIs{remote.Web3})
    // IPC RPC
    ipc, err := util.StartRPC(com, remote.Ipc{"/path/to"})
    if err != nil {
        logger.Fatalln(err)
    }
    ipc.EnableAPIs(remote.APIs{remote.Web3, remote.Admin, remote.Personal})

    // start up whisper
    whisper, err := util.StartWhisper(node.Net())
    if err != nil {
        logger.Fatalln(err)
    }
    whisper.Post("stuff")

    swarm, err := util.StartSwarm(com, ipfs.Provider())
    if err != nil {
        logger.Fatalln(err)
    }

    in, err := os.Open("/path/to/source")
    if err != nil {
        logger.Fatalln(err)
    }
    defer in.Close()

    hash, size := swarm.Copy(swarm, in)

    // get default eventer
    eventer := node.DefaultEventer()
    eventer.Post(struct{ T string }{"my async event"})
    eventer.PostSync(struct{ T string }{"my sync event"})

    // let eth handle shutdowns
    eth.WaitForShutdown()
}
```

