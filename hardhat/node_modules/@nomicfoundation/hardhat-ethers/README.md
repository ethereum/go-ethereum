[![npm](https://img.shields.io/npm/v/@nomicfoundation/hardhat-ethers.svg)](https://www.npmjs.com/package/@nomicfoundation/hardhat-ethers) [![hardhat](https://hardhat.org/buidler-plugin-badge.svg?1)](https://hardhat.org)

# hardhat-ethers

[Hardhat](https://hardhat.org) plugin for integration with [ethers.js](https://github.com/ethers-io/ethers.js/).

## What

This plugin brings to Hardhat the Ethereum library `ethers.js`, which allows you to interact with the Ethereum blockchain in a simple way.

## Installation

```bash
npm install --save-dev @nomicfoundation/hardhat-ethers ethers
```

And add the following statement to your `hardhat.config.js`:

```js
require("@nomicfoundation/hardhat-ethers");
```

Or, if you are using TypeScript, add this to your `hardhat.config.ts`:

```js
import "@nomicfoundation/hardhat-ethers";
```

## Tasks

This plugin creates no additional tasks.

## Environment extensions

This plugins adds an `ethers` object to the Hardhat Runtime Environment.

This object has the [same API](https://docs.ethers.org/v6/single-page/) as `ethers.js`, with some extra Hardhat-specific functionality.

### Provider object

A `provider` field is added to `ethers`, which is an [`ethers.Provider`](https://docs.ethers.org/v6/single-page/#api_providers__Provider) automatically connected to the selected network.

### Helpers

These helpers are added to the `ethers` object:

```typescript
interface Libraries {
  [libraryName: string]: string;
}

interface FactoryOptions {
  signer?: ethers.Signer;
  libraries?: Libraries;
}

function deployContract(name: string, constructorArgs?: any[], signer?: ethers.Signer): Promise<ethers.Contract>;

function getContractFactory(name: string, signer?: ethers.Signer): Promise<ethers.ContractFactory>;

function getContractFactory(name: string, factoryOptions: FactoryOptions): Promise<ethers.ContractFactory>;

function getContractFactory(abi: any[], bytecode: ethers.utils.BytesLike, signer?: ethers.Signer): Promise<ethers.ContractFactory>;

function getContractAt(name: string, address: string, signer?: ethers.Signer): Promise<ethers.Contract>;

function getContractAt(abi: any[], address: string, signer?: ethers.Signer): Promise<ethers.Contract>;

function getSigners() => Promise<ethers.Signer[]>;

function getSigner(address: string) => Promise<ethers.Signer>;

function getImpersonatedSigner(address: string) => Promise<ethers.Signer>;

function getContractFactoryFromArtifact(artifact: Artifact, signer?: ethers.Signer): Promise<ethers.ContractFactory>;

function getContractFactoryFromArtifact(artifact: Artifact, factoryOptions: FactoryOptions): Promise<ethers.ContractFactory>;

function getContractAtFromArtifact(artifact: Artifact, address: string, signer?: ethers.Signer): Promise<ethers.Contract>;
```

The [`Contract`s](https://docs.ethers.org/v6/single-page/#api_contract__Contract) and [`ContractFactory`s](https://docs.ethers.org/v6/single-page/#api_contract__ContractFactory) returned by these helpers are connected to the first [signer](https://docs.ethers.org/v6/single-page/#api_providers__Signer) returned by `getSigners` by default.

## Usage

There are no additional steps you need to take for this plugin to work.

Install it and access ethers through the Hardhat Runtime Environment anywhere you need it (tasks, scripts, tests, etc). For example, in your `hardhat.config.js`:

```js
require("@nomicfoundation/hardhat-ethers");

// task action function receives the Hardhat Runtime Environment as second argument
task(
  "blockNumber",
  "Prints the current block number",
  async (_, { ethers }) => {
    const blockNumber = await ethers.provider.getBlockNumber();
    console.log("Current block number: " + blockNumber);
  }
);

module.exports = {};
```

And then run `npx hardhat blockNumber` to try it.

Read the documentation on the [Hardhat Runtime Environment](https://hardhat.org/hardhat-runner/docs/advanced/hardhat-runtime-environment) to learn how to access the HRE in different ways to use ethers.js from anywhere the HRE is accessible.

### Library linking

Some contracts need to be linked with libraries before they are deployed. You can pass the addresses of their libraries to the `getContractFactory` function with an object like this:

```js
const contractFactory = await this.env.ethers.getContractFactory("Example", {
  libraries: {
    ExampleLib: "0x...",
  },
});
```

This allows you to create a contract factory for the `Example` contract and link its `ExampleLib` library references to the address `"0x..."`.

To create a contract factory, all libraries must be linked. An error will be thrown informing you of any missing library.

## Troubleshooting

### Events are not being emitted

Ethers.js polls the network to check if some event was emitted (except when a `WebSocketProvider` is used; see below). This polling is done every 4 seconds. If you have a script or test that is not emitting an event, it's likely that the execution is finishing before the event is detected by the polling mechanism.

If you are connecting to a Hardhat node using a `WebSocketProvider`, events should be emitted immediately. But keep in mind that you'll have to create this provider manually, since Hardhat only supports configuring networks via http. That is, you can't add a `localhost` network with a URL like `ws://127.0.0.1:8545`.
