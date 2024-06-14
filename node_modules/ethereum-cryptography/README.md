# ethereum-cryptography

[![npm version][1]][2]
[![Travis CI][3]][4]
[![license][5]][6]
[![Types][7]][8]

This npm package contains all the cryptographic primitives normally used when
developing Javascript/TypeScript applications and tools for Ethereum.

Pure Javascript implementations of all the primitives are included, so it can
be used out of the box for web applications and libraries.

In Node, it takes advantage of the built-in and N-API based implementations
whenever possible.

The cryptographic primitives included are:

* [Pseudorandom number generation](#pseudorandom-number-generation-submodule)
* [Keccak](#keccak-submodule)
* [Scrypt](#scrypt-submodule)
* [PBKDF2](#pbkdf2-submodule)
* [SHA-256](#sha-256-submodule)
* [RIPEMD-160](#ripemd-160-submodule)
* [BLAKE2b](#blake2b-submodule)
* [AES](#aes-submodule)
* [Secp256k1](#secp256k1-submodule)
* [Hierarchical Deterministic keys derivation](#hierarchical-deterministic-keys-submodule)
* [Seed recovery phrases](#seed-recovery-phrases)

## Installation

Via `npm`:

```bash
$ npm install ethereum-cryptography
```

Via `yarn`:

```bash
$ yarn add ethereum-cryptography
```

## Usage

This package has no single entry-point, but submodule for each cryptographic
primitive. Read each primitive's section of this document to learn how to use
them.

The reason for this is that importing everything from a single file will lead to
huge bundles when using this package for the web. This could be avoided through
tree-shaking, but the possibility of it not working properly on one of
[the supported bundlers](#browser-usage) is too high.

## Pseudorandom number generation submodule

The `random` submodule has functions to generate cryptographically strong
pseudo-random data in synchronous and asynchronous ways.

In Node, this functions are backed by [`crypto.randomBytes`](https://nodejs.org/api/crypto.html#crypto_crypto_randombytes_size_callback).

In the browser, [`crypto.getRandomValues`](https://developer.mozilla.org/en-US/docs/Web/API/Crypto/getRandomValues)
is used. If not available, this module won't work, as that would be insecure.

### Function types

```ts
function getRandomBytes(bytes: number): Promise<Buffer>;

function getRandomBytesSync(bytes: number): Buffer;
```

### Example usage

```js
const { getRandomBytesSync } = require("ethereum-cryptography/random");

console.log(getRandomBytesSync(32).toString("hex"));
```

## Keccak submodule

The `keccak` submodule has four functions that implement different variations of
the Keccak hashing algorithm. These are `keccak224`, `keccak256`, `keccak384`,
and `keccak512`.

### Function types

```ts
function keccak224(msg: Buffer): Buffer;

function keccak256(msg: Buffer): Buffer;

function keccak384(msg: Buffer): Buffer;

function keccak512(msg: Buffer): Buffer;
```

### Example usage

```js
const { keccak256 } = require("ethereum-cryptography/keccak");

console.log(keccak256(Buffer.from("Hello, world!", "ascii")).toString("hex"));
```

## Scrypt submodule

The `scrypt` submodule has two functions implementing the Scrypt key
derivation algorithm in synchronous and asynchronous ways. This algorithm is
very slow, and using the synchronous version in the browser is not recommended,
as it will block its main thread and hang your UI.

### Password encoding

Encoding passwords is a frequent source of errors. Please read
[these notes](https://github.com/ricmoo/scrypt-js/tree/0eb70873ddf3d24e34b53e0d9a99a0cef06a79c0#encoding-notes)
before using this submodule.

### Function types

```ts
function scrypt(password: Buffer, salt: Buffer, n: number, p: number, r: number, dklen: number): Promise<Buffer>;

function scryptSync(password: Buffer, salt: Buffer, n: number, p: number, r: number, dklen: number): Buffer;
```

### Example usage

```js
const { scryptSync } = require("ethereum-cryptography/scrypt");

console.log(
  scryptSync(
    Buffer.from("ascii password", "ascii"),
    Buffer.from("salt", "hex"),
    16,
    1,
    1,
    64
  ).toString("hex")
);
```

## PBKDF2 submodule

The `pbkdf2` submodule has two functions implementing the PBKDF2 key
derivation algorithm in synchronous and asynchronous ways. This algorithm is
very slow, and using the synchronous version in the browser is not recommended,
as it will block its main thread and hang your UI.

### Password encoding

Encoding passwords is a frequent source of errors. Please read
[these notes](https://github.com/ricmoo/scrypt-js/tree/0eb70873ddf3d24e34b53e0d9a99a0cef06a79c0#encoding-notes)
before using this submodule.

### Supported digests

In Node this submodule uses the built-in implementation and supports any digest
returned by [`crypto.getHashes`](https://nodejs.org/api/crypto.html#crypto_crypto_gethashes).

In the browser, it is tested to support at least `sha256`, the only digest
normally used with `pbkdf2` in Ethereum. It may support more.

### Function types

```ts
function pbkdf2(password: Buffer, salt: Buffer, iterations: number, keylen: number, digest: string): Promise<Buffer>;

function pbkdf2Sync(password: Buffer, salt: Buffer, iterations: number, keylen: number, digest: string): Buffer;
```

### Example usage

```js
const { pbkdf2Sync } = require("ethereum-cryptography/pbkdf2");

console.log(
  pbkdf2Sync(
    Buffer.from("ascii password", "ascii"),
    Buffer.from("salt", "hex"),
    4096,
    32,
    'sha256'
  ).toString("hex")
);
```

## SHA-256 submodule

The `sha256` submodule contains a single function implementing the SHA-256
hashing algorithm.

### Function types

```ts
function sha256(msg: Buffer): Buffer;
```

### Example usage

```js
const { sha256 } = require("ethereum-cryptography/sha256");

console.log(sha256(Buffer.from("message", "ascii")).toString("hex"));
```

## RIPEMD-160 submodule

The `ripemd160` submodule contains a single function implementing the
RIPEMD-160 hashing algorithm.

### Function types

```ts
function ripemd160(msg: Buffer): Buffer;
```

### Example usage

```js
const { ripemd160 } = require("ethereum-cryptography/ripemd160");

console.log(ripemd160(Buffer.from("message", "ascii")).toString("hex"));
```

## BLAKE2b submodule

The `blake2b` submodule contains a single function implementing the
BLAKE2b non-keyed hashing algorithm.

### Function types

```ts
function blake2b(input: Buffer, outputLength = 64): Buffer;
```

### Example usage

```js
const { blake2b } = require("ethereum-cryptography/blake2b");

console.log(blake2b(Buffer.from("message", "ascii")).toString("hex"));
```

## AES submodule

The `aes` submodule contains encryption and decryption functions implementing
the [Advanced Encryption Standard](https://en.wikipedia.org/wiki/Advanced_Encryption_Standard)
algorithm.

### Encrypting with passwords

AES is not supposed to be used directly with a password. Doing that will
compromise your users' security.

The `key` parameters in this submodule are meant to be strong cryptographic
keys. If you want to obtain such a key from a password, please use a
[key derivation function](https://en.wikipedia.org/wiki/Key_derivation_function)
like [pbkdf2](#pbkdf2-submodule) or [scrypt](#scrypt-submodule).

### Operation modes

This submodule works with different [block cipher modes of operation](https://en.wikipedia.org/wiki/Block_cipher_mode_of_operation). If you are using this module in a new
application, we recommend using the default.

While this module may work with any mode supported by OpenSSL, we only test it
with `aes-128-ctr`, `aes-128-cbc`, and `aes-256-cbc`. If you use another module
a warning will be printed in the console.

We only recommend using `aes-128-cbc` and `aes-256-cbc` to decrypt already
encrypted data.

### Padding plaintext messages

Some operation modes require the plaintext message to be a multiple of `16`. If
that isn't the case, your message has to be padded.

By default, this module automatically pads your messages according to [PKCS#7](https://tools.ietf.org/html/rfc2315).
Note that this padding scheme always adds at least 1 byte of padding. If you
are unsure what anything of this means, we **strongly** recommend you to use
the defaults.

If you need to encrypt without padding or want to use another padding scheme,
you can disable PKCS#7 padding by passing `false` as the last argument and
handling padding yourself. Note that if you do this and your operation mode
requires padding, `encrypt` will throw if your plaintext message isn't a
multiple of `16`.

This option is only present to enable the decryption of already encrypted data.
To encrypt new data, we recommend using the default.

### How to use the IV parameter

The `iv` parameter of the `encrypt` function must be unique, or the security
of the encryption algorithm can be compromissed.

You can generate a new `iv` using the `random` module.

Note that to decrypt a value, you have to provide the same `iv` used to encrypt
it.

### How to handle errors with this module

Sensitive information can be leaked via error messages when using this module.
To avoid this, you should make sure that the errors you return don't
contain the exact reason for the error. Instead, errors must report general
encryption/decryption failures.

Note that implementing this can mean catching all errors that can be thrown
when calling on of this module's functions, and just throwing a new generic
exception.

### Function types

```ts
function encrypt(msg: Buffer, key: Buffer, iv: Buffer, mode = "aes-128-ctr", pkcs7PaddingEnabled = true): Buffer;

function decrypt(cypherText: Buffer, key: Buffer, iv: Buffer, mode = "aes-128-ctr", pkcs7PaddingEnabled = true): Buffer
```

### Example usage

```js
const { encrypt } = require("ethereum-cryptography/aes");

console.log(
  encrypt(
    Buffer.from("message", "ascii"),
    Buffer.from("2b7e151628aed2a6abf7158809cf4f3c", "hex"),
    Buffer.from("f0f1f2f3f4f5f6f7f8f9fafbfcfdfeff", "hex")
  ).toString("hex")
);
```

## Secp256k1 submodule

The `secp256k1` submodule provides a library for elliptic curve operations on
the curve Secp256k1.

It has the exact same API than the version `4.x` of the [`secp256k1`](https://github.com/cryptocoinjs/secp256k1-node)
module from cryptocoinjs, with two added function to create private keys.

### Creating private keys

Secp256k1 private keys need to be cryptographycally secure random numbers with
certain caracteristics. If this is not the case, the security of Secp256k1 is
compromissed.

We strongly recommend to use this module to create new private keys.

### Function types

Functions to create private keys:

```ts
function createPrivateKey(): Promise<Uint8Array>;

function function createPrivateKeySync(): Uint8Array;
```

For the rest of the functions, pleasse read [`secp256k1`'s documentation](https://github.com/cryptocoinjs/secp256k1-node).

### Example usage

```js
const { createPrivateKeySync, ecdsaSign } = require("ethereum-cryptography/secp256k1");

const msgHash = Buffer.from(
  "82ff40c0a986c6a5cfad4ddf4c3aa6996f1a7837f9c398e17e5de5cbd5a12b28",
  "hex"
);

const privateKey = createPrivateKeySync();

console.log(Buffer.from(ecdsaSign(msgHash, privateKey).signature).toString("hex"));
```

## Hierarchical Deterministic keys submodule

The `hdkey` submodule provides a library for keys derivation according to
[BIP32](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki).

It has almost the exact same API than the version `1.x` of
[`hdkey` from cryptocoinjs](https://github.com/cryptocoinjs/hdkey),
but it's backed by this package's primitives, and has built-in TypeScript types.
Its only difference is that it has to be be used with a named import.

### Function types

This module exports a single class whose type is

```ts
class HDKey {
  public static HARDENED_OFFSET: number;
  public static fromMasterSeed(seed: Buffer, versions: Versions): HDKey;
  public static fromExtendedKey(base58key: string, versions: Versions): HDKey;
  public static fromJSON(json: { xpriv: string }): HDKey;

  public versions: Versions;
  public depth: number;
  public index: number;
  public chainCode: Buffer | null;
  public privateKey: Buffer | null;
  public publicKey: Buffer | null;
  public fingerprint: number;
  public parentFingerprint: number;
  public pubKeyHash: Buffer | undefined;
  public identifier: Buffer | undefined;
  public privateExtendedKey: string;
  public publicExtendedKey: string;

  private constructor(versios: Versions);
  public derive(path: string): HDKey;
  public deriveChild(index: number): HDKey;
  public sign(hash: Buffer): Buffer;
  public verify(hash: Buffer, signature: Buffer): boolean;
  public wipePrivateData(): this;
  public toJSON(): { xpriv: string; xpub: string };
}

interface Versions {
  private: number;
  public: number;
}
```

### Example usage

```js
const { HDKey } = require("ethereum-cryptography/hdkey");

const seed = "fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542";
const hdkey = HDKey.fromMasterSeed(Buffer.from(seed, "hex"));
const childkey = hdkey.derive("m/0/2147483647'/1");

console.log(childkey.privateExtendedKey);
```

## Seed recovery phrases

The `bip39` submodule provides functions to generate, validate and use seed
recovery phrases according to [BIP39](https://github.com/bitcoin/bips/blob/master/bip-0039.mediawiki).

### Function types

```ts
function generateMnemonic(wordlist: string[], strength: number = 128): string;

function mnemonicToEntropy(mnemonic: string, wordlist: string[]): Buffer;

function entropyToMnemonic(entropy: Buffer, wordlist: string[]): string;

function validateMnemonic(mnemonic: string, wordlist: string[]): boolean;

async function mnemonicToSeed(mnemonic: string, passphrase: string = ""): Promise<Buffer>;

function mnemonicToSeedSync(mnemonic: string, passphrase: string = ""): Buffer;
```

### Word lists

This submodule also contains the word lists defined by BIP39 for Czech, English,
French, Italian, Japanese, Korean, Simplified and Traditional Chinese, and
Spanish. These are not imported by default, as that would increase bundle sizes
too much. Instead, you should import and use them explicitly.

The word lists are exported as a `wordlist` variable in each of these submodules:

* `ethereum-cryptography/bip39/wordlists/czech.js`

* `ethereum-cryptography/bip39/wordlists/english.js`

* `ethereum-cryptography/bip39/wordlists/french.js`

* `ethereum-cryptography/bip39/wordlists/italian.js`

* `ethereum-cryptography/bip39/wordlists/japanese.js`

* `ethereum-cryptography/bip39/wordlists/korean.js`

* `ethereum-cryptography/bip39/wordlists/simplified-chinese.js`

* `ethereum-cryptography/bip39/wordlists/spanish.js`

* `ethereum-cryptography/bip39/wordlists/traditional-chinese.js`

### Example usage

```js

const { generateMnemonic } = require("ethereum-cryptography/bip39");
const { wordlist } = require("ethereum-cryptography/bip39/wordlists/english");

console.log(generateMnemonic(wordlist));
```

## Browser usage

This package works with all the major Javascript bundlers. It is
tested with `webpack`, `Rollup`, `Parcel`, and `Browserify`.

### Rollup setup

Using this library with Rollup requires the following plugins:

[`@rollup/plugin-commonjs`](https://www.npmjs.com/package/@rollup/plugin-commonjs)
[`@rollup/plugin-json`](https://www.npmjs.com/package/@rollup/plugin-json)
[`@rollup/plugin-node-resolve`](https://www.npmjs.com/package/@rollup/plugin-node-resolve)
[`rollup-plugin-node-builtins`](https://www.npmjs.com/package/rollup-plugin-node-builtins)
[`rollup-plugin-node-globals`](https://www.npmjs.com/package/rollup-plugin-node-globals)

These can be used by setting your `plugins` array like this:

```js
  plugins: [
    commonjs(),
    json(),
    nodeGlobals(),
    nodeBuiltins(),
    resolve({
      browser: true,
      preferBuiltins: false,
    }),
  ]
```

## Missing cryptographic primitives

This package intentionally excludes the the cryptographic primitives necessary
to implement the following EIPs:

* [EIP 196: Precompiled contracts for addition and scalar multiplication on the elliptic curve alt_bn128](https://eips.ethereum.org/EIPS/eip-196)
* [EIP 197: Precompiled contracts for optimal ate pairing check on the elliptic curve alt_bn128](https://eips.ethereum.org/EIPS/eip-197)
* [EIP 198: Big integer modular exponentiation](https://eips.ethereum.org/EIPS/eip-198)
* [EIP 152: Add Blake2 compression function `F` precompile](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-152.md)

Feel free to open an issue if you want this decision to be reconsidered, or if
you found another primitive that is missing.

## Security audit

This library has been audited by [Trail of Bits](https://www.trailofbits.com/).
You can see the results of the audit and the changes implemented as a result of
it in [`audit/`](./audit).

## License

`ethereum-cryptography` is released under [the MIT License](./LICENSE).

[1]: https://img.shields.io/npm/v/ethereum-cryptography.svg
[2]: https://www.npmjs.com/package/ethereum-cryptography
[3]: https://img.shields.io/travis/ethereum/js-ethereum-cryptography/master.svg?label=Travis%20CI
[4]: https://travis-ci.org/ethereum/js-ethereum-cryptography
[5]: https://img.shields.io/npm/l/ethereum-cryptography
[6]: https://github.com/ethereum/js-ethereum-cryptography/blob/master/packages/ethereum-cryptography/LICENSE
[7]: https://img.shields.io/npm/types/ethereum-cryptography.svg
[8]: https://www.npmjs.com/package/ethereum-cryptography
