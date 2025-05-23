# Hardhat Ignition Visualize UI

> Warning: this package is used internally within Hardhat Ignition, and is
> not intended to be used directly.

The website used in Hardhat Ignition's `visualize` task for visualising
a deployment.

## Development

A development server can be run from the root of this package with:

```sh
pnpm dev
```

By default in development the deployment in `./public/deployment.json` is used,
to overwrite this example deployment, update the module in
`./examples/ComplexModule.js` and run the regenerate command:

```sh
pnpm regenerate-deployment-example
```

## Contributing

Contributions are always welcome! Feel free to open any issue or send a pull request.

Go to [CONTRIBUTING.md](https://github.com/NomicFoundation/hardhat-ignition/blob/main/CONTRIBUTING.md) to learn about how to set up Hardhat Ignition's development environment.

## Feedback, help and news

[Hardhat Ignition on Discord](https://hardhat.org/ignition-discord): for questions and feedback.

Follow [Hardhat](https://twitter.com/HardhatHQ) and [Nomic Foundation](https://twitter.com/NomicFoundation) on Twitter.
