# secp256k1-node

This module provides native bindings to [bitcoin-core/secp256k1](https://github.com/bitcoin-core/secp256k1). In browser [elliptic](https://github.com/indutny/elliptic) will be used as fallback.

Works on node version 10.0.0 or greater, because use [N-API](https://nodejs.org/api/n-api.html).

## Installation

##### from npm

`npm install secp256k1`

##### from git

```
git clone git@github.com:cryptocoinjs/secp256k1-node.git
cd secp256k1-node
git submodule update --init
npm install
```

##### Windows

The easiest way to build the package on windows is to install [windows-build-tools](https://github.com/felixrieseberg/windows-build-tools).

Or install the following software:

  * Git: https://git-scm.com/download/win
  * nvm: https://github.com/coreybutler/nvm-windows
  * Python 2.7: https://www.python.org/downloads/release/python-2712/
  * Visual C++ Build Tools: http://landinghub.visualstudio.com/visual-cpp-build-tools (Custom Install, and select both Windows 8.1 and Windows 10 SDKs)

And run commands:

```
npm config set msvs_version 2015 --global
npm install npm@next -g
```

Based on:

  * https://github.com/nodejs/node-gyp/issues/629#issuecomment-153196245
  * https://github.com/nodejs/node-gyp/issues/972

## Usage

* [API Reference (v4.x)](API.md) (current version)
* [API Reference (v3.x)](https://github.com/cryptocoinjs/secp256k1-node/blob/v3.x/API.md)
* [API Reference (v2.x)](https://github.com/cryptocoinjs/secp256k1-node/blob/v2.x/API.md)

##### Private Key generation, Public Key creation, signature creation, signature verification

```js
const { randomBytes } = require('crypto')
const secp256k1 = require('secp256k1')
// or require('secp256k1/elliptic')
//   if you want to use pure js implementation in node

// generate message to sign
// message should have 32-byte length, if you have some other length you can hash message
// for example `msg = sha256(rawMessage)`
const msg = randomBytes(32)

// generate privKey
let privKey
do {
  privKey = randomBytes(32)
} while (!secp256k1.privateKeyVerify(privKey))

// get the public key in a compressed format
const pubKey = secp256k1.publicKeyCreate(privKey)

// sign the message
const sigObj = secp256k1.ecdsaSign(msg, privKey)

// verify the signature
console.log(secp256k1.ecdsaVerify(sigObj.signature, msg, pubKey))
// => true
```

\* **.verify return false for high signatures**

##### Get X point of ECDH

```js
const { randomBytes } = require('crypto')
// const secp256k1 = require('./elliptic')
const secp256k1 = require('./')

// generate privKey
function getPrivateKey () {
  while (true) {
    const privKey = randomBytes(32)
    if (secp256k1.privateKeyVerify(privKey)) return privKey
  }
}

// generate private and public keys
const privKey = getPrivateKey()
const pubKey = secp256k1.publicKeyCreate(getPrivateKey())

// compressed public key from X and Y
function hashfn (x, y) {
  const pubKey = new Uint8Array(33)
  pubKey[0] = (y[31] & 1) === 0 ? 0x02 : 0x03
  pubKey.set(x, 1)
  return pubKey
}

// get X point of ecdh
const ecdhPointX = secp256k1.ecdh(pubKey, privKey, { hashfn }, Buffer.alloc(33))
console.log(ecdhPointX.toString('hex'))
```

## LICENSE

This library is free and open-source software released under the MIT license.
