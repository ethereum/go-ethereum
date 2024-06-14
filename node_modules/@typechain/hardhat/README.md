<p align="center">
  <img src="https://github.com/Neufund/TypeChain/blob/d82f3cc644a11e22ca8e42505c16f035e2f2555d/docs/images/typechain-logo.png?raw=true" width="300" alt="TypeChain">
  <h3 align="center">TypeChain Hardhat plugin</h3>
  <p align="center">Zero-config TypeChain support for Hardhat</p>

  <p align="center">
    <a href="https://github.com/ethereum-ts/TypeChain/actions"><img alt="Build Status" src="https://github.com/ethereum-ts/TypeChain/workflows/CI/badge.svg"></a>
    <img alt="Downloads" src="https://img.shields.io/npm/dm/typechain.svg">
    <a href="https://github.com/prettier/prettier"><img alt="Prettier" src="https://img.shields.io/badge/code_style-prettier-ff69b4.svg"></a>
    <a href="/package.json"><img alt="Software License" src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square"></a>
  </p>
</p>

# Description

Automatically generate TypeScript bindings for smartcontracts while using [Hardhat](https://hardhat.org/).

# Installation

If you use Ethers do:

```bash
npm install --save-dev typechain @typechain/hardhat @typechain/ethers-v6
```

If you're a Truffle user you need:

```bash
npm install --save-dev typechain @typechain/hardhat @typechain/truffle-v5
```

And add the following statements to your `hardhat.config.js`:

```javascript
require('@typechain/hardhat')
require('@nomicfoundation/hardhat-ethers')
require('@nomicfoundation/hardhat-chai-matchers')
```

Or, if you use TypeScript, add this to your `hardhat.config.ts`:

```typescript
import '@typechain/hardhat'
import '@nomicfoundation/hardhat-ethers'
import '@nomicfoundation/hardhat-chai-matchers'
```

Here's a sample `tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "es2018",
    "module": "commonjs",
    "strict": true,
    "esModuleInterop": true,
    "outDir": "dist",
    "resolveJsonModule": true
  },
  "include": ["./scripts", "./test", "./typechain-types"],
  "files": ["./hardhat.config.ts"]
}
```

Now typings should be automatically generated each time contract recompilation happens.

**Warning**: before running it for the first time you need to do `hardhat clean`, otherwise TypeChain will think that
there is no need to generate any typings. This is because this plugin will attempt to do incremental generation and
generate typings only for changed contracts. You should also do `hardhat clean` if you change any TypeChain related
config option.

## Features

- **Zero Config Usage** - Run the _compile_ task as normal, and Typechain artifacts will automatically be generated in a
  root directory called `typechain-types`.
- **Incremental generation** - Only recompiled files will have their types regenerated
- **Frictionless** - return type of `ethers.getContractFactory` will be typed properly - no need for casts

## Tasks

This plugin overrides the _compile_ task and automatically generates new Typechain artifacts on each compilation.

There is an optional flag `--no-typechain` which can be passed in to skip Typechain compilation.

This plugin also adds the `typechain` task to hardhat:

```
hardhat typechain # always regenerates typings to all files
```

## Configuration

This plugin extends the `hardhatConfig` optional `typechain` object. The object contains two fields, `outDir` and
`target`. `outDir` is the output directory of the artifacts that TypeChain creates (defaults to `typechain`). `target`
is one of the targets specified by the TypeChain [docs](https://github.com/ethereum-ts/TypeChain#cli) (defaults to
`ethers`).

This is an example of how to set it:

```js
module.exports = {
  typechain: {
    outDir: 'src/types',
    target: 'ethers-v6',
    alwaysGenerateOverloads: false, // should overloads with full signatures like deposit(uint256) be generated always, even if there are no overloads?
    externalArtifacts: ['externalArtifacts/*.json'], // optional array of glob patterns with external artifacts to process (for example external libs from node_modules)
    dontOverrideCompile: false // defaults to false
  },
}
```

## Usage

`npx hardhat compile` - Compiles and generates Typescript typings for your contracts. Example Ethers + Hardhat Chai Matchers test that
uses typedefs for contracts:

```ts
import { ethers } from 'hardhat'
import chai from 'chai'

import { Counter } from '../src/types/Counter'

const { expect } = chai

describe('Counter', () => {
  let counter: Counter

  beforeEach(async () => {
    // 1
    const signers = await ethers.getSigners()

    // 2
    counter = await ethers.deployContract("Counter")

    // 3
    const initialCount = await counter.getCount()
    expect(initialCount).to.eq(0)
  })

  // 4
  describe('count up', async () => {
    it('should count up', async () => {
      await counter.countUp()
      let count = await counter.getCount()
      expect(count).to.eq(1)
    })
  })

  describe('count down', async () => {
    // 5 - this throw a error with solidity ^0.8.0
    it('should fail', async () => {
      await counter.countDown()
    })

    it('should count down', async () => {
      await counter.countUp()

      await counter.countDown()
      const count = await counter.getCount()
      expect(count).to.eq(0)
    })
  })
})
```

## Examples

- [starter kit](https://github.com/rhlsthrm/typescript-solidity-dev-starter-kit)
- [example-ethers](https://github.com/ethereum-ts/TypeChain/tree/master/examples/hardhat)
- [example-truffle](https://github.com/ethereum-ts/TypeChain/tree/master/examples/hardhat-truffle-v5)
- @paulrberg's [solidity-template](https://github.com/paulrberg/solidity-template)

Original work done by [@RHLSTHRM](https://twitter.com/RHLSTHRM).

## Troubleshooting

Using the types generated by this plugin can lead to Hardhat failing to run. The reason is that the types are not
avialable for loading the config, and that's required to generate the types.

To workaround this issue, you can run `TS_NODE_TRANSPILE_ONLY=1 npx hardhat compile`.
