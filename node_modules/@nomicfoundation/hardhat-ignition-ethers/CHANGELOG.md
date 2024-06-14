# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

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
