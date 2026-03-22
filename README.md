# Lime Chain DevOps Take Home Task Solution Documentation

Expected initial rough time to finish the task - 1 week.

Sub tasks should take several hours each - but I lack some terminology and must find good/best practices for deployment and understand how the blockchain and the go-ethereum project works.

Original README.md file is [README_MAIN.md](./README_MAIN.md)

## Requirements based on research

Running local devnet:
- geth is an execution clinet and requires a consensus client to work - https://geth.ethereum.org/docs/getting-started/consensus-clients
    - geth might also require a genesis file/block to run?
    - does dev mode support all features of the mainnet?
    - additional information about Developer mode - https://geth.ethereum.org/docs/developers/dapp-developer/dev-mode
        - requires knowledge of Solidity and Smart Contract Deployment - https://docs.soliditylang.org/en/v0.8.35-pre.1/, https://ethereum.org/developers/tutorials/deploying-your-first-smart-contract/
        - geth also supports custom genesis configuration based on docs/guide `geth --dev dumpgenesis`
        - setting up a a whole devnet will require a lot of research and testing I will Keep it as simple as possible. TODO: Get back to this if I have /timeleft.

- consensus clients might require a validator client - https://ethereum.org/glossary/#consensus-client
    - consensus clients can also require a beacon chain?
    - Prism written in GO supports validator and beacon-chain - seems like it fits the requirements for the task, but how does it work? - https://prysm.offchainlabs.com/docs/install-prysm/install-with-docker/ 
- validator clients might require 32 eth to run



### Possible/Similar implementations:
- https://github.com/OffchainLabs/eth-pos-devnet/tree/master


## Phase 1 CI/CD GitHub Actions and Container Building
- Fork the go-ethereumn repo

- Set up GitHub actions with PR with label CI:Build
  - build new docker image of the given project - so we need a Dockerfile with multi-stage build
  - upload to a container registry
  - libc might not be required in the build for CGO_ENABLED - but we we must check and note this that we can do it.

- set up docker-compose that runs local devnet with the new image 

Readings:
- Artifact attestations - https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations

## Phase 2 

- Research hardhat
- Create Sample HardHat Project - following docs - I guess I dont have to learn the whole framework
- 



# Go Ethereum

Tasks:
- Build the image
- Set up docker compose for the local devnet

One of the tools that we can use based on docs is Kurtosis  that has this package: - https://github.com/ethpandaops/ethereum-package but the task requires us to run it in a docker-compose.yaml file.

The requirements to run a devnet based on research: 




TODO readings:
- https://geth.ethereum.org/docs/getting-started
- https://geth.ethereum.org/docs/fundamentals/node-architecture
- https://geth.ethereum.org/docs/fundamentals

MONITORING - https://geth.ethereum.org/docs/monitoring/dashboards


- Choose a stable release branch - v1.17.1

run a test build with the default provided image:

```bash
docker build -t go-etherium:1.17.1 -f Dockerfile .
```

Security:
- https://geth.ethereum.org/docs/fundamentals/security

```bash
# Block:
- 8545 # for the JSON-RPC requests
# Allow:
- TCP 30303
- UDP 30303
```
Exposing api endpoints require:
- proxies,
- WAFs/Firewall
- App-level filtering
- rate limits
- logging
- tls termination
- monitoring

All ports:
- 8545 TCP, used by the HTTP based JSON RPC API
- 8546 TCP, used by the WebSocket based JSON RPC API
- 8547 TCP, used by the GraphQL API
- 30303 TCP and UDP, used by the P2P protocol running the network




---

## New Terms and Tech I need to learn


### go-ethereum
The official Go implementation of the Ethereum protocol, also known as geth. It includes a command-line interface and a library for building Ethereum applications in Go.

geth - The Etherium go client implementation
clef - signing tool for geth
devp2p - utility to interact with nodes on the networking layer without running a whole blockchain
abigen - source code generator to convert Ethereium contract definitions into easy-to-use, compile type-safe Go packages. Can also accept Solidity soruce files.
evm - developer utility to interact with the Ethereum Virtual Machine (EVM) without running a whole blockchain
rlpdump - dev utility tool to convert binary RLP(Recursive Length Prefix) dumps to user friendlier representation.

### DApps and Smart Contracts
DApps - decentralized applicaiton - can operate atonomously, typically thoruhg the use of smart contracts, that run on a blockchain or other dustributed legder system.

DApps use Smart contracts which are programs that run on the blockchain and execute operations. Multiple smart contracts can run one one DApp but in order to deploy them they need gas - which is the currency that is used for deploying and executing them. 

AN complex smart contract of a DAppp that operats on the Ethereum blockchain may fail to be deployed if it costs too much gas, leading to lower throughput and longer wait times for execution.

Operation:
- Dapps use consesus mechanisms over the network - proof-of-work(POW) and proof-of-stake(POS).
POW - Mining consensus - with computational power
POS - consensus mechanism that supports DApps through validatiors that secure the network by having a stake and a percent ownership over the application.

### Genesis Block

### Clients

- execution client
- consensus client
- validator client


- Consensus clients -  (such as Prysm, Teku, Nimbus, Lighthouse, Lodestar) run Ethereum's proof-of-stake consensus algorithm allowing the network to reach agreement about the head of the Beacon Chain. Consensus clients do not participate in validating/broadcasting transactions or executing state transitions. This is done by execution clients. Consensus clients do not attest to, or propose new blocks. This is done by the validator client which is an optional add-on to the consensus client.

Validator - A node in a proof-of-stake system responsible for storing data, processing transactions, and adding new blocks to the blockchain. To activate validator software, you need to be able to stake 32 ETH. More on staking in Ethereum.