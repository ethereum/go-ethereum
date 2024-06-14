# @ethereumjs/common

[![NPM Package][common-npm-badge]][common-npm-link]
[![GitHub Issues][common-issues-badge]][common-issues-link]
[![Actions Status][common-actions-badge]][common-actions-link]
[![Code Coverage][common-coverage-badge]][common-coverage-link]
[![Discord][discord-badge]][discord-link]

| Resources common to all EthereumJS implementations. |
| --------------------------------------------------- |

Note: this `README` reflects the state of the library from `v2.0.0` onwards. See `README` from the [standalone repository](https://github.com/ethereumjs/ethereumjs-common) for an introduction on the last preceding release.

## Installation

To obtain the latest version, simply require the project using `npm`:

```shell
npm install @ethereumjs/common
```

## Usage

### import / require

import (ESM, TypeScript):

```ts
import { Chain, Common, Hardfork } from '@ethereumjs/common'
```

require (CommonJS, Node.js):

```ts
const { Common, Chain, Hardfork } = require('@ethereumjs/common')
```

### Parameters

All parameters can be accessed through the `Common` class, instantiated with an object containing either the `chain` (e.g. 'Chain.Mainnet') or the `chain` together with a specific `hardfork` provided:

```ts
// ./examples/common.ts#L1-L7

import { Chain, Common, Hardfork } from '@ethereumjs/common'

// With enums:
const commonWithEnums = new Common({ chain: Chain.Mainnet, hardfork: Hardfork.London })

// (also possible with directly passing in strings:)
const commonWithStrings = new Common({ chain: 'mainnet', hardfork: 'london' })
```

If no hardfork is provided, the common is initialized with the default hardfork.

Current `DEFAULT_HARDFORK`: `Hardfork.Shanghai`

Here are some simple usage examples:

```ts
// ./examples/common.ts#L9-L23

// Instantiate with the chain (and the default hardfork)
let c = new Common({ chain: Chain.Mainnet })
console.log(`The gas price for ecAdd is ${c.param('gasPrices', 'ecAddGas')}`) // 500

// Chain and hardfork provided
c = new Common({ chain: Chain.Mainnet, hardfork: Hardfork.Byzantium })
console.log(`The miner reward under PoW on Byzantium us ${c.param('pow', 'minerReward')}`) // 3000000000000000000

// Get bootstrap nodes for chain/network
console.log('Below are the known bootstrap nodes')
console.log(c.bootstrapNodes()) // Array with current nodes

// Instantiate with an EIP activated
c = new Common({ chain: Chain.Mainnet, eips: [4844] })
console.log(`EIP 4844 is active -- ${c.isActivatedEIP(4844)}`)
```

## Browser

With the breaking release round in Summer 2023 we have added hybrid ESM/CJS builds for all our libraries (see section below) and have eliminated many of the caveats which had previously prevented a frictionless browser usage.

It is now easily possible to run a browser build of one of the EthereumJS libraries within a modern browser using the provided ESM build. For a setup example see [./examples/browser.html](./examples/browser.html).

## API

### Docs

See the API documentation for a full list of functions for accessing specific chain and
depending hardfork parameters. There are also additional helper functions like
`paramByBlock (topic, name, blockNumber)` or `hardforkIsActiveOnBlock (hardfork, blockNumber)`
to ease `blockNumber` based access to parameters.

Generated TypeDoc API [Documentation](./docs/README.md)

### Hybrid CJS/ESM Builds

With the breaking releases from Summer 2023 we have started to ship our libraries with both CommonJS (`cjs` folder) and ESM builds (`esm` folder), see `package.json` for the detailed setup.

If you use an ES6-style `import` in your code files from the ESM build will be used:

```ts
import { EthereumJSClass } from '@ethereumjs/[PACKAGE_NAME]'
```

If you use Node.js specific `require`, the CJS build will be used:

```ts
const { EthereumJSClass } = require('@ethereumjs/[PACKAGE_NAME]')
```

Using ESM will give you additional advantages over CJS beyond browser usage like static code analysis / Tree Shaking which CJS can not provide.

### Buffer -> Uint8Array

With the breaking releases from Summer 2023 we have removed all Node.js specific `Buffer` usages from our libraries and replace these with [Uint8Array](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Uint8Array) representations, which are available both in Node.js and the browser (`Buffer` is a subclass of `Uint8Array`).

We have converted existing Buffer conversion methods to Uint8Array conversion methods in the [@ethereumjs/util](https://github.com/ethereumjs/ethereumjs-monorepo/tree/master/packages/util) `bytes` module, see the respective README section for guidance.

### BigInt Support

Starting with v4 the usage of [BN.js](https://github.com/indutny/bn.js/) for big numbers has been removed from the library and replaced with the usage of the native JS [BigInt](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/BigInt) data type (introduced in `ES2020`).

Please note that number-related API signatures have changed along with this version update and the minimal build target has been updated to `ES2020`.

## Events

The `Common` class has a public property `events` which contains an `EventEmitter`. Following events are emitted on which you can react within your code:

| Event             | Description                                                |
| ----------------- | ---------------------------------------------------------- |
| `hardforkChanged` | Emitted when a hardfork change occurs in the Common object |

## Setup

### Chains

The `chain` can be set in the constructor like this:

```ts
const c = new Common({ chain: Chain.Mainnet })
```

Supported chains:

- `mainnet` (`Chain.Mainnet`)
- `goerli` (`Chain.Goerli`)
- `sepolia` (`Chain.Sepolia`) (`v2.6.1`+)
- `holesky` (`Chain.Holesky`) (`v4.1.0`+)
- Private/custom chain parameters

The following chain-specific parameters are provided:

- `name`
- `chainId`
- `networkId`
- `consensusType` (e.g. `pow` or `poa`)
- `consensusAlgorithm` (e.g. `ethash` or `clique`)
- `consensusConfig` (depends on `consensusAlgorithm`, e.g. `period` and `epoch` for `clique`)
- `genesis` block header values
- `hardforks` block numbers
- `bootstrapNodes` list
- `dnsNetworks` list ([EIP-1459](https://eips.ethereum.org/EIPS/eip-1459)-compliant list of DNS networks for peer discovery)

To get an overview of the different parameters have a look at one of the chain configurations in the `chains.ts` configuration
file, or to the `Chain` type in [./src/types.ts](./src/types.ts).

### Working with private/custom chains

There are two distinct APIs available for setting up custom(ized) chains.

#### Basic Chain Customization / Predefined Custom Chains

There is a dedicated `Common.custom()` static constructor which allows for an easy instantiation of a Common instance with somewhat adopted chain parameters, with the main use case to adopt on instantiating with a deviating chain ID (you can use this to adopt other chain parameters as well though). Instantiating a custom common instance with its own chain ID and inheriting all other parameters from `mainnet` can now be as easily done as:

```ts
// ./examples/common.ts#L25-L27

// Instantiate common with custom chainID
const commonWithCustomChainId = Common.custom({ chainId: 1234 })
console.log(`The current chain ID is ${commonWithCustomChainId.chainId}`)
```

The `custom()` method also takes a string as a first input (instead of a dictionary). This can be used in combination with the `CustomChain` enum dict which allows for the selection of predefined supported custom chains for an easier `Common` setup of these supported chains:

```ts
const common = Common.custom(CustomChain.ArbitrumRinkebyTestnet)
```

The following custom chains are currently supported:

- `PolygonMainnet`
- `PolygonMumbai`
- `ArbitrumRinkebyTestnet`
- `xDaiChain`
- `OptimisticKovan`
- `OptimisticEthereum`

`Common` instances created with this simplified `custom()` constructor can't be used in all usage contexts (the HF configuration is very likely not matching the actual chain) but can be useful for specific use cases, e.g. for sending a tx with `@ethereumjs/tx` to an L2 network (see the `Tx` library [README](https://github.com/ethereumjs/ethereumjs-monorepo/tree/master/packages/tx) for a complete usage example).

#### Activate with a single custom Chain setup

If you want to initialize a `Common` instance with a single custom chain which is then directly activated
you can pass a dictionary - conforming to the parameter format described above - with your custom chain
values to the constructor using the `chain` parameter or the `setChain()` method, here is some example:

```ts
// ./examples/customChain.ts

import { Common } from '@ethereumjs/common'
import myCustomChain1 from './genesisData/testnet.json'

// Add custom chain config
const common1 = new Common({ chain: myCustomChain1 })
console.log(`Common is instantiated with custom chain parameters - ${common1.chainName()}`)
```

#### Initialize using customChains Array

A second way for custom chain initialization is to use the `customChains` constructor option. This
option comes with more flexibility and allows for an arbitrary number of custom chains to be initialized on
a common instance in addition to the already supported ones. It also allows for an activation-independent
initialization, so you can add your chains by adding to the `customChains` array and either directly
use the `chain` option to activate one of the custom chains passed or activate a build in chain
(e.g. `mainnet`) and switch to other chains - including the custom ones - by using `Common.setChain()`.

```ts
// ./examples/customChains.ts

import { Common } from '@ethereumjs/common'
import myCustomChain1 from './genesisData/testnet.json'
import myCustomChain2 from './genesisData/testnet2.json'
// Add two custom chains, initial mainnet activation
const common1 = new Common({ chain: 'mainnet', customChains: [myCustomChain1, myCustomChain2] })
console.log(`Common is instantiated with mainnet parameters - ${common1.chainName()}`)
common1.setChain('testnet1')
console.log(`Common is set to use testnet parameters - ${common1.chainName()}`)
// Add two custom chains, activate customChain1
const common2 = new Common({
  chain: 'testnet2',
  customChains: [myCustomChain1, myCustomChain2],
})

console.log(`Common is instantiated with testnet2 parameters - ${common1.chainName()}`)
```

Starting with v3 custom genesis states should be passed to the [Blockchain](../blockchain/) library directly.

#### Initialize using Geth's genesis json

For lots of custom chains (for e.g. devnets and testnets), you might come across a genesis json config which
has both config specification for the chain as well as the genesis state specification. You can derive the
common from such configuration in the following manner:

```ts
// ./examples/fromGeth.ts

import { Common } from '@ethereumjs/common'
import { hexToBytes } from '@ethereumjs/util'

import genesisJson from './genesisData/post-merge.json'

const genesisHash = hexToBytes('0x3b8fb240d288781d4aac94d3fd16809ee413bc99294a085798a589dae51ddd4a')
// Load geth genesis json file into lets say `genesisJson` and optional `chain` and `genesisHash`
const common = Common.fromGethGenesis(genesisJson, { chain: 'customChain', genesisHash })
// If you don't have `genesisHash` while initiating common, you can later configure common (for e.g.
// after calculating it via `blockchain`)
common.setForkHashes(genesisHash)

console.log(`The London forkhash for this custom chain is ${common.forkHash('london')}`)
```

### Hardforks

The `hardfork` can be set in constructor like this:

```ts
// ./examples/common.ts#L1-L4

import { Chain, Common, Hardfork } from '@ethereumjs/common'

// With enums:
const commonWithEnums = new Common({ chain: Chain.Mainnet, hardfork: Hardfork.London })
```

### Active Hardforks

There are currently parameter changes by the following past and future hardfork by the
library supported:

- `chainstart` (`Hardfork.Chainstart`)
- `homestead` (`Hardfork.Homestead`)
- `dao` (`Hardfork.Dao`)
- `tangerineWhistle` (`Hardfork.TangerineWhistle`)
- `spuriousDragon` (`Hardfork.SpuriousDragon`)
- `byzantium` (`Hardfork.Byzantium`)
- `constantinople` (`Hardfork.Constantinople`)
- `petersburg` (`Hardfork.Petersburg`) (aka `constantinopleFix`, apply together with `constantinople`)
- `istanbul` (`Hardfork.Instanbul`)
- `muirGlacier` (`Hardfork.MuirGlacier`)
- `berlin` (`Hardfork.Berlin`) (since `v2.2.0`)
- `london` (`Hardfork.London`) (since `v2.4.0`)
- `merge` (`Hardfork.Merge`) (`DEFAULT_HARDFORK`) (since `v2.5.0`)
- `shanghai` (`Hardfork.Shanghai`) (since `v3.1.0`)
- `cancun` (`Hardfork.Cancun`) (since `v4.0.0`)

### Future Hardforks

The next upcoming HF `Hardfork.Prague` is currently not yet supported by this library.

### Parameter Access

For hardfork-specific parameter access with the `param()` and `paramByBlock()` functions
you can use the following `topics`:

- `gasConfig`
- `gasPrices`
- `vm`
- `pow`
- `sharding`

See one of the hardfork configurations in the `hardforks.ts` file
for an overview. For consistency, the chain start (`chainstart`) is considered an own
hardfork.

The hardfork configurations above `chainstart` only contain the deltas from `chainstart` and
shouldn't be accessed directly until you have a specific reason for it.

### EIPs

Starting with the `v2.0.0` release of the library, EIPs are now native citizens within the library
and can be activated like this:

```ts
const c = new Common({ chain: Chain.Mainnet, eips: [4844] })
```

The following EIPs are currently supported:

- [EIP-1153](https://eips.ethereum.org/EIPS/eip-1153) - Transient storage opcodes (Cancun)
- [EIP-1559](https://eips.ethereum.org/EIPS/eip-1559) - Fee market change for ETH 1.0 chain
- [EIP-2315](https://eips.ethereum.org/EIPS/eip-2315) - Simple subroutines for the EVM (`outdated`)
- [EIP-2537](https://eips.ethereum.org/EIPS/eip-2537) - BLS precompiles (removed in v4.0.0, see latest v3 release)
- [EIP-2565](https://eips.ethereum.org/EIPS/eip-2565) - ModExp gas cost
- [EIP-2718](https://eips.ethereum.org/EIPS/eip-2718) - Transaction Types
- [EIP-2929](https://eips.ethereum.org/EIPS/eip-2929) - gas cost increases for state access opcodes
- [EIP-2930](https://eips.ethereum.org/EIPS/eip-2930) - Optional access list tx type
- [EIP-3074](https://eips.ethereum.org/EIPS/eip-3074) - AUTH and AUTHCALL opcodes
- [EIP-3198](https://eips.ethereum.org/EIPS/eip-3198) - Base fee Opcode
- [EIP-3529](https://eips.ethereum.org/EIPS/eip-3529) - Reduction in refunds
- [EIP-3540](https://eips.ethereum.org/EIPS/eip-3540) - EVM Object Format (EOF) v1 (`outdated`)
- [EIP-3541](https://eips.ethereum.org/EIPS/eip-3541) - Reject new contracts starting with the 0xEF byte
- [EIP-3554](https://eips.ethereum.org/EIPS/eip-3554) - Difficulty Bomb Delay to December 2021 (only PoW networks)
- [EIP-3607](https://eips.ethereum.org/EIPS/eip-3607) - Reject transactions from senders with deployed code
- [EIP-3651](https://eips.ethereum.org/EIPS/eip-3651) - Warm COINBASE (Shanghai)
- [EIP-3670](https://eips.ethereum.org/EIPS/eip-3670) - EOF - Code Validation (`outdated`)
- [EIP-3675](https://eips.ethereum.org/EIPS/eip-3675) - Upgrade consensus to Proof-of-Stake
- [EIP-3855](https://eips.ethereum.org/EIPS/eip-3855) - Push0 opcode (Shanghai)
- [EIP-3860](https://eips.ethereum.org/EIPS/eip-3860) - Limit and meter initcode (Shanghai)
- [EIP-4345](https://eips.ethereum.org/EIPS/eip-4345) - Difficulty Bomb Delay to June 2022
- [EIP-4399](https://eips.ethereum.org/EIPS/eip-4399) - Supplant DIFFICULTY opcode with PREVRANDAO (Merge)
- [EIP-4788](https://eips.ethereum.org/EIPS/eip-4788) - Beacon block root in the EVM (Cancun)
- [EIP-4844](https://eips.ethereum.org/EIPS/eip-4844) - Shard Blob Transactions (Cancun) (`experimental`)
- [EIP-4895](https://eips.ethereum.org/EIPS/eip-4895) - Beacon chain push withdrawals as operations (Shanghai)
- [EIP-5133](https://eips.ethereum.org/EIPS/eip-5133) - Delaying Difficulty Bomb to mid-September 2022 (Gray Glacier)
- [EIP-5656](https://eips.ethereum.org/EIPS/eip-5656) - MCOPY - Memory copying instruction (Cancun)
- [EIP-6780](https://eips.ethereum.org/EIPS/eip-6780) - SELFDESTRUCT only in same transaction (Cancun)
- [EIP-7516](https://eips.ethereum.org/EIPS/eip-7516) - BLOBBASEFEE opcode (Cancun)

### Bootstrap Nodes

You can use `common.bootstrapNodes()` function to get nodes for a specific chain/network.

## EthereumJS

See our organizational [documentation](https://ethereumjs.readthedocs.io) for an introduction to `EthereumJS` as well as information on current standards and best practices. If you want to join for work or carry out improvements on the libraries, please review our [contribution guidelines](https://ethereumjs.readthedocs.io/en/latest/contributing.html) first.

## License

[MIT](https://opensource.org/licenses/MIT)

[discord-badge]: https://img.shields.io/static/v1?logo=discord&label=discord&message=Join&color=blue
[discord-link]: https://discord.gg/TNwARpR
[common-npm-badge]: https://img.shields.io/npm/v/@ethereumjs/common.svg
[common-npm-link]: https://www.npmjs.com/package/@ethereumjs/common
[common-issues-badge]: https://img.shields.io/github/issues/ethereumjs/ethereumjs-monorepo/package:%20common?label=issues
[common-issues-link]: https://github.com/ethereumjs/ethereumjs-monorepo/issues?q=is%3Aopen+is%3Aissue+label%3A"package%3A+common"
[common-actions-badge]: https://github.com/ethereumjs/ethereumjs-monorepo/workflows/Common/badge.svg
[common-actions-link]: https://github.com/ethereumjs/ethereumjs-monorepo/actions?query=workflow%3A%22Common%22
[common-coverage-badge]: https://codecov.io/gh/ethereumjs/ethereumjs-monorepo/branch/master/graph/badge.svg?flag=common
[common-coverage-link]: https://codecov.io/gh/ethereumjs/ethereumjs-monorepo/tree/master/packages/common
