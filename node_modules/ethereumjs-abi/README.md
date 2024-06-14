# ethereumjs-abi

[![NPM Package](https://img.shields.io/npm/v/ethereumjs-abi.svg?style=flat-square)](https://www.npmjs.org/package/ethereumjs-abi)
[![Build Status](https://img.shields.io/travis/ethereumjs/ethereumjs-abi.svg?branch=master&style=flat-square)](https://travis-ci.org/ethereumjs/ethereumjs-abi)
[![Coverage Status](https://img.shields.io/coveralls/ethereumjs/ethereumjs-abi.svg?style=flat-square)](https://coveralls.io/r/ethereumjs/ethereumjs-abi)
[![js-standard-style](https://img.shields.io/badge/code%20style-standard-brightgreen.svg)](http://standardjs.com/)
[![Gitter](https://img.shields.io/gitter/room/ethereum/ethereumjs-lib.svg?style=flat-square)](https://gitter.im/ethereum/ethereumjs-lib) or #ethereumjs on freenode


Module implementing the [Ethereum ABI](https://github.com/ethereum/wiki/wiki/Ethereum-Contract-ABI) in Javascript. Can be used with RPC libraries for communication or with ethereumjs-vm to implement a fully fledged simulator.

## Usage

#### Manual encoding and decoding

There are three methods of interest:
- ```methodID``` to create a function signature
- ```rawEncode``` to encode fields and
- ```rawDecode``` to decode fields

Example code:
```js
var abi = require('ethereumjs-abi')

// returns the encoded binary (as a Buffer) data to be sent
var encoded = abi.rawEncode([ "address" ], [ "0x0000000000000000000000000000000000000000" ])

// returns the decoded array of arguments
var decoded = abi.rawDecode([ "address" ], data)
```

#### Encoding and decoding aided by the JSON ABI definition

Planned for the future is supporting the JSON ABI definition:

```js
var abi = require('ethereumjs-abi')

// need to have the ABI definition in JSON as per specification
var tokenAbi = [{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"success","type":"bool"}],"type":"function"},{"inputs":[],"type":"constructor"}]

var encoded = abi.encode(tokenAbi, "balanceOf(uint256 address)", [ "0x0000000000000000000000000000000000000000" ])

var decoded = abi.decode(tokenAbi, "balanceOf(uint256 address)", data)
```

#### Simple encoding and decoding

```js
var abi = require('ethereumjs-abi')

// returns the encoded binary (as a Buffer) data to be sent
var encoded = abi.simpleEncode("balanceOf(address):(uint256)", "0x0000000000000000000000000000000000000000")

// returns the decoded array of arguments
var decoded = abi.simpleDecode("balanceOf(address):(uint256)", data)
```

#### Solidity *tightly packed* formats

This library also supports creating Solidity's tightly packed data constructs, which are used together with ```sha3```, ```sha256``` and ```ripemd160``` to create hashes.

Solidity code:
```js
contract HashTest {
  function testSha3() returns (bytes32) {
   address addr1 = 0x43989fb883ba8111221e89123897538475893837;
   address addr2 = 0;
   uint val = 10000;
   uint timestamp = 1448075779;

   return sha3(addr1, addr2, val, timestamp); // will return 0xc3ab5ca31a013757f26a88561f0ff5057a97dfcc33f43d6b479abc3ac2d1d595
 }
}
```

Creating the same hash using this library:
```js
var abi = require('ethereumjs-abi')
var BN = require('bn.js')

abi.soliditySHA3(
    [ "address", "address", "uint", "uint" ],
    [ new BN("43989fb883ba8111221e89123897538475893837", 16), 0, 10000, 1448075779 ]
).toString('hex')
```

For the same data structure:
* sha3 will return ```0xc3ab5ca31a013757f26a88561f0ff5057a97dfcc33f43d6b479abc3ac2d1d595```
* sha256 will return ```0x344d8cb0711672efbdfe991f35943847c1058e1ecf515ff63ad936b91fd16231```
* ripemd160 will return ```0x000000000000000000000000a398cc72490f72048efa52c4e92067e8499672e7``` (NOTE: it is 160bits, left padded to 256bits)

Note that ```ripemd160()``` in Solidity returns bytes20 and if you cast it to bytes32, it will be right padded with zeroes.

#### Using Serpent types

Serpent uses a different notation for the types, even though it will serialize to the same ABI.

We provide two helpers to convert between these notations:
* ```fromSerpent```: convert a Serpent notation to the ABI notation
* ```toSerpent```: the other way around

Example usage:
```js
abi.fromSerpent('s')    // [ 'bytes' ]
abi.fromSerpent('i')    // [ 'int256' ]
abi.fromSerpent('a')    // [ 'int256[]' ]
abi.fromSerpent('b8')   // [ 'bytes8' ]
abi.fromSerpent('b8i')  // [ 'bytes8', 'int256' ]

abi.toSerpent([ 'bytes' ])             // 's'
abi.toSerpent([ 'int256' ])            // 'i'
abi.toSerpent([ 'int256[]' ])          // 'a'
abi.toSerpent([ 'bytes8' ])            // 'b8'
abi.toSerpent([ 'bytes8', 'int256' ])  // 'b8i'
```

It is to be used in conjunction with ```rawEncode``` and ```rawDecode```:

```js
var encoded = abi.rawEncode(abi.fromSerpent("i"), [ "0x0000000000000000000000000000000000000000" ])

var decoded = abi.rawDecode([...abi.fromSerpent("i"), ...abi.fromSerpent("i")], data)
```

Note: Serpent uses arbitary binary fields. If you want to store strings it is preferable to ensure it is stored as UTF8. `Buffer.from(<string>, 'utf8')` can be used to ensure it is properly encoded.

## Contributing

I am more than happy to receive improvements. Please send me a pull request or reach out on email or twitter.

There is a lot missing, grep for *FIXME* in the source code to find inspiration.

## EthereumJS

See our organizational [documentation](https://ethereumjs.readthedocs.io) for an introduction to `EthereumJS` as well as information on current standards and best practices.

If you want to join for work or do improvements on the libraries have a look at our [contribution guidelines](https://ethereumjs.readthedocs.io/en/latest/contributing.html).

## License

    Copyright (C) 2015 Alex Beregszaszi

    Permission is hereby granted, free of charge, to any person obtaining a copy of
    this software and associated documentation files (the "Software"), to deal in
    the Software without restriction, including without limitation the rights to
    use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
    the Software, and to permit persons to whom the Software is furnished to do so,
    subject to the following conditions:

    The above copyright notice and this permission notice shall be included in all
    copies or substantial portions of the Software.

    THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
    IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
    FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
    COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
    IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
    CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
