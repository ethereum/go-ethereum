---
title: Node architecture
description: Introduction to how Ethereum nodes are organized and where Geth fits.
---

## Node architecture

Geth is an [execution client](https://ethereum.org/en/developers/docs/nodes-and-clients/#execution-clients). Originally, an execution client alone was enough to run a full Ethereum node. However, ever since Ethereum turned off [proof-of-work](https://ethereum.org/en/developers/docs/consensus-mechanisms/pow/) and implemented [proof-of-stake](https://ethereum.org/en/developers/docs/consensus-mechanisms/pow/), Geth has needed to be coupled to another piece of software called a [“consensus client”](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients) in order to keep track of the Ethereum blockchain.

The execution client (Geth) is responsible for transaction handling, transaction gossip, state management and supporting the Ethereum Virtual Machine ([EVM])(https://ethereum.org/en/developers/docs/evm/). However, Geth is **not** responsible for block building, block gossiping or handling consensus logic. These are in the remit of the consensus client.

The relationship between the two Ethereum clients is shown in the schematic below. The two clients each connect to their own respective peer-to-peer (P2P) networks. This is because the execution clients gossip transactions over their P2P network enabling them to manage their local transaction pool. The consensus clients gossip blocks over their P2P network, enabling consensus and chain growth.

![node-architecture](/images/docs/node_architecture.png)

For this two-client structure to work, consensus clients must be able to pass bundles of transactions to Geth to be executed. Executing the transactions locally is how the client validates that the transactions do not violate any Ethereum rules and that the proposed update to Ethereum’s state is correct. Likewise, when the node is selected to be a block producer the consensus client must be able to request bundles of transactions from Geth to include in the new block. This inter-client communication is handled by a local RPC connection using the [engine API](https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md).

## What does Geth do?

As an execution client, Geth is responsible for creating the execution payloads - the bundles of transactions - that consensus clients include in their blocks. Geth is also responsible for re-executing transactions that arrive in new blocks to ensure they are valid. Executing transactions is done on Geth's embedded computer, known as the Ethereum Virtual Machine (EVM).

Geth also offers a user-interface to Ethereum by exposing a set of RPC methods that enable users to query the Ethereum blockchain, submit transactions and deploy smart contracts using the command line, programmatically using Geth's built-in console, web3 development frameworks such as Hardhat and Truffle or via web-apps and wallets.

In summary, Geth is: - a user gateway to Ethereum - home to the Ethereum Virtual Machine, Ethereum's state and transaction pool.

Read more about [proof-of-stake](https://ethereum.org/en/developers/docs/consensus-mechanisms/pos/).
