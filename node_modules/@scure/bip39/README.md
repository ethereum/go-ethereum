# scure-bip39

Secure, [audited](#security) & minimal implementation of BIP39 mnemonic phrases.

Compared to popular `bip39` package, scure-bip39:

- Is 491KB all-bundled instead of 1.3MB
- Uses 2 dependencies instead of 15
- Wordlists are 157KB instead of 315KB
- Had an external security [audit](#security) by Cure53 on Jan 5, 2022

Check out [scure-bip32](https://github.com/paulmillr/scure-bip32) if you need
hierarchical deterministic wallets ("HD Wallets").

### This library belongs to *scure*

> **scure** — secure audited packages for every use case.

- Independent security audits
- All releases are signed with PGP keys
- Check out all libraries:
  [base](https://github.com/paulmillr/scure-base),
  [bip32](https://github.com/paulmillr/scure-bip32),
  [bip39](https://github.com/paulmillr/scure-bip39)

## Usage

> npm install @scure/bip39

Or

> yarn add @scure/bip39

```js
import * as bip39 from '@scure/bip39';
import { wordlist } from '@scure/bip39/wordlists/english';

// Generate x random words. Uses Cryptographically-Secure Random Number Generator.
const mn = bip39.generateMnemonic(wordlist);
console.log(mn);

// Reversible: Converts mnemonic string to raw entropy in form of byte array.
const ent = bip39.mnemonicToEntropy(mn, wordlist)

// Reversible: Converts raw entropy in form of byte array to mnemonic string.
bip39.entropyToMnemonic(ent, wordlist);

// Validates mnemonic for being 12-24 words contained in `wordlist`.
bip39.validateMnemonic(mn, wordlist);

// Irreversible: Uses KDF to derive 64 bytes of key data from mnemonic + optional password.
await bip39.mnemonicToSeed(mn, 'password');
bip39.mnemonicToSeedSync(mn, 'password');
```

This submodule contains the word lists defined by BIP39 for Czech, English, French, Italian, Japanese, Korean, Simplified and Traditional Chinese, and Spanish. These are not imported by default, as that would increase bundle sizes too much. Instead, you should import and use them explicitly.

```typescript
function generateMnemonic(wordlist: string[], strength?: number): string;
function mnemonicToEntropy(mnemonic: string, wordlist: string[]): Uint8Array;
function entropyToMnemonic(entropy: Uint8Array, wordlist: string[]): string;
function validateMnemonic(mnemonic: string, wordlist: string[]): boolean;
function mnemonicToSeed(mnemonic: string, passphrase?: string): Promise<Uint8Array>;
function mnemonicToSeedSync(mnemonic: string, passphrase?: string): Uint8Array;
```

All wordlists:

```typescript
import { wordlist } from '@scure/bip39/wordlists/czech';
import { wordlist } from '@scure/bip39/wordlists/english';
import { wordlist } from '@scure/bip39/wordlists/french';
import { wordlist } from '@scure/bip39/wordlists/italian';
import { wordlist } from '@scure/bip39/wordlists/japanese';
import { wordlist } from '@scure/bip39/wordlists/korean';
import { wordlist } from '@scure/bip39/wordlists/simplified-chinese';
import { wordlist } from '@scure/bip39/wordlists/spanish';
import { wordlist } from '@scure/bip39/wordlists/traditional-chinese';
```

## Security

The library has been audited by Cure53 on Jan 5, 2022. Check out the audit [PDF](./audit/2022-01-05-cure53-audit-nbl2.pdf) & [URL](https://cure53.de/pentest-report_hashing-libs.pdf). See [changes since audit](https://github.com/paulmillr/scure-bip39/compare/1.0.0..main).

1. The library was initially developed for [js-ethereum-cryptography](https://github.com/ethereum/js-ethereum-cryptography)
2. At commit [ae00e6d7](https://github.com/ethereum/js-ethereum-cryptography/commit/ae00e6d7d24fb3c76a1c7fe10039f6ecd120b77e), it
  was extracted to a separate package called `micro-bip39`
3. After the audit we've decided to use NPM namespace for security. Since `@micro` namespace was taken, we've renamed the package to `@scure/bip39`

## License

[MIT License](./LICENSE)

Copyright (c) 2022 Patricio Palladino, Paul Miller (paulmillr.com)
