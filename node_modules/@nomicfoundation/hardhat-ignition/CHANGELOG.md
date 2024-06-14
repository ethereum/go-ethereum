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

## 0.14.0 - 2024-02-21

### Added

- Support BigInt values in Module Parameter files by encoding them as strings with the format /d+n/ ([#663](https://github.com/NomicFoundation/hardhat-ignition/issues/663))

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

- Add support for setting the default sender account from tests and scripts ([#639](https://github.com/NomicFoundation/hardhat-ignition/issues/639))
- Add support for setting the default sender from the cli ([#620](https://github.com/NomicFoundation/hardhat-ignition/issues/620))

### Changed

- Split out `ethers` support from `@nomicfoundation/hardhat-plugin`, to allow opting for either **ethers** or **Viem** in tests and scripts. If you were using `hre.ignition.deploy(...)` in tests or scripts you will need to install, and require in your Hardhat config, the `@nomicfoundation/hardhat-plugin-ethers` package. For more details on our [Viem support see our guide](https://hardhat.org/ignition/docs/guides/viem). ([#612](https://github.com/NomicFoundation/hardhat-ignition/pull/612))

## 0.12.0 - 2023-12-05

### Added

- Add support for verification, see our [verification guide](https://hardhat.org/ignition/docs/guides/verify) for more information ([#630](https://github.com/NomicFoundation/hardhat-ignition/issues/630))

### Changed

- Improved the error for fee exceeding block gas limit ([#594](https://github.com/NomicFoundation/hardhat-ignition/issues/594))

## 0.11.2 - 2023-11-06

### Added

- Support account values in _send_ `to` in Module API ([#618](https://github.com/NomicFoundation/hardhat-ignition/issues/618))
- Validation check for duplicate module ids ([#608](https://github.com/NomicFoundation/hardhat-ignition/issues/608))

### Fixed

- Fix `ContractAt`s being recorded to `deployed_addresses.json` ([#607](https://github.com/NomicFoundation/hardhat-ignition/issues/607))

## 0.11.1 - 2023-10-30

### Added

- Give visual indication that there was a gas bump in `deploy` task ([#587](https://github.com/NomicFoundation/hardhat-ignition/issues/587))

### Changed

- When displaying an Ethereum Address at the cli, show in checksum format ([#600](https://github.com/NomicFoundation/hardhat-ignition/issues/600))
- Show a better message when no futures were executed during a rerun ([#586](https://github.com/NomicFoundation/hardhat-ignition/issues/586))
- Less confusing message for no modules folder error ([#602](https://github.com/NomicFoundation/hardhat-ignition/issues/602))

## 0.11.0 - 2023-10-23

First public launch 🚀

### Added

- Display batching information in the visualize report ([#494](https://github.com/NomicFoundation/hardhat-ignition/issues/494))
- Update styling of visualize report ([#493](https://github.com/NomicFoundation/hardhat-ignition/issues/493))
- Expand Module API so value and from support staticCall/readEventArg as values ([#455](https://github.com/NomicFoundation/hardhat-ignition/issues/455))
- Support fully qualified contract names ([#563](https://github.com/NomicFoundation/hardhat-ignition/pull/563))

### Changed

- More compact command-line UI ([#495](https://github.com/NomicFoundation/hardhat-ignition/issues/495))
- Ignition tasks are now scoped e.g. `npx hardhat ignition deploy` ([#442](https://github.com/NomicFoundation/hardhat-ignition/issues/442))
- The `ignition-info` task has been renamed to `status`, and now shows the status of the current deployment not just the successes ([#496](https://github.com/NomicFoundation/hardhat-ignition/issues/496))
- Modules now have to be inside a `modules` subfolder within the `./ignition` folder ([#511](https://github.com/NomicFoundation/hardhat-ignition/issues/511))
- Import of `buildModule` now from `@nomicfoundation/hardhat-ignition/modules` ([#540](https://github.com/NomicFoundation/hardhat-ignition/issues/540))
- A warning is displayed if you are running against an in-process hardhat network ([#553](https://github.com/NomicFoundation/hardhat-ignition/issues/553))
- The default deployment id is now `chain-<CHAINID>` ([#551](https://github.com/NomicFoundation/hardhat-ignition/issues/551))
- The `contractAt` signature overload for artifact has been changed to match other artifact overload signatures ([#557](https://github.com/NomicFoundation/hardhat-ignition/issues/557))

### Fixed

- Improve loading times for other non-ignition hh tasks, when Ignition is included ([#13](https://github.com/NomicFoundation/hardhat-ignition/issues/13))
- Only open visualization report if supported ([#504](https://github.com/NomicFoundation/hardhat-ignition/issues/504))
- Fixed nonce check failure on rerun ([#506](https://github.com/NomicFoundation/hardhat-ignition/issues/506))
- Ensure future's senders meet nonce sync checks ([#411](https://github.com/NomicFoundation/hardhat-ignition/issues/411))
- Show all deployed contracts at the end of a deployment ([#480](https://github.com/NomicFoundation/hardhat-ignition/issues/480))
- Rerun blocked by sent transactions message IGN403 ([#574](https://github.com/NomicFoundation/hardhat-ignition/issues/574))
- Rerun over multiple batches trigger error IGN405 ([#576](https://github.com/NomicFoundation/hardhat-ignition/issues/576))

## 0.4.0 - 2023-09-15

### Added

- Advanced cli UI for deploy ([#401](https://github.com/NomicFoundation/ignition/pull/401))
- Store artifact debug files as part of deployment directory ([#473](https://github.com/NomicFoundation/ignition/pull/473))

### Changed

- Changed npm package name to `@nomicfoundation/hardhat-ignition`
- rename the `plan` task to `visualize` ([#471](https://github.com/NomicFoundation/ignition/pull/471))
- Constrain module ids and action ids to better support storing deployments on windows ([#466](https://github.com/NomicFoundation/ignition/pull/466))
- Rename `use-verbose` flag on `deploy` task to `simple-text-ui` ([#444](https://github.com/NomicFoundation/ignition/pull/444))

### Fixed

- Fix batch completion on non-automining chains ([#467](https://github.com/NomicFoundation/ignition/pull/467))

## 0.3.0 - 2023-08-30

### Added

- Support eip-1559 style transactions ([#8](https://github.com/NomicFoundation/ignition/issues/8))
- Automatic gas bumping of transactions ([#294](https://github.com/NomicFoundation/ignition/issues/294))
- Improve validation based on artifacts ([#390](https://github.com/NomicFoundation/ignition/issues/390))

### Changed

- Switch peer dependency from ethers v5 to ethers v6 ([#338](https://github.com/NomicFoundation/ignition/issues/338))
- Deprecate support for node 14, and add support for node 20 ([#370](https://github.com/NomicFoundation/ignition/issues/370))

## 0.2.0 - 2023-08-16

### Added

- The execution config is now exposed through deploy and wired into the `hardhat-ignition` plugin config.

### Fixed

- Switch default deploy configurations depending on whether the current network is automined.

## 0.1.2 - 2023-07-31

### Fixed

- Fix validation error when using the result of a `staticCall` as the address of a `contractAt`/`contractAtFromArtifact` ([#354](https://github.com/NomicFoundation/ignition/issues/357))
- Fix bug in `staticCall` execution logic preventing successful execution

## 0.1.1 - 2023-07-30

### Fixed

- Fix validation error when using the result of a `readEventArgument` as the address of a `contractAt`/`contractAtFromArtifact` ([#354](https://github.com/NomicFoundation/ignition/issues/354))

## 0.1.0 - 2023-07-27

### Added

- Rerunning now uses a reconciliation phase to allow more leeway in changing a module between runs
- Deployments against real networks are recorded to `./ignition/deployments/<deploy-id>`, including recording the deployed addresses and the artifacts (i.e. abi, build-info etc) used for each contract deploy

### Changed

- The _Module API_ has went through considerable restructuring, including _breaking changes_, please see the `./docs` for more details
- The plan task has been enhanced to give a module centric view rather than the lower level execution that was previously shown
- The _ui_ during a deployment has been reduced to showing the results, the full UI will be brought back in a coming release

## 0.0.13 - 2023-04-18

### Added

- Support static calls in the Module API ([#85](https://github.com/NomicFoundation/ignition/issues/85))
- Add command, `ignition-info`, to list previously deployed contracts ([#111](https://github.com/NomicFoundation/ignition/issues/111))

## 0.0.12 - 2023-04-04

### Fixed

- Support recursive types in `m.call` args ([#186](https://github.com/NomicFoundation/ignition/issues/186))

## 0.0.11 - 2023-03-29

### Changed

- Replace `m.getBytesForArtifact("Foo")` with `m.getArtifact("Foo")` in the module api ([#155](https://github.com/NomicFoundation/ignition/issues/155))

### Fixed

- Fix libraries in plan ([#131](https://github.com/NomicFoundation/ignition/issues/131))

## 0.0.10 - 2023-03-14

### Added

- Make Hardhat network accounts available within modules ([#166](https://github.com/NomicFoundation/ignition/pull/166))

### Changed

- Show file/line/column against validation errors, so that module problems can more easily be traced back to the source code ([#160](https://github.com/NomicFoundation/ignition/pull/160))

## 0.0.9 - 2023-03-02

### Added

- Support defining modules in typescript ([#101](https://github.com/NomicFoundation/ignition/issues/101))
- Allow rerunning deployment while ignoring journal history through a `--force` flag ([#132](https://github.com/NomicFoundation/ignition/issues/132))

### Changed

- Do not ask for confirmation when deploying to a hardhat node ([#134](https://github.com/NomicFoundation/ignition/issues/134))

## 0.0.8 - 2023-02-16

### Added

- Allow file paths to ignition modules on cli to support cli completions for file discovery ([#102](https://github.com/NomicFoundation/ignition/issues/102))

### Changed

- Rename config option `gasIncrementPerRetry` to `gasPriceIncrementPerRetry` for clarity ([#143](https://github.com/NomicFoundation/ignition/pull/143))

### Fixed

- Improve error messages display during module validation ([#141](https://github.com/NomicFoundation/ignition/issues/141))
- Ban passing async functions to `buildModule` ([#138](https://github.com/NomicFoundation/ignition/issues/138))

## 0.0.7 - 2023-01-31

### Fixed

- Resolve parameter args for deployed contracts during execution ([#125](https://github.com/NomicFoundation/ignition/pull/125))

## 0.0.6 - 2023-01-20

### Added

- Support rerunning deployments that errored or went to on-hold on a previous run ([#70](https://github.com/NomicFoundation/ignition/pull/70))
- Support sending `ETH` to a contract without having to make a call/deploy ([#79](https://github.com/NomicFoundation/ignition/pull/79))
- Confirm dialog on deploys to non-hardhat networks ([#95](https://github.com/NomicFoundation/ignition/issues/95))

### Changed

- Rename the `awaitEvent` action in the api to `event` ([#108](https://github.com/NomicFoundation/ignition/issues/108))

## 0.0.5 - 2022-12-20

### Added

- Add params section to deploy display in cli ([#74](https://github.com/NomicFoundation/ignition/pull/74))
- Expose config for pollingInterval ([#75](https://github.com/NomicFoundation/ignition/pull/75))
- Support `getBytesForArtifact` in deployment api ([#76](https://github.com/NomicFoundation/ignition/pull/76))
- Support use of emitted event args as futures for later deployment api calls ([#77](https://github.com/NomicFoundation/ignition/pull/77))
- Support event params futures in `contractAt` ([#78](https://github.com/NomicFoundation/ignition/pull/78))

### Fixed

- Fix for planning on modules with deploys from artifacts ([#73](https://github.com/NomicFoundation/ignition/pull/73))

## 0.0.4 - 2022-11-22

### Added

- Support setting module params from JSON file ([#64](https://github.com/NomicFoundation/ignition/pull/64))

## 0.0.3 - 2022-11-09

### Added

- Allow modules to depend on other calls ([#53](https://github.com/NomicFoundation/ignition/pull/53))
- Allow depending on a module ([#54](https://github.com/NomicFoundation/ignition/pull/54))

### Changed

- Dependening on returned module contract equivalent to depending on the module ([#55](https://github.com/NomicFoundation/ignition/pull/55))

## 0.0.2 - 2022-10-26

### Added

- Add `deploy` task to hardhat via the plugin
- Add `plan` task to hardhat via the plugin
