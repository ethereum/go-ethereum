---
title: Node architecture
description: Introduction to how Ethereum nodes are organized and where Geth fits.
---

An Ethereum node is composed of two clients: an [execution client](https://ethereum.org/en/developers/docs/nodes-and-clients/#execution-clients) and a [consensus client](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients). Geth is an [execution client](https://ethereum.org/en/developers/docs/nodes-and-clients/#execution-clients). Originally, an execution client alone was enough to run a full Ethereum node. However, ever since Ethereum turned off [proof-of-work](https://ethereum.org/en/developers/docs/consensus-mechanisms/pow/) and implemented [proof-of-stake](https://ethereum.org/en/developers/docs/consensus-mechanisms/pow/), Geth has needed to be coupled to another piece of software called a [“consensus client”](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients) in order to keep track of the Ethereum blockchain.

The execution client (Geth) is responsible for transaction handling, transaction gossip, state management and supporting the Ethereum Virtual Machine [EVM](https://ethereum.org/en/developers/docs/evm/). However, Geth is **not** responsible for block building, block gossiping or handling consensus logic. These are in the remit of the consensus client.

The relationship between the two Ethereum clients is shown in the schematic below. The two clients each connect to their own respective peer-to-peer (P2P) networks. This is because the execution clients gossip transactions over their P2P network enabling them to manage their local transaction pool. The consensus clients gossip blocks over their P2P network, enabling consensus and chain growth.

![node-architecture](/images/docs/node-architecture-text-background.png)

For this two-client structure to work, consensus clients must be able to pass bundles of transactions to Geth to be executed. Executing the transactions locally is how the client validates that the transactions do not violate any Ethereum rules and that the proposed update to Ethereum’s state is correct. Likewise, when the node is selected to be a block producer the consensus client must be able to request bundles of transactions from Geth to include in the new block. This inter-client communication is handled by a local RPC connection using the [engine API](https://github.com/ethereum/execution-apis/blob/main/src/engine/).

## What does Geth do? {#what-does-geth-do}

As an execution client, Geth is responsible for creating the execution payloads - the list of transactions, updated state trie plus other execution related data - that consensus clients include in their blocks. Geth is also responsible for re-executing transactions that arrive in new blocks to ensure they are valid. Executing transactions is done on Geth's embedded computer, known as the Ethereum Virtual Machine (EVM).

Geth also offers a user-interface to Ethereum by exposing a set of [RPC methods](/docs/interacting-with-geth/rpc/) that enable users to query the Ethereum blockchain, submit transactions and deploy smart contracts. Often, the RPC calls are abstracted by a library such as [Web3js](https://web3js.readthedocs.io/en/v1.8.0/) or [Web3py](https://web3py.readthedocs.io/en/v5/) for example in Geth's built-in Javascript console, development frameworks or web-apps.

## What does the consensus client do? {#consensus-client}

The consensus client deals with all the logic that enables a node to stay in sync with the Ethereum network. This includes receiving blocks from peers and running a fork choice algorithm to ensure the node always follows the chain with the greatest accumulation of attestations (weighted by validator effective balances). The consensus client has its own peer-to-peer network, separate from the network that connects execution clients to each other. The consensus clients share blocks and attestations over their peer-to-peer network. The consensus client itself does not participate in attesting to or proposing blocks - this is done by a validator which is an optional add-on to a consensus client. A consensus client without a validator only keeps up with the head of the chain, allowing the node to stay synced. This enables a user to transact with Ethereum using their execution client, confident that they are on the right chain.

## Validators {#validators}

Validators can be added to consensus clients if 32 ETH have been sent to the deposit contract. The validator client comes bundled with the consensus client and can be added to a node at any time. The validator handles attestations and block proposals. They enable a node to accrue rewards or lose ETH via penalties or slashing. Running the validator software also makes a node eligible to be selected to propose a new block.

Read more about [proof-of-stake](https://ethereum.org/en/developers/docs/consensus-mechanisms/pos/).
