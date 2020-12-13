---
title: Go Contract Bindings
---

**[Please note, events are not yet implemented as they need some RPC subscription
features that are still under review.]**

The original roadmap and/or dream of the Ethereum platform was to provide a solid, high
performing client implementation of the consensus protocol in various languages, which
would provide an RPC interface for JavaScript DApps to communicate with, pushing towards
the direction of the Mist browser, through which users can interact with the blockchain.

Although this was a solid plan for mainstream adoption and does cover quite a lot of use
cases that people come up with (mostly where people manually interact with the blockchain),
it eludes the server side (backend, fully automated, devops) use cases where JavaScript is
usually not the language of choice given its dynamic nature.

This page introduces the concept of server side native Dapps: Go language bindings to any
Ethereum contract that is compile time type safe, highly performant and best of all, can
be generated fully automatically from a contract ABI and optionally the EVM bytecode.

*This page is written in a more beginner friendly tutorial style to make it easier for
people to start out with writing Go native Dapps. The used concepts will be introduced
gradually as a developer would need/encounter them. However, we do assume the reader
is familiar with Ethereum in general, has a fair understanding of Solidity and can code
Go.*

## Token contract 

To avoid falling into the fallacy of useless academic examples, we're going to take the
official Token contract as the base for introducing the Go
native bindings. If you're unfamiliar with the contract, skimming the linked page should
probably be enough, the details aren't relevant for now. *In short the contract implements
a custom token that can be deployed on top of Ethereum.* To make sure this tutorial doesn't
go stale if the linked website changes, the Solidity source code of the Token contract is
also available at [`token.sol`](https://gist.github.com/karalabe/08f4b780e01c8452d989).

### Go binding generator

Interacting with a contract on the Ethereum blockchain from Go (or any other language for
a matter of fact) is already possible via the RPC interfaces exposed by Ethereum clients.
However, writing the boilerplate code that translates decent Go language constructs into
RPC calls and back is extremely time consuming and also extremely brittle: implementation
bugs can only be detected during runtime and it's almost impossible to evolve a contract
as even a tiny change in Solidity can be painful to port over to Go.

To avoid all this mess, the go-ethereum implementation introduces a source code generator
that can convert Ethereum ABI definitions into easy to use, type-safe Go packages. Assuming
you have a valid Go development environment set up, `godep` installed and the go-ethereum
repository checked out correctly, you can build the generator with:

```
$ cd $GOPATH/src/github.com/ethereum/go-ethereum
$ godep go install ./cmd/abigen
```

### Generating the bindings

The single essential thing needed to generate a Go binding to an Ethereum contract is the
contract's ABI definition `JSON` file. For our `Token` contract tutorial you can obtain this
either by compiling the Solidity code yourself (e.g. via @chriseth's [online Solidity compiler](https://chriseth.github.io/browser-solidity/)), or you can download our pre-compiled [`token.abi`](https://gist.github.com/karalabe/b8dfdb6d301660f56c1b).

To generate a binding, simply call:

```
$ abigen --abi token.abi --pkg main --type Token --out token.go
```

Where the flags are:

 * `--abi`: Mandatory path to the contract ABI to bind to
 * `--pgk`: Mandatory Go package name to place the Go code into
 * `--type`: Optional Go type name to assign to the binding struct
 * `--out`: Optional output path for the generated Go source file (not set = stdout)

This will generate a type-safe Go binding for the Token contract. The generated code will
look something like [`token.go`](https://gist.github.com/karalabe/5839509295afa4f7e2215bc4116c7a8f),
but please generate your own as this will change as more work is put into the generator.

### Accessing an Ethereum contract

To interact with a contract deployed on the blockchain, you'll need to know the `address`
of the contract itself, and need to specify a `backend` through which to access Ethereum.
The binding generator provides out of the box an RPC backend through which you can attach
to an existing Ethereum node via IPC, HTTP or WebSockets.

We'll use the foundation's Unicorn token contract deployed
on the testnet to demonstrate calling contract methods. It is deployed at the address
`0x21e6fc92f93c8a1bb41e2be64b4e1f88a54d3576`.

To run the snippet below, please ensure a Geth instance is running and attached to the
Morden test network where the above mentioned contract was deployed. Also please update
the path to the IPC socket below to the one reported by your own local Geth node.

```go
package main

import (
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// Create an IPC based RPC connection to a remote node
	conn, err := ethclient.Dial("/home/karalabe/.ethereum/testnet/geth.ipc")
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	// Instantiate the contract and display its name
	token, err := NewToken(common.HexToAddress("0x21e6fc92f93c8a1bb41e2be64b4e1f88a54d3576"), conn)
	if err != nil {
		log.Fatalf("Failed to instantiate a Token contract: %v", err)
	}
	name, err := token.Name(nil)
	if err != nil {
		log.Fatalf("Failed to retrieve token name: %v", err)
	}
	fmt.Println("Token name:", name)
}
```

And the output (yay):

```
Token name: Testnet Unicorn
```

If you look at the method invoked to read the token name `token.Name(nil)`, it required
a parameter to be passed, even though the original Solidity contract requires none. This
is a `*bind.CallOpts` type, which can be used to fine tune the call.

 * `Pending`: Whether to access pending contract state or the current stable one
 * `GasLimit`: Place a limit on the computing resources the call might consume

### Transacting with an Ethereum contract

Invoking a method that changes contract state (i.e. transacting) is a bit more involved,
as a live transaction needs to be authorized and broadcast into the network. **Opposed
to the conventional way of storing accounts and keys in the node we attach to, Go bindings
require signing transactions locally and do not delegate this to a remote node.** This is
done so to facilitate the general direction of the Ethereum community where accounts are
kept private to DApps, and not shared (by default) between them.

Thus to allow transacting with a contract, your code needs to implement a method that
given an input transaction, signs it and returns an authorized output transaction. Since
most users have their keys in the [Web3 Secret Storage](https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition) format, the `bind` package contains a small utility method 
(`bind.NewTransactor(keyjson, passphrase)`) that can create an authorized transactor from
a key file and associated password, without the user needing to implement key signing himself.

Changing the previous code snippet to send one unicorn to the zero address:

```go
package main

import (
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const key = `paste the contents of your *testnet* key json here`

func main() {
	// Create an IPC based RPC connection to a remote node and instantiate a contract binding
	conn, err := ethclient.Dial("/home/karalabe/.ethereum/testnet/geth.ipc")
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	token, err := NewToken(common.HexToAddress("0x21e6fc92f93c8a1bb41e2be64b4e1f88a54d3576"), conn)
	if err != nil {
		log.Fatalf("Failed to instantiate a Token contract: %v", err)
	}
	// Create an authorized transactor and spend 1 unicorn
	auth, err := bind.NewTransactor(strings.NewReader(key), "my awesome super secret password")
	if err != nil {
		log.Fatalf("Failed to create authorized transactor: %v", err)
	}
	tx, err := token.Transfer(auth, common.HexToAddress("0x0000000000000000000000000000000000000000"), big.NewInt(1))
	if err != nil {
		log.Fatalf("Failed to request token transfer: %v", err)
	}
	fmt.Printf("Transfer pending: 0x%x\n", tx.Hash())
}
```

And the output (yay):

```
Transfer pending: 0x4f4aaeb29ed48e88dd653a81f0b05d4df64a86c99d4e83b5bfeb0f0006b0e55b
```

*Note, with high probability you won't have any testnet unicorns available to spend, so the
above program will fail with an error. Send at least 2.014 testnet(!) Ethers to the foundation
testnet tipjar `0xDf7D0030bfed998Db43288C190b63470c2d18F50` to receive a unicorn token and
you'll be able to see the above code run without an error!*

Similar to the method invocations in the previous section which only read contract state,
transacting methods also require a mandatory first parameter, a `*bind.TransactOpts` type,
which authorizes the transaction and potentially fine tunes it:

 * `From`: Address of the account to invoke the method with (mandatory)
 * `Signer`: Method to sign a transaction locally before broadcasting it (mandatory)
 * `Nonce`: Account nonce to use for the transaction ordering (optional)
 * `GasLimit`: Place a limit on the computing resources the call might consume (optional)
 * `GasPrice`: Explicitly set the gas price to run the transaction with (optional)
 * `Value`: Any funds to transfer along with the method call (optional)

The two mandatory fields are automatically set by the `bind` package if the auth options are
constructed using `bind.NewTransactor`. The nonce and gas related fields are automatically
derived by the binding if they are not set. An unset value is assumed to be zero.

### Pre-configured contract sessions

As mentioned in the previous two sections, both reading as well as state modifying contract
calls require a mandatory first parameter which can both authorize as well as fine tune some
of the internal parameters. However, most of the time we want to use the same parameters and
issue transactions with the same account, so always constructing the call/transact options or
passing them along with the binding can become unwieldy.

To avoid these scenarios, the generator also creates specialized wrappers that can be pre-
configured with tuning and authorization parameters, allowing all the Solidity defined methods
to be invoked without needing an extra parameter.

These are named analogous to the original contract type name, just suffixed with `Sessions`:

```go
// Wrap the Token contract instance into a session
session := &TokenSession{
	Contract: token,
	CallOpts: bind.CallOpts{
		Pending: true,
	},
	TransactOpts: bind.TransactOpts{
		From:     auth.From,
		Signer:   auth.Signer,
		GasLimit: big.NewInt(3141592),
	},
}
// Call the previous methods without the option parameters
session.Name()
session.Transfer("0x0000000000000000000000000000000000000000"), big.NewInt(1))
```

### Deploying contracts to Ethereum

Interacting with existing contracts is nice, but let's take it up a notch and deploy
a brand new contract onto the Ethereum blockchain! To do so however, the contract ABI
we used to generate the binding is not enough. We need the compiled bytecode too to
allow deploying it.

To get the bytecode, either go back to the online compiler with which you may generate it,
or alternatively download our [`token.bin`](https://gist.github.com/karalabe/026548f6a5f5f97b54de).
You'll need to rerun the Go generator with the bytecode included for it to create deploy
code too:

```
$ abigen --abi token.abi --pkg main --type Token --out token.go --bin token.bin
```

This will generate something similar to [`token.go`](https://gist.github.com/karalabe/2153b087c1f80f651fd87dd4c439fac4).
If you quickly skim this file, you'll find an extra `DeployToken` function that was just
injected compared to the previous code. Beside all the parameters specified by Solidity,
it also needs the usual authorization options to deploy the contract with and the Ethereum
backend to deploy the contract through.

Putting it all together would result in:

```go
package main

import (
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
)

const key = `paste the contents of your *testnet* key json here`

func main() {
	// Create an IPC based RPC connection to a remote node and an authorized transactor
	conn, err := rpc.NewIPCClient("/home/karalabe/.ethereum/testnet/geth.ipc")
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	auth, err := bind.NewTransactor(strings.NewReader(key), "my awesome super secret password")
	if err != nil {
		log.Fatalf("Failed to create authorized transactor: %v", err)
	}
	// Deploy a new awesome contract for the binding demo
	address, tx, token, err := DeployToken(auth, conn), new(big.Int), "Contracts in Go!!!", 0, "Go!")
	if err != nil {
		log.Fatalf("Failed to deploy new token contract: %v", err)
	}
	fmt.Printf("Contract pending deploy: 0x%x\n", address)
	fmt.Printf("Transaction waiting to be mined: 0x%x\n\n", tx.Hash())

	// Don't even wait, check its presence in the local pending state
	time.Sleep(250 * time.Millisecond) // Allow it to be processed by the local node :P

	name, err := token.Name(&bind.CallOpts{Pending: true})
	if err != nil {
		log.Fatalf("Failed to retrieve pending name: %v", err)
	}
	fmt.Println("Pending name:", name)
}
```

And the code performs as expected: it requests the creation of a brand new Token contract
on the Ethereum blockchain, which we can either wait for to be mined or as in the above code
start calling methods on it in the pending state :)

```
Contract pending deploy: 0x46506d900559ad005feb4645dcbb2dbbf65e19cc
Transaction waiting to be mined: 0x6a81231874edd2461879b7280ddde1a857162a744e3658ca7ec276984802183b

Pending name: Contracts in Go!!!
```

## Bind Solidity directly

If you've followed the tutorial along until this point you've probably realized that
every contract modification needs to be recompiled, the produced ABIs and bytecodes
(especially if you need multiple contracts) individually saved to files and then the
binding executed for them. This can become a quite bothersome after the Nth iteration,
so the `abigen` command supports binding from Solidity source files directly (`--sol`),
which first compiles the source code (via `--solc`, defaulting to `solc`) into it's
constituent components and binds using that.

Binding the official Token contract [`token.sol`](https://gist.github.com/karalabe/08f4b780e01c8452d989)
would then entail to running:

```
$ abigen --sol token.sol --pkg main --out token.go
```

*Note: Building from Solidity (`--sol`) is mutually exclusive with individually setting
the bind components (`--abi`, `--bin` and `--type`), as all of them are extracted from
the Solidity code and produced build results directly.*

Building a contract directly from Solidity has the nice side effect that all contracts
contained within a Solidity source file are built and bound, so if your file contains many
contract sources, each and every one of them will be available from Go code. The sample
Token solidity file results in [`token.go`](https://gist.github.com/karalabe/c22aab73194ba7da834ab5b379621031).

### Project integration (i.e. `go generate`)

The `abigen` command was made in such a way as to play beautifully together with existing
Go toolchains: instead of having to remember the exact command needed to bind an Ethereum
contract into a Go project, we can leverage `go generate` to remember all the nitty-gritty
details.

Place the binding generation command into a Go source file before the package definition:

```
//go:generate abigen --sol token.sol --pkg main --out token.go
```

After which whenever the Solidity contract is modified, instead of needing to remember and
run the above command, we can simply call `go generate` on the package (or even the entire
source tree via `go generate ./...`), and it will correctly generate the new bindings for us.

## Blockchain simulator

Being able to deploy and access already deployed Ethereum contracts from within native Go
code is an extremely powerful feature, but there is one facet with developing native code
that not even the testnet lends itself well to: *automatic unit testing*. Using go-ethereum
internal constructs it's possible to create test chains and verify them, but it is unfeasible
to do high level contract testing with such low level mechanisms.

To sort out this last issue that would make it hard to run (and test) native DApps, we've also
implemented a *simulated blockchain*, that can be set as a backend to native contracts the same
way as a live RPC backend could be: `backends.NewSimulatedBackend(genesisAccounts)`.

```go
package main

import (
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	// Generate a new random account and a funded simulator
	key, _ := crypto.GenerateKey()
	auth := bind.NewKeyedTransactor(key)

	sim := backends.NewSimulatedBackend(core.GenesisAccount{Address: auth.From, Balance: big.NewInt(10000000000)})

	// Deploy a token contract on the simulated blockchain
	_, _, token, err := DeployMyToken(auth, sim, new(big.Int), "Simulated blockchain tokens", 0, "SBT")
	if err != nil {
		log.Fatalf("Failed to deploy new token contract: %v", err)
	}
	// Print the current (non existent) and pending name of the contract
	name, _ := token.Name(nil)
	fmt.Println("Pre-mining name:", name)

	name, _ = token.Name(&bind.CallOpts{Pending: true})
	fmt.Println("Pre-mining pending name:", name)

	// Commit all pending transactions in the simulator and print the names again
	sim.Commit()

	name, _ = token.Name(nil)
	fmt.Println("Post-mining name:", name)

	name, _ = token.Name(&bind.CallOpts{Pending: true})
	fmt.Println("Post-mining pending name:", name)
}
```

And the output (yay):

```
Pre-mining name: 
Pre-mining pending name: Simulated blockchain tokens
Post-mining name: Simulated blockchain tokens
Post-mining pending name: Simulated blockchain tokens
```

Note, that we don't have to wait for a local private chain miner, or testnet miner to
integrate the currently pending transactions. When we decide to mine the next block,
we simply `Commit()` the simulator.
