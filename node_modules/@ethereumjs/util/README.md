# @ethereumjs/util

[![NPM Package][util-npm-badge]][util-npm-link]
[![GitHub Issues][util-issues-badge]][util-issues-link]
[![Actions Status][util-actions-badge]][util-actions-link]
[![Code Coverage][util-coverage-badge]][util-coverage-link]
[![Discord][discord-badge]][discord-link]

A collection of utility functions for Ethereum. It can be used in Node.js and in the browser with [browserify](http://browserify.org/).

## Installation

To obtain the latest version, simply require the project using `npm`:

```shell
npm install @ethereumjs/util
```

## Usage

```js
import assert from 'assert'
import { isValidChecksumAddress, unpadBuffer } from '@ethereumjs/util'

assert.ok(isValidChecksumAddress('0x2F015C60E0be116B1f0CD534704Db9c92118FB6A'))

assert.ok(unpadBuffer(Buffer.from('000000006600', 'hex')).equals(Buffer.from('6600', 'hex')))
```

## API

### Documentation

Read the [API docs](docs/).

### Modules

- [account](src/account.ts)
  - Account class
  - Private/public key and address-related functionality (creation, validation, conversion)
- [address](src/address.ts)
  - Address class and type
- [bytes](src/bytes.ts)
  - Byte-related helper and conversion functions
- [constants](src/constants.ts)
  - Exposed constants
    - e.g. `KECCAK256_NULL_S` for string representation of Keccak-256 hash of null
- hash
  - This module has been removed with `v8`, please use [ethereum-cryptography](https://github.com/ethereum/js-ethereum-cryptography) directly instead
- [signature](src/signature.ts)
  - Signing, signature validation, conversion, recovery
- [types](src/types.ts)
  - Helpful TypeScript types
- [internal](src/internal.ts)
  - Internalized helper methods
- [withdrawal](src/withdrawal.ts)
  - Withdrawal class (EIP-4895)

### BigInt Support

Starting with v8 the usage of [BN.js](https://github.com/indutny/bn.js/) for big numbers has been removed from the library and replaced with the usage of the native JS [BigInt](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/BigInt) data type (introduced in `ES2020`).

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

```typescript
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
