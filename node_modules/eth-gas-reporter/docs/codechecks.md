## Guide to using Codechecks

This reporter comes with a [codechecks](http://codechecks.io) CI integration that
displays a pull request's gas consumption changes relative to its target branch in the Github UI.
It's like coveralls for gas. The codechecks service is free for open source and maintained by MakerDao engineer [@krzkaczor](https://github.com/krzkaczor).

![Screen Shot 2019-06-18 at 12 25 49 PM](https://user-images.githubusercontent.com/7332026/59713894-47298900-91c5-11e9-8083-233572787cfa.png)

## Setup

- Enable your project on [codechecks.io](https://codechecks.io/). Check out the
  [getting started guide](https://github.com/codechecks/docs/blob/master/getting-started.md). (All
  you really have to do is toggle your repo 'on' and copy-paste a token into your CI environment
  variables settings.)

- Install the codechecks client library as a dev dependency:

```
npm install --save-dev @codechecks/client
```

- Add a `codechecks.yml` to your project's root directory as below:

```yml
checks:
  - name: eth-gas-reporter/codechecks
```

- Run `codechecks` as a step in your build

```yml
# CircleCI Example
steps:
  - checkout
  - run: npm install
  - run: npm test
  - run: npx codechecks

# Travis
script:
  - npm test
  - npx codechecks
```

- You're done! :elephant:

### Multiple reports (for different CI jobs)

For each report, create a codechecks.yml file, e.g

```
codechecks.testing.yml
codechecks.production.yml
```

Use the `name` option in your `.yml` config to individuate the report:

```yml
# codechecks.production.yml
checks:
  - name: eth-gas-reporter/codechecks
    options:
      name: production
```

When running `codechecks` as a command in CI, specify the relevant codechecks config `.yml`

```yml
production:
  docker:
    - image: circleci/node:10.13.0
  steps:
    - checkout
    - run: npm install
    - run: npm test
    - run: npx codechecks codechecks.production.yml
```

### Failure thresholds

You can ask Codechecks to report the CI run as a failure by using the `maxMethodDiff` and
`maxDeploymentDiff` reporter options. These set the integer percentage difference
over which an increase in gas usage by any method (or deployment) is forbidden.

**Examples**

```js
// truffle-config.js
mocha: {
  reporter: "eth-gas-reporter",
  reporterOptions: {
    maxMethodDiff: 25,
  }
}

// buidler.config.js
gasReporter: {
  maxMethodDiff: 25,
}
```

### Codechecks is new :wrench:

Codechecks is new and some of its quirks are still being ironed out:

- If you're using CircleCI and the report seems to be missing from the first
  build of a pull request, you can [configure your codechecks.yml's branch setting](https://github.com/codechecks/docs/blob/master/configuration.md#settings) to make it work as expected.
- Both Travis and Circle must be configured to run on commit/push
  (this is true by default and will only be a problem if you've turned those builds off to save resources.)

### Diff Report Example

Something like this will be displayed in the `checks` tab of your GitHub pull request.
Increases in gas usage relative to the PR's target branch are highlighted with a red cross, decreases are
highlighted with a green check.

## Deployments

|                         |       Gas |                                                                      |            Diff | Diff % | Block % | chf avg cost |
| :---------------------- | --------: | :------------------------------------------------------------------: | --------------: | -----: | ------: | -----------: |
| **ConvertLib**          |   111,791 |                                                                      |               0 |      0 |   1.4 % |         0.48 |
| **EtherRouter**         |   278,020 |                                                                      |               0 |      0 |   3.5 % |         1.20 |
| **Factory**             |   324,331 | ![passed](https://travis-ci.com/images/stroke-icons/icon-passed.png) | [**-14,222**]() |     4% |   4.1 % |         1.40 |
| **MetaCoin**            |   358,572 | ![failed](https://travis-ci.com/images/stroke-icons/icon-failed.png) | [**+73,534**]() |    26% |   4.5 % |         1.55 |
| **MultiContractFileA**  |    90,745 |                                                                      |               0 |      0 |   1.1 % |         0.39 |
| **MultiContractFileB**  |    90,745 |                                                                      |               0 |      0 |   1.1 % |         0.39 |
| **Resolver**            |   430,580 |                                                                      |               0 |      0 |   5.4 % |         1.86 |
| **VariableConstructor** | 1,001,890 |                                                                      |               0 |      0 |  12.5 % |         4.34 |
| **VariableCosts**       |   930,528 |                                                                      |               0 |      0 |  11.6 % |         4.03 |
| **VersionA**            |    88,665 |                                                                      |               0 |      0 |   1.1 % |         0.38 |
| **Wallet**              |   217,795 |                                                                      |               0 |      0 |   2.7 % |         0.94 |

## Methods

|                              |     Gas |                                                                      |          Diff | Diff % | Calls | chf avg cost |
| :--------------------------- | ------: | :------------------------------------------------------------------: | ------------: | -----: | ----: | -----------: |
| **EtherRouter**              |         |                                                                      |               |        |       |              |
|        *setResolver*         |  43,192 |                                                                      |             0 |      0 |     1 |         0.19 |
| **Factory**                  |         |                                                                      |               |        |       |              |
|        *deployVersionB*      | 107,123 |                                                                      |             0 |      0 |     1 |         0.46 |
| **MetaCoin**                 |         |                                                                      |               |        |       |              |
|        *sendCoin*            |  51,019 |                                                                      |             0 |      0 |     1 |         0.22 |
| **MultiContractFileA**       |         |                                                                      |               |        |       |              |
|        *hello*               |  41,419 |                                                                      |             0 |      0 |     1 |         0.18 |
| **MultiContractFileB**       |         |                                                                      |               |        |       |              |
|        *goodbye*             |  41,419 |                                                                      |             0 |      0 |     1 |         0.18 |
| **Resolver**                 |         |                                                                      |               |        |       |              |
|        *register*            |  37,633 |                                                                      |             0 |      0 |     2 |         0.16 |
| **VariableCosts**            |         |                                                                      |               |        |       |              |
|        *addToMap*            |  90,341 |                                                                      |             0 |      0 |     7 |         0.39 |
|        *methodThatThrows*    |  41,599 |                                                                      |             0 |      0 |     2 |         0.18 |
|        *otherContractMethod* |  57,407 |                                                                      |             0 |      0 |     1 |         0.25 |
|        *removeFromMap*       |  36,481 |                                                                      |             0 |      0 |     8 |         0.16 |
|        *sendPayment*         |  32,335 |                                                                      |             0 |      0 |     1 |         0.14 |
|        *setString*           |  28,787 | ![passed](https://travis-ci.com/images/stroke-icons/icon-passed.png) | [**-2156**]() |     8% |     4 |         0.12 |
|        *transferPayment*     |  32,186 |                                                                      |             0 |      0 |     1 |         0.14 |
| **VersionA**                 |         |                                                                      |               |        |       |              |
|        *setValue*            |  25,663 |                                                                      |             0 |      0 |     1 |         0.11 |
| **VersionB**                 |         |                                                                      |               |        |       |              |
|        *setValue*            |  25,685 |                                                                      |             0 |      0 |     1 |         0.11 |
| **Wallet**                   |         |                                                                      |               |        |       |              |
|        *sendPayment*         |  32,181 |                                                                      |             0 |      0 |     1 |         0.14 |
|        *transferPayment*     |  32,164 |                                                                      |             0 |      0 |     1 |         0.14 |

## Build Configuration

| Option                 | Settings              |
| ---------------------- | --------------------- |
| solc: version          | 0.5.0+commit.1d4f565a |
| solc: optimized        | false                 |
| solc: runs             | 200                   |
| gas: block limit       | 8,000,000             |
| gas: price             | 21 gwei/gas           |
| gas: currency/eth rate | 206.15 chf/eth        |

### Gas Reporter JSON output

The gas reporter now writes the data it collects as JSON to a file at `./gasReporterOutput.json` whenever the environment variable `CI` is set to true. You can see an example of this output [here](https://github.com/cgewecke/eth-gas-reporter/blob/master/docs/gasReporterOutput.md).
You may find it useful as a base to generate more complex or long running gas analyses, develop CI integrations with, or make nicer tables.
