# ethereumjs-util

[![NPM Package][util-npm-badge]][util-npm-link]
[![GitHub Issues][util-issues-badge]][util-issues-link]
[![Actions Status][util-actions-badge]][util-actions-link]
[![Code Coverage][util-coverage-badge]][util-coverage-link]
[![Discord][discord-badge]][discord-link]

A collection of utility functions for Ethereum. It can be used in Node.js and in the browser with [browserify](http://browserify.org/).

# INSTALL

`npm install ethereumjs-util`

# USAGE

```js
import assert from 'assert'
import { isValidChecksumAddress, unpadBuffer, BN } from 'ethereumjs-util'

assert.ok(isValidChecksumAddress('0x2F015C60E0be116B1f0CD534704Db9c92118FB6A'))

assert.ok(unpadBuffer(Buffer.from('000000006600', 'hex')).equals(Buffer.from('6600', 'hex')))

assert.ok(new BN('dead', 16).add(new BN('101010', 2)).eqn(57047))
```

# API

## Documentation

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
- [hash](src/hash.ts)
  - Hash functions
- [object](src/object.ts)
  - Helper function for creating a binary object (`DEPRECATED`)
- [signature](src/signature.ts)
  - Signing, signature validation, conversion, recovery
- [types](src/types.ts)
  - Helpful TypeScript types
- [internal](src/internal.ts)
  - Internalized helper methods
- [externals](src/externals.ts)
  - Re-exports of `BN`, `rlp`

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

```js
import { stripHexPrefix } from 'ethereumjs-util'
```

### Re-Exports

`ethereumjs-util` re-exports the following commonly-used libraries:

- [BN.js](https://github.com/indutny/bn.js) (version `5.x`)
- [rlp](https://github.com/ethereumjs/rlp) (version `2.x`)

They can be imported by name:

```js
import { BN, rlp } from 'ethereumjs-util'
```

# EthereumJS

See our organizational [documentation](https://ethereumjs.readthedocs.io) for an introduction to `EthereumJS` as well as information on current standards and best practices.

If you want to join for work or do improvements on the libraries have a look at our [contribution guidelines](https://ethereumjs.readthedocs.io/en/latest/contributing.html).

# LICENSE

MPL-2.0

[util-npm-badge]: https://img.shields.io/npm/v/ethereumjs-util.svg
[util-npm-link]: https://www.npmjs.org/package/ethereumjs-util
[util-issues-badge]: https://img.shields.io/github/issues/ethereumjs/ethereumjs-monorepo/package:%20util?label=issues
[util-issues-link]: https://github.com/ethereumjs/ethereumjs-monorepo/issues?q=is%3Aopen+is%3Aissue+label%3A"package%3A+util"
[util-actions-badge]: https://github.com/ethereumjs/ethereumjs-monorepo/workflows/Util/badge.svg
[util-actions-link]: https://github.com/ethereumjs/ethereumjs-monorepo/actions?query=workflow%3A%22Util%22
[util-coverage-badge]: https://codecov.io/gh/ethereumjs/ethereumjs-monorepo/branch/master/graph/badge.svg?flag=util
[util-coverage-link]: https://codecov.io/gh/ethereumjs/ethereumjs-monorepo/tree/master/packages/util
[discord-badge]: https://img.shields.io/static/v1?logo=discord&label=discord&message=Join&color=blue
[discord-link]: https://discord.gg/TNwARpR
