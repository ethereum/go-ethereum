# scure-bip32

Secure, [audited](#security) & minimal implementation of BIP32 hierarchical deterministic (HD) wallets.

Compared to popular `hdkey` package, scure-bip32:

- Is 418KB all-bundled instead of 5.9MB
- Uses 3 dependencies instead of 24
- Had an external security [audit](#security) by Cure53 on Jan 5, 2022

Check out [scure-bip39](https://github.com/paulmillr/scure-bip39) if you need mnemonic phrases. See [micro-ed25519-hdkey](https://github.com/paulmillr/micro-ed25519-hdkey) if you need SLIP-0010/BIP32 HDKey implementation.

### This library belongs to *scure*

> **scure** — secure audited packages for every use case.

- Independent security audits
- All releases are signed with PGP keys
- Check out all libraries:
  [base](https://github.com/paulmillr/scure-base),
  [bip32](https://github.com/paulmillr/scure-bip32),
  [bip39](https://github.com/paulmillr/scure-bip39)

## Usage

> npm install @scure/bip32

Or

> yarn add @scure/bip32

This module exports a single class `HDKey`, which should be used like this:

```ts
const { HDKey } = require("@scure/bip32");
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
Its only difference is that it has to be be used with a named import.
The implementation is [loosely based on hdkey, which has MIT License](#LICENSE).

## Security

The library has been audited by Cure53 on Jan 5, 2022. Check out the audit [PDF](./audit/2022-01-05-cure53-audit-nbl2.pdf) & [URL](https://cure53.de/pentest-report_hashing-libs.pdf). See [changes since audit](https://github.com/paulmillr/scure-bip32/compare/1.0.1..main).

1. The library was initially developed for [js-ethereum-cryptography](https://github.com/ethereum/js-ethereum-cryptography)
2. At commit [ae00e6d7](https://github.com/ethereum/js-ethereum-cryptography/commit/ae00e6d7d24fb3c76a1c7fe10039f6ecd120b77e), it
  was extracted to a separate package called `micro-bip32`
3. After the audit we've decided to use NPM namespace for security. Since `@micro` namespace was taken, we've renamed the package to `@scure/bip32`

## License

[MIT License](./LICENSE)

Copyright (c) 2022 Patricio Palladino, Paul Miller (paulmillr.com)
