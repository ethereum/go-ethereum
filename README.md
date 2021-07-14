# PluGeth

PluGeth is a fork of the [Go Ethereum Client](https://github.com/ethereum/go-ethereum)
(Geth) that implements a plugin architecture, allowing developers to extend
Geth's  capabilities in a number of different ways using plugins, rather than
having to create additional, new forks of Geth.

## WARNING: UNSTABLE API

Right now PluGeth is in early development. We are still settling on some of the
plugin APIs, and are not yet making official releases. From an operational
perspective, PluGeth should be as stable as upstream Geth less whatever
instability is added by plugins you might run. But if you plan to run PluGeth
today, be aware that future updates will likely break your plugins.

## System Requirements

System requirements will vary depending on which network you are connecting to.
On the Ethereum mainnet, you should have at least 8 GB RAM, 2 CPUs, and 350 GB
of SSD disks.

PluGeth relies on Golang's Plugin implementation, which is only supported on
Linux, FreeBSD, and macOS. Windows support is unlikely to be added in the
foreseeable future.

## Design Goals

The upstream Geth client exists primarily to serve as a client for the Ethereum
mainnet, though it also supports a number of popular testnets. Supporting the
Ethereum mainnet is a big enough challenge in its own right that the Geth team
generally avoids changes to support other networks, or to provide features only
a small handful of users would be interested in.

The result is that many projects have forked Geth. Some implement their own
consensus protocols or alter the behavior of the EVM to support other networks.
Others are designed to extract information from the Ethereum mainnet in ways
the standard Geth client does not support.

Creating numerous different forks to fill a variety of different needs comes
with a number of drawbacks. Forks tend to drift apart from each other. Many
networks that forked from Geth long ago have stopped merging updates from Geth;
this makes some sense, given that those networks have moved in different
directions than Geth and merging upstream changes while properly maintaining
consensus rules of an existing network could prove quite challenging. But not
merging changes from upstream can mean that security updates are easily missed,
especially when the upstream team [obscures security updates as optimizations](https://blog.openrelay.xyz/vulnerability-lifecycle-framework-geth/)
as a matter of process.

PluGeth aims to provide a single Geth fork that developers can choose to extend
rather than forking the Geth project. Out of the box, PluGeth behaves exactly
like upstream Geth, but by installing plugins written in Golang, developers can
extend its functionality in a wide variety of way.

## Anatomy of a Plugin

Plugins for Plugeth use Golang's [Native Plugin System](https://golang.org/pkg/plugin/).
Plugin modules must export variables using specific names and types. These will
be processed by the plugin loader, and invoked at certain points during Geth's
operations.

### API

#### Flags

* **Name**: Flags
* **Type**: [flag.FlagSet](https://golang.org/pkg/flag/#FlagSet)
* **Behavior**: This FlagSet will be parsed and your plugin will be able to
  access the resulting flags. Note that if any flags are provided, certain
  checks are disabled within Geth to avoid failing due to unexpected flags.

#### Subcommands

* **Name**: Subcommands
* **Type**: map[string]func(ctx [*cli.Context](https://pkg.go.dev/github.com/urfave/cli#Context), args []string) error
* **Behavior**: If Geth is invoked with `geth YOUR_COMMAND`, the plugin loader
  will look for `YOUR_COMMAND` within this map, and invoke the corresponding
  function. This can be useful for certain behaviors like manipulating Geth's
  database without having to build a separate binary.


#### Initialize

* **Name**: Initialize
* **Type**: func(*cli.Context, *PluginLoader)
* **Behavior**: Called as soon as the plugin is loaded, with the cli context
  and a reference to the plugin loader. This is your plugin's opportunity to
  initialize required variables as needed. Note that using the context object
  you can check arguments, and optionally can manipulate arguments if needed
  for your plugin.

#### InitializeNode

* **Name**: InitializeNode
* **Type**: func(*node.Node, interfaces.Backend)
* **Behavior**: This is called as soon as the Geth node is initialized. The
 `*node.Node` object represents the running node with p2p and RPC capabilities,
 while the Backend gives you access to a wide array of data you may need to
 access.

#### Tracers

* **Name**: Tracer
* **Type**: map[string]TracerResult
* **Behavior**: When calling debug.traceX functions (such as debug_traceCall
  and debug_traceTransaction) the tracer can be specified as a key to this map
  and the tracer used  will be the TracerResult specified here. TracerResult
  objects must match the interface:

```
// CaptureStart is called at the start of each transaction
CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {}
// CaptureState is called for each opcode
CaptureState(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {}
// CaptureFault is called when an error occurs in the EVM
CaptureFault(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {}
// CaptureEnd is called at the end of each transaction
CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {}
// GetResult should return a JSON serializable result object to respond to the trace call
GetResult() (interface{}, error) {}

```

* **Caution**: Modifying of the values passed into tracer functions can alter
  the results of the EVM execution in unpredictable ways. Additionally, some
  objects may be reused across calls, so data you wish to capture should be
  copied rather than retained by reference.

#### LiveTracer

* **Name**: LiveTracer
* **Type**: vm.Tracer
* **Behavior**: This tracer is used for tracing transactions as they are
  processed within blocks. Note that if a block does not validate, some
  transactions may be processed that don't end up in blocks, so be sure to
  check transactions against finalized blocks.

The interface for a vm.Tracer is similar to a TracerResult (above), but does
not require a `GetResult()` function.

#### GetAPIs

* **Name**: GetAPIs
* **Type**: func(*node.Node, interfaces.Backend) []rpc.API
* **Behavior**: This allows you to register new RPC methods to run within Geth.
* **Example**:

The GetAPIs function itself will generally be fairly brief, and will looks
something like this:

```
func GetAPIs(stack *node.Node, backend plugins.Backend) []rpc.API {
  return []rpc.API{
   {
     Namespace: "mynamespace",
     Version:	 "1.0",
     Service:	 &MyService{backend},
     Public:		true,
   },
 }
}
```

The bulk of the implementation will be in the `MyService` struct. MyService
should be a struct with public functions. These functions can have two
different types of signatures:

* RPC Calls: For straight RPC calls, a function should have a `context.Context`
  object as the first argument, followed by an arbitrary number of JSON
  marshallable arguments, and return either a single JSON marshal object, or a
  JSON marshallable object and an error. The RPC framework will take care of
  decoding inputs to this function and encoding outputs, and if the error is
  non-nil it will serve an error response.
* Subscriptions: For subscriptions (supported on IPC and websockets), a
  function should have a `context.Context` object as the first argument
  followed by an arbitrary number of JSON marshallable arguments, and should
  return an `*rpc.Subscription` object. The subscription object can be created
  with `rpcSub := notifier.CreateSubscription()`, and JSON marshallable data
  can be sent to the subscriber with `notifier.Notify(rpcSub.ID, b)`.

A very simple MyService might look like:

```
type MyService struct{}

func (h *MyService) HelloWorld(ctx context.Context) string {
  return "Hello World"
}
```

And the client could then access this with an rpc call to `mynaespace_helloWorld`.


#### PreProcessBlock
* **Name**: PreProcessBlock
* **Type**: func(*types.Block)
* **Behavior**: Invoked before the transactions of a block are processed.

#### PreProcessTransaction
* **Name**: PreProcessTransaction
* **Type**: func(*types.Transaction, *types.Block, int)
* **Behavior**: Invoked before each individual transaction of a block is
  processed.

#### BlockProcessingError
* **Name**: BlockProcessingError
* **Type**: func(*types.Transaction, *types.Block, error)
* **Behavior**: Invoked if an error occurs while processing a transaction. This
  only applies to errors that would invalidate the block were this transaction
  included, not errors such as reverts or opcode errors.

#### PostProcessTransaction
* **Name**: PostProcessTransaction
* **Type**: func(*types.Transaction, *types.Block, int, *types.Receipt)
* **Behavior**: Invoked after each individual transaction of a block is processed.

#### PostProcessBlock
* **Name**: PostProcessBlock
* **Type**: func(*types.Block)
* **Behavior**: Invoked after all transactions of a block are processed. Note
  that this does not mean that the block can be considered canonical - it may
  end up being uncled or side-chained. You should rely on `NewHead` to
  determine which blocks are canonical.

#### NewHead
* **Name**: NewHead
* **Type**: func(*types.Block, common.Hash, []*types.Log)
* **Behavior**: Invoked when a new block becomes the canonical latest block.
  Note that if several blocks are processed in a group (such as during a reorg)
  this may not be called for each block. You should track the prior latest head
  if you need to process intermediate blocks.

#### NewSideBlock
* **Name**: NewSideBlock
* **Type**: func(*types.Block, common.Hash, []*types.Log)
* **Behavior**: Invoked when a block is side-chained. Blocks passed to this
  method are non-canonical blocks

#### Reorg
* **Name**: Reorg
* **Type**: func(common *types.Block, oldChain, newChain types.Blocks)
* **Behavior**: Invoked when a chain reorg occurs (at least one block is
  removed and one block is added). `oldChain` is a list of removed blocks,
  `newChain` is a list of newly added blocks, and `common` is the latest block
  that is an ancestor to both oldChain and newChain.

#### StateUpdate
* **Name**: StateUpdate
* **Type**: func(root common.Hash, parentRoot common.Hash, destructs map[common.Hash]struct{}, accounts map[common.Hash][]byte, storage map[common.Hash]map[common.Hash][]byte)
* **Behavior**: Invoked for each new block, StateUpdate provides the changes to
  the blockchain state. `root` corresponds to the state root of the new block.
  `parentRoot` corresponds to the state root of the parent block. `destructs`
  serves as a set of accounts that self-destructed in this block. `accounts`
  maps the hash of each account address to the SlimRLP encoding of the account
  data. `storage` maps the hash of each account to a map of that account's
  stored data.

#### AppendAncient
* **Name**: AppendAncient
* **Type**: func(number uint64, hash, header, body, receipts, td []byte)
* **Behavior**: Invoked when the freezer moves a block from LevelDB to the
  ancients database. `number` is the number of the block. `hash` is the 32 byte
  hash of the block as a raw `[]byte`. `header`, `body`, and `receipts` are the
  RLP encoded versions of their respective block elements. `td` is the byte
  encoded total difficulty of the block.


## Extending The Plugin API

When extending the plugin API, a primary concern is leaving a minimal footprint
in the core Geth codebase to avoid future merge conflicts. To achieve this,
when we want to add a hook within some existing Geth code, we create a
`plugin_hooks.go` in the same package. For example, in the core/rawdb package
we have:

```
// This file is part of the package we are adding hooks to
package rawdb

// Import whatever is necessary
import (
  "github.com/ethereum/go-ethereum/plugins"
  "github.com/ethereum/go-ethereum/log"
)


// PluginAppendAncient is the public plugin hook function, available for testing
func PluginAppendAncient(pl *plugins.PluginLoader, number uint64, hash, header, body, receipts, td []byte) {
  fnList := pl.Lookup("AppendAncient", func(item interface{}) bool {
    _, ok := item.(func(number uint64, hash, header, body, receipts, td []byte))
    return ok
  })
  for _, fni := range fnList {
    if fn, ok := fni.(func(number uint64, hash, header, body, receipts, td []byte)); ok {
      fn(number, hash, header, body, receipts, td)
    }
  }
}

// pluginAppendAncient is the private plugin hook function
func pluginAppendAncient(number uint64, hash, header, body, receipts, td []byte) {
  if plugins.DefaultPluginLoader == nil {
		log.Warn("Attempting AppendAncient, but default PluginLoader has not been initialized")
    return
  }
  PluginAppendAncient(plugins.DefaultPluginLoader, number, hash, header, body, receipts, td)
}
```

### The Public Plugin Hook Function

The public plugin hook function should follow the naming convention
`Plugin$HookName`. The first argument should be a *plugins.PluginLoader,
followed by any arguments required by the functions to be provided by nay
plugins implementing this hook.

The plugin hook function should use `PluginLoader.Lookup("$HookName", func(item interface{}) bool`
to get a list of the plugin-provided functions to be invoked. The provided
function should verify that the provided function implements the expected
interface. After the first time a given hook is looked up through the plugin
loader, the PluginLoader will cache references to those hooks.

Given the function list provided by the plugin loader, the public plugin hook
function should iterate over the list, cast the elements to the appropriate
type, and call the function with the provided arguments.

Unless there is a clear justification to the contrary, the function should be
called in the current goroutine. Plugins may choose to spawn off a separate
goroutine as appropriate, but for the sake of thread safety we should generally
not assume that plugins will be implemented in a threadsafe manner. If a plugin
degrades the performance of Geth significantly, that will generally be obvious,
and plugin authors can take appropriate measures to improve performance. If a
plugin introduces thread safety issues, those can go unnoticed during testing.

### The Private Plugin Hook Function

The private plugin hook function should bear the same name as the public plugin
hook function, but with a lower case first letter. The signature should match
the public plugin hook function, except that the first argument referencing the
PluginLoader should be removed. It should invoke the public plugin hook
function on `plugins.DefaultPluginLoader`. It should always verify that the
DefaultPluginLoader is non-nil, log warning and return if the
DefaultPluginLoader has not been initialized.

### In-Line Invocation

Within the Geth codebase, the private plugin hook function should be invoked
with the appropriate arguments in a single line, to minimize unexpected
conflicts merging the upstream geth codebase into plugeth.

### Contact Us

While we can imagine lots of ways plugins might like to extract or change
information in Geth, we're trying not to go too crazy with the plugin API based
purely on hypotheticals. The Plugin API in its current form reflects the needs
of projects currently building on PluGeth, and we're happy to extend it for
people who are building something. If you're trying to do something that isn't
supported by the current plugin system, we're happy to help. Reach out to us on
[Discord](https://discord.gg/Epf7b7Gr) and we'll help you figure out how to
make it work.
