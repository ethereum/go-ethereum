# binary-parse-stream

  Painless streaming binary protocol parsers using generators.

## Installation

    npm install binary-parse-stream

## Synchronous

  This module uses the exact same generator interface as [binary-parse-stream](https://github.com/nathan7/binary-parse-stream), which presents a synchronous interface to a generator parser.

## Usage

```js
const BinaryParseStream = require('binary-parse-stream')
const {One} = BinaryParseStream // -1
```

  BinaryParseStream is a TransformStream that consumes buffers and outputs objects on the other end.
  It expects your subclass to implement a `_parse` method that is a generator.
  When your generator yields a number, it'll be fed a buffer of that length from the input.
  If it yields -1, it'll be given the value of the first byte instead of a single-byte buffer.
  When your generator returns, the return value will be pushed to the output side.

## Example

  The following module parses a protocol that consists of a 32-bit unsigned big-endian type parameter, an unsigned 8-bit length parameter, and a buffer of the specified length.
  It outputs `{type, buf}` objects.

```js
class SillyProtocolParseStream extends BinaryParseStream {
  constructor(options) {
    super(options)
    this.count = 0
  }

  *_parse() {
    const type = (yield 4).readUInt32BE(0, true)
    const length = yield -1
    const buf = yield length
    this.count++
    return {type, buf}
  }
}
```

  There is also a shorter syntax for when you don't want to explicitly subclass:  `BinaryParseStream.extend(function*())`.

