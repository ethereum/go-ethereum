## Advanced Topics

### Configuration for non-buidler, non-truffle projects

The reporter's only strict requirements are:

- Mocha
- The Ethereum client it connects to is _in a separate process_ and accepts calls over
  http. (You cannot use ganache-core as an in-process provider, for example.)

Apart from that, it should be possible to run the reporter in any environment by configuring
the following:

- The root directory to begin searching for `.sol` files in via the `src` option.

- The client `url` the reporter uses to send calls.

- The method the reporter uses to acquire necessary info from solc compilation artifacts.
  Truffle and Buidler are supported out of the box but you can also use the `artifactType`
  option to define a function which meets your use case. This method
  receives a contract name (ex: `MetaCoin`) and must return an object as below:

```js
// Example function
function myArtifactProcessor(contractName){...}

// Output
{
  // Required
  abi: []
  bytecode: "0xabc.." // solc: "0x" + contract.evm.bytecode.object

  // Optional
  deployedBytecode: "0xabc.." // solc: "0x" + contract.evm.deployedBytecode.object
  metadata: {
    compiler: {
      version: "0.5.8"
    },
    settings: {
      optimizer: {
        enabled: true,
        runs: 500
      }
    }
  }
}
```

Example artifact handlers can be found [here](https://github.com/cgewecke/eth-gas-reporter/blob/master/lib/artifactor.js).

### Resolving method identities when using proxy contracts

Many projects use a proxy contract strategy like
[EtherRouter](https://github.com/PeterBorah/ether-router) or
[ZeppelinOS](https://docs.zeppelinos.org/docs/start.html) to manage their upgradeability requirements.
In practice this means method calls are routed through the
proxy's fallback function and forwarded to the contract system's current implementation.

You can define a helper method for the `proxyResolver` option
which makes matching methods to contracts in these cases possible.
The reporter automatically detects proxy use when methods are called
on a contract whose ABI does not include their signature. It then
invokes `proxyResolver` to make additional calls to the router contract and establish the true
identity of the transaction target.

**Resources**

- An [implementation](https://github.com/cgewecke/eth-gas-reporter/blob/master/lib/etherRouter.js) for EtherRouter.
- The [code](https://github.com/cgewecke/eth-gas-reporter/blob/master/lib/transactionWatcher.js) which consumes the proxyResolver.

PRs are welcome if you have a proxy mechanism you'd like supported by default.

### Gas Reporter JSON output

The gas reporter now writes the data it collects to a JSON file at `./gasReporterOutput.json` whenever the environment variable `CI` is set to true. An example of this output is [here](https://github.com/cgewecke/eth-gas-reporter/blob/master/docs/gasReporterOutput.md).
You may find it useful as an input to more complex / long running gas analyses, better CI integrations, etc.
