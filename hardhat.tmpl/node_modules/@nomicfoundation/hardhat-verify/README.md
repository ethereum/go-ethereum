[![npm](https://img.shields.io/npm/v/@nomicfoundation/hardhat-verify.svg)](https://www.npmjs.com/package/@nomicfoundation/hardhat-verify) [![hardhat](https://hardhat.org/buidler-plugin-badge.svg?1)](https://hardhat.org)

# hardhat-verify

[Hardhat](https://hardhat.org) plugin to verify the source of code of deployed contracts.

## What

This plugin helps you verify the source code for your Solidity contracts. At the moment, it supports [Etherscan](https://etherscan.io)-based explorers, explorers compatible with its API like [Blockscout](https://www.blockscout.com/) and [Sourcify](https://sourcify.dev/).

It's smart and it tries to do as much as possible to facilitate the process:

- Just provide the deployment address and constructor arguments, and the plugin will detect locally which contract to verify.
- If your contract uses Solidity libraries, the plugin will detect them and deal with them automatically. You don't need to do anything about them.
- A simulation of the verification process will run locally, allowing the plugin to detect and communicate any mistakes during the process.
- Once the simulation is successful the contract will be verified using the Etherscan API.

## Installation

```bash
npm install --save-dev @nomicfoundation/hardhat-verify
```

And add the following statement to your `hardhat.config.js`:

```js
require("@nomicfoundation/hardhat-verify");
```

Or, if you are using TypeScript, add this to your `hardhat.config.ts`:

```js
import "@nomicfoundation/hardhat-verify";
```

## Tasks

This plugin provides the `verify` task, which allows you to verify contracts through Sourcify and Etherscan's service.

## Environment extensions

This plugin does not extend the environment.

## Usage

You need to add the following Etherscan and Sourcify configs to your `hardhat.config.js` file:

```js
module.exports = {
  networks: {
    mainnet: { ... }
  },
  etherscan: {
    // Your API key for Etherscan
    // Obtain one at https://etherscan.io/
    apiKey: "YOUR_ETHERSCAN_API_KEY"
  },
  sourcify: {
    // Disabled by default
    // Doesn't need an API key
    enabled: true
  }
};
```

Alternatively you can specify more than one block explorer API key, by passing an object under the `apiKey` property, see [`Multiple API keys and alternative block explorers`](#multiple-api-keys-and-alternative-block-explorers).

Lastly, run the `verify` task, passing the address of the contract, the network where it's deployed, and the constructor arguments that were used to deploy it (if any):

```bash
npx hardhat verify --network mainnet DEPLOYED_CONTRACT_ADDRESS "Constructor argument 1"
```

### Complex arguments

When the constructor has a complex argument list, you'll need to write a javascript module that exports the argument list. The expected format is the same as a constructor list for an [ethers contract](https://docs.ethers.org/v6/single-page/#api_contract__Contract). For example, if you have a contract like this:

```solidity
struct Point {
  uint x;
  uint y;
}

contract Foo {
  constructor (uint x, string s, Point memory point, bytes b) { ... }
}
```

then you can use an `arguments.js` file like this:

```js
module.exports = [
  50,
  "a string argument",
  {
    x: 10,
    y: 5,
  },
  // bytes have to be 0x-prefixed
  "0xabcdef",
];
```

Where the third argument represents the value for the `point` parameter.

The module can then be loaded by the `verify` task when invoked like this:

```bash
npx hardhat verify --constructor-args arguments.js DEPLOYED_CONTRACT_ADDRESS
```

### Libraries with undetectable addresses

Some library addresses are undetectable. If your contract uses a library only in the constructor, then its address cannot be found in the deployed bytecode.

To supply these missing addresses, you can create a javascript module that exports a library dictionary and pass it through the `--libraries` parameter:

```bash
hardhat verify --libraries libraries.js OTHER_ARGS
```

where `libraries.js` looks like this:

```js
module.exports = {
  SomeLibrary: "0x...",
};
```

### Multiple API keys and alternative block explorers

If your project targets multiple EVM-compatible networks that have different explorers, you'll need to set multiple API keys.

To configure the API keys for the chains you are using, provide an object under `etherscan/apiKey` with the identifier of each chain as the key. **This is not necessarily the same name that you are using to define the network**. For example, if you are going to verify contracts in Ethereum mainnet, Optimism and Arbitrum, your config would look like this:

```js
module.exports = {
  networks: {
    mainnet: { ... },
    testnet: { ... }
  },
  etherscan: {
    apiKey: {
        mainnet: "YOUR_ETHERSCAN_API_KEY",
        optimisticEthereum: "YOUR_OPTIMISTIC_ETHERSCAN_API_KEY",
        arbitrumOne: "YOUR_ARBISCAN_API_KEY",
    }
  }
};
```

To see the full list of supported networks, run `npx hardhat verify --list-networks`. The identifiers shown there are the ones that should be used as keys in the `apiKey` object.

### Adding support for other networks

If the chain you are using is not in the list, you can manually add the necessary information to verify your contracts on it. For this you need three things: the chain id of the network, the URL of the verification endpoint, and the URL of the explorer.

For example, if Goerli wasn't supported, you could add it like this:

```
etherscan: {
  apiKey: {
    goerli: "<goerli-api-key>"
  },
  customChains: [
    {
      network: "goerli",
      chainId: 5,
      urls: {
        apiURL: "https://api-goerli.etherscan.io/api",
        browserURL: "https://goerli.etherscan.io"
      }
    }
  ]
}
```

Keep in mind that the name you are giving to the network in `customChains` is the same one that has to be used in the `apiKey` object.

To see which custom chains are supported, run `npx hardhat verify --list-networks`.

### Verifying on Sourcify

To verify a contract using Sourcify, you need to add to your Hardhat config:

```js
sourcify: {
  enabled: true,
  // Optional: specify a different Sourcify server
  apiUrl: "https://sourcify.dev/server",
  // Optional: specify a different Sourcify repository
  browserUrl: "https://repo.sourcify.dev",
}
```

and then run the `verify` task, passing the address of the contract and the network where it's deployed:

```bash
npx hardhat verify --network mainnet DEPLOYED_CONTRACT_ADDRESS
```

**Note:** Constructor arguments are not required for Sourcify verification, but you'll need to provide them if you also have Etherscan verification enabled.

To disable Sourcify verification and suppress related messages, set `enabled` to `false`:

```js
sourcify: {
  enabled: false,
}
```

### Using programmatically

To call the verification task from within a Hardhat task or script, use the `"verify:verify"` subtask. Assuming the same contract as [above](#complex-arguments), you can run the subtask like this:

```js
await hre.run("verify:verify", {
  address: contractAddress,
  constructorArguments: [
    50,
    "a string argument",
    {
      x: 10,
      y: 5,
    },
    "0xabcdef",
  ],
});
```

If the verification is not successful, an error will be thrown.

#### Providing libraries from a script or task

If your contract has libraries with undetectable addresses, you may pass the libraries parameter with a dictionary specifying them:

```js
hre.run("verify:verify", {
  // other args
  libraries: {
    SomeLibrary: "0x...",
  }
}
```

#### Advanced Usage: Using the Etherscan and Sourcify classes from another plugin

Both Etherscan and Sourcify classes can be imported from the plugin for direct use.

- **Etherscan Class Usage**

  ```js
  import { Etherscan } from "@nomicfoundation/hardhat-verify/etherscan";

  const instance = new Etherscan(
    "abc123def123", // Etherscan API key
    "https://api.etherscan.io/api", // Etherscan API URL
    "https://etherscan.io" // Etherscan browser URL
  );

  if (!instance.isVerified("0x123abc...")) {
    const { message: guid } = await instance.verify(
      // Contract address
      "0x123abc...",
      // Contract source code
      '{"language":"Solidity","sources":{"contracts/Sample.sol":{"content":"// SPDX-Lic..."}},"settings":{ ... }}',
      // Contract name
      "contracts/Sample.sol:MyContract",
      // Compiler version
      "v0.8.19+commit.7dd6d404",
      // Encoded constructor arguments
      "0000000000000000000000000000000000000000000000000000000000000032"
    );

    await sleep(1000);
    const verificationStatus = await instance.getVerificationStatus(guid);

    if (verificationStatus.isSuccess()) {
      const contractURL = instance.getContractUrl("0x123abc...");
      console.log(
        `Successfully verified contract "MyContract" on Etherscan: ${contractURL}`
      );
    }
  }
  ```

- **Sourcify Class Usage**

  ```js
  import { Sourcify } from "@nomicfoundation/hardhat-verify/sourcify";

  const instance = new Sourcify(
    1,
    "https://sourcify.dev/server",
    "https://repo.sourcify.dev"
  ); // Set chainId

  if (!instance.isVerified("0x123abc...")) {
    const sourcifyResponse = await instance.verify("0x123abc...", {
      "metadata.json": "{...}",
      "otherFile.sol": "...",
    });
    if (sourcifyResponse.isOk()) {
      const contractURL = instance.getContractUrl(
        "0x123abc...",
        sourcifyResponse.status
      );
      console.log(`Successfully verified contract on Sourcify: ${contractURL}`);
    }
  }
  ```

## How it works

The plugin works by fetching the bytecode in the given address and using it to check which contract in your project corresponds to it. Besides that, some sanity checks are performed locally to make sure that the verification won't fail.

## Known limitations

- Adding, removing, moving or renaming new contracts to the hardhat project or reorganizing the directory structure of contracts after deployment may alter the resulting bytecode in some solc versions. See this [Solidity issue](https://github.com/ethereum/solidity/issues/9573) for further information.
