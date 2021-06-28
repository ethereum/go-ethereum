# PluGeth

PluGeth is a fork of the [Go Ethereum Client](https://github.com/ethereum/go-ethereum)
(Geth) that implements a plugin architecture, allowing developers to extend
Geth's  capabilities in a number of different ways using plugins, rather than
having to create additional, new forks of Geth.

## WARNING: UNSTABLE API

Right now PluGeth is in early development. We are still settling on some of the
plugin APIs, and are not yet making official releases. From an operational
perspective, PluGeth should be as stable as upstream Geth, less whatever
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

#### Tracers

* **Name**: Tracer
* **Type**: map[string]TracerResult
* **Behavior**: When calling debug.traceX functions (such as debug_traceCall
  and debug_traceTransaction) the tracer can be specified as a key to this map
  and the tracer used  will be the TracerResult specified here. TracerResult
  objects must match the interface:

```
// CaptureStart is called at the start of each transaction
CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
// CaptureState is called for each opcode
CaptureState(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
// CaptureFault is called when an error occurs in the EVM
CaptureFault(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
// CaptureEnd is called at the end of each transaction
CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {
// GetResult should return a JSON serializable result object to respond to the trace call
GetResult() (interface{}, error) {

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


## Extending The Plugin API

While we can imagine lots of ways plugins might like to extract or change
information in Geth, we're trying not to go too crazy with the plugin API based
purely on hypotheticals. The Plugin API in its current form reflects the needs
of projects currently building on PluGeth, and we're happy to extend it for
people who are building something. If you're trying to do something that isn't
supported by the current plugin system, we're happy to help. Reach out to us on
[Discord](https://discord.gg/Epf7b7Gr) and we'll help you figure out how to
make it work.
