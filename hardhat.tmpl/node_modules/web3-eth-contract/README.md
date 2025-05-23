<p align="center">
  <img src="assets/logo/web3js.jpg" width="500" alt="web3.js" />
</p>

# web3.js - Eth Contract Package

![ES Version](https://img.shields.io/badge/ES-2020-yellow)
![Node Version](https://img.shields.io/badge/node-14.x-green)
[![NPM Package][npm-image]][npm-url]
[![Downloads][downloads-image]][npm-url]

This is a sub-package of [web3.js][repo].

`web3-eth-contract` contains the contract package used in `web3-eth`.

## Installation

You can install the package either using [NPM](https://www.npmjs.com/package/web3-eth-contract) or using [Yarn](https://yarnpkg.com/package/web3-eth-contract)

### Using NPM

```bash
npm install web3-eth-contract
```

### Using Yarn

```bash
yarn add web3-eth-contract
```

## Getting Started

-   :writing_hand: If you have questions [submit an issue](https://github.com/ChainSafe/web3.js/issues/new) or join us on [Discord](https://discord.gg/yjyvFRP)
    ![Discord](https://img.shields.io/discord/593655374469660673.svg?label=Discord&logo=discord)

## Prerequisites

-   :gear: [NodeJS](https://nodejs.org/) (LTS/Fermium)
-   :toolbox: [Yarn](https://yarnpkg.com/)/[Lerna](https://lerna.js.org/)

## Usage

You can initialize the typesafe Contract API instance with the following.

```ts
import { Contract } from 'web3-eth-contract';

const abi = [...] as const;

const contract = new Contract(abi);
```

-   We prefer that you use `web3.eth.Contract` API in normal usage.
-   The use of `as const` is necessary to have fully type-safe interface for the contract.
-   As the ABIs are not extensive in size, we suggest declaring them `as const` in your TS project.
-   This approach is more flexible and seamless compared to other approaches of off-line compiling ABIs to TS interfaces (such as [TypeChain](https://github.com/dethcrypto/TypeChain).

## Compatibility

We have tested the TypeScript interface support for the ABIs compiled with solidity version `v0.4.x` and above. If you face any issue regarding the contract typing, please create an issue to report to us.

The TypeScript support for fixed length array types are supported up 30 elements. See more details [here](https://github.com/ChainSafe/web3.js/blob/nh%2F4562-contract-typing/packages/web3-eth-abi/src/number_map_type.ts#L1). This limitation is only to provide more performant developer experience in IDEs. In future we may come up with a workaround to avoid this limitation. If you have any idea feel free to share.

## Package.json Scripts

| Script           | Description                                        |
| ---------------- | -------------------------------------------------- |
| clean            | Uses `rimraf` to remove `dist/`                    |
| build            | Uses `tsc` to build package and dependent packages |
| lint             | Uses `eslint` to lint package                      |
| lint:fix         | Uses `eslint` to check and fix any warnings        |
| format           | Uses `prettier` to format the code                 |
| test             | Uses `jest` to run unit tests                      |
| test:integration | Uses `jest` to run tests under `/test/integration` |
| test:unit        | Uses `jest` to run tests under `/test/unit`        |

[docs]: https://docs.web3js.org/
[repo]: https://github.com/web3/web3.js/tree/4.x/packages/web3-eth-contract
[npm-image]: https://img.shields.io/github/package-json/v/web3/web3.js/4.x?filename=packages%2Fweb3-eth-contract%2Fpackage.json
[npm-url]: https://npmjs.org/package/web3-eth-contract
[downloads-image]: https://img.shields.io/npm/dm/web3-eth-contract?label=npm%20downloads
