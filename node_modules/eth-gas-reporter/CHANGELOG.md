## Changelog: eth-gas-reporter

# 0.2.26 / 2023-09-29

- Replace request-promise-native with axios / avoid default price API calls (https://github.com/cgewecke/eth-gas-reporter/issues/299)
- Remove request package (https://github.com/cgewecke/eth-gas-reporter/issues/297)
- Bump ethers version (https://github.com/cgewecke/eth-gas-reporter/issues/296)
- Update Mocha to v10 (https://github.com/cgewecke/eth-gas-reporter/issues/295)

# 0.2.23 / 2021-11-26

- Add notes to README about missing price data & remote data fetching race condition
- Add support for multiple gas price tokens (BNB, MATIC, AVAX, HR, MOVR) (https://github.com/cgewecke/eth-gas-reporter/pull/251)
- Make @codechecks/client peer dep optional (https://github.com/cgewecke/eth-gas-reporter/pull/257)
- Update @solidity-parser/parser to 0.14.0 (https://github.com/cgewecke/eth-gas-reporter/pull/261)

# 0.2.22 / 2021-03-04

- Update @solidity-parser/parser to ^0.12.0 (support Panic keyword in catch blocks) (https://github.com/cgewecke/eth-gas-reporter/issues/243)

# 0.2.21 / 2021-02-16

- Fix missing truffle migration deployments data (https://github.com/cgewecke/eth-gas-reporter/issues/240)
- Upgrade solidity-parser/parser to 0.11.1 (https://github.com/cgewecke/eth-gas-reporter/issues/239)

# 0.2.20 / 2020-12-01

- Add support for remote contracts data pre-loading (hardhat-gas-reporter feature)

# 0.2.19 / 2020-10-29

- Delegate contract loading/parsing to artifactor & make optional (#227)

# 0.2.18 / 2020-10-13

- Support multiple codechecks reports per CI run
- Add CI error threshold options: maxMethodDiff, maxDeploymentDiff
- Add async collection methods for BuidlerEVM
- Update solidity-parser/parser to 0.8.0 (contribution: @vicnaum)
- Update dev deps / use Node 12 in CI

# 0.2.17 / 2020-04-13

- Use @solidity-parser/parser for better solc 0.6.x parsing
- Upgrade Mocha to ^7.1.1 (to remove minimist vuln warning)
- Stop crashing when parser or ABI Encoder fails
- Update @ethersproject/abi to ^5.0.0-beta.146 (and unpin)

# 0.2.16 / 2020-03-18

- Use new coinmarketcap data API / make api key configurable. Old (un-gated) API has been taken offline.
- Fix crashing when artifact transactionHash is stale after deleting previously migrated contracts

# 0.2.15 / 2020-02-12

- Use parser-diligence to parse Solidity 0.6.x
- Add option to show full method signature

# 0.2.14 / 2019-12-01

- Add ABIEncoderV2 support by using @ethersproject/abi for ABI processing

# 0.2.12 / 2019-09-30

- Add try/catch block for codechecks.getValue so it doesn't throw when server is down.
- Pin parser-antlr to 0.4.7

# 0.2.11 / 2019-08-27

- Fix syntax err on unresolved provider error msg (contribution: gnidan)
- Add unlock-protocol funding ymls
- Update abi-decoder deps / web3

# 0.2.10 / 2019-08-08

- Small codechecks table formatting improvements
- Fix syntax error when codechecks errors on missing gas report

# 0.2.9 / 2019-07-30

- Optimize post-transaction data collection (reduce # of calls & cache addresses)
- Catch codechecks server errors

# 0.2.8 / 2019-07-27

- Render codechecks CI table as markdown

# 0.2.7 / 2019-07-27

- Fix block limit basis bug
- Fix bug affecting Truffle < v5.0.10 (crash because metadata not defined)
- Add percentage diff columns to codechecks ci table / make table narrower
- Slightly randomize gas consumption in tests
- Begin running codechecks in CI for own tests

# 0.2.6 / 2019-07-16

- Stopped using npm-shrinkwrap, because it seemed to correlate w/ weird installation problems
- Fix bug which caused outputFile option to crash due to misnamed variable

# 0.2.5 / 2019-07-15

- Upgrade lodash for because of vulnerability report (contribution @ppoliani)

# 0.2.4 / 2019-07-08

- Update abi-decoder to 2.0.1 to fix npm installation bug with bignumber.js fork

# 0.2.3 / 2019-07-04

- Bug fix to invoke user defined artifactType methods correctly

# 0.2.2 / 2019-07-02

- Add documentation about codechecks, buidler, advanced use cases.
- Add artifactType option as a user defined function so people use with any compilation artifacts.
- Add codechecks integration
- Add buidler plugin integration
- Remove shelljs due to GH security warning, execute ls command manually

# 0.2.1 / 2019-06-19

- Upgrade mocha from 4.1.0 to 5.2.0
- Report solc version and settings info
- Add EtherRouter method resolver logic (as option and example)
- Add proxyResolver option & support discovery of delegated method calls identity
- Add draft of 0x artifact handler
- Add url option for non-truffle, non-buidler use
- Add buidler truffle-v5 plugin support (preface to gas-reporter plugin in next release)
- Completely reorganize and refactor

# 0.2.0 / 2019-05-07

- Add E2E tests in CI
- Restore logic that matches tx signatures to contracts as a fallback when it's impossible to
  be certain which contract was called (contribution @ItsNickBarry)
- Fix bug which crashed reporter when migrations linked un-deployed contracts

# 0.1.12 / 2018-09-14

- Allow contracts to share method signatures (contribution @wighawag)
- Collect gas data for Migrations deployments (contribution @wighawag)
- Add ability allow to specify a different src folder for contracts (contribution @wighawag)
- Handle in-memory provider error correctly / use spec reporter if sync calls impossible (contribution @wighawag)
- Default to only showing invoked methods in report

# 0.1.10 / 2018-07-18

- Update mocha from 3.5.3 to 4.10.0 (contribution ldub)
- Update truffle to truffle@next to fix mocha issues (contribution ldub)
- Modify binary checking to allow very long bytecodes / large contracts (contribution ben-kaufman)

# 0.1.9 / 2018-06-27

- Fix bug that caused test gas to include before hook gas consumption totals

# 0.1.8 / 2018-06-26

- Add showTimeSpent option to also show how long each test took (contribution @ldub)
- Update cli-table2 to cli-table3 (contribution @DanielRuf)

# 0.1.7 / 2018-05-27

- Support reStructured text code-block output

# 0.1.5 / 2018-05-15

- Support multi-contract files by parsing files w/ solidity-parser-antlr

# 0.1.4 / 2018-05-14

- Try to work around web3 websocket provider by attempting connection over http://.
  `requestSync` doesn't support this otherwise.
- Detect and identify binaries with library links, add to the deployments table
- Add scripts to run geth in CI (not enabled)

# 0.1.2 / 2018-04-20

- Make compatible with Web 1.0 by creating own sync RPC wrapper. (Contribution: @area)

# 0.1.1 / 2017-12-19

- Use mochas own reporter options instead of .ethgas (still supported)
- Add onlyCalledMethods option
- Add outputFile option
- Add noColors option

# 0.1.0 / 2017-12-10

- Require config gas price to be expressed in gwei (breaking change)
- Use eth gas station API for gas price (it's more accurate)
- Fix bug that caused table not to print if any test failed.

# 0.0.15 / 2017-12-09

- Fix ascii colorization bug that caused crashed during table generation. (Use colors/safe).

# 0.0.14 / 2017-11-30

- Fix bug that caused the error report at the end of test run not to be printed.

# 0.0.13 / 2017-11-15

- Filter throws by receipt.status if possible
- Use testrpc 6.0.2 in tests, add view and pure methods to tests.

# 0.0.12 / 2017-10-28

- Add config. Add gasPrice and currency code options
- Improve table clarity
- Derive block.gasLimit from rpc

# 0.0.11 / 2017-10-23

- Add Travis CI
- Fix bug that crashed reported when truffle could not find required file

# 0.0.10 / 2017-10-22

- Add examples

# 0.0.10 / 2017-10-22

- Filter deployment calls that throw from the stats

# 0.0.8 / 2017-10-22

- Filter method calls that throw from the stats
- Add deployment stats
- Add number of calls column

# 0.0.6 / 2017-10-14

- Stop showing zero gas usage in mocha output
- Show currency rates and gwei gas price rates in table header
  \* Alphabetize table
- Fix bug caused by unused methods reporting NaN
- Fix failure to round avg gas use in table
- Update dev deps to truffle4 beta

# 0.0.5 / 2017-10-12

- Thanks
- Update image
- Finish table formatting
- Add some variable gas consumption contracts
- Working table
- Get map to work in the runner
- Get gasStats file and percentage of limit working
- Test using npm install
- Add gasPrice data fetch, config logic
- More tests
- Abi encoding map.

# 0.0.4 / 2017-10-01

- Add visual inspection test
- Fix bug that counted gas consumed in the test hooks
