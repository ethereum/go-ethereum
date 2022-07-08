---
title: The Merge
sort_key: A2
---

The Merge is probably the most significant upgrade to Ethereum since Homestead. This page explains what The Merge is
and how it will affect Geth users. Instructions on how to configure an Ethereum node in
anticipation of The Merge are provided on the [Consensus Clients page](/docs/interface/consensus-clients).

## What is The Merge

[The Merge](https://ethereum.org/en/upgrades/merge/) is an upcoming upgrade to Ethereum that swaps out its current
proof-of-work (PoW) consensus mechanism for a proof-of-stake (PoS) mechanism. The term "Merge" refers
to the fact that the initial phase of the transition involved launching a PoS chain in parallel with
the existing PoW Ethereum Mainnet. This PoS chain, known as the Beacon Chain, has been executing the PoS
consensus logic since November 2020 but without integrating real Ethereum transaction data. The Merge refers
to the moment when Ethereum's existing blockchain and the Beacon Chain join together to form one unified chain.
At the moment of the merge, execution clients will switch off their proof-of-work and block gossiping functions 
and hand responsibility for all consensus and fork choice logic over to consensus clients. This is a profound
change to how Ethereum operates and it will require node operators to run a consensus client in addition to
Geth.

## How will Geth change?

Geth is an execution client. Historically, an execution client alone has been enough to run a full Ethereum node.
However, when The Merge happens, Geth will not be able to track the Ethereum chain on its own. Instead, it will need to 
be coupled to another piece of software called a ["consensus client"][con-client-link]. In this configuration, 
the execution client will be responsible for transaction handling, transaction gossip, state management and
the Ethereum Virtual Machine (EVM). However, Geth will no longer be responsible for block building, block
gossiping or handling consensus logic. These will be in the remit of the consensus client.

For Geth users that intend to continue to run full nodes after The Merge, it is sensible to start running
a consensus client in advance, so that The Merge can happen smoothly. There are five consensus clients available, all
of which connect to Geth in the same way. Instructions for this are available on the 
[Consensus Clients page](/docs/interface/consensus-clients).


## Client architecture

The relationship between the two Ethereum clients is shown in the schematic below. The two clients each connect
to their own respective peer-to-peer (P2P) networks. This is because the execution clients gossip transactions over
their P2P network enabling them to manage their local mempool. The consensus clients gossip blocks over their P2P
network, enabling consensus and chain growth.

![Client schematic](/static/images/client-architecture.png)

For this two-client structure to work, consensus clients must be able to pass bundles of transactions to Geth
to be executed. Executing the transactions locally is how the client validates that the transactions do not
violate any Ethereum rules and that the proposed update to Ethereum's state is correct. Likewise, when the node
is selected to be a block producer the consensus client must be able to request bundles of transactions from Geth to
include in the new block. This inter-client communication is handled by a local RPC connection using the 
[`engine` API][engine-api-link] which is part of the JSON-RPC API exposed by Geth. 

## Transition

The transition from PoW to PoS will happen when a pre-announced total difficulty is reached by the chain. 
This is different to other forks which usually occur at a certain scheduled block number. The total difficulty threshold 
that will trigger the Merge is also known as the [*Terminal
Total Difficulty* (TTD)](https://ethereum.org/en/glossary/#terminal-total-difficulty). In
case of an emergency delay, the TTD can be overriden in Geth using the `--override.terminaltotaldifficulty` command-line
flag. Once the merge block exists, Geth will no longer be able to follow the head of the chain without a consensus
client. If Geth is not connected to a consensus client it will simply stall at the merge block. 
Assuming a consensus client is connected in advance, the two clients will automatically
handle the merge together with no disruption to the user.

## Using Geth after The Merge

Although The Merge is a profound change to Ethereum's underlying achitecture, there will be minimal changes to how Geth
users interact with Ethereum. At The Merge responsibility for consensus logic and block propagation are handed over to 
the consensus layer, but all of Geth’s other functionality remains intact. This means transactions, contract deployments 
and data queries can still be handled by Geth using the same methods as before. This includes interacting with Geth using
the JSON_RPC_API directly using tools such as [curl](https//curl.se), third party libraries such as
[Web3.js][web3js-link] or [Web3.py][web3py-link] in development frameworks, e.g. [Truffle][truffle-link], [Hardhat][hardhat-link],
[Brownie][brownie-link], [Foundry][foundry-link] or using Web3.js in Geth's built-in Javascript console.
Much more information about the Javascript console is available on the [Javascript Console page](/docs/interface/javascript-console).

## Summary

The Merge is an upcoming upgrade to Ethereum that swaps the existing PoW consensus mechanism for a PoS consensus
mechanism. This changes the client software organization for Ethereum nodes. After The Merge, nodes are required to
run both execution and consensus clients that each have responsibility for specific parts of the Ethereum protocol
and communicate with each other over a local RPC connection. Although there is some necessary configuration in advance
of The Merge, the experience for Geth users should change minimally as a result of The Merge.

[engine-api-link]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md
[cl-list]: https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients
[web3py-link]: https://web3py.readthedocs.io/en/stable/web3.main.html
[web3js-link]: https://web3js.readthedocs.io/en/v1.2.9/
[brownie-link]: https://eth-brownie.readthedocs.io/en/stable/
[truffle-link]: https://trufflesuite.com/
[hardhat-link]: https://hardhat.org/
[foundry-link]: https://github.com/foundry-rs/foundry)
[con-client-link]:https://ethereum.org/en/glossary/#consensus-client
