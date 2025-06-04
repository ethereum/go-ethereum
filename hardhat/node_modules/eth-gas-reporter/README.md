# eth-gas-reporter

[![npm version](https://badge.fury.io/js/eth-gas-reporter.svg)](https://badge.fury.io/js/eth-gas-reporter)
[![Build Status](https://travis-ci.org/cgewecke/eth-gas-reporter.svg?branch=master)](https://travis-ci.org/cgewecke/eth-gas-reporter)
[![Codechecks](https://raw.githubusercontent.com/codechecks/docs/master/images/badges/badge-default.svg?sanitize=true)](https://codechecks.io)
[![buidler](https://buidler.dev/buidler-plugin-badge.svg?1)](https://github.com/cgewecke/buidler-gas-reporter)

**A Mocha reporter for Ethereum test suites:**

- Gas usage per unit test.
- Metrics for method calls and deployments.
- National currency costs of deploying and using your contract system.
- CI integration with [codechecks](http://codechecks.io)
- Simple installation for Truffle and Buidler
- Use ETH, BNB, MATIC, AVAX, HT or MOVR price to calculate the gas price.

### Example output

![Screen Shot 2019-06-24 at 4 54 47 PM](https://user-images.githubusercontent.com/7332026/60059336-fa502180-96a0-11e9-92b8-3dd436a9b2f1.png)

### Installation and Config

**[Truffle](https://www.trufflesuite.com/docs)**

```
npm install --save-dev eth-gas-reporter
```

```javascript
/* truffle-config.js */
module.exports = {
  networks: { ... },
  mocha: {
    reporter: 'eth-gas-reporter',
    reporterOptions : { ... } // See options below
  }
};
```

**[Buidler](https://buidler.dev)**

```
npm install --save-dev buidler-gas-reporter
```

```javascript
/* buidler.config.js */
usePlugin('buidler-gas-reporter');

module.exports = {
  networks: { ... },
  gasReporter: { ... } // See options below
};
```

**Other**

This reporter should work with any build platform that uses Mocha and
connects to an Ethereum client running as a separate process. There's more on advanced use cases
[here](https://github.com/cgewecke/eth-gas-reporter/blob/master/docs/advanced.md).

### Continuous Integration (Travis and CircleCI)

This reporter comes with a [codechecks](http://codechecks.io) CI integration that
displays a pull request's gas consumption changes relative to its target branch in the Github UI.
It's like coveralls for gas. The codechecks service is free for open source and maintained by MakerDao engineer [@krzkaczor](https://github.com/krzkaczor).

Complete [set-up guide here](https://github.com/cgewecke/eth-gas-reporter/blob/master/docs/codechecks.md) (it's easy).

![Screen Shot 2019-06-18 at 12 25 49 PM](https://user-images.githubusercontent.com/7332026/59713894-47298900-91c5-11e9-8083-233572787cfa.png)

### Options

:warning: **CoinMarketCap API change** :warning:

Beginning March 2020, CoinMarketCap requires an API key to access currency market
price data. The reporter uses an unprotected . You can get your own API key [here][55] and set it with the `coinmarketcap` option. (This service's free tier allows 10k reqs/mo)

In order to retrieve the gas price of a particular blockchain, you can configure the `token` and `gasPriceApi` (API key rate limit may apply).

**NOTE**: HardhatEVM and ganache-cli implement the Ethereum blockchain. To get accurate gas measurements for other chains you may need to run your tests against development clients developed specifically for those networks.

| Option             | Type                   | Default                                                                    | Description                                                                                                                                                                                                                                  |
| ------------------ | ---------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| currency           | _String_               | 'EUR'                                                                      | National currency to represent gas costs in. Exchange rates loaded at runtime from the `coinmarketcap` api. Available currency codes can be found [here](https://coinmarketcap.com/api/documentation/v1/#section/Standards-and-Conventions). |
| coinmarketcap      | _String_               | (unprotected API key)                                                      | [API key][55] to use when fetching current market price data. (Use this if you stop seeing price data)                                                                                                                                       |
| gasPrice           | _Number_               | (varies)                                                                   | Denominated in `gwei`. Default is loaded at runtime from the `eth gas station` api                                                                                                                                                           |
| token              | _String_               | 'ETH'                                                                      | The reference token for gas price                                                                                                                                                                                                            |
| gasPriceApi        | _String_               | [Etherscan](https://api.etherscan.io/api?module=proxy&action=eth_gasPrice) | The API endpoint to retrieve the gas price. Find below other networks.                                                                                                                                                                       |
| outputFile         | _String_               | stdout                                                                     | File path to write report output to                                                                                                                                                                                                          |
| forceConsoleOutput | _Boolean_              | false                                                                      | Print report output on console                                                                                                                                                                                                               |
| noColors           | _Boolean_              | false                                                                      | Suppress report color. Useful if you are printing to file b/c terminal colorization corrupts the text.                                                                                                                                       |
| onlyCalledMethods  | _Boolean_              | true                                                                       | Omit methods that are never called from report.                                                                                                                                                                                              |
| rst                | _Boolean_              | false                                                                      | Output with a reStructured text code-block directive. Useful if you want to include report in RTD                                                                                                                                            |
| rstTitle           | _String_               | ""                                                                         | Title for reStructured text header (See Travis for example output)                                                                                                                                                                           |
| showTimeSpent      | _Boolean_              | false                                                                      | Show the amount of time spent as well as the gas consumed                                                                                                                                                                                    |
| excludeContracts   | _String[]_             | []                                                                         | Contract names to exclude from report. Ex: `['Migrations']`                                                                                                                                                                                  |
| src                | _String_               | "contracts"                                                                | Folder in root directory to begin search for `.sol` files. This can also be a path to a subfolder relative to the root, e.g. "planets/annares/contracts"                                                                                     |
| url                | _String_               | `web3.currentProvider.host`                                                | RPC client url (ex: "http://localhost:8545")                                                                                                                                                                                                 |
| proxyResolver      | _Function_             | none                                                                       | Custom method to resolve identity of methods managed by a proxy contract.                                                                                                                                                                    |
| artifactType       | _Function_ or _String_ | "truffle-v5"                                                               | Compilation artifact format to consume. (See [advanced use](https://github.com/cgewecke/eth-gas-reporter/blob/master/docs/advanced.md).)                                                                                                     |
| showMethodSig      | _Boolean_              | false                                                                      | Display complete method signatures. Useful when you have overloaded methods you can't tell apart.                                                                                                                                            |
| maxMethodDiff      | _Number_               | undefined                                                                  | Codechecks failure threshold, triggered when the % diff for any method is greater than `number` (integer)                                                                                                                                    |
| maxDeploymentDiff  | _Number_               | undefined                                                                  | Codechecks failure threshold, triggered when the % diff for any deployment is greater than `number` (integer)                                                                                                                                |

[55]: https://coinmarketcap.com/api/pricing/

#### `token` and `gasPriceApi` options example

| Network            | token | gasPriceApi                                                            |
| ------------------ | ----- | ---------------------------------------------------------------------- |
| Ethereum (default) | ETH   | https://api.etherscan.io/api?module=proxy&action=eth_gasPrice          |
| Binance            | BNB   | https://api.bscscan.com/api?module=proxy&action=eth_gasPrice           |
| Polygon            | MATIC | https://api.polygonscan.com/api?module=proxy&action=eth_gasPrice       |
| Avalanche          | AVAX  | https://api.snowtrace.io/api?module=proxy&action=eth_gasPrice          |
| Heco               | HT    | https://api.hecoinfo.com/api?module=proxy&action=eth_gasPrice          |
| Moonriver          | MOVR  | https://api-moonriver.moonscan.io/api?module=proxy&action=eth_gasPrice |

These APIs have [rate limits](https://docs.etherscan.io/support/rate-limits). Depending on the usage, it might require an [API Key](https://docs.etherscan.io/getting-started/viewing-api-usage-statistics).

> NB: Any gas price API call which returns a JSON-RPC response formatted like this is supported: `{"jsonrpc":"2.0","id":73,"result":"0x6fc23ac00"}`.

### Advanced Use

An advanced use guide is available [here](https://github.com/cgewecke/eth-gas-reporter/blob/master/docs/advanced.md). Topics include:

- Getting accurate gas data when using proxy contracts like EtherRouter or ZeppelinOS.
- Configuring the reporter to work with non-truffle, non-buidler projects.

### Example Reports

- [gnosis/gnosis-contracts](https://github.com/cgewecke/eth-gas-reporter/blob/master/docs/gnosis.md)
- [windingtree/LifToken](https://github.com/cgewecke/eth-gas-reporter/blob/master/docs/lifToken.md)

### Usage Notes

- Requires Node >= 8.
- You cannot use `ganache-core` as an in-process provider for your test suite. The reporter makes sync RPC calls
  while collecting data and your tests will hang unless the client is launched as a separate process.
- Method calls that throw are filtered from the stats.
- Contracts that are only ever created by other contracts within Solidity are not shown in the deployments table.

### Troubleshooting

- [Missing price data](./docs/missingPriceData.md)

### Contributions

Feel free to open PRs or issues. There is an integration test and one of the mock test cases is expected to fail. If you're adding an option, you can vaildate it in CI by adding it to the mock options config located [here](https://github.com/cgewecke/eth-gas-reporter/blob/master/mock/config-template.js#L13-L19).

### Credits

All the ideas in this utility have been borrowed from elsewhere. Many thanks to:

- [@maurelian](https://github.com/maurelian) - Mocha reporting gas instead of time is his idea.
- [@cag](https://github.com/cag) - The table borrows from / is based his gas statistics work for the Gnosis contracts.
- [Neufund](https://github.com/Neufund/ico-contracts) - Block limit size ratios for contract deployments and euro pricing are borrowed from their `ico-contracts` test suite.

### Contributors

- [@cgewecke](https://github.com/cgewecke)
- [@rmuslimov](https://github.com/rmuslimov)
- [@area](https://github.com/area)
- [@ldub](https://github.com/ldub)
- [@ben-kaufman](https://github.com/ben-kaufman)
- [@wighawag](https://github.com/wighawag)
- [@ItsNickBarry](https://github.com/ItsNickBarry)
- [@krzkaczor](https://github.com/krzkaczor)
- [@ppoliani](https://github.com/@ppoliani)
- [@gnidan](https://github.com/gnidan)
- [@fodisi](https://github.com/fodisi)
- [@vicnaum](https://github.com/vicnaum)
- [@markmiro](https://github.com/markmiro)
- [@lucaperret](https://github.com/lucaperret)
- [@ChristopherDedominici](https://github.com/ChristopherDedominici)
