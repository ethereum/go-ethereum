---
title: Puppeth
description: introduction to the private-network boot-strapping tool, Puppeth
---

<Note>Puppeth was [removed from Geth](https://github.com/ethereum/go-ethereum/pull/26581) in January 2023.</Note>

Puppeth was a tool for quickly spinning up and managing private development networks. The user was guided through the process by a command line wizard instead of having to configure the network manually. However, this tool has been discontinued and removed from the Geth repository. 

This page demonstrates how to start a private proof-of-authority network with all the nodes running on the local machine. Other configurations are also possible, for example nodes can be spread over multiple (virtual) machines and the consensus mechanism can be proof-of-work.

Instructions for setting up a private network using Ethash or Clique are available on our [private networks page](/docs/fundamentals/private-network.md).