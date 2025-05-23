<p align="center">
  <img src="assets/logo/web3js.jpg" width="500" alt="web3.js" />
</p>

# web3-validator

![ES Version](https://img.shields.io/badge/ES-2020-yellow)
![Node Version](https://img.shields.io/badge/node-14.x-green)
[![NPM Package][npm-image]][npm-url]
[![Downloads][downloads-image]][npm-url]

This is a sub-package of [web3.js][repo].

`web3-validator` contains functions for validating objects.

## Installation

You can install the package either using [NPM](https://www.npmjs.com/package/web3-validator) or using [Yarn](https://yarnpkg.com/package/web3-validator)

### Using NPM

```bash
npm install web3-validator
```

### Using Yarn

```bash
yarn add web3-validator
```

## Getting Started

-   :writing_hand: If you have questions [submit an issue](https://github.com/ChainSafe/web3.js/issues/new) or join us on [Discord](https://discord.gg/yjyvFRP)
    ![Discord](https://img.shields.io/discord/593655374469660673.svg?label=Discord&logo=discord)

### Usage

You can use the validator by importing as and using to validate;

```ts
import { validator } from 'web3-validator';

// To validate and throw
validator.validate(['uint8', 'string'], [val1, val2]);

// To validate and return error
const errors = validator.validate(['uint8', 'string'], [val1, val2], { silent: true });
```

To see more examples of schema you can use to validate check [following file](./test/fixtures/abi_to_json_schema.ts).

Following eth types are supported to validate.

| Type  | Input As                        | Description                                                                                                                                               |
| ----- | ------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| uint  | `number`, `string`, `HexString` | Unsigned integer, all ethereum compatible variants are also supported e.g. `uint8`, `uint256`. You can also use array specifiers as `uint[]` or `uint[2]` |
| int   | `number`, `string`, `HexString` | Signed integer, all ethereum compatible variants are also supported e.g. `int8`, `int256`. You can also use array specifiers as `int[]` or `int[2]`       |
| bytes | `HexString`, `Uint8Array`       | Raw bytes. You can also use fixed length bytes as `bytes[2]`                                                                                              |

| string | `string` | String values |
| address | `string`, `HexString` | Ethereum network compatible address |
| bloom | `string`, `HexString` | Check if a given string is a Eth bloom |
| tuple | `array` | You can specify any tuple as nested arrays. e.g. `['uint', 'string']`. For a custom tuple or array tuple you can use syntax e.g. `['tuple[3]', ['uint', 'string']]` |

For the ethereum compatible data values should be passed as arrays e.g. for schema `['uint', 'string']` value should be passed as `[2, 'my-string']`.

You can also pass full ABI schema for the validation. e.g.

```json
[{ "name": "owner", "type": "address" }]
```

The implementation of the validator is extension of [JSON-Schema-Draft07](https://json-schema.org/draft-07/json-schema-release-notes.html) with a custom keyword `eth`. So you can use JSON-Schema compatible schema to validate any object based data as well.

## Prerequisites

-   :gear: [NodeJS](https://nodejs.org/) (LTS/Fermium)
-   :toolbox: [Yarn](https://yarnpkg.com/)/[Lerna](https://lerna.js.org/)

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
[repo]: https://github.com/web3/web3.js/tree/4.x/packages/web3-validator
[npm-image]: https://img.shields.io/github/package-json/v/web3/web3.js/4.x?filename=packages%2Fweb3-validator%2Fpackage.json
[npm-url]: https://npmjs.org/package/web3-validator
[downloads-image]: https://img.shields.io/npm/dm/web3-validator?label=npm%20downloads
