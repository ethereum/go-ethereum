# @ethereumjs/util

[![NPM Package][util-npm-badge]][util-npm-link]
[![GitHub Issues][util-issues-badge]][util-issues-link]
[![Actions Status][util-actions-badge]][util-actions-link]
[![Code Coverage][util-coverage-badge]][util-coverage-link]
[![Discord][discord-badge]][discord-link]

| A collection of utility functions for Ethereum. |
| ----------------------------------------------- |

## Installation

To obtain the latest version, simply require the project using `npm`:

```shell
npm install @ethereumjs/util
```

## Usage

This package contains the following modules providing respective helper methods, classes and commonly re-used constants.

All helpers are re-exported from the root level and deep imports are not necessary. So an import can be done like this:

```ts
import { hexToBytes, isValidChecksumAddress } from '@ethereumjs/util'
```

### Module: [account](src/account.ts)

Class representing an `Account` and providing private/public key and address-related functionality (creation, validation, conversion).

```ts
// ./examples/account.ts

import { Account } from '@ethereumjs/util'

const account = Account.fromAccountData({
  nonce: '0x02',
  balance: '0x0384',
  storageRoot: '0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421',
  codeHash: '0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470',
})
console.log(`Account with nonce=${account.nonce} and balance=${account.balance} created`)
```

### Module: [address](src/address.ts)

Class representing an Ethereum `Address` with instantiation helpers and validation methods.

```ts
// ./examples/address.ts

import { Address } from '@ethereumjs/util'

const address = Address.fromString('0x2f015c60e0be116b1f0cd534704db9c92118fb6a')
console.log(`Ethereum address ${address.toString()} created`)
```

### Module: [blobs](src/blobs.ts)

Module providing helpers for 4844 blobs and versioned hashes.

```ts
// ./examples/blobs.ts

import { bytesToHex, computeVersionedHash, getBlobs } from '@ethereumjs/util'

const blobs = getBlobs('test input')

console.log('Created the following blobs:')
console.log(blobs)

const commitment = new Uint8Array([1, 2, 3])
const blobCommitmentVersion = 0x01
const versionedHash = computeVersionedHash(commitment, blobCommitmentVersion)

console.log(`Versioned hash ${bytesToHex(versionedHash)} computed`)
```

### Module: [bytes](src/bytes.ts)

Byte-related helper and conversion functions.

```ts
// ./examples/bytes.ts

import { bytesToBigInt } from '@ethereumjs/util'

const bytesValue = new Uint8Array([97])
const bigIntValue = bytesToBigInt(bytesValue)

console.log(`Converted value: ${bigIntValue}`)
```

### Module: [constants](src/constants.ts)

Exposed constants (e.g. `KECCAK256_NULL_S` for string representation of Keccak-256 hash of null)

```ts
// ./examples/constants.ts

import { BIGINT_2EXP96, KECCAK256_NULL_S } from '@ethereumjs/util'

console.log(`The keccak-256 hash of null: ${KECCAK256_NULL_S}`)
console.log(`BigInt constants (performance), e.g. BIGINT_2EXP96: ${BIGINT_2EXP96}`)
```

### Module: [db](src/db.ts)

