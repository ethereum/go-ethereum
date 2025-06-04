# Changelog

## 0.15.11

### Patch Changes

- cf16afa: Override max-width to fix ignition graph centering

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

## 0.11.0 - 2023-10-23

### Added

- Display batching information in the visualize report ([#494](https://github.com/NomicFoundation/hardhat-ignition/issues/494))
- Update styling of visualize report ([#493](https://github.com/NomicFoundation/hardhat-ignition/issues/493))

## 0.4.0 - 2023-09-15

### Changed

- Now published on the `@nomicfoundation` namespace

## 0.1.0 - 2023-07-27

### Added

- A UI used to display plans of Ignition deployments
