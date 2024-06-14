# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [4.0.1]
### Fixed
- Fix mistake in TYPED_MESSAGE_SCHEMA ([#243](https://github.com/MetaMask/eth-sig-util/pull/243))
  - The schema changed in v4 in a way that accidentally disallowed "reference types" (i.e. custom types) apart from the primary type. Reference types are now once again allowed.

## [4.0.0]
### Added
- **BREAKING**: Add subpath exports ([#214](https://github.com/MetaMask/eth-sig-util/pull/214), [#211](https://github.com/MetaMask/eth-sig-util/pull/211))
  - This is breaking because it prevents the import of modules that are not exposed as subpath exports.
- Add `salt` to the EIP-712 `domain` type ([#176](https://github.com/MetaMask/eth-sig-util/pull/176))
- Add additional unit tests ([#146](https://github.com/MetaMask/eth-sig-util/pull/146), [#164](https://github.com/MetaMask/eth-sig-util/pull/164), [#167](https://github.com/MetaMask/eth-sig-util/pull/167), [#169](https://github.com/MetaMask/eth-sig-util/pull/169), [#172](https://github.com/MetaMask/eth-sig-util/pull/172), [#177](https://github.com/MetaMask/eth-sig-util/pull/177), [#180](https://github.com/MetaMask/eth-sig-util/pull/180), [#170](https://github.com/MetaMask/eth-sig-util/pull/170), [#171](https://github.com/MetaMask/eth-sig-util/pull/171), [#178](https://github.com/MetaMask/eth-sig-util/pull/178), [#173](https://github.com/MetaMask/eth-sig-util/pull/173), [#182](https://github.com/MetaMask/eth-sig-util/pull/182), [#184](https://github.com/MetaMask/eth-sig-util/pull/184), [#185](https://github.com/MetaMask/eth-sig-util/pull/185), [#187](https://github.com/MetaMask/eth-sig-util/pull/187))
- Improve documentation ([#157](https://github.com/MetaMask/eth-sig-util/pull/157), [#177](https://github.com/MetaMask/eth-sig-util/pull/177), [#174](https://github.com/MetaMask/eth-sig-util/pull/174), [#180](https://github.com/MetaMask/eth-sig-util/pull/180), [#178](https://github.com/MetaMask/eth-sig-util/pull/178), [#181](https://github.com/MetaMask/eth-sig-util/pull/181), [#186](https://github.com/MetaMask/eth-sig-util/pull/186), [#212](https://github.com/MetaMask/eth-sig-util/pull/212), [#207](https://github.com/MetaMask/eth-sig-util/pull/207), [#213](https://github.com/MetaMask/eth-sig-util/pull/213))

### Changed
- **BREAKING**: Consolidate `signTypedData` and `recoverTypedSignature` functions ([#156](https://github.com/MetaMask/eth-sig-util/pull/156))
  - The functions `signTypedDataLegacy`, `signTypedData`, and `signTypedData_v4` have been replaced with a single `signTypedData` function with a `version` parameter. The `version` parameter determines which type of signature you get.
    - If you used `signTypedDataLegacy`, switch to `signTypedData` with the version `V1`.
    - If you used `signTypedData`, switch to `signTypedData` with the version `V3`.
    - If you used `signTypedData_v4`, switch to `signTypedData` with the version `V4`.
  - The functions `recoverTypedSignatureLegacy`, `recoverTypedSignature`, and `recoverTypedSignature_v4` have been replaced with a single `recoverTypedSignature` function.
    - If you used `recoverTypedSignatureLegacy`, switch to `recoverTypedMessage` with the version `V1`.
    - If you used `recoverTypedMessage`, switch to `recoverTypedMessage` with the version `V3`.
    - If you used `recoverTypedSignature_v4`, switch to `recoverTypedMessage` with the version `V4`.
- **BREAKING**: Rename `TypedDataUtils.sign` to `TypedDataUtils.eip712Hash` ([#104](https://github.com/MetaMask/eth-sig-util/pull/104))
  - This function never actually signed anything. It just created a hash that was later signed. The new name better reflects what the function does.
- **BREAKING**: Move package under `@metamask` npm organization ([#162](https://github.com/MetaMask/eth-sig-util/pull/162))
  - Update your `require` and `import` statements to import `@metamask/eth-sig-util` rather than `eth-sig-util`.
- **BREAKING**: Simplify function type signatures ([#198](https://github.com/MetaMask/eth-sig-util/pull/198))
  - This is only a breaking change for TypeScript projects that were importing types used by the function signatures. The types should be far simpler now.
  - The `TypedData` has been updated to be more restrictive (it only allows valid typed data now), and it was renamed to `TypedDataV1`
- **BREAKING**: Replace `MsgParams` parameters with "options" parameters ([#204](https://github.com/MetaMask/eth-sig-util/pull/204))
  - This affects the following functions:
    - `personalSign`
    - `recoverPersonalSignature`
    - `extractPublicKey`
    - `encrypt`
    - `encryptSafely`
    - `decrypt`
    - `decryptSafely`
    - `signTypedData`
    - `recoverTypedSignature`
  - All parameters are passed in as a single "options" object now, instead of the `MsgParams` type that was used for most of these functions previously. Read each function signature carefully to ensure you are correctly passing in parameters.
  - `personalSign` example:
    - Previously it was called like this: `personalSign(privateKey, { data })`
    - Now it is called like this: `personalSign({ privateKey, data })`
- **BREAKING**: Rename `Version` type to `SignTypedDataVersion` ([#218](https://github.com/MetaMask/eth-sig-util/pull/218))
- **BREAKING**: Rename `EIP712TypedData` type to `TypedDataV1Field` ([#218](https://github.com/MetaMask/eth-sig-util/pull/218))
- Add `signTypedData` version validation ([#201](https://github.com/MetaMask/eth-sig-util/pull/201))
- Add validation to check that parameters aren't nullish ([#205](https://github.com/MetaMask/eth-sig-util/pull/205))
- Enable inline sourcemaps ([#159](https://github.com/MetaMask/eth-sig-util/pull/159))
- Update `ethereumjs-util` to v6 ([#138](https://github.com/MetaMask/eth-sig-util/pull/138), [#195](https://github.com/MetaMask/eth-sig-util/pull/195))
- Allow `TypedDataUtils` functions to be called unbound ([#152](https://github.com/MetaMask/eth-sig-util/pull/152))
- Update minimum `tweetnacl-util` version ([#155](https://github.com/MetaMask/eth-sig-util/pull/155))
- Add Solidity types to JSON schema for `signTypedData` ([#189](https://github.com/MetaMask/eth-sig-util/pull/189))
- Replace README API docs with generated docs ([#213](https://github.com/MetaMask/eth-sig-util/pull/213))

## [3.0.1] - 2021-02-04
### Changed
- Update `ethereumjs-abi` ([#96](https://github.com/MetaMask/eth-sig-util/pull/96))
- Remove unused dependencies ([#117](https://github.com/MetaMask/eth-sig-util/pull/117))
- Update minimum `tweetnacl` to latest version ([#123](https://github.com/MetaMask/eth-sig-util/pull/123))

## [3.0.0] - 2020-11-09
### Changed
- [**BREAKING**] Migrate to TypeScript ([#74](https://github.com/MetaMask/eth-sig-util/pull/74))
- Fix package metadata ([#81](https://github.com/MetaMask/eth-sig-util/pull/81)
- Switch from Node.js v8 to Node.js v10 ([#76](https://github.com/MetaMask/eth-sig-util/pull/77) and [#80](https://github.com/MetaMask/eth-sig-util/pull/80))


## [2.5.4] - 2021-02-04
### Changed
- Update `ethereumjs-abi` ([#121](https://github.com/MetaMask/eth-sig-util/pull/121))
- Remove unused dependencies ([#120](https://github.com/MetaMask/eth-sig-util/pull/120))
- Update minimum `tweetnacl` to latest version ([#124](https://github.com/MetaMask/eth-sig-util/pull/124))

## [2.5.3] - 2020-03-16 [WITHDRAWN]
### Changed
- [**BREAKING**] Migrate to TypeScript ([#74](https://github.com/MetaMask/eth-sig-util/pull/74))
- Fix package metadata ([#81](https://github.com/MetaMask/eth-sig-util/pull/81)
- Switch from Node.js v8 to Node.js v10 ([#76](https://github.com/MetaMask/eth-sig-util/pull/77) and [#80](https://github.com/MetaMask/eth-sig-util/pull/80))

[Unreleased]: https://github.com/MetaMask/eth-sig-util/compare/v4.0.1...HEAD
[4.0.1]: https://github.com/MetaMask/eth-sig-util/compare/v4.0.0...v4.0.1
[4.0.0]: https://github.com/MetaMask/eth-sig-util/compare/v3.0.1...v4.0.0
[3.0.1]: https://github.com/MetaMask/eth-sig-util/compare/v3.0.0...v3.0.1
[3.0.0]: https://github.com/MetaMask/eth-sig-util/compare/v2.5.4...v3.0.0
[2.5.4]: https://github.com/MetaMask/eth-sig-util/compare/v2.5.3...v2.5.4
[2.5.3]: https://github.com/MetaMask/eth-sig-util/releases/tag/v2.5.3
