# SYNOPSIS

[![NPM Package](https://img.shields.io/npm/v/rlp.svg)](https://www.npmjs.org/package/rlp)
[![Actions Status](https://github.com/ethereumjs/rlp/workflows/Build/badge.svg)](https://github.com/ethereumjs/rlp/actions)
[![Coverage Status](https://img.shields.io/coveralls/ethereumjs/rlp.svg)](https://coveralls.io/r/ethereumjs/rlp)
[![Discord](https://img.shields.io/static/v1?logo=discord&label=discord&message=Join&color=blue)](https://discord.gg/TNwARpR)

[Recursive Length](https://github.com/ethereum/wiki/wiki/RLP) Prefix Encoding for Node.js.

## INSTALL

`npm install rlp`

install with `-g` if you want to use the cli.

## USAGE

```typescript
import * as assert from 'assert'
import * as rlp from 'rlp'

const nestedList = [[], [[]], [[], [[]]]]
const encoded = rlp.encode(nestedList)
const decoded = rlp.decode(encoded)
assert.deepEqual(nestedList, decoded)
```

## API

`rlp.encode(plain)` - RLP encodes an `Array`, `Buffer` or `String` and returns a `Buffer`.

`rlp.decode(encoded, [skipRemainderCheck=false])` - Decodes an RLP encoded `Buffer`, `Array` or `String` and returns a `Buffer` or an `Array` of `Buffers`. If `skipRemainderCheck` is enabled, `rlp` will just decode the first rlp sequence in the buffer. By default, it would throw an error if there are more bytes in Buffer than used by rlp sequence.

## CLI

`rlp decode <hex string>`  
`rlp encode <json String>`

## TESTS

Tests use mocha.

To run tests and linting: `npm test`

To auto fix linting problems use: `npm run lint:fix`

## CODE COVERAGE

Install dev dependencies
`npm install`

Run
`npm run coverage`

The results are at
`coverage/lcov-report/index.html`

# EthereumJS

See our organizational [documentation](https://ethereumjs.readthedocs.io) for an introduction to `EthereumJS` as well as information on current standards and best practices.

If you want to join for work or do improvements on the libraries have a look at our [contribution guidelines](https://ethereumjs.readthedocs.io/en/latest/contributing.html).
