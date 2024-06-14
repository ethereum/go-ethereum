![hardhat_Ignition_banner](https://github.com/NomicFoundation/hardhat-ignition/assets/24030/cc73227b-8791-4bb3-bc9a-a39be69d215f)
[![npm](https://img.shields.io/npm/v/@nomicfoundation/hardhat-ignition.svg)](https://www.npmjs.com/package/@nomicfoundation/hardhat-ignition) [![hardhat](https://hardhat.org/buidler-plugin-badge.svg?1)](https://hardhat.org) [![Open in Dev Containers](https://img.shields.io/static/v1?label=Dev%20Containers&message=Open&color=blue&logo=visualstudiocode)](https://vscode.dev/redirect?url=vscode://ms-vscode-remote.remote-containers/cloneInVolume?url=https://github.com/NomicFoundation/hardhat-ignition.git)

---

# Hardhat Ignition

Hardhat Ignition is a declarative system for deploying smart contracts on Ethereum. It enables you to define smart contract instances you want to deploy, and any operation you want to run on them. By taking over the deployment and execution, Hardhat Ignition lets you focus on your project instead of getting caught up in the deployment details.

Built by the [Nomic Foundation](https://nomic.foundation/) for the Ethereum community.

Join the Hardhat Ignition channel of our [Hardhat Community Discord server](https://hardhat.org/ignition-discord) to stay up to date on new releases and tutorials.

## Installation

```bash
# ethers users
npm install --save-dev @nomicfoundation/hardhat-ignition-ethers

# viem users
npm install --save-dev @nomicfoundation/hardhat-ignition-viem
```

Import the plugin in your `hardhat.config.js``:

```js
// ethers users
require("@nomicfoundation/hardhat-ignition-ethers");

// viem users
require("@nomicfoundation/hardhat-ignition-viem");
```

Or if you are using TypeScript, in your `hardhat.config.ts``:

```js
// ethers users
import "@nomicfoundation/hardhat-ignition-ethers";

// viem users
import "@nomicfoundation/hardhat-ignition-viem";
```

## Documentation

On [Hardhat Ignition's website](https://hardhat.org/ignition) you will find guides for:

- [Getting started](https://hardhat.org/ignition/docs/getting-started)
- [Creating Modules](https://hardhat.org/ignition/docs/guides/creating-modules)
- [Deploying a module](https://hardhat.org/ignition/docs/guides/deploy)
- [Visualizing your module](https://hardhat.org/ignition/docs/guides/visualize)
- [Handling errors](https://hardhat.org/ignition/docs/guides/error-handling)
- [Modifying an existing module](https://hardhat.org/ignition/docs/guides/modifications)
- [Using Hardhat Ignition in your tests](https://hardhat.org/ignition/docs/guides/tests)

## Contributing

Contributions are always welcome! Feel free to open any issue or send a pull request.

Go to [CONTRIBUTING.md](https://github.com/NomicFoundation/hardhat-ignition/blob/main/CONTRIBUTING.md) to learn about how to set up Hardhat Ignition's development environment.

## Feedback, help and news

[Hardhat Ignition on Discord](https://hardhat.org/ignition-discord): for questions and feedback.

Follow [Hardhat](https://twitter.com/HardhatHQ) and [Nomic Foundation](https://twitter.com/NomicFoundation) on Twitter.
