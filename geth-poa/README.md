# geth-poa

Tool for spinning up a POA ethereum sidechain bridged to goerli via [hyperlane](https://www.hyperlane.xyz/) token warp route.

## TODOs

* add container which stress tests network with txs
* merge branch with local anvil l1 once hyperlane v3 is released -> useful for mev-commit docker testing 

## Metrics

Metrics recorded by bootnode are exposed to host at http://127.0.0.1:6060/debug/metrics

## Key Summary

Testnet private keys are in notion, and need to be copied into an .env.

All relevant accounts are funded on sidechain genesis, you may need to fund these accounts on L1 with faucets. See [hyperlane docs](https://docs.hyperlane.xyz/docs/deploy/deploy-hyperlane#1.-setup-keys).

## Contract deployer

Address:    `0xBe3dEF3973584FdcC1326634aF188f0d9772D57D`

Note if the relayer is emitting errors related to unexpected contract routing, try using a new deployer key pair. It's likely that the current bridge contract deployments are clashing with previous deployments.   

You'd need to replace `Address` above, the allocs field of `genesis.json`, and the `CONTRACT_DEPLOYER_PRIVATE_KEY` in `.env`.

## Validator Accounts (also POA signers)

### Node1

Address:     `0xd9cd8E5DE6d55f796D980B818D350C0746C25b97`

### Node2

Address:     `0x788EBABe5c3dD422Ef92Ca6714A69e2eabcE1Ee4`

## Relayer

Address:     `0x0DCaa27B9E4Db92F820189345792f8eC5Ef148F6`

## Create2 Deployment Proxy

A Create2 deployment proxy is can be deployed to this chain at `0x4e59b44847b379578588920ca78fbf26c0b4956c`. see more [here](https://github.com/primevprotocol/deterministic-deployment-proxy). Note this proxy is required to deploy the whitelist bridge contract, and is consistent to foundry's suggested process for create2 deployment. The deployment signer, `0x3fab184622dc19b6109349b94811493bf2a45362` is funded on genesis.

## Local Run

1. To run the local setup, set the .env file with the keys specified in .env.example.
2. Run `$ make up-dev-build` to run the whole stack including bridge, or `$ make up-dev-settlement` to bring up only the settlement layer.

## Starter .env file
To get a standard starter .env file from primev internal development, [click here.](https://www.notion.so/Private-keys-and-env-for-settlement-layer-245a4f3f4fe040a7b72a6be91131d9c2?pvs=4)
