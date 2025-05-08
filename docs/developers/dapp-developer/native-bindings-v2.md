---
title: Go Contract Bindings (v2)
description: Introduction to generating contract bindings with abigen v2
---


This page introduces the concept of server-side native dapps. Geth provides the tools required to generate [Go](https://github.com/golang/go/wiki#getting-started-with-go) language bindings to any Ethereum contract that is compile-time type-safe, highly performant, and can be generated completely automatically from a compiled contract.

Interacting with a contract on the Ethereum blockchain from Go is already possible via the RPC interfaces exposed by Ethereum clients. However, writing boilerplate code that translates Go language constructs to and from ABI-encoded packed data used by contract methods is time-consuming and brittle - implementation bugs can only be detected during runtime, and it's almost impossible to evolve a contract as even a tiny change in Solidity is awkward to port over to Go. Therefore, Geth provides tools for easily converting contract code into Go code that can be used directly in Go applications.



This page provides an introduction to generating Go contract bindings and using them in a simple Go application.

## Prerequisites {#prerequisites}

This page is fairly beginner-friendly and designed for people starting out with writing Go native dapps. The core concepts will be introduced gradually as a developer would encounter them. However, some basic familiarity with [Ethereum](https://ethereum.org), [Solidity](https://docs.soliditylang.org/en/v0.8.15/) and [Go](https://go.dev/) is assumed.

## What is an ABI? {#what-is-an-abi}

Ethereum smart contracts have a schema that defines its functions and returns types as a JSON file. This JSON file is known as an _Application Binary Interface_, or ABI. The ABI acts as a specification for precisely how to encode data sent to a contract and how to decode the data the contract sends back. The ABI is the only essential piece of information required to generate Go bindings. Go developers can then use the bindings to interact with the contract from their Go application without having to deal directly with data encoding and decoding. An ABI is generated when a contract is compiled.

## Abigen: Go binding generator {#abigen}

Geth includes a source code generator called `abigen` that can convert Ethereum ABI definitions into easy-to-use, type-safe Go packages. With a valid Go development environment set up and the go-ethereum repository checked out correctly, `abigen` can be built as follows:

```sh
go install github.com/ethereum/go-ethereum/cmd/abigen@latest
```

### Generating the bindings {#generating-bindings}

A contract is required to demonstrate the binding generator. The contract `Storage.sol` implements two very simple functions: `store` updates a user-defined `uint256` to the contract's storage, and `retrieve` displays the value stored in the contract to the user. The Solidity code is as follows:

```solidity
// SPDX-License-Identifier: GPL-3.0

pragma solidity >0.7.0 < 0.9.0;
/**
* @title Storage
* @dev store or retrieve a variable value
*/

contract Storage {

	uint256 value;

	function store(uint256 number) public{
		value = number;
	}

	function retrieve() public view returns (uint256){
		return value;
	}
}
```

This contract can be pasted into a text file and saved as `Storage.sol`.

The following code snippet shows how an ABI can be generated for `Storage.sol` using the Solidity compiler `solc`.

```shell
solc --combined-json abi,bin Storage.sol > Storage.abi
```

The ABI can also be generated in other ways such as using the `compile` commands in development frameworks such as [Foundry](https://book.getfoundry.sh/), [Hardhat](https://hardhat.org/) and [Brownie](https://eth-brownie.readthedocs.io/en/stable/) or in the online IDE [Remix](https://remix.ethereum.org/). ABIs for existing verified contracts can be downloaded from [Etherscan](https://etherscan.io/).

The "combined json" for `Storage.sol` (ABI definition and deployer bytecode in `Storage.abi`) looks as follows:

```json
{
  "contracts": {
    "Storage.sol:Storage": {
      "abi": [
        {
          "inputs": [],
          "name": "retrieve",
          "outputs": [
            {
              "internalType": "uint256",
              "name": "",
              "type": "uint256"
            }
          ],
          "stateMutability": "view",
          "type": "function"
        },
        {
          "inputs": [
            {
              "internalType": "uint256",
              "name": "number",
              "type": "uint256"
            }
          ],
          "name": "store",
          "outputs": [],
          "stateMutability": "nonpayable",
          "type": "function"
        }
      ],
      "bin": "6080604052348015600e575f5ffd5b506101298061001c5f395ff3fe6080604052348015600e575f5ffd5b50600436106030575f3560e01c80632e64cec11460345780636057361d14604e575b5f5ffd5b603a6066565b60405160459190608d565b60405180910390f35b606460048036038101906060919060cd565b606e565b005b5f5f54905090565b805f8190555050565b5f819050919050565b6087816077565b82525050565b5f602082019050609e5f8301846080565b92915050565b5f5ffd5b60af816077565b811460b8575f5ffd5b50565b5f8135905060c78160a8565b92915050565b5f6020828403121560df5760de60a4565b5b5f60ea8482850160bb565b9150509291505056fea26469706673582212200857cb05506ae0a0b1cb7a5f2c713313775ab4ec0c0c8e02a0d5c9ce9dd77fd364736f6c634300081c0033"
    }
  },
  "version": "0.8.28+commit.7893614a.Darwin.appleclang"
}
```

The contract binding can then be generated by passing the ABI to `abigen` as follows:

```sh
$ abigen --v2 --combined-json build/Storage.abi --pkg main --type Storage --out Storage.go
```

Where the flags are:

- `--v2`: Mandatory flag to specify the newest version of the binding generator.
- `--combined-json`: Mandatory path to the contract combined json (ABI + deployer bytecode) to bind to
- `--pkg`: Mandatory Go package name to place the Go code into
- `--type`: Optional Go type name to assign to the binding struct
- `--out`: Optional output path for the generated Go source file (not set = stdout)

This will generate a type-safe Go binding for the Storage contract. The generated code will look something like the snippet below, the full version of which can be viewed [here](https://github.com/jwasinger/abigen2-examples/blob/main/example_1/Storage.go).

```go
package main

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = common.Big1
	_ = types.BloomLookup
	_ = abi.ConvertType
)

// StorageMetaData contains all meta data concerning the Storage contract.
var StorageMetaData = bind.MetaData{
	ABI:     "[{\"inputs\":[],\"name\":\"retrieve\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"number\",\"type\":\"uint256\"}],\"name\":\"store\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// Storage is an auto generated Go binding around an Ethereum contract.
type Storage struct {
	abi abi.ABI
}

// NewStorage creates a new instance of Storage.
func NewStorage() *Storage {
	parsed, err := StorageMetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &Storage{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *Storage) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewContractInstance(backend, addr, c.abi)
}

// Retrieve is a free data retrieval call binding the contract method 0x2e64cec1.
//
// Solidity: function retrieve() view returns(uint256)
func (storage *Storage) PackRetrieve() []byte {
	enc, err := storage.abi.Pack("retrieve")
	if err != nil {
		panic(err)
	}
	return enc
}

func (storage *Storage) UnpackRetrieve(data []byte) (*big.Int, error) {
...
}
```

Generated code contains all the bindings necessary to interact with the contract from a Go application using APIs that Geth already provides.  The following sections will demonstrate examples for common use-cases.

### Deploying contracts to Ethereum {#deploying-contracts}

In the previous section, the contract ABI was sufficient for generating the contract bindings from its ABI. However, deploying the contract requires some additional information in the form of the compiled bytecode.

The bytecode is obtained by running the compiler again but this passing the `--bin` flag, e.g.

```sh
solc --bin Storage.sol -o build
```

Then `abigen` can be run again, this time passing `Storage.bin`:

```sh
$ abigen --v2 --abi build/Storage.abi --pkg main --type Storage --out Storage.go --bin build/Storage.bin
```

This will generate something similar to the bindings generated in the previous section. However, the contract metadata object now has the deployer bytecode embedded:

```go
// StorageMetaData contains all meta data concerning the Storage contract.
var StorageMetaData = bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"retrieve\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"number\",\"type\":\"uint256\"}],\"name\":\"store\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	ID:  "e338b353ed1163ea35b3bea455d45b50d7",
	Bin: "0x6080604052348015600e575f5ffd5b506101298061001c5f395ff3fe6080604052348015600e575f5ffd5b50600436106030575f3560e01c80632e64cec11460345780636057361d14604e575b5f5ffd5b603a6066565b60405160459190608d565b60405180910390f35b606460048036038101906060919060cd565b606e565b005b5f5f54905090565b805f8190555050565b5f819050919050565b6087816077565b82525050565b5f602082019050609e5f8301846080565b92915050565b5f5ffd5b60af816077565b811460b8575f5ffd5b50565b5f8135905060c78160a8565b92915050565b5f6020828403121560df5760de60a4565b5b5f60ea8482850160bb565b9150509291505056fea2646970667358221220708ec609f9c9f0df1ca6db6712f37455e91bc9f302d896f19f530f12bb6db51864736f6c634300081e0033",
}
```

View the full file [here](https://github.com/jwasinger/abigen2-examples/blob/main/example_1/Storage.go).

The new `StorageMetaData` can can be used to deploy the contract to an Ethereum testnet from a Go application. To do this requires incorporating the bindings into a Go application that also handles account management, authorization and Ethereum backend to deploy the contract through. Specifically, this requires:

1. A running Geth node connected to an Ethereum testnet
2. An account in the keystore prefunded with enough ETH to cover gas costs for deploying and interacting with the contract

Assuming these prerequisites exist, a new `ethclient` can be instantiated with the local Geth node's ipc file, providing access to the testnet from the Go application. The key can be instantiated as a variable in the application by copying the JSON object from the keyfile in the keystore.

Putting it all together would result in:

```go
package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/ethclient"
	"io"
	"os"
	"path"
	"strings"
)

// NOTE: do not EVER embed secrets in the source code like this in production code
var (
	key = "{\"address\":\"85755d82a3adc23598b887a6c33a2508b4b71e5a\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"ciphertext\":\"9631b0f23356c128bb8116207fda102951ba802da2a4e6f4cc1fc4f80c5e424a\",\"cipherparams\":{\"iv\":\"d3c8ceabd915abd50e85deafe15979aa\"},\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":262144,\"p\":1,\"r\":8,\"salt\":\"217cae8c2d765f3d6bfdbab3a8f145364fd2adf10183e9fbaea4b3ffb00404d9\"},\"mac\":\"4ccf3a9847f41af5ae15d0c67a3929e3375476f8b1982ff24041c810d7aa76ab\"},\"id\":\"ee4c195b-c2ea-4951-9f33-28502fa85734\",\"version\":3}"
	passphrase = "asdfasdfasdf"
)

func main() {
	// Create an IPC-based RPC connection to a remote node and an authorized transactor
	// NOTE update the path to the ipc file!
	conn, err := ethclient.Dial("/home/go-ethereum/sepolia/geth.ipc")
	if err != nil {
		panic(fmt.Errorf("Failed to connect to the Ethereum client: %v", err))
	}
	// Retrieve the current chain ID
	chainID, err := conn.ChainID(context.Background())
	if err != nil {
		panic(fmt.Errorf("Failed to retrieve chain ID: %v", err))
	}

	// create auth for tx signing from the key on disk
	json, err := io.ReadAll(strings.NewReader(key))
	if err != nil {
		panic(fmt.Errorf("failed to read key: %v", err))
	}
	key, err := keystore.DecryptKey(json, passphrase)
	if err != nil {
		panic(fmt.Errorf("failed to decrypt key: %v", err))
	}
	auth := bind.NewKeyedTransactor(key.PrivateKey, chainID)

	// set up params to deploy an instance of the Storage contract
	deployParams := bind.DeploymentParams{
		Contracts: []*bind.MetaData{&StorageMetaData},
	}

	// use the default deployer: it simply creates, signs and submits the deployment transactions
	deployer := bind.DefaultDeployer(auth, conn)

	// create and submit the contract deployment
	deployRes, err := bind.LinkAndDeploy(&deployParams, deployer)
	if err != nil {
		panic(fmt.Errorf("error submitting contract: %v", err))
	}

	address, tx := deployRes.Addresses[StorageMetaData.ID], deployRes.Txs[StorageMetaData.ID]

	fmt.Printf("contract pending deploy: 0x%x\n", address)
	fmt.Printf("transaction waiting to be mined: 0x%x\n", tx.Hash())

	// create a BoundContract instance to interact with the pending contract
	storageABI, _ := StorageMetaData.ParseABI()
	contract := Storage{*storageABI}
	instance := contract.Instance(conn, address)

	// perform an eth_call on the pending contract
	val, err := bind.Call(instance, &bind.CallOpts{Pending: true}, contract.PackRetrieve(), contract.UnpackRetrieve)
	if err != nil {
		panic(fmt.Errorf("call returned error: %v", err))
	}
	fmt.Printf("call to method retrieve returned result: %d\n", val)

	// wait for the pending contract to be deployed on-chain
	if _, err := bind.WaitDeployed(context.Background(), conn, tx.Hash()); err != nil {
		panic(fmt.Errorf("failed waiting for contract deployment: %v", err))
	}
	fmt.Println("contract deployed successfully")
}
```

Running this code requests the creation of a brand new `Storage` contract on the Sepolia testnet. The contract functions can be called while the contract is waiting to be included in a block.

```sh
contract pending deploy: 0x98c54be290f1b8446aff970e2d9489466b03122e
transaction waiting to be mined: 0x8d61d260a81696d3b54e9ae55ce4db5c5df99af4d9d1c26c139bccc6db4663af
call to method retrieve returned result: 0
contract deployed successfully
```

Once the contract deployment has been included in a validated block, the contract exists permanently at its deployment address and can now be interacted with from other applications without ever needing to be redeployed.

### Accessing an Ethereum contract {#accessing-contracts}

To interact with a contract already deployed on the blockchain, the deployment `address` of `Storage.sol` is required and a `backend` through which to access Ethereum must be defined.

As in the previous section, a Geth node running on an Ethereum testnet (recommend Sepolia) and an account with some test ETH to cover gas are required.

Again, an instance of `ethclient` can be created, passing the path to Geth's ipc file. In the example below this backend is assigned to the variable `conn`.

```go
// Create an IPC-based RPC connection to a remote node
// NOTE update the path to the ipc file!
conn, err := ethclient.Dial("/home/go-ethereum/sepolia/geth.ipc")
if err != nil {
	panic(fmt.Errorf("Failed to connect to the Ethereum client: %v", err))
}
```

To interact with the contract, we instantiate a `Storage` instance which which holds the ABI, and use this together with `conn` and `address` to create a `BoundContract`:


```go
storageABI, _ := StorageMetaData.ParseABI()
contract := Storage{*storageABI}
instance := contract.Instance(conn, address)
```

In the EVM, invocation of a Solidity contract method consists of a call to the contract with ABI-encoded call data that specifies which method to execute and what the inputs are.

`Storage.go` provides functions for each contract method that craft this call data.  For contract methods that take inputs, these methods handle encoding of the Go inputs. If a contract method returns values, `Storage.go` defines a method to decode the raw output to Go types. 


The `Call` method provided by the `bind` package requires an additional parameter.  This is a `*bind.CallOpts` type, which can be used to fine-tune the call. If no adjustments to the call are required, pass `nil`. Adjustments to the call include:

- `Pending`: Whether to access the pending contract state or the current stable one
- `GasLimit`: Place a limit on the computing resources the call might consume

So to call the `Retrieve()` function in the Go application:

```go
callOpts := bind2.CallOpts{Pending: true}
val, err := bind2.Call(instance, &callOpts, contract.PackRetrieve(), contract.UnpackRetrieve)
if err != nil {
    panic(fmt.Errorf("call returned error: %v", err))
}
fmt.Printf("value: ", value)
```

The output will be something like:

```terminal
value: 0
```

### Transacting with an Ethereum contract {#transacting-with-contract}

Invoking a method that changes contract state (i.e. transacting) is a bit more involved, as a live transaction needs to be authorized and broadcast into the network.

Thus, to allow transacting with a contract, your code needs to implement a method that given an input transaction, signs it and returns an authorized output transaction. Since most users have their keys in the [Web3 Secret Storage](https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition) format, the `bind` package contains a small utility method (`bind.NewTransactor(keyjson, passphrase)`) that can create an authorized transactor from a key file and associated password, without the user needing to implement key signing themselves.

Changing the previous code snippet to update the value stored in the contract:

```go
package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"io"
	"math/big"
	"os"
	"path"
	"strings"
)

// NOTE: do not EVER embed secrets in the source code like this in production code
var (
	key        = "{\"address\":\"85755d82a3adc23598b887a6c33a2508b4b71e5a\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"ciphertext\":\"9631b0f23356c128bb8116207fda102951ba802da2a4e6f4cc1fc4f80c5e424a\",\"cipherparams\":{\"iv\":\"d3c8ceabd915abd50e85deafe15979aa\"},\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":262144,\"p\":1,\"r\":8,\"salt\":\"217cae8c2d765f3d6bfdbab3a8f145364fd2adf10183e9fbaea4b3ffb00404d9\"},\"mac\":\"4ccf3a9847f41af5ae15d0c67a3929e3375476f8b1982ff24041c810d7aa76ab\"},\"id\":\"ee4c195b-c2ea-4951-9f33-28502fa85734\",\"version\":3}"
	passphrase = "asdfasdfasdf"
	address    = common.HexToAddress("0x98c54BE290f1B8446afF970e2D9489466b03122e")
)

func main() {
	// Create an IPC-based RPC connection to a remote node
	// NOTE update the path to the ipc file!
	conn, err := ethclient.Dial("/home/go-ethereum/sepolia/geth.ipc")
	if err != nil {
		panic(fmt.Errorf("Failed to connect to the Ethereum client: %v", err))
	}
	// Retrieve the current chain ID
	chainID, err := conn.ChainID(context.Background())
	if err != nil {
		panic(fmt.Errorf("failed to retrieve chain ID: %v", err))
	}
    
	// create auth for tx signing from the key on disk
	json, err := io.ReadAll(strings.NewReader(key))
	if err != nil {
		panic(fmt.Errorf("failed to read key: %v", err))
	}
	key, err := keystore.DecryptKey(json, passphrase)
	if err != nil {
		panic(fmt.Errorf("failed to decrypt key: %v", err))
	}

	address := common.HexToAddress("0x98c54BE290f1B8446afF970e2D9489466b03122e")

	// create a BoundContract instance to interact with the pending contract
	storageABI, _ := StorageMetaData.ParseABI()
	contract := Storage{*storageABI}
	instance := contract.Instance(conn, address)

	// Create an authorized transactor
	auth := bind.NewKeyedTransactor(key.PrivateKey, chainID)

	// send a transaction which calls the store function
	tx, err := bind.Transact(instance, auth, contract.PackStore(big.NewInt(42069)))
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %v", err))
	}

	// wait for transaction inclusion
	if _, err := bind.WaitMined(context.Background(), conn, tx.Hash()); err != nil {
		panic(fmt.Errorf("error waiting for tx inclusion: %v", err))
	}

	fmt.Println("transaction invoking store method was successfully included")
}
```

Unlike the method invocations in the previous section which only read contract state, transacting methods require a mandatory authorization parameter, a `*bind.TransactOpts` type, which authorizes the transaction and potentially fine-tunes it:

- `From`: Address of the account to invoke the method with (mandatory)
- `Signer`: Method to sign a transaction locally before broadcasting it (mandatory)
- `Nonce`: Account nonce to use for the transaction ordering (optional)
- `GasLimit`: Place a limit on the computing resources the call might consume (optional)
- `GasPrice`: Explicitly set the gas price to run the transaction with (optional)
- `Value`: Any funds to transfer along with the method call (optional)

The two mandatory fields are automatically set by the `bind` package if the auth options are constructed using `bind.NewTransactor`. The nonce and gas related fields are automatically derived by the binding if they are not set. Unset values are assumed to be zero.

### Project integration (`go generate`) {#project-integration}

The `abigen` command was designed to integrate easily into existing Go toolchains: instead of having to remember the exact command needed to bind an Ethereum contract to a Go project, `go generate` can handle all the fine details.

Place the binding generation command into a Go source file before the package definition:

```go
//go:generate abigen --v2 --sol Storage.sol --pkg main --out Storage.go
```

After that, whenever the Solidity contract is modified, instead of remembering and running the above command, we can simply call `go generate` on the package (or even the entire source tree via `go generate ./...`), and it will correctly generate the new bindings for us.

## Blockchain simulator {#blockchain-simulator}

Being able to deploy and access deployed Ethereum contracts from native Go code is a powerful feature. However, using public testnets as a backend does not lend itself well to _automated unit testing_. Therefore, Geth also implements a _simulated blockchain_ that can be set as a backend to native contracts like a live RPC backend. The code snippet below shows how this can be used as a backend in a Go application.

```go
package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
)

// NOTE: do not EVER embed secrets in the source code like this in production code
var (
	key = "{\"address\":\"85755d82a3adc23598b887a6c33a2508b4b71e5a\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"ciphertext\":\"9631b0f23356c128bb8116207fda102951ba802da2a4e6f4cc1fc4f80c5e424a\",\"cipherparams\":{\"iv\":\"d3c8ceabd915abd50e85deafe15979aa\"},\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":262144,\"p\":1,\"r\":8,\"salt\":\"217cae8c2d765f3d6bfdbab3a8f145364fd2adf10183e9fbaea4b3ffb00404d9\"},\"mac\":\"4ccf3a9847f41af5ae15d0c67a3929e3375476f8b1982ff24041c810d7aa76ab\"},\"id\":\"ee4c195b-c2ea-4951-9f33-28502fa85734\",\"version\":3}"
	passphrase = "asdfasdfasdf"
)

func main() {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Errorf("failed to generate key: %v", err))
	}

	// Since we are using a simulated backend, we will get the chain ID
	// from the same place that the simulated backend gets it.
	chainID := params.AllDevChainProtocolChanges.ChainID

	auth := bind.NewKeyedTransactor(key, chainID)

	sim := simulated.NewBackend(map[common.Address]types.Account{
		auth.From: {Balance: big.NewInt(9e18)},
	})

	// set up params to deploy an instance of the Storage contract
	deployParams := bind.DeploymentParams{
		Contracts: []*bind.MetaData{&StorageMetaData},
	}

	// use the default deployer: it simply creates, signs and submits the deployment transactions
	deployer := bind.DefaultDeployer(auth, sim.Client())

	// create and submit the contract deployment
	deployRes, err := bind.LinkAndDeploy(&deployParams, deployer)
	if err != nil {
		panic(fmt.Errorf("error submitting contract: %v", err))
	}

	address, tx := deployRes.Addresses[StorageMetaData.ID], deployRes.Txs[StorageMetaData.ID]

	// call Commit to make the simulated backend mine a block
	sim.Commit()

	// wait for the pending contract to be deployed on-chain
	if _, err := bind.WaitDeployed(context.Background(), sim.Client(), tx.Hash()); err != nil {
		panic(fmt.Errorf("failed waiting for contract deployment: %v", err))
	}
	fmt.Printf("contract deployed at address 0x%x\n", address)

	// create a BoundContract instance to interact with the pending contract
	storageABI, _ := StorageMetaData.ParseABI()
	contract := Storage{*storageABI}
	instance := contract.Instance(sim.Client(), address)

	// perform an eth_call on the pending contract
	val, err := bind.Call(instance, &bind.CallOpts{Pending: true}, contract.PackRetrieve(), contract.UnpackRetrieve)
	if err != nil {
		panic(fmt.Errorf("call returned error: %v", err))
	}
	fmt.Printf("call to method retrieve returned result: %d\n", val)
}
```

## Summary {#summary}

To make interacting with Ethereum contracts easier for Go developers, Geth provides tools that generate contract bindings automatically. This makes interacting with Ethereum contracts accessible from Go native applications.

The full, executable code for the examples shown in this tutorial are available [here](https://github.com/jwasinger/abigen2-examples).
