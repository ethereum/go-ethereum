---
title: What is Geth
root: ..
---

## What is Geth?

Geth (go-ethereum) is a [Go](https://go.dev/) implementation of [Ethereum](http://ethereum.org) - a 
gateway into the decentralized web.

Running Geth alongside a consensus client turns a computer into an Ethereum node. 
Nodes communicate with one another, agreeing on the data they should each add to their local databases. 
Ethereum itself is the network of connected nodes running Ethereum software.


## Why run a node?

Running your own node enables you to use Ethereum in a truly private, self-sufficient and trustless 
manner. You don't need to trust information you receive because you can verify the data yourself 
using your Geth instance. 

**"Don't trust, verify"**

![node basic](/assets/node-basic.png)

Your node verifies all changes to its database by itself. This means: 

- you don’t have to trust any other nodes in the network.
- You never have to leak your addresses and balances to other nodes.
- You can use Ethereum securely and privately. Most wallet software can be pointed to your own local node.
- You can program your own custom RPC endpoints and make your own modifications to the source code.
- You get low latency, fast access to Ethereum.

A large and diverse set of nodes independently verifying new information is critical for Ethereum’s health, 
security and operational resiliency.

**If you run a full node, the whole Ethereum network benefits.**


## Node architecture

Geth is an [execution client](https://ethereum.org/en/developers/docs/nodes-and-clients/#execution-clients). 
Originally, an execution client alone was enough to run a full Ethereum node.
However, ever since Ethereum turned off proof-of-work and implemented proof-of-stake,
Geth must to be coupled to another piece of software called a 
[“consensus client”](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients).

The execution client is responsible for transaction handling, transaction gossip, state management and 
the Ethereum Virtual Machine (EVM). However, Geth is **not** responsible for block building, block gossiping 
or handling consensus logic. These are in the remit of the consensus client.

The relationship between the two Ethereum clients is shown in the schematic below. The two clients each 
connect to their own respective peer-to-peer (P2P) networks. This is because the execution clients gossip 
transactions over their P2P network enabling them to manage their local transaction pool. The consensus clients 
gossip blocks over their P2P network, enabling consensus and chain growth.

![node-architecture](/assets/node_architecture.png)

For this two-client structure to work, consensus clients must be able to pass bundles of transactions to 
Geth to be executed. Executing the transactions locally is how the client validates that the transactions 
do not violate any Ethereum rules and that the proposed update to Ethereum’s state is correct. Likewise, 
when the node is selected to be a block producer the consensus client must be able to request bundles of 
transactions from Geth to include in the new block. This inter-client communication is handled by a local 
RPC connection using the engine API which is part of the JSON-RPC API exposed by Geth.



## What does Geth do?

As an execution client, Geth is responsible for creating the execution payloads - the bundles of transactions -
that consensus clients include in their blocks. Geth is also responsible for re-executing transactions that arrive
in new blocks to ensure they are valid. Executing transactions is done on Geth's embedded computer, known as the 
Ethereum Virtual Machine (EVM).

Geth also offers a user-interface to Ethereum by exposing a set of RPC methods that enable users to query the 
Ethereum blockchain, submit transactions and deploy smart contracts using the command line, programmatically 
using Geth's built-in console, web3 development frameworks such as Hardhat and Truffle or via web-apps and wallets.

In summary, Geth is:
	- a user gateway to Ethereum 
	- home to the Ethereum Virtual Machine, Ethereum's state and transaction pool.





