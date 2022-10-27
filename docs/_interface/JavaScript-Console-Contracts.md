---
title: JavaScript Console 2 - Contracts
sort_key: D
---

The [Introduction to the Javascript console](src/pages/docs/interacting-with-geth/javascript-console.md) 
page outlined how a Javascript console can be attached to Geth to provide a more 
user-friendly interface to Ethereum than interacting directly with the JSON-RPC API. 
This page will describe how to deploy contracts and interact with contracts using 
the attached console. This page will assume the Javascript console is attached to 
a running Geth instance using IPC. Clef should be used to manage accounts. 

## Deploying a contract

First we need a contract to deploy. We can use the well-known `Storage.sol` contract 
written in Solidity. The following Solidity code can be copied and pasted into a text 
editor and saved as `go-ethereum/storage-contract/Storage.sol`.

```Solidity
// SPDX License-Identifier: GPL 3.0

pragma solidity ^0.8.0;

contract Storage{

    uint256 value = 5;
    
    function set(uint256 number) public{
        value = number;
    }

    function retrieve() public view returns (uint256){
        return value;
    }
}
```

The contract needs to be compiled before Geth can understand it. Compiling the 
contract creates an [Application Binary Interface](https://docs.soliditylang.org/en/v0.4.24/abi-spec.html) 
and the contract bytecode. This requires a Solidity compiler (e.g. `solc`) to be 
installed on the local machine. Then, compile and save the ABI and bytecode to a 
new `build` subdirectory using the following terminal commands:

```sh
cd ~/go-ethereum/storage-contract
solc --bin Storage.sol -o build
solc --abi Storage.sol -o build
```

The outputs look as follows:

Storage.bin:
```sh
608060405234801561001057600080fd5b50610150806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80632e64cec11461003b5780636057361d14610059575b600080fd5b610043610075565b60405161005091906100a1565b60405180910390f35b610073600480360381019061006e91906100ed565b61007e565b005b60008054905090565b8060008190555050565b6000819050919050565b61009b81610088565b82525050565b60006020820190506100b66000830184610092565b92915050565b600080fd5b6100ca81610088565b81146100d557600080fd5b50565b6000813590506100e7816100c1565b92915050565b600060208284031215610103576101026100bc565b5b6000610111848285016100d8565b9150509291505056fea264697066735822122031443f2fb748bdb27e539fdbeb0c6f575aec50508baaa7e4dbeb08577ef19b3764736f6c63430008110033
```

Storage.abi:
```json
[{"inputs":[],"name":"retrieve","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"number","type":"uint256"}],"name":"store","outputs":[],"stateMutability":"nonpayable","type":"function"}]
```

These are all the data required to deploy the contract using the Geth Javascript 
console. Open the Javascript console using `./geth attach geth.ipc`. 

Now, for convenice we can store the abi and bytecode in variables in the console:

```js
var abi = [{"inputs":[],"name":"retrieve","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"number","type":"uint256"}],"name":"store","outputs":[],"stateMutability":"nonpayable","type":"function"}]

var bytecode = "608060405234801561001057600080fd5b50610150806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80632e64cec11461003b5780636057361d14610059575b600080fd5b610043610075565b60405161005091906100a1565b60405180910390f35b610073600480360381019061006e91906100ed565b61007e565b005b60008054905090565b8060008190555050565b6000819050919050565b61009b81610088565b82525050565b60006020820190506100b66000830184610092565b92915050565b600080fd5b6100ca81610088565b81146100d557600080fd5b50565b6000813590506100e7816100c1565b92915050565b600060208284031215610103576101026100bc565b5b6000610111848285016100d8565b9150509291505056fea264697066735822122031443f2fb748bdb27e539fdbeb0c6f575aec50508baaa7e4dbeb08577ef19b3764736f6c63430008110033"
```

The ABI can be used to create an instance of the contract:

```js
var contract = eth.contract(abi)
```

This contract instance can then be deployed to the blockchain. This is done 
using `eth.sendTransaction`, passing the contract bytecode in the `data` field. 
For convenience we can create a transaction JSON object first, then pass it to 
`eth.sendTransaction` later. Let's use the first account in `eth.accounts` as the 
sender. The amount of gas to include can be determined using `eth.estimateGas`:

```js
var gas = eth.estimateGas({data: bytecode})
```

**Note that each command that touches accounts will require approval in Clef unless 
a custom rule has been implemented.**

The bytecode, gas and address of the sender can be bundled together into an object
that will be passed to the contract's `new()` method which deploys the contract.

```js
var tx = {'from': eth.accounts[0], data: bytecode, gas: gas}
var deployed_contract = contract.new(tx)
```

The transaction hash and deployment address can now been viewed in the console by 
entering the variable name (in this case `deployed_contract`):

```js
{
  abi:[{
    inputs: [],
    name: "retrieve",
    outputs: [{...}],
    stateMutability: "view",
    type: "function"
  },{
    inputs: [],
    name: "store",
    outputs: [{...}],
    stateMutability: "nonpayable",
    type: "function"
  }],
  address: "0x2d6505f8b1130a22a5998cd31788bf6c751247f",
  transactionHash: "0x5040a8916b23b76696ea9eba5b072546e1112cc481995219081fc86f5b911bf3",
  allEvents: function bound(),
  retrieve: function bound(),
  store: function bound()
}
```

Passing the transaction hash to `eth.getTransaction()` returns more detailed deployment 
transaction details. To interact with the contract, create an instance by passing the 
deployment address to `contract.at()` then call the methods.

```js
var instance = contract.at("0x2d6505f8b1130a22a5998cd31788bf6c751247f")
// store() alters the state and therefore requires sendTransaction()
contract.set.sendTransaction(42, {from: eth.accounts[0], gas: 1000000})
// retrieve does not alter state so it can be executed using call()
contract.retrieve().call()

>> 2
```

## Summary 

This page demonstrated how to create, compile, deploy and interact with an Ethereum 
smart contract using Geth's Javascript console.