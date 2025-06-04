# Changelog

## 0.15.11

### Patch Changes

- 23280b8: Resolve all dependencies when using submodules in `after`
- 1a6203d: Add arbitrum sepolia to chain config

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

- Support `maxPriorityFeePerGas` as a configuration parameter ([#728](https://github.com/NomicFoundation/hardhat-ignition/issues/728))
- Use RPC call `eth_maxPriorityFeePerGas` in gas fee calculations when available ([#743](https://github.com/NomicFoundation/hardhat-ignition/issues/743))
- Support zero gas fee chains (like private Besu chains), thanks @jimthematrix ([#730](https://github.com/NomicFoundation/hardhat-ignition/pull/730))

### Fixed

- Use pre-EIP-1559 transactions for Polygon to avoid dropped transactions ([#735](https://github.com/NomicFoundation/hardhat-ignition/issues/735))

## 0.15.1 - 2024-04-04

### Added

- Add a configurable upper limit for the maxFeePerGas ([#685](https://github.com/NomicFoundation/hardhat-ignition/issues/685))
- Update `ignition status` core function to display chainId ([#668](https://github.com/NomicFoundation/hardhat-ignition/issues/668))

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

### Fixed

- Fix bug with `process.stdout` being used in a non-tty context ([#654](https://github.com/NomicFoundation/hardhat-ignition/pull/654))

## 0.13.0 - 2023-12-13

### Added

- Enhance types around artifacts and ABIs to better support `Viem` type inference ([#612](https://github.com/NomicFoundation/hardhat-ignition/pull/612))

### Fixed

- Fix bug with default sender account not being recognised due to case sensitivity ([#640](https://github.com/NomicFoundation/hardhat-ignition/pull/640))

## 0.12.0 - 2023-12-05

### Added

- Add support for verification, see our [verification guide](https://hardhat.org/ignition/docs/guides/verify) for more information ([#630](https://github.com/NomicFoundation/hardhat-ignition/issues/630))

### Changed

- Improved the error for fee exceeding block gas limit ([#594](https://github.com/NomicFoundation/hardhat-ignition/issues/594))

## 0.11.2 - 2023-11-06

### Added

- Support account values in _send_ `to` in Module API ([#618](https://github.com/NomicFoundation/hardhat-ignition/issues/618))

### Fixed

- Fix `ContractAt`s being recorded to `deployed_addresses.json` ([#607](https://github.com/NomicFoundation/hardhat-ignition/issues/607))

## 0.11.1 - 2023-10-30

### Added

- Give visual indication that there was a gas bump in `deploy` task ([#587](https://github.com/NomicFoundation/hardhat-ignition/issues/587))

### Changed

- When displaying an Ethereum Address at the cli, show in checksum format ([#600](https://github.com/NomicFoundation/hardhat-ignition/issues/600))

## 0.11.0 - 2023-10-23

First public launch 🚀

### Added

- Expand Module API so value and from support staticCall/readEventArg as values ([#455](https://github.com/NomicFoundation/hardhat-ignition/issues/455))
- Support fully qualified contract names ([#563](https://github.com/NomicFoundation/hardhat-ignition/pull/563))

### Changed

- The `contractAt` signature overload for artifact has been changed to match other artifact overload signatures ([#557](https://github.com/NomicFoundation/hardhat-ignition/issues/557))

### Fixed

- Fixed nonce check failure on rerun ([#506](https://github.com/NomicFoundation/hardhat-ignition/issues/506))
- Ensure future's senders meet nonce sync checks ([#411](https://github.com/NomicFoundation/hardhat-ignition/issues/411))
- Show all deployed contracts at the end of a deployment ([#480](https://github.com/NomicFoundation/hardhat-ignition/issues/480))
- Rerun blocked by sent transactions message IGN403 ([#574](https://github.com/NomicFoundation/hardhat-ignition/issues/574))
- Rerun over multiple batches trigger error IGN405 ([#576](https://github.com/NomicFoundation/hardhat-ignition/issues/576))

## 0.4.0 - 2023-09-15

### Added

- Store artifact debug files as part of deployment directory ([#473](https://github.com/NomicFoundation/ignition/pull/473))

### Changed

- Changed npm package name to `@nomicfoundation/ignition-core`
- Constrain module ids and action ids to better support storing deployments on windows ([#466](https://github.com/NomicFoundation/ignition/pull/466))

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

## 0.0.8 - 2023-02-16

### Changed

- Rename config option `gasIncrementPerRetry` to `gasPriceIncrementPerRetry` for clarity ([#143](https://github.com/NomicFoundation/ignition/pull/143))

### Fixed

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

- Expose config for pollingInterval ([#75](https://github.com/NomicFoundation/ignition/pull/75))
- Support `getBytesForArtifact` in deployment api ([#76](https://github.com/NomicFoundation/ignition/pull/76))
- Support use of emitted event args as futures for later deployment api calls ([#77](https://github.com/NomicFoundation/ignition/pull/77))
- Support event params futures in `contractAt` ([#78](https://github.com/NomicFoundation/ignition/pull/78))

### Fixed

- Fix for planning on modules with deploys from artifacts ([#73](https://github.com/NomicFoundation/ignition/pull/73))

## 0.0.4 - 2022-11-22

### Added

- Pass eth as `value` on deploy or call ([#60](https://github.com/NomicFoundation/ignition/pull/60))
- Pass parameters for `value` ([#66](https://github.com/NomicFoundation/ignition/pull/66))

## 0.0.3 - 2022-11-09

### Added

- Allow modules to depend on other calls ([#53](https://github.com/NomicFoundation/ignition/pull/53))
- Allow depending on a module ([#54](https://github.com/NomicFoundation/ignition/pull/54))

### Changed

- Dependening on returned module contract equivalent to depending on the module ([#55](https://github.com/NomicFoundation/ignition/pull/55))

## 0.0.2 - 2022-10-26

### Added

- Deploy a module to a ephemeral local hardhat node
- Generate example execution graph for plans
