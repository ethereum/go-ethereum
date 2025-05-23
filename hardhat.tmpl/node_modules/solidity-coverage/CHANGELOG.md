# Changelog

0.8.11 / 2024-03-07
===================
  * Check all SWAP opcodes for inst. hashes when viaIR is true (https://github.com/sc-forks/solidity-coverage/issues/873)

0.8.10 / 2024-02-29
===================
  * Check all PUSH opcodes for instr. hashes when viaIR is true (https://github.com/sc-forks/solidity-coverage/issues/871)

0.8.9 / 2024-02-27
==================
  * Fix duplicate hash logic (https://github.com/sc-forks/solidity-coverage/issues/868)
  * Improve organization of edge case code in collector (https://github.com/sc-forks/solidity-coverage/issues/869)

0.8.8 / 2024-02-21
==================
  * Coerce sources path to absolute path if necessary (https://github.com/sc-forks/solidity-coverage/issues/866)
  * Only inject file-level instr. for first pragma in file (https://github.com/sc-forks/solidity-coverage/issues/865)

0.8.7 / 2024-02-09
==================
  * Documentation Cleanup & Improvements for 0.8.7 release
    (https://github.com/sc-forks/solidity-coverage/issues/859)
  * Add tests for file-level function declarations
    (https://github.com/sc-forks/solidity-coverage/issues/858)
  * Add try / catch unit tests (https://github.com/sc-forks/solidity-coverage/issues/857)
  * Fix test project configs for viaIR detection in overrides
    (https://github.com/sc-forks/solidity-coverage/issues/856)
  * Enable coverage when viaIR compiler flag is true
    (https://github.com/sc-forks/solidity-coverage/issues/854)
  * Add missing onPreCompile hook
    (https://github.com/sc-forks/solidity-coverage/issues/851)
  * Remove ganache-cli related code from API & tests
    (https://github.com/sc-forks/solidity-coverage/pull/849)
  * Add command option to specify the source files to run the coverage on
    (https://github.com/sc-forks/solidity-coverage/pull/838)

0.8.6 / 2024-01-28
==================
  * Add test for multi-contract files with inheritance
    (https://github.com/sc-forks/solidity-coverage/issues/836)
  * Add test for modifiers with post-conditions (https://github.com/sc-forks/solidity-coverage/issues/835)
  * Document Istanbul check-coverage cli command
    (https://github.com/sc-forks/solidity-coverage/issues/834)
  * Throw error when mocha parallel is set to true
    (https://github.com/sc-forks/solidity-coverage/issues/833)
  * Fix instrumentation error for virtual modifiers
    (https://github.com/sc-forks/solidity-coverage/issues/832)
  * Add test for file level `using for` statements
    (https://github.com/sc-forks/solidity-coverage/issues/831)
  * Fix chained ternary conditionals instrumentation
    (https://github.com/sc-forks/solidity-coverage/issues/830)
  * Update faq.md with an optimizer config workaround
    (https://github.com/sc-forks/solidity-coverage/issues/822)
  * Upgrade solidity-parser to 0.18.0 (https://github.com/sc-forks/solidity-coverage/issues/829)
  * Perform ternary conditional injections before branch injections
    (https://github.com/sc-forks/solidity-coverage/issues/828)
  * Add drips funding config (https://github.com/sc-forks/solidity-coverage/issues/827)

0.8.5 / 2023-09-21
==================
  * Update contributor list (https://github.com/sc-forks/solidity-coverage/issues/812)
  * Add dependabot config (https://github.com/sc-forks/solidity-coverage/issues/759)
  * Add a package description to package.json (https://github.com/sc-forks/solidity-coverage/issues/775)
  * change .solcoverjs occurrences to .solcover.js (https://github.com/sc-forks/solidity-coverage/issues/777)
  * Remove all mentions to buidler (https://github.com/sc-forks/solidity-coverage/issues/778)
  * Update HH dev dep & fix Zeppelin E2E test (https://github.com/sc-forks/solidity-coverage/issues/811)
  * Update mocha version to 10.2.0, fix deprecated debug package (https://github.com/sc-forks/solidity-coverage/issues/810)

0.8.4 / 2023-07-04
==================
  * Update solidity-parser to 0.16.0 (https://github.com/sc-forks/solidity-coverage/issues/802)

0.8.3 / 2023-06-22
==================
  * Updates for Hardhat v2.15.0  (https://github.com/sc-forks/solidity-coverage/pull/796)

0.8.1 / 2022-09-06
===================
  * Restore web3-utils (https://github.com/sc-forks/solidity-coverage/issues/743)

0.8.0 / 2022-09-05
==================

* See release notes at: https://github.com/sc-forks/solidity-coverage/releases/tag/v0.8.0

0.7.21 / 2022-04-24
===================
  * Add support for UncheckedStatement blocks (https://github.com/sc-forks/solidity-coverage/issues/712)
  * Lazy load hardhat plugin resources (https://github.com/sc-forks/solidity-coverage/issues/711)

0.7.20 / 2022-02-15
===================
  * Remove early V7 Truffle patches  (https://github.com/sc-forks/solidity-coverage/issues/693)

0.7.19 / 2022-02-09
===================
  * Update solidity-parser/parser to 0.14.0 - supports solidity user-defined types (https://github.com/sc-forks/solidity-coverage/issues/689)

0.7.18 2022-01-17
=================
* Add solcOptimizerDetails option to help workaround "stack too deep" errors (https://github.com/sc-forks/solidity-coverage/issues/683)
* Add __SOLIDITY_COVERAGE_RUNNING variable on global HH env for identifying solidity-coverage task from other tasks (https://github.com/sc-forks/solidity-coverage/issues/682)
* Fix hardhat_reset (https://github.com/sc-forks/solidity-coverage/issues/667)
* Add new hook and make temporary contracts directory configurable (https://github.com/sc-forks/solidity-coverage/issues/664)
* Use internal visibility for fn level hash method defs (https://github.com/sc-forks/solidity-coverage/issues/660)

0.7.16 / 2021-03-04
===================
  * Update @solidity-parser/parser to ^0.12.0 (and support Panic keyword in catch clauses) (https://github.com/sc-forks/solidity-coverage/issues/621)

0.7.15 / 2021-02-16
===================
  * Fix ctrl-c not exiting on Hardhat (https://github.com/sc-forks/solidity-coverage/issues/616)
  * Fix bug instrumentation bug w/ "pragma abicoder v2" (https://github.com/sc-forks/solidity-coverage/issues/615)
  * Update changelog: v0.7.14

0.7.14 / 2021-01-14
===================
  * Support file scoped function definitions for solc >= 0.7.4
  * Upgrade @solidity-parser/parser to v0.11.0

0.7.13 / 2020-12-03
===================
  * Use default artifact paths for hardhat >= 2.0.4 (Fixes hardhat-deploy bug)

0.7.12 / 2020-11-16
===================
  * Add Hardhat plugin support and allow coverage to run on HardhatEVM (https://github.com/sc-forks/solidity-coverage/pull/548)
  * Allow truffle projects to contain vyper contracts (https://github.com/sc-forks/solidity-coverage/issues/571)
  * Locate .coverage_contracts correctly for subfolder paths
    (https://github.com/sc-forks/solidity-coverage/issues/570)
  * Replace Web3 with thin rpc wrapper (https://github.com/sc-forks/solidity-coverage/issues/568)
  * Stop reporting assert statements as branches (https://github.com/sc-forks/solidity-coverage/issues/556)
  * Upgrade @truffle/provider to 0.2.24 (https://github.com/sc-forks/solidity-coverage/issues/550)
  * Upgrade solidity-parser/parser to 0.8.1 (https://github.com/sc-forks/solidity-coverage/issues/549)


0.7.11 / 2020-10-08
========================
  * Upgrade Web3 to ^1.3.0, ganache-cli to 6.11.0
  * Make statement and function coverage measurement optional

0.7.10 / 2020-08-18
==================
  * Bump parser to 0.7.0 (Solidity 0.7.0)

0.7.9 / 2020-06-28
==================
  * Fix --testfiles glob handling (Buidler)

0.7.8 / 2020-06-24
==================
  * Track statements in try/catch blocks correctly

0.7.7 / 2020-06-10
==================

  * Recommend using --network in buidler docs
  * Fix html report function highlighting
  * Stop instrumenting receive() for statements / fns (to save gas)
  * Lazy load ganache in buidler plugin (fix for buidler source-maps)
  * Unset useLiteralContent in buidler compilation settings
  * Support multi-contract files w/ inheritance for solc 0.6.x

0.7.5 / 2020-04-30
==================

  * Auto disable buidler-gas-reporter (fixes hang when both plugins are present)
  * Upgrade @solidity-parser/parser to ^0.6.0 (for solc 0.6.5 parsing)


0.7.4 / 2020-04-09
==================

  * Use @solidity-parser/parser (^0.5.2) for better 0.6.x parsing (https://github.com/sc-forks/solidity-coverage/issues/495)
  * Allowing providerOptions gasLimit and allowUnlimitedContractSize to override defaults (https://github.com/sc-forks/solidity-coverage/issues/494)

0.7.3 / 2020-04-06
==================

  * Use empty string for default cli flag values (Buidler) (https://github.com/sc-forks/solidity-coverage/issues/490)
  * Get test files after compileComplete hook runs (Truffle) (https://github.com/sc-forks/solidity-coverage/issues/485)
  * Fix skipFiles option for Windows OS (https://github.com/sc-forks/solidity-coverage/issues/486)
  * Allow modifier string arguments containing "{" (https://github.com/sc-forks/solidity-coverage/issues/480)
  * Allow base contract string constructor args with open curly braces
    (https://github.com/sc-forks/solidity-coverage/issues/479)
  * Bump handlebars from 4.1.2 to 4.5.3
  * Bump kind-of from 6.0.2 to 6.0.3

0.7.2 / 2020-02-12
==================
  * Use solidity-parser-diligence (parse Solidity 0.6.x syntax)
  * Upgrade backup ganache-cli to 6.9.0
  * Upgrade Web3 to 1.2.6

0.7.1 / 2020-01-07
==================
  * Add missing hash bang to .bin warning so 'npx solidity-coverage' produces correct message.
  * Use istanbul fork (because deprecated)

0.7.0 / 2019-12-31
==================
  * New architecture / Truffle plugin (See PR #372 for details)
  * Expose solidity-coverage/api (See PR #421 for details)
  * Add Buidler plugin (See PR #421 for details)

0.6.5 - 0.67 / 2019-09-12
=========================
  * Ugrapde testrpc-sc to 6.5.1-sc.1 (using eth-sig-util 2.3.0)
  * Pin solidity-parser-antlr to 0.4.7

0.6.4 / 2019-08-01
==================
  * Upgrade testrpc-sc to track ganache-cli 6.5.1

0.6.3 / 2019-07-17
==================
  * Fix parsing bug for soldity `type` keyword (upgrade solidity-parser-antlr to ^0.4.7)

0.6.2 / 2019-07-13
==================
  * Fix coverage for solidity `send` and `transfer` (upgrade testrpc-sc to 6.4.5-sc.3)

0.6.1 / 2019-07-12
==================
  * Fix bug preprocessing unbracketed if else statements

0.6.0 / 2019-07-11
==================
  * Add E2E vs Zeppelin and Colony in CI
  * Add misc regression tests
  * Rebase testrpc-sc on ganache-cli 6.4.5, using --allowUnlimitedContractSize and --emitFreeLogs by default.
  * Make pre-processor (and instrumentation step) much much faster
  * Transition to using solidity-parser-antlr (@area)

0.5.11 / 2018-08-31
===================
  * Support org namespaces / subfolders for `copyPackages` option (contribution @bingen)

0.5.10 / 2018-08-29
==================
  * Add deep skip option to provide relief from instrumentation hangs for contracts like Oraclize

0.5.8 / 2018-08-26
=================
  * Fix instrumentation algo so SC doesn't instrument entire codebase twice (contribution @sohkai)

0.5.7 / 2018-08-07
==================
  * Add new parameter for build directory path (contribution @DimitarSD)
  * Upgrade testrpc-sc to 6.1.6 (to support reason strings) (@area)
  * Switch to CircleCI v2 (@area)

0.5.5 / 2018-07-01
==================
  * Also copy files starting with '.' into coverage environment - they might be necessary.
  * Parse pragma ABIEncoderV2

0.5.4 / 2018-05-23
==================
  * Use require.resolve() to get the path of testrpc-sc (lerna + yarn workspaces compatibility) (contribution @vdrg)

0.5.3 / 2018-05-22
=================
  * Add -L flag when copying packages specified in the copyPackages option (following symlinks | lerna + yarn workspaces compatibility) (contribution
    @vdrg)

0.5.2 / 2018-05-20
==================
  * Silence security warnings coming from the parser by upgrading mocha there to v4.
  * Kill testrpc w/ tree-kill so that the childprocess actually dies in linux.

0.5.0 / 2018-04-20
==================
  * Update README for 0.5.0
  * Cleanup stdout/stderr streams on exit. This might stop testrpc-sc from running as a background
    zombie on Linux systems, post-exit.
  * Support `constructor` keyword
  * Prefix instrumentation events with `emit` keyword
  * (Temporarily) remove support for ternary conditional branch coverage. Solidity no longer allows
    us to emit instrumentationevents within the grammatical construction @area devised to
    make this possible.

0.4.15 / 2018-03-28
===================
  * Update parser to allow `emit` keyword (contribution @andresilva).

0.4.14 / 2018-03-15
====================
* Fix misc bugs related to testrpc-sc using an older version of `ganache-cli` to webpack testrpc-sc
  by bumping to testrpc-sc 6.1.2

0.4.13 / 2018-03-15
====================
  * Fix bug introduced in 0.4.12 that broke internal rpc server launch. (contribution @andresilva)

0.4.12 / 2018-03-13
===================
  * Fix bug that caused parser to crash on JS future-reserved keywords like `class`
  * Use newer more stable `ganache-core` as core of testrpc-sc.
  * Update instrumentation to allow interface contracts.

0.4.11 / 2018-03-04
===================
  * Add @vdrg to contributor list
  * Update parser to allow function types as function parameters (contribution: vrdg)

0.4.10 / 2018-02-24
===================
  * Fix bug that corrupted the line coverage alignment in reports when view/pure modifiers
    occupied their own line.

0.4.9 / 2018-01-23
==================
  * Fix bug that ommitted line-coverage for lines with trailing '//' comment

0.4.8 / 2018-01-02
==================

  * Fix windows html report bug caused by failure to pass path relativized mapping to Istanbul

0.4.5 - 0.4.7 / 2017-12-21
==================

  * Fix parsing bug preventing fn definition in structs. Bump parser to 0.4.4

0.4.4 / 2017-12-19
==================
  * Fix build folder management by only deleting its contracts folder (contribution: ukstv)
  * Document problems/solutions when Truffle must be a local dependency.

0.4.3 / 2017-12-08
==================

  * Stop requiring that `view` `pure` and `constant` methods be invoked by `.call` (solution: spalladino @ zeppelin)
  * Add ability to include specific node_modules packages (contribution: e11io), dramatically speeding
    up coverage env generation for larger projects.
  * Add ability to skip instrumentation for an entire folder.

0.4.2 / 2017-11-20
==================

  * Bug fix to gracefully handle *.sol files that are invalid Solidity during pre-processing.

0.4.1 / 2017-11-19
==================

  * Bug fix to allow `constant` keyword for variables by only removing visibility modifiers from
    functions. Uses the preprocessor walking over the AST rather than a regex

0.4.0 / 2017-11-08 (Compatible with testrpc >= 6.0 / pragma 0.4.18 and above)
==================

  * Bug fix to accommodate strict enforcement of constant and view modifiers in pragma 0.4.18

0.3.5 / 2017-11-07 (Compatible with testrpc >= 6.0 / pragma 0.4.17 and below)
==================

  * Bug fix to accomodate Truffle's simplified interface for view and constant
  * Bug fix to accomodate enforcement of EIP 170 (max contract bytes === 24576)

0.3.0 / 2017-11-05
===================

  * Add sanity checks for contracts and coverageEnv folders
  * Update testrpc-sc to 6.0.3 (Byzantium fork)

0.2.7 / 2017-10-12
=================
  * Fix bug that prevented overloading abstract pure methods imported from outside the
    contracts directory (@elenadimitrova)
  * Begin using npm published solidity-parser-sc / allow upgrading with yarn (@elenadimitrova)
  * Update README and FAQ for Windows users

0.2.6 / 2017-10-11
=================
  * Permit view and pure modifiers
  * Permit experimental pragma
  * Upgrade development deps to truffle4 beta and solc 0.4.17
  * Fix bug causing large suites that use the internal testrpc launch to crash mysteriously
    by increasing testrpc-sc stdout buffer size. (@rudolfix / Neufund contribution)
  * Fix bugs that made tool (completely) unrunnable and report unreadable on Windows. (@phiferd contribution)
  * Fix bug that caused tool to crash when reading the events log from very large test suites by
    reading the log line by line as a stream. (@rudolfix / Neufund contribution)

0.2.5 / 2017-10-02
=================
  * Revert vm stipend fix, it was corrupting balances. `send` & `transfer` to instrumented fallback
    will fail now though.

0.2.4 / 2017-09-22
=================

  * Fix bug where sigint handler wasn't declared early enough in the exec script, resulting
    in occasional failure to cleanup.

0.2.3 / 2017-09-13
=================

  * Add unit tests for transfers and sends to instrumented fallback fns.
  * Fix bug in testrpc-sc causing transfers/sends to fail with instrumented fallback fns.

0.2.2 / 2017-08-21
=================

  * Allow truffle.js to be named truffle-config.js for windows compatibility
  * Remove old logic that handled empty function body instrumentation (lastchar special case)
  * Correctly instrument comments near function's {

0.2.1 / 2017-07-29
=================

  * Verify testrpc-sc's location in node_modules before invoking (yarn / package-lock.json fix)
  * Fix lcov html report bug introduced in 0.1.9

0.2.0 / 2017-07-26
=================

  * Stop ignoring package-lock.json - npm version wants it
  * Add testrpc-sc publication docs
  * Update to use testrpc-sc v 4.0.1 (tracking testrpc 4.0.1)

0.1.10 / 2017-07-25
==================

  * Cover assert/require statements as if they were if statements (which, secretly, they are)
  * Add documentation justifying the above changes
  * Upgraded solc to 0.4.13
  * Switch to using ethereumjs-vm-sc in order to be able to test case where asserts and requires fail

0.1.9 / 2017-07-23
==================

  * Pin testrpc-sc to its 3.0.3 branch as a safe-haven while we upgrade to testrpc v4
  * Add changelog
  * Simplify and reorder README, add CI integration guide
  * Add testrpc-sc signing test and lint
  * Clear cache on CI, add Maurelian to contributor list
  * exec.js refactor: modularized and moved logic to lib/app.js
  * More informative TestRPC failure logging

0.1.8 / 2017-07-13
==================

  * Add Alan Lu as contributor
    Also remove mysterious crash known issue, since he fixed it.
  * Fix testrpc-sc race condition
  * Test command runs after TestRPC starts listening
  * Improved mock test command
  * Added test for race condition
  * README updates: remove require info, add memory info
  * Add Invalid JSON RPC note to known issues in README

0.1.7 / 2017-07-05
==================

  * Instrument empty contract bodies correctly
  * Instrument conditional assignment to MemberExpressions

0.1.6 / 2017-07-03
==================

  * Add gas estimation example. Pin truffle to 3.2.5
  * Allow files to be skipped during coverage
    While ordinarily we shouldn't want to do these, it is possible to
    construct valid contracts using assembly that break when the coverage
    events are injected.

0.1.5 / 2017-06-26
==================

  * Fix istanbul exiting error
  * Fix tuple parsing / update tests

0.1.4 / 2017-06-26
==================

  * Change testrpc path for yarn compatibility
  * Small exec.js cleanup, clarify port options settings in README
  * Unit test copying project into env
  * Copy all directories when setting up coverageEnv
    The exception is `node_modules`, which must have copyNodeModules
    set to `true` in .solcover.js in order to be included.

0.1.3 / 2017-06-21
==================

  * Stop crashing on encounters with non-truffle projects

0.1.2 / 2017-06-21
==================

  * Add repository field to package, use cache again on CI

0.1.1 / 2017-06-20
==================

  * Remove events warning, update package webpage, misc rewordings
  * Fix testrpc filter tests. Disable yarn
  * Add (disabled) events filter unit test
  * Add truffle as dev dep, re-yarn
  * Add topic logging to coverageMap
  * Add yarn.lock, use yarn on CI
  * Use coverage network port if avail (& unit test).
  * Edits to HDWalletProvider notes
  * Add npm version badge, update known issues
  * Pin SP to sc-forks#master (has post-install script to build parser)
  * Remove parse.js dummy node list, order nodes alphabetically
    Note: This change exposes the fact that a number of cases aren't actually being checked in the
    parse table. Possible test-deficits / parse-table logic defects here.
  * Remove parse.js dummy node list, order nodes alphabetically
  * add waffle.io badge

0.1.0 / 2017-05-13
==================

  * Change install instructions, small edits & formatting

0.0.1 / 2017-05-13
==================

  * Move files into /lib and /bin
  * Use node_modules/.bin/testrpc-sc
  * Disambiguate package name & fix readme option params
  * Edit readme to reflect repo name, add options, contributors, contrib guidelines
  * Update to Truffle3, refactor CLI, add CLI unit tests, fix misc bugs
  * Disable two "config" tests for CI - multiple testrpc launches max out container memory limit
  * Rename "run" folders/files "cli" for consistency
  * Fix broken chained call handling, add unit tests to verify cases get coverage
  * Add unit test for arbitrary testing command config option, remove test flush
  * Allow testrpc options string in config, rename run to cli (test sequencing fix)
  * Update README with known issues, links to zeppelin example report
