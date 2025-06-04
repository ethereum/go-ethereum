# Changelog

## 0.15.11

### Patch Changes

- 23280b8: Resolve all dependencies when using submodules in `after`

## 0.15.10

### Patch Changes

- d96c003: Fix for bug when we fail to save transaction hash

## 0.15.9 - 2024-12-18

### Added

- Standard Ignition UI can now be enabled when deploying via Hardhat scripts by setting `displayUi: true` in the `deploy` function options, @zoeyTM ([#843](https://github.com/NomicFoundation/hardhat-ignition/pull/843))
- Ignition modules can now be set as a dependency in the `after` option of futures, @zoeyTM ([#828](https://github.com/NomicFoundation/hardhat-ignition/pull/828))
- The `ignition transactions` command output will now include a link to view each transaction on the configured block explorer, @zoeyTM ([#849](https://github.com/NomicFoundation/hardhat-ignition/pull/849))
- Module parameters can now be directly imported from a JSON file when deploying via Hardhat scripts by passing an absolute path to the file to the `parameters` option, @zoeyTM ([#850](https://github.com/NomicFoundation/hardhat-ignition/pull/850))

### Fixed

- Properly handle errors when verifying deployments that use external artifacts, @zoeyTM ([#848](https://github.com/NomicFoundation/hardhat-ignition/pull/848))
- Fix issue with `ignition status` command not working with deployments that use external artifacts, @zoeyTM ([#846](https://github.com/NomicFoundation/hardhat-ignition/pull/846))

## 0.15.8 - 2024-11-22

### Fixed

- `transactions` command now properly serializes `bigint` values, @zoeyTM ([#837](https://github.com/NomicFoundation/hardhat-ignition/pull/837))
- Additional validations added for global parameters, @kanej ([#832](https://github.com/NomicFoundation/hardhat-ignition/pull/832))

## 0.15.7 - 2024-10-24

### Added

- New CLI command `ignition transactions` to list all transactions sent for a given deployment ID, @zoeyTM ([#821](https://github.com/NomicFoundation/hardhat-ignition/pull/821))
- Module parameters can now be set at the global level using `$global`, @zoeyTM ([#819](https://github.com/NomicFoundation/hardhat-ignition/pull/819))

### Fixed

- Gas fields are now properly set for Optimistic BNB, @zoeyTM ([#826](https://github.com/NomicFoundation/hardhat-ignition/pull/826))
- Corrected resolution of artifacts when using fully qualified names in deployment modules, @kanej ([#822](https://github.com/NomicFoundation/hardhat-ignition/pull/822))

## 0.15.6 - 2024-09-25

### Added

- Updates to the visualization UI, including the ability to zoom and pan the mermaid diagram ([#810](https://github.com/NomicFoundation/hardhat-ignition/pull/810))
- `gasPrice` and `disableFeeBumping` config fields added as part of our L2 gas logic update ([#808](https://github.com/NomicFoundation/hardhat-ignition/pull/808))
- Debug logging for communication errors with Hardhat Ledger ([#792](https://github.com/NomicFoundation/hardhat-ignition/pull/792))
- JSON5 support for module parameters, thanks @erhant ([#800](https://github.com/NomicFoundation/hardhat-ignition/pull/800))
- Add `writeLocalhostDeployment` flag to allow saving deployment artifacts when deploying to the ephemeral Hardhat network, thanks @SebastienGllmt ([#816](https://github.com/NomicFoundation/hardhat-ignition/pull/816))

### Fixed

- Replace `this` with the class itself in `ViemIgnitionHelper`, thanks @iosh ([#796](https://github.com/NomicFoundation/hardhat-ignition/pull/796))

## 0.15.5 - 2024-06-17

### Added

- New function `m.encodeFunctionCall` ([#761](https://github.com/NomicFoundation/hardhat-ignition/pull/761))

### Fixed

- Adjusted regex to allow calling overloaded functions with array parameters ([#774](https://github.com/NomicFoundation/hardhat-ignition/pull/774))
- Handle anvil response for `hardhat_setBalance` when deploying with create2 ([#773](https://github.com/NomicFoundation/hardhat-ignition/pull/773))
- Properly resolve `verify` logic when dealing with circular or very deeply nested imports ([#772](https://github.com/NomicFoundation/hardhat-ignition/pull/772))
- Exclude BNB Test Chain from zero fee configuration in gas fee logic, thanks @MukulKolpe ([#768](https://github.com/NomicFoundation/hardhat-ignition/pull/768))

## 0.15.4 - 2024-05-14

### Fixed

- Reconcile address parameters with mismatched casings ([#748](https://github.com/NomicFoundation/hardhat-ignition/pull/748))
- Display better error messages for insufficient funds ([#754](https://github.com/NomicFoundation/hardhat-ignition/pull/754))

## 0.15.3 - 2024-05-09

### Fixed

- Exclude BNB Chain from zero fee configuration in gas fee logic, thanks @magicsih ([#755](https://github.com/NomicFoundation/hardhat-ignition/pull/755))

## 0.15.2 - 2024-05-02

### Added

- Improved support for deploying via a Ledger Hardware wallet, [see our guide for details](https://hardhat.org/ignition/docs/guides/ledger) ([#720](https://github.com/NomicFoundation/hardhat-ignition/issues/720))
- Support `maxPriorityFeePerGas` as a configuration parameter ([#728](https://github.com/NomicFoundation/hardhat-ignition/issues/728))
- Use RPC call `eth_maxPriorityFeePerGas` in gas fee calculations when available ([#743](https://github.com/NomicFoundation/hardhat-ignition/issues/743))
- Support zero gas fee chains (like private Besu chains), thanks @jimthematrix ([#730](https://github.com/NomicFoundation/hardhat-ignition/pull/730))

### Fixed

- Use pre-EIP-1559 transactions for Polygon to avoid dropped transactions ([#735](https://github.com/NomicFoundation/hardhat-ignition/issues/735))

## 0.15.1 - 2024-04-04

### Added

- Add a configurable upper limit for the maxFeePerGas ([#685](https://github.com/NomicFoundation/hardhat-ignition/issues/685))
- Support writing and reading from deployments folder within tests and scripts ([#704](https://github.com/NomicFoundation/hardhat-ignition/pull/704))
- Add `ignition deployments` task to list all the current deployments ([#646](https://github.com/NomicFoundation/hardhat-ignition/issues/646))

### Changed

- Deploying to a cleared local hardhat node ignores previous deployment ([#650](https://github.com/NomicFoundation/hardhat-ignition/issues/650))

### Fixed

- More resilent automine check ([#721](https://github.com/NomicFoundation/hardhat-ignition/issues/721))
- `getCode` usage brought in line with Ethereum RPC standard ([#715](https://github.com/NomicFoundation/hardhat-ignition/issues/715))
- Fixed unexpected next nonce on revert ([#676](https://github.com/NomicFoundation/hardhat-ignition/issues/676))
- Reduce sources being passed to etherscan for verification ([#706](https://github.com/NomicFoundation/hardhat-ignition/issues/706))

## 0.15.0 - 2024-03-13

### Added

- Support `create2` through strategies, for more details see [our `create2` guide](https://hardhat.org/ignition/docs/guides/create2). ([#629](https://github.com/NomicFoundation/hardhat-ignition/issues/629))

## 0.13.2 - 2024-01-25

### Fixed

- Add memory pool lookup retry to reduce errors from slow propogation ([#667](https://github.com/NomicFoundation/hardhat-ignition/pull/667))

### Added

- Improve Module API typescript doc comments to enhance intellisense experience ([#642](https://github.com/NomicFoundation/hardhat-ignition/issues/642))
- Support module parameters taking accounts as the default value ([673](https://github.com/NomicFoundation/hardhat-ignition/issues/673))

## 0.13.1 - 2023-12-19

### Added

- New flag `--reset` for `ignition deploy` to wipe the existing deployment state before running ([#649](https://github.com/NomicFoundation/hardhat-ignition/issues/649))

### Fixed

- Fix bug with `process.stdout` being used in a non-tty context ([#644](https://github.com/NomicFoundation/hardhat-ignition/issues/644))

## 0.13.0 - 2023-12-13

### Added

- Add `@nomicfoundation/hardhat-plugin-ethers` package, that adds an `ignition` object to the Hardhat Runtime Environment that supports deploying Ignition modules and returning deployed contracts as [Ethers](https://docs.ethers.org) contract instances ([#612](https://github.com/NomicFoundation/hardhat-ignition/pull/612))
- Add support for setting the default sender account from tests and scripts ([#639](https://github.com/NomicFoundation/hardhat-ignition/issues/639))
