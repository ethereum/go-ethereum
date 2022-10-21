---
title: The Merge
sort_key: A2
---

The Merge was probably the most significant upgrade to Ethereum since Homestead. This page explains what The Merge was
and how it affected Geth users.

## What was The Merge

[The Merge](https://ethereum.org/en/upgrades/merge/) was an upgrade to Ethereum that swapped out its original
proof-of-work (PoW) based consensus mechanism for a proof-of-stake based (PoS) mechanism. The term "Merge" refers
to the fact that the initial phase of the transition involved launching a PoS chain in parallel with the
PoW Ethereum Mainnet. That PoS chain, known as the Beacon Chain, was been executing the PoS consensus logic but 
without including any real Ethereum transaction data. The Merge was the moment when Ethereum's blockchain 
and the Beacon Chain joined together to form one unified chain. At the moment of The Merge, execution clients 
switched off their proof-of-work and block gossiping functions and handed responsibility for all consensus and 
fork choice logic over to consensus clients. This was a profound change to how Ethereum operates. Now that The Merge
is done, node operators are required to run a consensus client in addition to an execution client such as Geth.

## How did Geth change?

Geth is an execution client. Historically, running an execution client alone was enough to turn a computer 
into a full Ethereum node. However, since The Merge, Geth has not been able to track the Ethereum chain on 
its own. Instead, it needs to be coupled to another piece of software called a ["consensus client"][con-client-link]. 
The execution client is responsible for transaction handling, transaction gossip, state management and
the Ethereum Virtual Machine (EVM). However, Geth is no longer responsible for block proposals or 
handling consensus logic. These are in the remit of the consensus client.

There are five consensus clients available, all of which connect to Geth in the same way. 
Instructions for this are available on the [Consensus Clients page](/docs/interface/consensus-clients).


## Client architecture

The relationship between the two Ethereum clients is shown in the schematic below. The two clients each connect
to their own respective peer-to-peer (P2P) networks. This is because the execution clients gossip transactions over
their P2P network enabling them to manage their local mempool. The consensus clients gossip blocks over their P2P
network, enabling consensus and chain growth.

![Client schematic](/static/images/client-architecture.png)

For this two-client structure to work, consensus clients must be able to pass bundles of transactions to Geth
to be executed. Executing the transactions locally is how the client validates that the transactions do not
violate any Ethereum rules and that the proposed update to Ethereum's state is correct. Likewise, when the node
is selected to be a block producer the consensus client must be able to request executable data from Geth including bundles of transactions and metadata to
include in the new block and a resulting state change. This inter-client communication is handled by a local RPC 
connection using the [`engine` API][engine-api-link] which is part of the JSON-RPC API exposed by Geth. 

## Using Geth since The Merge

Although The Merge was a profound change to Ethereum's underlying achitecture, there were minimal changes to how Geth
users interact with Ethereum. At The Merge responsibility for consensus logic and block propagation were handed over to 
the consensus layer, but all of Geth’s other functionality remains intact. This means transactions, contract deployments 
and data queries can still be handled by Geth using the same methods as before. This includes interacting with Geth using
the JSON_RPC_API directly using tools such as [curl](https//curl.se), third party libraries such as
[Web3.js][web3js-link] or [Web3.py][web3py-link] in development frameworks, e.g. [Truffle][truffle-link], [Hardhat][hardhat-link],
[Brownie][brownie-link], [Foundry][foundry-link] or using Web3.js in Geth's built-in Javascript console.
Much more information about the Javascript console is available on the [Javascript Console page](/docs/interface/javascript-console).

## Summary

The Merge was an upgrade to Ethereum that swapped the original PoW based consensus mechanism for a PoS based consensus
mechanism. This changed the client software organization for Ethereum nodes. Since The Merge, both execution and consensus 
clients are required. Each client has responsibility for specific parts of the Ethereum protocol and communicate with each 
other over a local RPC connection.

[engine-api-link]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md
[cl-list]: https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients
[web3py-link]: https://web3py.readthedocs.io/en/stable/web3.main.html
[web3js-link]: https://web3js.readthedocs.io/en/v1.2.9/
[brownie-link]: https://eth-brownie.readthedocs.io/en/stable/
[truffle-link]: https://trufflesuite.com/
[hardhat-link]: https://hardhat.org/
[foundry-link]: https://github.com/foundry-rs/foundry)
[con-client-link]:https://ethereum.org/en/glossary/#consensus-client
