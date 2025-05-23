[![npm](https://img.shields.io/npm/v/@nomicfoundation/hardhat-network-helpers.svg)](https://www.npmjs.com/package/@nomicfoundation/hardhat-network-helpers)

# Hardhat Network Helpers

Hardhat Network Helpers is a library that provides a set of utility functions to interact with the [Hardhat Network](https://hardhat.org/hardhat-network/docs). You can read their full documentation [here](https://hardhat.org/hardhat-network-helpers/docs).

### Installation

We recommend using npm 7 or later:

```
npm install --save-dev @nomicfoundation/hardhat-network-helpers
```

### Usage

Import it and use it in any of your files. For example, this [Hardhat script](https://hardhat.org/hardhat-runner/docs/advanced/scripts) mines some blocks and then prints the block number.

```js
const helpers = require("@nomicfoundation/hardhat-network-helpers");

async function main() {
  // mine 100 blocks
  await helpers.mine(100);

  console.log("The current block number is", await helpers.time.latestBlock());
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
```