DB interface for database abstraction (Blockchain, Trie), see e.g. [@ethereumjs/trie recipes](https://github.com/ethereumjs/ethereumjs-monorepo/tree/master/packages/trie/recipes/level.ts)) for usage.

### Module: [genesis](src/genesis.ts)

Genesis related interfaces and helpers.

### Module: [internal](src/internal.ts)

Internalized simple helper methods like `isHexPrefixed`. Note that methods from this module might get deprectared in the future.

### Module: [kzg](src/kzg.ts)

KZG interface (used for 4844 blob txs), see [@ethereumjs/tx](https://github.com/ethereumjs/ethereumjs-monorepo/tree/master/packages/tx/README.md#kzg-setup) README for main usage instructions.

### Module: [mapDB](src/mapDB.ts)

Simple map DB implementation using the `DB` interface (see above).

### Module: [signature](src/signature.ts)

Functionality for signing, signature validation, conversion, recovery.

```ts
// ./examples/signature.ts

import { bytesToHex, ecrecover, hexToBytes } from '@ethereumjs/util'

const chainId = BigInt(3) // Ropsten

const echash = hexToBytes('0x82ff40c0a986c6a5cfad4ddf4c3aa6996f1a7837f9c398e17e5de5cbd5a12b28')
const r = hexToBytes('0x99e71a99cb2270b8cac5254f9e99b6210c6c10224a1579cf389ef88b20a1abe9')
const s = hexToBytes('0x129ff05af364204442bdb53ab6f18a99ab48acc9326fa689f228040429e3ca66')
const v = BigInt(41)

const pubkey = ecrecover(echash, v, r, s, chainId)

console.log(`Recovered public key ${bytesToHex(pubkey)} from valid signature values`)
```

### Module: [types](src/types.ts)

Various TypeScript types. Direct usage is not recommended, type structure might change in the future.

### Module: [withdrawal](src/withdrawal.ts)

Class representing an `EIP-4895` `Withdrawal` with different constructors as well as conversion and output helpers.

```ts
// ./examples/withdrawal.ts

import { Withdrawal } from '@ethereumjs/util'

const withdrawal = Withdrawal.fromWithdrawalData({
  index: 0n,
  validatorIndex: 65535n,
  address: '0x0000000000000000000000000000000000000000',
  amount: 0n,
})

console.log('Withdrawal object created:')
console.log(withdrawal.toJSON())
```

## Browser

With the breaking release round in Summer 2023 we have added hybrid ESM/CJS builds for all our libraries (see section below) and have eliminated many of the caveats which had previously prevented a frictionless browser usage.

It is now easily possible to run a browser build of one of the EthereumJS libraries within a modern browser using the provided ESM build. For a setup example see [./examples/browser.html](./examples/browser.html).

## API

### Documentation

Read the [API docs](docs/).

### Upgrade Helpers in bytes-Module

Depending on the extend of `Buffer` usage within your own libraries and other planning considerations, there are the two upgrade options to do the switch to `Uint8Array` yourself or keep `Buffer` and do transitions for input and output values.

We have updated the `@ethereumjs/util` `bytes` module with helpers for the most common conversions:

```ts
Buffer.alloc(97) // Allocate a Buffer with length 97
new Uint8Array(97) // Allocate a Uint8Array with length 97

Buffer.from('342770c0', 'hex') // Convert a hex string to a Buffer
hexToBytes('0x342770c0') // Convert a prefixed hex string to a Uint8Array, Util.hexToBytes()

`0x${myBuffer.toString('hex')}` // Convert a Buffer to a prefixed hex string
bytesToHex(myUint8Array) // Convert a Uint8Array to a prefixed hex string

intToBuffer(9) // Convert an integer to a Buffer, old (removed)
intToBytes(9) // Convert an integer to a Uint8Array, Util.intToBytes()
bytesToInt(myUint8Array) // Convert a Uint8Array to an integer, Util.bytesToInt()

bigIntToBytes(myBigInt) // Convert a BigInt to a Uint8Array, Util.bigIntToBytes()
bytesToBigInt(myUint8Array) // Convert a Uint8Array to a BigInt, Util.bytesToInt()

utf8ToBytes(myUtf8String) // Converts a UTF-8 string to a Uint8Array, Util.utf8ToBytes()
bytesToUtf8(myUint8Array) // Converts a Uint8Array to a UTF-8 string, Util.bytesToUtf8()

toBuffer(v: ToBufferInputTypes) // Converts various byte compatible types to Buffer, old (removed)
toBytes(v: ToBytesInputTypes) // Converts various byte compatible types to Uint8Array, Util.toBytes()
```

Helper methods can be imported like this:

```ts
import { hexToBytes } from '@ethereumjs/util'
```

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

Starting with Util v8 the usage of [BN.js](https://github.com/indutny/bn.js/) for big numbers has been removed from the library and replaced with the usage of the native JS [BigInt](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/BigInt) data type (introduced in `ES2020`).

Please note that number-related API signatures have changed along with this version update and the minimal build target has been updated to `ES2020`.

### ethjs-util methods

The following methods are available by an internalized version of the [ethjs-util](https://github.com/ethjs/ethjs-util) package (`MIT` license), see [internal.ts](src/internal.ts). The original package is not maintained any more and the original functionality will be replaced by own implementations over time (starting with the `v7.1.3` release, October 2021).

- arrayContainsArray
- getBinarySize
- stripHexPrefix
- isHexPrefixed
- isHexString
- padToEven
- fromAscii
- fromUtf8
- toUtf8
- toAscii
- getKeys

They can be imported by name:

```ts
import { stripHexPrefix } from '@ethereumjs/util'
```

## EthereumJS

See our organizational [documentation](https://ethereumjs.readthedocs.io) for an introduction to `EthereumJS` as well as information on current standards and best practices. If you want to join for work or carry out improvements on the libraries, please review our [contribution guidelines](https://ethereumjs.readthedocs.io/en/latest/contributing.html) first.

## License

[MPL-2.0](<https://tldrlegal.com/license/mozilla-public-license-2.0-(mpl-2)>)

[util-npm-badge]: https://img.shields.io/npm/v/@ethereumjs/util.svg
[util-npm-link]: https://www.npmjs.org/package/@ethereumjs/util
[util-issues-badge]: https://img.shields.io/github/issues/ethereumjs/ethereumjs-monorepo/package:%20util?label=issues
[util-issues-link]: https://github.com/ethereumjs/ethereumjs-monorepo/issues?q=is%3Aopen+is%3Aissue+label%3A"package%3A+util"
[util-actions-badge]: https://github.com/ethereumjs/ethereumjs-monorepo/workflows/Util/badge.svg
[util-actions-link]: https://github.com/ethereumjs/ethereumjs-monorepo/actions?query=workflow%3A%22Util%22
[util-coverage-badge]: https://codecov.io/gh/ethereumjs/ethereumjs-monorepo/branch/master/graph/badge.svg?flag=util
[util-coverage-link]: https://codecov.io/gh/ethereumjs/ethereumjs-monorepo/tree/master/packages/util
[discord-badge]: https://img.shields.io/static/v1?logo=discord&label=discord&message=Join&color=blue
[discord-link]: https://discord.gg/TNwARpR
