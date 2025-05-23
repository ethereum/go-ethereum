# solidity-coverage

[![Gitter chat](https://badges.gitter.im/sc-forks/solidity-coverage.svg)][18]
![npm (tag)](https://img.shields.io/npm/v/solidity-coverage/latest)
[![CircleCI](https://circleci.com/gh/sc-forks/solidity-coverage.svg?style=svg)][20]
[![codecov](https://codecov.io/gh/sc-forks/solidity-coverage/branch/master/graph/badge.svg)][21]
[![Hardhat](https://hardhat.org/buidler-plugin-badge.svg?1)][26]


## Code coverage for Solidity testing
![coverage example][22]

+ For more details about what this is, how it works and potential limitations,
  see [the accompanying article][16].
+ `solidity-coverage` is [Solcover][17]

## Requirements

+ Hardhat >= 2.11.0

## Install
```
$ yarn add solidity-coverage --dev
```

**Require** the plugin in `hardhat.config.js` ([Hardhat docs][26])
```javascript
require('solidity-coverage')
```

Or, if you are using TypeScript, add this to your hardhat.config.ts:
```ts
import 'solidity-coverage'
```

**Resources**:
+ [0.8.0 release notes][31]

## Run
```
npx hardhat coverage [command-options]
```

### Trouble shooting

**Missing or unexpected coverage?** Make sure you're using the latest plugin version and run:
```sh
$ npx hardhat clean
$ npx hardhat compile
$ npx hardhat coverage
```

**Typescript compilation errors?**
```sh
$ npx hardhat compile
$ TS_NODE_TRANSPILE_ONLY=true npx hardhat coverage
```

**Weird test failures or plugin conflicts?**

```sh
# Setting the `SOLIDITY_COVERAGE` env variable tells the coverage plugin to configure the provider
# early in the hardhat task cycle, minimizing conflicts with other plugins or `extendEnvironment` hooks

$ SOLIDITY_COVERAGE=true npx hardhat coverage
```

**Additional Help**
+ [FAQ][1007]
+ [Advanced Topics][1005]


## Command Options
| Option <img width=200/> | Example <img width=750/>| Description <img width=1000/> |
|--------------|------------------------------------|--------------------------------|
| testfiles  | `--testfiles "test/registry/*.ts"` | Test file(s) to run. (Globs must be enclosed by quotes and use [globby matching patterns][38])|
| sources | `--sources myFolder` or `--sources myFile.sol` | Path to *single* folder or file to target for coverage. Path is relative to Hardhat's `paths.sources` (usually `contracts/`) |
| solcoverjs | `--solcoverjs ./../.solcover.js` | Relative path from working directory to config. Useful for monorepo packages that share settings. (Path must be "./" prefixed) |
| matrix   | `--matrix` | Generate a JSON object that maps which mocha tests hit which lines of code. (Useful as an input for some fuzzing, mutation testing and fault-localization algorithms.) [More...][39]|

[<sup>*</sup> Advanced use][14]

## Config Options

Additional options can be specified in a `.solcover.js` config file located in the root directory
of your project.

**Example:**
```javascript
module.exports = {
  skipFiles: ['Routers/EtherRouter.sol']
};
```

| Option <img width=200/>| Type <img width=200/> | Default <img width=1000/> | Description <img width=1000/> |
| ------ | ---- | ------- | ----------- |
| skipFiles | *Array* | `[]` | Array of contracts or folders (with paths expressed relative to the `contracts` directory) that should be skipped when doing instrumentation.(ex: `[ "Routers", "Networks/Polygon.sol"]`) :warning: **RUN THE HARDHAT CLEAN COMMAND AFTER UPDATING THIS**  |
| irMinimum | *Boolean* | `false` | Speeds up test execution times when solc is run in `viaIR` mode. If your project successfully compiles while generating coverage with this option turned on (it may not!) it's worth using |
| modifierWhitelist | *String[]* | `[]` | List of modifier names (ex: `onlyOwner`) to exclude from branch measurement. (Useful for modifiers which prepare something instead of acting as a gate.)) |
| mocha | *Object* | `{ }` | [Mocha options][3] to merge into existing mocha config. `grep` and `invert` are useful for skipping certain tests under coverage using tags in the test descriptions. [More...][24]|
| measureStatementCoverage | *boolean* | `true` | Computes statement (in addition to line) coverage. [More...][34] |
| measureFunctionCoverage | *boolean* | `true` | Computes function coverage. [More...][34] |
| measureModifierCoverage | *boolean* | `true` | Computes each modifier invocation as a code branch. [More...][34] |
| **:printer:    OUTPUT** |  | |  |
| matrixOutputPath | *String* | `./testMatrix.json` | Relative path to write test matrix JSON object to. [More...][39]|
| mochaJsonOutputPath | *String* | `./mochaOutput.json` | Relative path to write mocha JSON reporter object to. [More...][39]|
| abiOutputPath | *String* | `./humanReadableAbis.json` | Relative path to write diff-able ABI data to |
| istanbulFolder | *String* | `./coverage` |  Folder location for Istanbul coverage reports. |
| istanbulReporter | *Array* | `['html', 'lcov', 'text', 'json']` | [Istanbul coverage reporters][2]  |
| silent | *Boolean* | false | Suppress logging output |
| **:recycle:    WORKFLOW HOOKS** |  | |  |
| onServerReady[<sup>*</sup>][14] | *Function* |   | Hook run *after* server is launched, *before* the tests execute. Useful if you need to use the Oraclize bridge or have setup scripts which rely on the server's availability. [More...][23] |
| onPreCompile[<sup>*</sup>][14] | *Function* |   | Hook run *after* instrumentation is performed, *before* the compiler is run. Can be used with the other hooks to be able to generate coverage reports on non-standard / customized directory structures, as well as contracts with absolute import paths. [More...][23] |
| onCompileComplete[<sup>*</sup>][14] | *Function* |  | Hook run *after* compilation completes, *before* tests are run. Useful if you have secondary compilation steps or need to modify built artifacts. [More...][23]|
| onTestsComplete[<sup>*</sup>][14] | *Function* |  | Hook run *after* the tests complete, *before* Istanbul reports are generated. [More...][23]|
| onIstanbulComplete[<sup>*</sup>][14] | *Function* |  | Hook run *after* the Istanbul reports are generated, *before* the coverage task completes. Useful if you need to clean resources up. [More...][23]|
| **:warning:    LOW LEVEL** |  | |  |
| configureYulOptimizer | *Boolean* | false | Setting to `true` lets you specify optimizer details (see next option). If no details are defined it defaults to turning on the yul optimizer and enabling stack allocation |
| solcOptimizerDetails | *Object* | `undefined` |Must be used in combination with `configureYulOptimizer`. Allows you to configure solc's [optimizer details][1001]. (See [FAQ: Running out of stack][1002] ). |


[<sup>*</sup> Advanced use][14]

## Viewing the reports:
+ You can find the Istanbul reports written to the `./coverage/` folder generated in your root directory.

## API

Solidity-coverage's core methods and many utilities are available as an API.

```javascript
const CoverageAPI = require('solidity-coverage/api');
```

[Documentation available here][28].

## Detecting solidity-coverage from another task

If you're writing another plugin or task, it can be helpful to detect whether coverage is running or not.
The coverage plugin sets a boolean variable on the globally injected hardhat environment object for this purpose.

```
hre.__SOLIDITY_COVERAGE_RUNNING === true
```

## Example reports
+ [openzeppelin-contracts][10] (Codecov)

## Funding

You can help fund solidity-coverage development through [DRIPS][1008]. It's a public goods protocol which helps distribute money to packages in your dependency tree. (It's great, check it out.)

## Contribution Guidelines

Contributions are welcome! If you're opening a PR that adds features or options *please consider
writing full [unit tests][11] for them*. (We've built simple fixtures for almost everything
and are happy to add some for your case if necessary).


Set up the development environment with:
```
$ git clone https://github.com/sc-forks/solidity-coverage.git
$ yarn
```

[2]: https://istanbul.js.org/docs/advanced/alternative-reporters/
[3]: https://mochajs.org/api/mocha
[4]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/faq.md#running-out-of-gas
[5]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/faq.md#running-out-of-memory
[6]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/faq.md#running-out-of-time
[7]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/faq.md#continuous-integration
[8]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/faq.md#notes-on-branch-coverage
[9]: https://sc-forks.github.io/metacoin/
[10]: https://app.codecov.io/gh/OpenZeppelin/openzeppelin-contracts
[11]: https://github.com/sc-forks/solidity-coverage/tree/master/test/units
[12]: https://github.com/sc-forks/solidity-coverage/issues
[13]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/faq.md#notes-on-gas-distortion
[14]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/advanced.md
[15]: #config-options
[16]: https://blog.colony.io/code-coverage-for-solidity-eecfa88668c2
[17]: https://github.com/JoinColony/solcover
[18]: https://gitter.im/sc-forks/solidity-coverage?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge
[19]: https://badge.fury.io/js/solidity-coverage
[20]: https://circleci.com/gh/sc-forks/solidity-coverage
[21]: https://codecov.io/gh/sc-forks/solidity-coverage
[22]: https://cdn-images-1.medium.com/max/800/1*uum8t-31bUaa6dTRVVhj6w.png
[23]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/advanced.md#workflow-hooks
[24]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/advanced.md#skipping-tests
[25]: https://github.com/sc-forks/solidity-coverage/issues/417
[26]: https://hardhat.org/
[27]: https://www.trufflesuite.com/docs
[28]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/api.md
[29]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/upgrade.md#upgrading-from-06x-to-070
[30]: https://github.com/sc-forks/solidity-coverage/tree/0.6.x-final#solidity-coverage
[31]: https://github.com/sc-forks/solidity-coverage/releases/tag/v0.7.0
[33]: https://github.com/sc-forks/moloch
[34]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/advanced.md#reducing-the-instrumentation-footprint
[35]: https://github.com/OpenZeppelin/openzeppelin-contracts/blob/e5fbbda9bac49039847a7ed20c1d966766ecc64a/scripts/coverage.js
[36]: https://hardhat.org/
[37]: https://github.com/sc-forks/solidity-coverage/blob/master/HARDHAT_README.md
[38]: https://github.com/sindresorhus/globby#globbing-patterns
[39]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/advanced.md#generating-a-test-matrix
[1001]: https://docs.soliditylang.org/en/v0.8.0/using-the-compiler.html#input-description
[1002]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/faq.md#running-out-of-stack
[1003]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/advanced.md#parallelization-in-ci
[1004]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/advanced.md#coverage-threshold-checks
[1005]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/advanced.md
[1007]: https://github.com/sc-forks/solidity-coverage/blob/master/docs/faq.md
[1008]: https://www.drips.network/app/projects/github/sc-forks/solidity-coverage

