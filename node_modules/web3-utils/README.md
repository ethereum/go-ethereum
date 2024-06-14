# web3-utils

[![NPM Package][npm-image]][npm-url]

This is a sub-package of [web3.js][repo].

This contains useful utility functions for Dapp developers.

Please read the [documentation][docs] for more.

## Installation

You can install the package either using [NPM](https://www.npmjs.com/package/web3-utils) or using [Yarn](https://yarnpkg.com/package/web3-utils)

### Using NPM

```bash
npm install web3-utils
```

### Using Yarn

```bash
yarn add web3-utils
```

## Usage

```js
const Web3Utils = require('web3-utils');
console.log(Web3Utils);
{
    sha3: function(){},
    soliditySha3: function(){},
    isAddress: function(){},
    ...
}
```

## Types

All the TypeScript typings are placed in the `types` folder.

[docs]: http://web3js.readthedocs.io/en/1.0/
[repo]: https://github.com/ethereum/web3.js
[npm-image]: https://img.shields.io/npm/v/web3-utils.svg
[npm-url]: https://npmjs.org/package/web3-utils
