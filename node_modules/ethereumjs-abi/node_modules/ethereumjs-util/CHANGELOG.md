# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
(modification: no type change headlines) and this project adheres to
[Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [6.2.1] - 2020-07-17

This release replaces the native `secp256k1` and `keccak` dependencies with
[ethereum-cryptopgraphy](https://github.com/ethereum/js-ethereum-cryptography)
which doesn't need native compilation.

[6.2.1]: https://github.com/ethereumjs/ethereumjs-util/compare/v6.2.0...v6.2.1

## [6.2.0] - 2019-11-06

This release comes with a new file structure, related functionality is now broken
down into separate files (like `account.js`) allowing for more oversight and
modular integration. All functionality is additionally exposed through an
aggregating `index.js` file, so this version remains backwards-compatible.

Overview on the new structure:

- `account`: Private/public key and address-related functionality
  (creation, validation, conversion)
- `byte`: Byte-related helper and conversion functions
- `constants`: Exposed constants (e.g. `KECCAK256_NULL_S` for the string
  representation of the Keccak-256 hash of null)
- `hash`: Hash functions
- `object`: Helper function for creating a binary object (`DEPRECATED`)
- `signature`: Signing, signature validation, conversion, recovery

See associated PRs [#182](https://github.com/ethereumjs/ethereumjs-util/pull/182)
and [#179](https://github.com/ethereumjs/ethereumjs-util/pull/179).

**Features**

- `account`: Added `EIP-1191` address checksum algorithm support for
  `toChecksumAddress()`,
  PR [#204](https://github.com/ethereumjs/ethereumjs-util/pull/204)

**Bug Fixes**

- `bytes`: `toBuffer()` conversion function now throws if strings aren't
  `0x`-prefixed hex values making the behavior of `toBuffer()` more predictable
  respectively less error-prone (you might generally want to check cases in your
  code where you eventually allowed non-`0x`-prefixed input before),
  PR [#197](https://github.com/ethereumjs/ethereumjs-util/pull/197)

**Dependencies / Environment**

- Dropped Node `6`, added Node `11` and `12` to officially supported Node versions,
  PR [#207](https://github.com/ethereumjs/ethereumjs-util/pull/207)
- Dropped `safe-buffer` dependency,
  PR [#182](https://github.com/ethereumjs/ethereumjs-util/pull/182)
- Updated `rlp` dependency from `v2.0.0` to `v2.2.3` (`TypeScript` improvements
  for RLP hash functionality),
  PR [#187](https://github.com/ethereumjs/ethereumjs-util/pull/187)
- Made `@types/bn.js` a `dependency` instead of a `devDependency`,
  PR [#205](https://github.com/ethereumjs/ethereumjs-util/pull/205)
- Updated `keccak256` dependency from `v1.4.0` to `v2.0.0`, PR [#168](https://github.com/ethereumjs/ethereumjs-util/pull/168)

[6.2.0]: https://github.com/ethereumjs/ethereumjs-util/compare/v6.1.0...v6.2.0

## [6.1.0] - 2019-02-12

First **TypeScript** based release of the library, now also including a
**type declaration file** distributed along with the package published,
see PR [#170](https://github.com/ethereumjs/ethereumjs-util/pull/170).

**Bug Fixes**

- Fixed a bug in `isValidSignature()` not correctly returning `false`
  if passed an `s`-value greater than `secp256k1n/2` on `homestead` or later.
  If you use the method signature with more than three arguments (so not just
  passing in `v`, `r`, `s` and use it like `isValidSignature(v, r, s)` and omit
  the optional args) please read the thread from
  PR [#171](https://github.com/ethereumjs/ethereumjs-util/pull/171) carefully
  and check your code.

**Development**

- Updated `@types/node` to Node `11` types,
  PR [#175](https://github.com/ethereumjs/ethereumjs-util/pull/175)
- Changed browser from Chrome to ChromeHeadless,
  PR [#156](https://github.com/ethereumjs/ethereumjs-util/pull/156)

[6.1.0]: https://github.com/ethereumjs/ethereumjs-util/compare/v6.0.0...v6.1.0

## [6.0.0] - 2018-10-08

- Support for `EIP-155` replay protection by adding an optional `chainId` parameter
  to `ecsign()`, `ecrecover()`, `toRpcSig()` and `isValidSignature()`, if present the  
  new signature format relying on the `chainId` is used, see PR [#143](https://github.com/ethereumjs/ethereumjs-util/pull/143)
- New `generateAddress2()` for `CREATE2` opcode (`EIP-1014`) address creation
  (Constantinople HF), see PR [#146](https://github.com/ethereumjs/ethereumjs-util/pull/146)
- [BREAKING] Fixed signature to comply with Geth and Parity in `toRpcSig()` changing
  `v` from 0/1 to 27/28, this changes the resulting signature buffer, see PR [#139](https://github.com/ethereumjs/ethereumjs-util/pull/139)
- [BREAKING] Remove deprecated `sha3`-named constants and methods (see `v5.2.0` release),
  see PR [#154](https://github.com/ethereumjs/ethereumjs-util/pull/154)

[6.0.0]: https://github.com/ethereumjs/ethereumjs-util/compare/v5.2.0...v6.0.0

## [5.2.0] - 2018-04-27

- Rename all `sha3` hash related constants and functions to `keccak`, see
  [this](https://github.com/ethereum/EIPs/issues/59) EIP discussion for context
  (tl;dr: Ethereum uses a slightly different hash algorithm then in the official
  `SHA-3` standard)
- Renamed constants:
  - `SHA3_NULL_S` -> `KECCAK256_NULL_S`
  - `SHA3_NULL` -> `KECCAK256_NULL`
  - `SHA3_RLP_ARRAY_S` -> `KECCAK256_RLP_ARRAY_S`
  - `SHA3_RLP_ARRAY` -> `KECCAK256_RLP_ARRAY`
  - `SHA3_RLP_S` -> `KECCAK256_RLP_S`
  - `SHA3_RLP` -> `KECCAK256_RLP`
- Renamed functions:
  - `sha3()` -> `keccak()` (number of bits determined in arguments)
- New `keccak256()` alias function for `keccak(a, 256)`
- The usage of the `sha`-named versions is now `DEPRECATED` and the related
  constants and functions will be removed on the next major release `v6.0.0`

[5.2.0]: https://github.com/ethereumjs/ethereumjs-util/compare/v5.1.5...v5.2.0

## [5.1.5] - 2018-02-28

- Fix `browserify` issue leading to 3rd-party build problems, PR [#119](https://github.com/ethereumjs/ethereumjs-util/pull/119)

[5.1.5]: https://github.com/ethereumjs/ethereumjs-util/compare/v5.1.4...v5.1.5

## [5.1.4] - 2018-02-03

- Moved to `ES5` Node distribution version for easier toolchain integration, PR [#114](https://github.com/ethereumjs/ethereumjs-util/pull/114)
- Updated `isPrecompile()` with Byzantium precompile address range, PR [#115](https://github.com/ethereumjs/ethereumjs-util/pull/115)

[5.1.4]: https://github.com/ethereumjs/ethereumjs-util/compare/v5.1.3...v5.1.4

## [5.1.3] - 2018-01-03

- `ES6` syntax updates
- Dropped Node `5` support
- Moved babel to dev dependencies, switched to `env` preset
- Usage of `safe-buffer` instead of Node `Buffer`
- Do not allow capital `0X` as valid address in `isValidAddress()`
- New methods `zeroAddress()` and `isZeroAddress()`
- Updated dependencies

[5.1.3]: https://github.com/ethereumjs/ethereumjs-util/compare/v5.1.2...v5.1.3

## [5.1.2] - 2017-05-31

- Add browserify for `ES2015` compatibility
- Fix hex validation

[5.1.2]: https://github.com/ethereumjs/ethereumjs-util/compare/v5.1.1...v5.1.2

## [5.1.1] - 2017-02-10

- Use hex utils from `ethjs-util`
- Move secp vars into functions
- Dependency updates

[5.1.1]: https://github.com/ethereumjs/ethereumjs-util/compare/v5.1.0...v5.1.1

## [5.1.0] - 2017-02-04

- Fix `toRpcSig()` function
- Updated Buffer creation (`Buffer.from`)
- Dependency updates
- Fix npm error
- Use `keccak` package instead of `keccakjs`
- Helpers for `eth_sign` RPC call

[5.1.0]: https://github.com/ethereumjs/ethereumjs-util/compare/v5.0.1...v5.1.0

## [5.0.1] - 2016-11-08

- Fix `bufferToHex()`

[5.0.1]: https://github.com/ethereumjs/ethereumjs-util/compare/v5.0.0...v5.0.1

## [5.0.0] - 2016-11-08

- Added `isValidSignature()` (ECDSA signature validation)
- Change `v` param in `ecrecover()` from `Buffer` to `int` (breaking change!)
- Fix property alias for setting with initial parameters
- Reject invalid signature lengths for `fromRpcSig()`
- Fix `sha3()` `width` param (byte -> bit)
- Fix overflow bug in `bufferToInt()`

[5.0.0]: https://github.com/ethereumjs/ethereumjs-util/compare/v4.5.0...v5.0.0

## [4.5.0] - 2016-17-12

- Introduced `toMessageSig()` and `fromMessageSig()`

[4.5.0]: https://github.com/ethereumjs/ethereumjs-util/compare/v4.4.1...v4.5.0

## Older releases:

- [4.4.1](https://github.com/ethereumjs/ethereumjs-util/compare/v4.4.0...v4.4.1) - 2016-05-20
- [4.4.0](https://github.com/ethereumjs/ethereumjs-util/compare/v4.3.1...v4.4.0) - 2016-04-26
- [4.3.1](https://github.com/ethereumjs/ethereumjs-util/compare/v4.3.0...v4.3.1) - 2016-04-25
- [4.3.0](https://github.com/ethereumjs/ethereumjs-util/compare/v4.2.0...v4.3.0) - 2016-03-23
- [4.2.0](https://github.com/ethereumjs/ethereumjs-util/compare/v4.1.0...v4.2.0) - 2016-03-18
- [4.1.0](https://github.com/ethereumjs/ethereumjs-util/compare/v4.0.0...v4.1.0) - 2016-03-08
- [4.0.0](https://github.com/ethereumjs/ethereumjs-util/compare/v3.0.0...v4.0.0) - 2016-02-02
- [3.0.0](https://github.com/ethereumjs/ethereumjs-util/compare/v2.0.0...v3.0.0) - 2016-01-20
