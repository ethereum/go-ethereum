# noble-hashes

Audited & minimal JS implementation of hash functions, MACs and KDFs.

- 🔒 [**Audited**](#security) by an independent security firm
- 🔻 Tree-shakeable: unused code is excluded from your builds
- 🏎 Fast: hand-optimized for caveats of JS engines
- 🔍 Reliable: chained / sliding window / DoS tests and fuzzing ensure correctness
- 🔁 No unrolled loops: makes it easier to verify and reduces source code size up to 5x
- 🦘 Includes SHA, RIPEMD, BLAKE, HMAC, HKDF, PBKDF, Scrypt, Argon2 & KangarooTwelve
- 🪶 48KB for everything, 5KB (2.5KB gzipped) for single-hash build

Take a glance at [GitHub Discussions](https://github.com/paulmillr/noble-hashes/discussions) for questions and support.
The library's initial development was funded by [Ethereum Foundation](https://ethereum.org/).

### This library belongs to _noble_ cryptography

> **noble cryptography** — high-security, easily auditable set of contained cryptographic libraries and tools.

- Zero or minimal dependencies
- Highly readable TypeScript / JS code
- PGP-signed releases and transparent NPM builds
- All libraries:
  [ciphers](https://github.com/paulmillr/noble-ciphers),
  [curves](https://github.com/paulmillr/noble-curves),
  [hashes](https://github.com/paulmillr/noble-hashes),
  [post-quantum](https://github.com/paulmillr/noble-post-quantum),
  4kb [secp256k1](https://github.com/paulmillr/noble-secp256k1) /
  [ed25519](https://github.com/paulmillr/noble-ed25519)
- [Check out homepage](https://paulmillr.com/noble/)
  for reading resources, documentation and apps built with noble

## Usage

> `npm install @noble/hashes`

> `deno add jsr:@noble/hashes`

> `deno doc jsr:@noble/hashes` # command-line documentation

We support all major platforms and runtimes.
For React Native, you may need a [polyfill for getRandomValues](https://github.com/LinusU/react-native-get-random-values).
A standalone file [noble-hashes.js](https://github.com/paulmillr/noble-hashes/releases) is also available.

```js
// import * from '@noble/hashes'; // Error: use sub-imports, to ensure small app size
import { sha256 } from '@noble/hashes/sha2'; // ESM & Common.js
sha256(new Uint8Array([1, 2, 3]); // returns Uint8Array
sha256('abc'); // == sha256(new TextEncoder().encode('abc'))

// Available modules
import { sha256, sha384, sha512, sha224, sha512_224, sha512_256 } from '@noble/hashes/sha2';
import { sha3_256, sha3_512, keccak_256, keccak_512, shake128, shake256 } from '@noble/hashes/sha3';
import { cshake256, turboshake256, kmac256, tuplehash256, k12, m14, keccakprg } from '@noble/hashes/sha3-addons';
import { ripemd160 } from '@noble/hashes/ripemd160';
import { blake3 } from '@noble/hashes/blake3';
import { blake2b } from '@noble/hashes/blake2b';
import { blake2s } from '@noble/hashes/blake2s';
import { blake256, blake512 } from '@noble/hashes/blake1';
import { hmac } from '@noble/hashes/hmac';
import { hkdf } from '@noble/hashes/hkdf';
import { pbkdf2, pbkdf2Async } from '@noble/hashes/pbkdf2';
import { scrypt, scryptAsync } from '@noble/hashes/scrypt';
import * as utils from '@noble/hashes/utils'; // bytesToHex, hexToBytes, etc
```

- [sha2: sha256, sha384, sha512](#sha2-sha256-sha384-sha512-and-others)
- [sha3: FIPS, SHAKE, Keccak](#sha3-fips-shake-keccak)
- [sha3-addons: cSHAKE, KMAC, K12, M14, TurboSHAKE](#sha3-addons-cshake-kmac-k12-m14-turboshake)
- [ripemd160](#ripemd160) | [blake, blake2b, blake2s, blake3](#blake-blake2b-blake2s-blake3) | [legacy: sha1, md5](#legacy-sha1-md5)
- MACs: [hmac](#hmac) | [sha3-addons kmac](#sha3-addons-cshake-kmac-k12-m14-turboshake) | [blake3 key mode](#blake2b-blake2s-blake3)
- KDFs: [hkdf](#hkdf) | [pbkdf2](#pbkdf2) | [scrypt](#scrypt) | [argon2](#argon2)
- [utils](#utils)
- [Security](#security) | [Speed](#speed) | [Contributing & testing](#contributing--testing) | [License](#license)

### Implementations

Hash functions:

- receive & return `Uint8Array`
- may receive `string` **(not hex)**, which is automatically utf8-encoded to `Uint8Array`
- support little-endian architecture; also experimentally big-endian
- can hash up to 4GB per chunk, with any amount of chunks
- can be constructed via `hash.create()` method
  - the result is `Hash` subclass instance, which has `update()` and `digest()` methods
  - `digest()` finalizes the hash and makes it no longer usable
- some of them can receive `options`:
  - second argument to hash function: `blake3('abc', { key: 'd', dkLen: 32 })`
  - first argument to class initializer: `blake3.create({ context: 'e', dkLen: 32 })`

#### sha2: sha256, sha384, sha512 and others

```typescript
import { sha256, sha384, sha512, sha224, sha512_224, sha512_256 } from '@noble/hashes/sha2';
// also available as aliases:
// import ... from '@noble/hashes/sha256'
// import ... from '@noble/hashes/sha512'

// Variant A:
const h1a = sha256('abc');

// Variant B:
const h1b = sha256
  .create()
  .update(Uint8Array.from([1, 2, 3]))
  .digest();

for (let hash of [sha384, sha512, sha224, sha512_224, sha512_256]) {
  const res1 = hash('abc');
  const res2 = hash
    .create()
    .update('def')
    .update(Uint8Array.from([1, 2, 3]))
    .digest();
}
```

See [RFC 4634](https://datatracker.ietf.org/doc/html/rfc4634) and
[the paper on truncated SHA512/256](https://eprint.iacr.org/2010/548.pdf).

#### sha3: FIPS, SHAKE, Keccak

```typescript
// prettier-ignore
import {
  sha3_224, sha3_256, sha3_384, sha3_512,
  keccak_224, keccak_256, keccak_384, keccak_512,
  shake128, shake256,
} from '@noble/hashes/sha3';
const h5a = sha3_256('abc');
const h5b = sha3_256
  .create()
  .update(Uint8Array.from([1, 2, 3]))
  .digest();
const h6a = keccak_256('abc');
const h7a = shake128('abc', { dkLen: 512 });
const h7b = shake256('abc', { dkLen: 512 });
```

See [FIPS-202](https://nvlpubs.nist.gov/nistpubs/FIPS/NIST.FIPS.202.pdf),
[Website](https://keccak.team/keccak.html).

Check out [the differences between SHA-3 and Keccak](https://crypto.stackexchange.com/questions/15727/what-are-the-key-differences-between-the-draft-sha-3-standard-and-the-keccak-sub)

#### sha3-addons: cSHAKE, KMAC, K12, M14, TurboSHAKE

```typescript
// prettier-ignore
import {
  cshake128, cshake256,
  turboshake128, turboshake256,
  kmac128, kmac256,
  tuplehash256, parallelhash256,
  k12, m14, keccakprg
} from '@noble/hashes/sha3-addons';
const h7c = cshake128('abc', { personalization: 'def' });
const h7d = cshake256('abc', { personalization: 'def' });
const h7e = kmac128('key', 'message');
const h7f = kmac256('key', 'message');
const h7h = k12('abc');
const h7g = m14('abc');
const h7t1 = turboshake128('abc');
const h7t2 = turboshake256('def', { D: 0x05 });
const h7i = tuplehash256(['ab', 'c']); // tuplehash(['ab', 'c']) !== tuplehash(['a', 'bc']) !== tuplehash(['abc'])
// Same as k12/blake3, but without reduced number of rounds. Doesn't speedup anything due lack of SIMD and threading,
// added for compatibility.
const h7j = parallelhash256('abc', { blockLen: 8 });
// pseudo-random generator, first argument is capacity. XKCP recommends 254 bits capacity for 128-bit security strength.
// * with a capacity of 254 bits.
const p = keccakprg(254);
p.feed('test');
const rand1b = p.fetch(1);
```

- Full [NIST SP 800-185](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-185.pdf):
  cSHAKE, KMAC, TupleHash, ParallelHash + XOF variants
- [Reduced-round Keccak](https://datatracker.ietf.org/doc/draft-irtf-cfrg-kangarootwelve/):
  - 🦘 K12 aka KangarooTwelve
  - M14 aka MarsupilamiFourteen
  - TurboSHAKE
- [KeccakPRG](https://keccak.team/files/CSF-0.1.pdf): Pseudo-random generator based on Keccak

#### ripemd160

```typescript
import { ripemd160 } from '@noble/hashes/ripemd160';
const hash8 = ripemd160('abc');
const hash9 = ripemd160
  .create()
  .update(Uint8Array.from([1, 2, 3]))
  .digest();
```

See [RFC 2286](https://datatracker.ietf.org/doc/html/rfc2286),
[Website](https://homes.esat.kuleuven.be/~bosselae/ripemd160.html)

#### blake, blake2b, blake2s, blake3

```typescript
import { blake224, blake256, blake384, blake512 } from '@noble/hashes/blake1';
import { blake2b } from '@noble/hashes/blake2b';
import { blake2s } from '@noble/hashes/blake2s';
import { blake3 } from '@noble/hashes/blake3';

const h_b1_224 = blake224('abc');
const h_b1_256 = blake256('abc');
const h_b1_384 = blake384('abc');
const h_b1_512 = blake512('abc');

const h10a = blake2s('abc');
const b2params = { key: new Uint8Array([1]), personalization: t, salt: t, dkLen: 32 };
const h10b = blake2s('abc', b2params);
const h10c = blake2s
  .create(b2params)
  .update(Uint8Array.from([1, 2, 3]))
  .digest();

// All params are optional
const h11 = blake3('abc', { dkLen: 256 });
const h11_mac = blake3('abc', { key: new Uint8Array(32) });
const h11_kdf = blake3('abc', { context: 'application name' });
```

- Blake1 is legacy hash, one of SHA3 proposals. It is rarely used anywhere. See [pdf](https://www.aumasson.jp/blake/blake.pdf).
- Blake2 is popular fast hash. blake2b focuses on 64-bit platforms while blake2s is for 8-bit to 32-bit ones. See [RFC 7693](https://datatracker.ietf.org/doc/html/rfc7693), [Website](https://www.blake2.net)
- Blake3 is faster, reduced-round blake2. See [Website & specs](https://blake3.io)

#### legacy: sha1, md5

SHA1 (RFC 3174) and MD5 (RFC 1321) legacy, broken hash functions.
Don't use them in a new protocol. What "broken" means:

- Collisions can be made with 2^18 effort in MD5, 2^60 in SHA1.
- No practical pre-image attacks (only theoretical, 2^123.4)
- HMAC seems kinda ok: https://datatracker.ietf.org/doc/html/rfc6151

```typescript
import { sha1, md5 } from '@noble/hashes/legacy';
const h12s = sha1('def');
const h12m = md5('xcb');
```

#### hmac

```typescript
import { hmac } from '@noble/hashes/hmac';
import { sha256 } from '@noble/hashes/sha2';
const mac1 = hmac(sha256, 'key', 'message');
const mac2 = hmac
  .create(sha256, Uint8Array.from([1, 2, 3]))
  .update(Uint8Array.from([4, 5, 6]))
  .digest();
```

Matches [RFC 2104](https://datatracker.ietf.org/doc/html/rfc2104).

#### hkdf

```typescript
import { hkdf } from '@noble/hashes/hkdf';
import { sha256 } from '@noble/hashes/sha2';
import { randomBytes } from '@noble/hashes/utils';
const inputKey = randomBytes(32);
const salt = randomBytes(32);
const info = 'application-key';
const hk1 = hkdf(sha256, inputKey, salt, info, 32);

// == same as
import * as hkdf from '@noble/hashes/hkdf';
import { sha256 } from '@noble/hashes/sha2';
const prk = hkdf.extract(sha256, inputKey, salt);
const hk2 = hkdf.expand(sha256, prk, info, dkLen);
```

Matches [RFC 5869](https://datatracker.ietf.org/doc/html/rfc5869).

#### pbkdf2

```typescript
import { pbkdf2, pbkdf2Async } from '@noble/hashes/pbkdf2';
import { sha256 } from '@noble/hashes/sha2';
const pbkey1 = pbkdf2(sha256, 'password', 'salt', { c: 32, dkLen: 32 });
const pbkey2 = await pbkdf2Async(sha256, 'password', 'salt', { c: 32, dkLen: 32 });
const pbkey3 = await pbkdf2Async(sha256, Uint8Array.from([1, 2, 3]), Uint8Array.from([4, 5, 6]), {
  c: 32,
  dkLen: 32,
});
```

Matches [RFC 2898](https://datatracker.ietf.org/doc/html/rfc2898).

#### scrypt

```typescript
import { scrypt, scryptAsync } from '@noble/hashes/scrypt';
const scr1 = scrypt('password', 'salt', { N: 2 ** 16, r: 8, p: 1, dkLen: 32 });
const scr2 = await scryptAsync('password', 'salt', { N: 2 ** 16, r: 8, p: 1, dkLen: 32 });
const scr3 = await scryptAsync(Uint8Array.from([1, 2, 3]), Uint8Array.from([4, 5, 6]), {
  N: 2 ** 17,
  r: 8,
  p: 1,
  dkLen: 32,
  onProgress(percentage) {
    console.log('progress', percentage);
  },
  maxmem: 2 ** 32 + 128 * 8 * 1, // N * r * p * 128 + (128*r*p)
});
```

Conforms to [RFC 7914](https://datatracker.ietf.org/doc/html/rfc7914),
[Website](https://www.tarsnap.com/scrypt.html)

- `N, r, p` are work factors. To understand them, see [the blog post](https://blog.filippo.io/the-scrypt-parameters/).
  `r: 8, p: 1` are common. JS doesn't support parallelization, making increasing p meaningless.
- `dkLen` is the length of output bytes e.g. `32` or `64`
- `onProgress` can be used with async version of the function to report progress to a user.
- `maxmem` prevents DoS and is limited to `1GB + 1KB` (`2**30 + 2**10`), but can be adjusted using formula: `N * r * p * 128 + (128 * r * p)`

Time it takes to derive Scrypt key under different values of N (2\*\*N) on Apple M4 (mobile phones can be 1x-4x slower):

| N pow | Time | RAM   |
| ----- | ---- | ----- |
| 16    | 0.1s | 64MB  |
| 17    | 0.2s | 128MB |
| 18    | 0.4s | 256MB |
| 19    | 0.8s | 512MB |
| 20    | 1.5s | 1GB   |
| 21    | 3.1s | 2GB   |
| 22    | 6.2s | 4GB   |
| 23    | 13s  | 8GB   |
| 24    | 27s  | 16GB  |

> [!NOTE]
> We support N larger than `2**20` where available, however,
> not all JS engines support >= 2GB ArrayBuffer-s.
> When using such N, you'll need to manually adjust `maxmem`, using formula above.
> Other JS implementations don't support large N-s.

#### argon2

```ts
import { argon2d, argon2i, argon2id } from '@noble/hashes/argon2';
const result = argon2id('password', 'saltsalt', { t: 2, m: 65536, p: 1, maxmem: 2 ** 32 - 1 });
```

Argon2 [RFC 9106](https://datatracker.ietf.org/doc/html/rfc9106) implementation.

> [!WARNING]
> Argon2 can't be fast in JS, because there is no fast Uint64Array.
> It is suggested to use [Scrypt](#scrypt) instead.
> Being 5x slower than native code means brute-forcing attackers have bigger advantage.

#### utils

```typescript
import { bytesToHex as toHex, randomBytes } from '@noble/hashes/utils';
console.log(toHex(randomBytes(32)));
```

- `bytesToHex` will convert `Uint8Array` to a hex string
- `randomBytes(bytes)` will produce cryptographically secure random `Uint8Array` of length `bytes`

## Security

The library has been independently audited:

- at version 1.0.0, in Jan 2022, by [Cure53](https://cure53.de)
  - PDFs: [website](https://cure53.de/pentest-report_hashing-libs.pdf), [in-repo](./audit/2022-01-05-cure53-audit-nbl2.pdf)
  - [Changes since audit](https://github.com/paulmillr/noble-hashes/compare/1.0.0..main).
  - Scope: everything, besides `blake3`, `sha3-addons`, `sha1` and `argon2`, which have not been audited
  - The audit has been funded by [Ethereum Foundation](https://ethereum.org/en/) with help of [Nomic Labs](https://nomiclabs.io)

It is tested against property-based, cross-library and Wycheproof vectors,
and is being fuzzed in [the separate repo](https://github.com/paulmillr/fuzzing).

If you see anything unusual: investigate and report.

### Constant-timeness

We're targetting algorithmic constant time. _JIT-compiler_ and _Garbage Collector_ make "constant time"
extremely hard to achieve [timing attack](https://en.wikipedia.org/wiki/Timing_attack) resistance
in a scripting language. Which means _any other JS library can't have
constant-timeness_. Even statically typed Rust, a language without GC,
[makes it harder to achieve constant-time](https://www.chosenplaintext.ca/open-source/rust-timing-shield/security)
for some cases. If your goal is absolute security, don't use any JS lib — including bindings to native ones.
Use low-level libraries & languages.

### Memory dumping

The library shares state buffers between hash
function calls. The buffers are zeroed-out after each call. However, if an attacker
can read application memory, you are doomed in any case:

- At some point, input will be a string and strings are immutable in JS:
  there is no way to overwrite them with zeros. For example: deriving
  key from `scrypt(password, salt)` where password and salt are strings
- Input from a file will stay in file buffers
- Input / output will be re-used multiple times in application which means it could stay in memory
- `await anything()` will always write all internal variables (including numbers)
  to memory. With async functions / Promises there are no guarantees when the code
  chunk would be executed. Which means attacker can have plenty of time to read data from memory
- There is no way to guarantee anything about zeroing sensitive data without
  complex tests-suite which will dump process memory and verify that there is
  no sensitive data left. For JS it means testing all browsers (incl. mobile),
  which is complex. And of course it will be useless without using the same
  test-suite in the actual application that consumes the library

### Supply chain security

- **Commits** are signed with PGP keys, to prevent forgery. Make sure to verify commit signatures
- **Releases** are transparent and built on GitHub CI. Make sure to verify [provenance](https://docs.npmjs.com/generating-provenance-statements) logs
  - Use GitHub CLI to verify single-file builds:
    `gh attestation verify --owner paulmillr noble-hashes.js`
- **Rare releasing** is followed to ensure less re-audit need for end-users
- **Dependencies** are minimized and locked-down: any dependency could get hacked and users will be downloading malware with every install.
  - We make sure to use as few dependencies as possible
  - Automatic dep updates are prevented by locking-down version ranges; diffs are checked with `npm-diff`
- **Dev Dependencies** are disabled for end-users; they are only used to develop / build the source code

For this package, there are 0 dependencies; and a few dev dependencies:

- micro-bmark, micro-should and jsbt are used for benchmarking / testing / build tooling and developed by the same author
- prettier, fast-check and typescript are used for code quality / test generation / ts compilation. It's hard to audit their source code thoroughly and fully because of their size

### Randomness

We're deferring to built-in
[crypto.getRandomValues](https://developer.mozilla.org/en-US/docs/Web/API/Crypto/getRandomValues)
which is considered cryptographically secure (CSPRNG).

In the past, browsers had bugs that made it weak: it may happen again.
Implementing a userspace CSPRNG to get resilient to the weakness
is even worse: there is no reliable userspace source of quality entropy.

### Quantum computers

Cryptographically relevant quantum computer, if built, will allow to
utilize Grover's algorithm to break hashes in 2^n/2 operations, instead of 2^n.

This means SHA256 should be replaced with SHA512, SHA3-256 with SHA3-512, SHAKE128 with SHAKE256 etc.

Australian ASD prohibits SHA256 and similar hashes [after 2030](https://www.cyber.gov.au/resources-business-and-government/essential-cyber-security/ism/cyber-security-guidelines/guidelines-cryptography).

## Speed

```sh
npm run bench:install && npm run bench
```

Benchmarks measured on Apple M4.

```
# 32B
sha256 x 1,968,503 ops/sec @ 508ns/op
sha512 x 740,740 ops/sec @ 1μs/op
sha3_256 x 287,686 ops/sec @ 3μs/op
sha3_512 x 288,267 ops/sec @ 3μs/op
k12 x 476,190 ops/sec @ 2μs/op
m14 x 423,190 ops/sec @ 2μs/op
blake2b x 464,252 ops/sec @ 2μs/op
blake2s x 766,871 ops/sec @ 1μs/op
blake3 x 879,507 ops/sec @ 1μs/op
ripemd160 x 1,757,469 ops/sec @ 569ns/op

# 1MB
sha256 x 331 ops/sec @ 3ms/op
sha512 x 129 ops/sec @ 7ms/op
sha3_256 x 38 ops/sec @ 25ms/op
sha3_512 x 20 ops/sec @ 47ms/op
k12 x 88 ops/sec @ 11ms/op
m14 x 62 ops/sec @ 15ms/op
blake2b x 69 ops/sec @ 14ms/op
blake2s x 57 ops/sec @ 17ms/op
blake3 x 72 ops/sec @ 13ms/op
ripemd160 x 264 ops/sec @ 3ms/op

# MAC
hmac(sha256) x 599,880 ops/sec @ 1μs/op
hmac(sha512) x 197,122 ops/sec @ 5μs/op
kmac256 x 87,981 ops/sec @ 11μs/op
blake3(key) x 796,812 ops/sec @ 1μs/op

# KDF
hkdf(sha256) x 259,942 ops/sec @ 3μs/op
blake3(context) x 424,808 ops/sec @ 2μs/op
pbkdf2(sha256, c: 2 ** 18) x 5 ops/sec @ 197ms/op
pbkdf2(sha512, c: 2 ** 18) x 1 ops/sec @ 630ms/op
scrypt(n: 2 ** 18, r: 8, p: 1) x 2 ops/sec @ 400ms/op
argon2id(t: 1, m: 256MB) 2881ms
```

Compare to native node.js implementation that uses C bindings instead of pure-js code:

```
# native (node) 32B
sha256 x 2,267,573 ops/sec
sha512 x 983,284 ops/sec
sha3_256 x 1,522,070 ops/sec
blake2b x 1,512,859 ops/sec
blake2s x 1,821,493 ops/sec
ripemd160 x 1,855,287 ops/sec
hmac(sha256) x 1,085,776 ops/sec
hkdf(sha256) x 312,109 ops/sec
# native (node) KDF
pbkdf2(sha256, c: 2 ** 18) x 5 ops/sec @ 197ms/op
pbkdf2(sha512, c: 2 ** 18) x 1 ops/sec @ 630ms/op
scrypt(n: 2 ** 18, r: 8, p: 1) x 2 ops/sec @ 378ms/op
```

It is possible to [make this library 4x+ faster](./benchmark/README.md) by
_doing code generation of full loop unrolls_. We've decided against it. Reasons:

- the library must be auditable, with minimum amount of code, and zero dependencies
- most method invocations with the lib are going to be something like hashing 32b to 64kb of data
- hashing big inputs is 10x faster with low-level languages, which means you should probably pick 'em instead

The current performance is good enough when compared to other projects; SHA256 takes only 900 nanoseconds to run.

## Contributing & testing

`test/misc` directory contains implementations of loop unrolling and md5.

- `npm install && npm run build && npm test` will build the code and run tests.
- `npm run lint` / `npm run format` will run linter / fix linter issues.
- `npm run bench` will run benchmarks, which may need their deps first (`npm run bench:install`)
- `npm run build:release` will build single file
- There is **additional** 20-min DoS test `npm run test:dos` and 2-hour "big" multicore test `npm run test:big`.
  See [our approach to testing](./test/README.md)

Additional resources:

- NTT hashes are outside of scope of the library. You can view some of them in different repos:
    - [Pedersen in micro-zk-proofs](https://github.com/paulmillr/micro-zk-proofs/blob/1ed5ce1253583b2e540eef7f3477fb52bf5344ff/src/pedersen.ts)
    - [Poseidon in noble-curves](https://github.com/paulmillr/noble-curves/blob/3d124dd3ecec8b6634cc0b2ba1c183aded5304f9/src/abstract/poseidon.ts)
- Check out [guidelines](https://github.com/paulmillr/guidelines) for coding practices
- See [paulmillr.com/noble](https://paulmillr.com/noble/) for useful resources, articles, documentation and demos
related to the library.

## License

The MIT License (MIT)

Copyright (c) 2022 Paul Miller [(https://paulmillr.com)](https://paulmillr.com)

See LICENSE file.
