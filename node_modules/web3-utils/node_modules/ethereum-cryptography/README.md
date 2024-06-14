# ethereum-cryptography

[![npm version][1]][2] [![license][3]][4]

[Audited](#security) pure JS library containing all Ethereum-related cryptographic primitives.

Included algorithms, implemented with just 5 [noble & scure](https://paulmillr.com/noble/) dependencies:

* [Hashes: SHA256, keccak-256, RIPEMD160, BLAKE2b](#hashes-sha256-keccak-256-ripemd160-blake2b)
* [KDFs: PBKDF2, Scrypt](#kdfs-pbkdf2-scrypt)
* [CSPRNG (Cryptographically Secure Pseudorandom Number Generator)](#csprng-cryptographically-strong-pseudorandom-number-generator)
* [secp256k1 elliptic curve](#secp256k1-curve)
* [BIP32 HD Keygen](#bip32-hd-keygen)
* [BIP39 Mnemonic phrases](#bip39-mnemonic-seed-phrase)
* [AES Encryption](#aes-encryption)

**April 2023 update:** v2.0 is out, switching
[noble-secp256k1](https://github.com/paulmillr/noble-secp256k1) to
[noble-curves](https://github.com/paulmillr/noble-curves),
which changes re-exported api of `secp256k1` submodule.
There have been no other changes.

**January 2022 update:** v1.0 has been released. We've rewritten the library from
scratch and [audited](#security) it. It became **6x smaller:** ~5,000 lines of
code instead of ~24,000 (with all deps); 650KB instead of 10.2MB.
5 dependencies by 1 author are now used, instead of 38 by 5 authors.

Check out [Upgrading](#upgrading) section and an article about the library:
[A safer, smaller, and faster Ethereum cryptography stack](https://medium.com/nomic-labs-blog/a-safer-smaller-and-faster-ethereum-cryptography-stack-5eeb47f62d79).

## Usage

Use NPM / Yarn in node.js / browser:

```bash
# NPM
npm install ethereum-cryptography

# Yarn
yarn add ethereum-cryptography
```

See [browser usage](#browser-usage) for information on using the package with major Javascript bundlers. It is
tested with **Webpack, Rollup, Parcel and Browserify**.

This package has no single entry-point, but submodule for each cryptographic
primitive. Read each primitive's section of this document to learn how to use
them.

The reason for this is that importing everything from a single file will lead to
huge bundles when using this package for the web. This could be avoided through
tree-shaking, but the possibility of it not working properly on one of
[the supported bundlers](#browser-usage) is too high.

```js
// Hashes
import { sha256 } from "ethereum-cryptography/sha256.js";
import { keccak256 } from "ethereum-cryptography/keccak.js";
import { ripemd160 } from "ethereum-cryptography/ripemd160.js";
import { blake2b } from "ethereum-cryptography/blake2b.js";

// KDFs
import { pbkdf2Sync } from "ethereum-cryptography/pbkdf2.js";
import { scryptSync } from "ethereum-cryptography/scrypt.js";

// Random
import { getRandomBytesSync } from "ethereum-cryptography/random.js";

// AES encryption
import { encrypt } from "ethereum-cryptography/aes.js";

// secp256k1 elliptic curve operations
import { secp256k1 } from "ethereum-cryptography/secp256k1.js";

// BIP32 HD Keygen, BIP39 Mnemonic Phrases
import { HDKey } from "ethereum-cryptography/hdkey.js";
import { generateMnemonic } from "ethereum-cryptography/bip39/index.js";
import { wordlist } from "ethereum-cryptography/bip39/wordlists/english.js";

// utilities
import { hexToBytes, toHex, utf8ToBytes } from "ethereum-cryptography/utils.js";
```

## Hashes: SHA256, keccak-256, RIPEMD160, BLAKE2b
```typescript
function sha256(msg: Uint8Array): Uint8Array;
function sha512(msg: Uint8Array): Uint8Array;
function keccak256(msg: Uint8Array): Uint8Array;
function ripemd160(msg: Uint8Array): Uint8Array;
function blake2b(msg: Uint8Array, outputLength = 64): Uint8Array;
```

Exposes following cryptographic hash functions:

- SHA2 (SHA256, SHA512)
- keccak-256 variant of SHA3 (also `keccak224`, `keccak384`,
and `keccak512`)
- RIPEMD160
- BLAKE2b

```js
import { sha256 } from "ethereum-cryptography/sha256.js";
import { sha512 } from "ethereum-cryptography/sha512.js";
import { keccak256, keccak224, keccak384, keccak512 } from "ethereum-cryptography/keccak.js";
import { ripemd160 } from "ethereum-cryptography/ripemd160.js";
import { blake2b } from "ethereum-cryptography/blake2b.js";

sha256(Uint8Array.from([1, 2, 3]))

// Can be used with strings
import { utf8ToBytes } from "ethereum-cryptography/utils.js";
sha256(utf8ToBytes("abc"))

// If you need hex
import { bytesToHex as toHex } from "ethereum-cryptography/utils.js";
toHex(sha256(utf8ToBytes("abc")))
```

## KDFs: PBKDF2, Scrypt

```ts
function pbkdf2(password: Uint8Array, salt: Uint8Array, iterations: number, keylen: number, digest: string): Promise<Uint8Array>;
function pbkdf2Sync(password: Uint8Array, salt: Uint8Array, iterations: number, keylen: number, digest: string): Uint8Array;
function scrypt(password: Uint8Array, salt: Uint8Array, N: number, p: number, r: number, dkLen: number, onProgress?: (progress: number) => void): Promise<Uint8Array>;
function scryptSync(password: Uint8Array, salt: Uint8Array, N: number, p: number, r: number, dkLen: number, onProgress?: (progress: number) => void)): Uint8Array;
```

The `pbkdf2` submodule has two functions implementing the PBKDF2 key
derivation algorithm in synchronous and asynchronous ways. This algorithm is
very slow, and using the synchronous version in the browser is not recommended,
as it will block its main thread and hang your UI. The KDF supports `sha256` and `sha512` digests.

The `scrypt` submodule has two functions implementing the Scrypt key
derivation algorithm in synchronous and asynchronous ways. This algorithm is
very slow, and using the synchronous version in the browser is not recommended,
as it will block its main thread and hang your UI.

Encoding passwords is a frequent source of errors. Please read
[these notes](https://github.com/ricmoo/scrypt-js/tree/0eb70873ddf3d24e34b53e0d9a99a0cef06a79c0#encoding-notes)
before using these submodules.

```js
import { pbkdf2 } from "ethereum-cryptography/pbkdf2.js";
import { utf8ToBytes } from "ethereum-cryptography/utils.js";
// Pass Uint8Array, or convert strings to Uint8Array
console.log(await pbkdf2(utf8ToBytes("password"), utf8ToBytes("salt"), 131072, 32, "sha256"));
```

```js
import { scrypt } from "ethereum-cryptography/scrypt.js";
import { utf8ToBytes } from "ethereum-cryptography/utils.js";
console.log(await scrypt(utf8ToBytes("password"), utf8ToBytes("salt"), 262144, 8, 1, 32));
```

## CSPRNG (Cryptographically strong pseudorandom number generator)

```ts
function getRandomBytes(bytes: number): Promise<Uint8Array>;
function getRandomBytesSync(bytes: number): Uint8Array;
```

The `random` submodule has functions to generate cryptographically strong
pseudo-random data in synchronous and asynchronous ways.

Backed by [`crypto.getRandomValues`](https://developer.mozilla.org/en-US/docs/Web/API/Crypto/getRandomValues) in browser and by [`crypto.randomBytes`](https://nodejs.org/api/crypto.html#crypto_crypto_randombytes_size_callback) in node.js. If backends are somehow not available, the module would throw an error and won't work, as keeping them working would be insecure.

```js
import { getRandomBytesSync } from "ethereum-cryptography/random.js";
console.log(getRandomBytesSync(32));
```

## secp256k1 curve

```ts
function getPublicKey(privateKey: Uint8Array, isCompressed = true): Uint8Array;
function sign(msgHash: Uint8Array, privateKey: Uint8Array): { r: bigint; s: bigint; recovery: number };
function verify(signature: Uint8Array, msgHash: Uint8Array, publicKey: Uint8Array): boolean
function getSharedSecret(privateKeyA: Uint8Array, publicKeyB: Uint8Array): Uint8Array;
function utils.randomPrivateKey(): Uint8Array;
```

The `secp256k1` submodule provides a library for elliptic curve operations on
the curve secp256k1. For detailed documentation, follow [README of `noble-curves`](https://github.com/paulmillr/noble-curves), which the module uses as a backend.

secp256k1 private keys need to be cryptographically secure random numbers with
certain characteristics. If this is not the case, the security of secp256k1 is
compromised. We strongly recommend using `utils.randomPrivateKey()` to generate them.

```js
import { secp256k1 } from "ethereum-cryptography/secp256k1.js";
(async () => {
  // You pass either a hex string, or Uint8Array
  const privateKey = "6b911fd37cdf5c81d4c0adb1ab7fa822ed253ab0ad9aa18d77257c88b29b718e";
  const messageHash = "a33321f98e4ff1c283c76998f14f57447545d339b3db534c6d886decb4209f28";
  const publicKey = secp256k1.getPublicKey(privateKey);
  const signature = secp256k1.sign(messageHash, privateKey);
  const isSigned = secp256k1.verify(signature, messageHash, publicKey);
})();
```

We're also providing a compatibility layer for users who want to upgrade
from `tiny-secp256k1` or `secp256k1` modules without hassle.
Check out [secp256k1 compatibility layer](#legacy-secp256k1-compatibility-layer).

## BIP32 HD Keygen

Hierarchical deterministic (HD) wallets that conform to BIP32 standard.
Also available as standalone package [scure-bip32](https://github.com/paulmillr/scure-bip32).

This module exports a single class `HDKey`, which should be used like this:

```ts
import { HDKey } from "ethereum-cryptography/hdkey.js";
const hdkey1 = HDKey.fromMasterSeed(seed);
const hdkey2 = HDKey.fromExtendedKey(base58key);
const hdkey3 = HDKey.fromJSON({ xpriv: string });

// props
[hdkey1.depth, hdkey1.index, hdkey1.chainCode];
console.log(hdkey2.privateKey, hdkey2.publicKey);
console.log(hdkey3.derive("m/0/2147483647'/1"));
const sig = hdkey3.sign(hash);
hdkey3.verify(hash, sig);
```

Note: `chainCode` property is essentially a private part
of a secret "master" key, it should be guarded from unauthorized access.

The full API is:

```ts
class HDKey {
  public static HARDENED_OFFSET: number;
  public static fromMasterSeed(seed: Uint8Array, versions: Versions): HDKey;
  public static fromExtendedKey(base58key: string, versions: Versions): HDKey;
  public static fromJSON(json: { xpriv: string }): HDKey;

  readonly versions: Versions;
  readonly depth: number = 0;
  readonly index: number = 0;
  readonly chainCode: Uint8Array | null = null;
  readonly parentFingerprint: number = 0;

  get fingerprint(): number;
  get identifier(): Uint8Array | undefined;
  get pubKeyHash(): Uint8Array | undefined;
  get privateKey(): Uint8Array | null;
  get publicKey(): Uint8Array | null;
  get privateExtendedKey(): string;
  get publicExtendedKey(): string;

  derive(path: string): HDKey;
  deriveChild(index: number): HDKey;
  sign(hash: Uint8Array): Uint8Array;
  verify(hash: Uint8Array, signature: Uint8Array): boolean;
  wipePrivateData(): this;
}

interface Versions {
  private: number;
  public: number;
}
```

The `hdkey` submodule provides a library for keys derivation according to
[BIP32](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki).

It has almost the exact same API than the version `1.x` of
[`hdkey` from cryptocoinjs](https://github.com/cryptocoinjs/hdkey),
but it's backed by this package's primitives, and has built-in TypeScript types.
Its only difference is that it has to be used with a named import.
The implementation is [loosely based on hdkey, which has MIT License](#LICENSE).

## BIP39 Mnemonic Seed Phrase

```ts
function generateMnemonic(wordlist: string[], strength: number = 128): string;
function mnemonicToEntropy(mnemonic: string, wordlist: string[]): Uint8Array;
function entropyToMnemonic(entropy: Uint8Array, wordlist: string[]): string;
function validateMnemonic(mnemonic: string, wordlist: string[]): boolean;
async function mnemonicToSeed(mnemonic: string, passphrase: string = ""): Promise<Uint8Array>;
function mnemonicToSeedSync(mnemonic: string, passphrase: string = ""): Uint8Array;
```

The `bip39` submodule provides functions to generate, validate and use seed
recovery phrases according to [BIP39](https://github.com/bitcoin/bips/blob/master/bip-0039.mediawiki).

Also available as standalone package [scure-bip39](https://github.com/paulmillr/scure-bip39).

```js
import { generateMnemonic } from "ethereum-cryptography/bip39/index.js";
import { wordlist } from "ethereum-cryptography/bip39/wordlists/english.js";
console.log(generateMnemonic(wordlist));
```

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
* `ethereum-cryptography/bip39/wordlists/portuguese.js`
* `ethereum-cryptography/bip39/wordlists/simplified-chinese.js`
* `ethereum-cryptography/bip39/wordlists/spanish.js`
* `ethereum-cryptography/bip39/wordlists/traditional-chinese.js`

## AES Encryption

```ts
function encrypt(msg: Uint8Array, key: Uint8Array, iv: Uint8Array, mode = "aes-128-ctr", pkcs7PaddingEnabled = true): Promise<Uint8Array>;
function decrypt(cypherText: Uint8Array, key: Uint8Array, iv: Uint8Array, mode = "aes-128-ctr", pkcs7PaddingEnabled = true): Promise<Uint8Array>;
```

The `aes` submodule contains encryption and decryption functions implementing
the [Advanced Encryption Standard](https://en.wikipedia.org/wiki/Advanced_Encryption_Standard)
algorithm.

### Encrypting with passwords

AES is not supposed to be used directly with a password. Doing that will
compromise your users' security.

The `key` parameters in this submodule are meant to be strong cryptographic
keys. If you want to obtain such a key from a password, please use a
[key derivation function](https://en.wikipedia.org/wiki/Key_derivation_function)
like [pbkdf2](#kdfs-pbkdf2-scrypt) or [scrypt](#kdfs-pbkdf2-scrypt).

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
of the encryption algorithm can be compromised.

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

### Example usage

```js
import { encrypt } from "ethereum-cryptography/aes.js";
import { hexToBytes, utf8ToBytes } from "ethereum-cryptography/utils.js";

console.log(
  encrypt(
    utf8ToBytes("message"),
    hexToBytes("2b7e151628aed2a6abf7158809cf4f3c"),
    hexToBytes("f0f1f2f3f4f5f6f7f8f9fafbfcfdfeff")
  )
);
```

## Browser usage

### Rollup setup

Using this library with Rollup requires the following plugins:

* [`@rollup/plugin-commonjs`](https://www.npmjs.com/package/@rollup/plugin-commonjs)
* [`@rollup/plugin-node-resolve`](https://www.npmjs.com/package/@rollup/plugin-node-resolve)

These can be used by setting your `plugins` array like this:

```js
  plugins: [
    commonjs(),
    resolve({
      browser: true,
      preferBuiltins: false,
    }),
  ]
```

## Legacy secp256k1 compatibility layer

**Warning:** use `secp256k1` instead. This module is only for users who upgraded
from ethereum-cryptography v0.1. It could be removed in the future.

The API of `secp256k1-compat` is the same as [secp256k1-node](https://github.com/cryptocoinjs/secp256k1-node):

```js
import { createPrivateKeySync, ecdsaSign } from "ethereum-cryptography/secp256k1-compat";
const msgHash = Uint8Array.from(
  "82ff40c0a986c6a5cfad4ddf4c3aa6996f1a7837f9c398e17e5de5cbd5a12b28",
  "hex"
);
const privateKey = createPrivateKeySync();
console.log(Uint8Array.from(ecdsaSign(msgHash, privateKey).signature));
```

## Missing cryptographic primitives

This package intentionally excludes the cryptographic primitives necessary
to implement the following EIPs:

* [EIP 196: Precompiled contracts for addition and scalar multiplication on the elliptic curve alt_bn128](https://eips.ethereum.org/EIPS/eip-196)
* [EIP 197: Precompiled contracts for optimal ate pairing check on the elliptic curve alt_bn128](https://eips.ethereum.org/EIPS/eip-197)
* [EIP 198: Big integer modular exponentiation](https://eips.ethereum.org/EIPS/eip-198)
* [EIP 152: Add Blake2 compression function `F` precompile](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-152.md)

Feel free to open an issue if you want this decision to be reconsidered, or if
you found another primitive that is missing.

## Upgrading

Upgrading from 1.0 to 2.0:

1. `secp256k1` module was changed massively:
  before, it was using [noble-secp256k1 1.7](https://github.com/paulmillr/noble-secp256k1);
  now it uses safer [noble-curves](https://github.com/paulmillr/noble-curves). Please refer
  to [upgrading section from curves README](https://github.com/paulmillr/noble-curves#upgrading).
  Main changes to keep in mind: a) `sign` now returns `Signature` instance
  b) `recoverPublicKey` got moved onto a `Signature` instance
2. node.js 14 and older support was dropped. Upgrade to node.js 16 or later.

Upgrading from 0.1 to 1.0: **Same functionality**, all old APIs remain the same except for the breaking changes:

1. We return `Uint8Array` from all methods that worked with `Buffer` before.
`Buffer` has never been supported in browsers, while `Uint8Array`s are supported natively in both
browsers and node.js.
2. We target runtimes with [bigint](https://caniuse.com/bigint) support,
which is Chrome 67+, Edge 79+, Firefox 68+, Safari 14+, node.js 10+. If you need to support older runtimes, use `ethereum-cryptography@0.1`
3. If you've used `secp256k1`, [rename it to `secp256k1-compat`](#legacy-secp256k1-compatibility-layer)

```js
import { sha256 } from "ethereum-cryptography/sha256.js";

// Old usage
const hasho = sha256(Buffer.from("string", "utf8")).toString("hex");

// New usage
import { toHex } from "ethereum-cryptography/utils.js";
const hashn = toHex(sha256("string"));

// If you have `Buffer` module and want to preserve it:
const hashb = Buffer.from(sha256("string"));
const hashbo = hashb.toString("hex");
```

## Security

Audited by Cure53 on Jan 5, 2022. Check out the audit [PDF](./audit/2022-01-05-cure53-audit-nbl2.pdf) & [URL](https://cure53.de/pentest-report_hashing-libs.pdf).

## License

`ethereum-cryptography` is released under The MIT License (MIT)

Copyright (c) 2021 Patricio Palladino, Paul Miller, ethereum-cryptography contributors

See [LICENSE](./LICENSE) file.

`hdkey` is loosely based on [hdkey](https://github.com/cryptocoinjs/hdkey),
which had [MIT License](https://github.com/cryptocoinjs/hdkey/blob/3f3c0b5cedb98f971835b5116ebea05b3c09422a/LICENSE)

Copyright (c) 2018 cryptocoinjs

[1]: https://img.shields.io/npm/v/ethereum-cryptography.svg
[2]: https://www.npmjs.com/package/ethereum-cryptography
[3]: https://img.shields.io/npm/l/ethereum-cryptography
[4]: https://github.com/ethereum/js-ethereum-cryptography/blob/master/packages/ethereum-cryptography/LICENSE
