# scure-bip32

Audited & minimal implementation of BIP32 hierarchical deterministic (HD) wallets over secp256k1.

- 🔒 [Audited](#security) by an independent security firm
- 🔻 Tree-shaking-friendly: use only what's necessary, other code won't be included
- 📦 ESM and common.js
- ➰ Only 3 audited dependencies by the same author:
  [noble-curves](https://github.com/paulmillr/noble-curves),
  [noble-hashes](https://github.com/paulmillr/noble-hashes),
  and [scure-base](https://github.com/paulmillr/scure-base)
- 🪶 300 lines. 90KB with all dependencies

Check out [scure-bip39](https://github.com/paulmillr/scure-bip39) if you need mnemonic phrases.
See [ed25519-keygen](https://github.com/paulmillr/ed25519-keygen) if you need SLIP-0010/BIP32 ed25519 hdkey implementation.

### This library belongs to *scure*

> **scure** — audited micro-libraries.

- Zero or minimal dependencies
- Highly readable TypeScript / JS code
- PGP-signed releases and transparent NPM builds
- Check out [homepage](https://paulmillr.com/noble/#scure) & all libraries:
  [base](https://github.com/paulmillr/scure-base),
  [bip32](https://github.com/paulmillr/scure-bip32),
  [bip39](https://github.com/paulmillr/scure-bip39),
  [btc-signer](https://github.com/paulmillr/scure-btc-signer),
  [starknet](https://github.com/paulmillr/scure-starknet)

## Usage

> npm install @scure/bip32

This module exports a single class `HDKey`, which should be used like this:

```ts
import { HDKey } from "@scure/bip32";
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

The module implements [bip32](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki) standard:
check it out for additional documentation.

The implementation is loosely based on cryptocoinjs/hdkey, [which has MIT License](#LICENSE).

## Security

The library has been independently audited:

- at version 1.0.1, in Jan 2022, by [cure53](https://cure53.de)
  - PDFs: [online](https://cure53.de/pentest-report_hashing-libs.pdf), [offline](./audit/2022-01-05-cure53-audit-nbl2.pdf)
  - [Changes since audit](https://github.com/paulmillr/scure-bip32/compare/1.0.0..main).
  - The audit has been funded by [Ethereum Foundation](https://ethereum.org/en/) with help of [Nomic Labs](https://nomiclabs.io)

The library was initially developed for [js-ethereum-cryptography](https://github.com/ethereum/js-ethereum-cryptography).
At commit [ae00e6d7](https://github.com/ethereum/js-ethereum-cryptography/commit/ae00e6d7d24fb3c76a1c7fe10039f6ecd120b77e),
it was extracted to a separate package called `micro-bip32`.
After the audit we've decided to use `@scure` NPM namespace for security.

## License

[MIT License](./LICENSE)

Copyright (c) 2022 Patricio Palladino, Paul Miller (paulmillr.com)
