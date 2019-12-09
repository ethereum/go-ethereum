---
title: Private Network
sort_key: B
---

A private Ethereum network is useful for dapp testing, privacy, starting a local network for a hackathon, and many other purposes. An Ethereum network is private if the nodes are not connected to a main network.

Setting up your own private network can be a complex task, you need to configure a genesis block, different node types, miners, monitoring, and much more. To help get you started, `puppeth` is a CLI tool that leads you though the steps to create a private network, building it with Docker containers 


aids in creating a new Ethereum network down to the genesis, bootnodes, signers, ethstats, faucet, dashboard and more, without the hassle that it would normally take to configure all these services one by one. Puppeth uses ssh to dial into remote servers, and builds its network components out of docker containers using docker-compose. The user is guided through the process via a command line wizard that does the heavy lifting and topology configuration automatically behind the scenes.

Puppeth is not a magic bullet. If you have large in-house Ethereum deployments based on your own orchestration tools, it’s always better to use existing infrastructure. However, if you need to create your own Ethereum network without the fuss, Puppeth might actually help you do that… fast. Everything is deployed into containers, so it will not litter your system with weird packages. That said, it’s Puppeth’s first release, so tread with caution and try not to deploy onto critical systems.
