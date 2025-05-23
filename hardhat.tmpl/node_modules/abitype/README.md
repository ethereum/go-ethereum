<br/>

<p align="center">
  <a href="https://abitype.dev">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/wagmi-dev/abitype/main/.github/logo-dark.svg">
      <img alt="ABIType logo" src="https://raw.githubusercontent.com/wagmi-dev/abitype/main/.github/logo-light.svg" width="auto" height="45">
    </picture>
  </a>
</p>

<div align="center">
  <a href="https://www.npmjs.com/package/abitype">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://img.shields.io/npm/v/abitype?colorA=21262d&colorB=21262d&style=flat">
      <img src="https://img.shields.io/npm/v/abitype?colorA=f6f8fa&colorB=f6f8fa&style=flat" alt="Version">
    </picture>
  </a>
  <a href="https://www.npmjs.com/package/abitype">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://img.shields.io/npm/dm/abitype?colorA=21262d&colorB=21262d&style=flat">
      <img src="https://img.shields.io/npm/dm/abitype?colorA=f6f8fa&colorB=f6f8fa&style=flat" alt="Downloads per month">
    </picture>
  </a>
  <a href="https://github.com/wagmi-dev/abitype/blob/main/LICENSE">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://img.shields.io/npm/l/abitype?colorA=21262d&colorB=21262d&style=flat">
      <img src="https://img.shields.io/npm/l/abitype?colorA=f6f8fa&colorB=f6f8fa&style=flat" alt="MIT License">
    </picture>
  </a>
  <a href="https://bestofjs.org/projects/abitype">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://img.shields.io/endpoint?colorA=21262d&colorB=21262d&style=flat&url=https://bestofjs-serverless.now.sh/api/project-badge?fullName=wagmi-dev%2Fabitype%26since=daily">
      <img src="https://img.shields.io/endpoint?colorA=f6f8fa&colorB=f6f8fa&style=flat&url=https://bestofjs-serverless.now.sh/api/project-badge?fullName=wagmi-dev%2Fabitype%26since=daily" alt="Best of JS">
    </picture>
  </a>
</div>

---

Strict TypeScript types for Ethereum ABIs. ABIType provides utilities and type definitions for ABI properties and values, covering the [Contract ABI Specification](https://docs.soliditylang.org/en/latest/abi-spec.html), as well as [EIP-712](https://eips.ethereum.org/EIPS/eip-712) Typed Data.

```ts
import { ExtractAbiFunctions } from 'abitype'
import { erc20Abi } from 'abitype/test'

type FunctionNames = ExtractAbiFunctionNames<typeof erc20Abi, 'view'>
//   ^? type FunctionNames = "symbol" | "name" | "allowance" | "balanceOf" | "decimals" | "totalSupply"

type TransferInputTypes = AbiParametersToPrimitiveTypes<
  // ^? type TransferInputTypes = readonly [`0x${string}`, bigint]
  ExtractAbiFunction<typeof erc20Abi, 'transfer'>['inputs']
>
```

Works great for adding blazing fast [autocomplete](https://twitter.com/awkweb/status/1555678944770367493) and type checking to functions, variables, or your own types. No need to generate types with third-party tools – just use your ABI and let TypeScript do the rest!

## TL;DR

ABIType might be a good option for your project if:

- You want to [typecheck](https://abitype.dev/api/types) your ABIs or EIP-712 Typed Data.
- You want to add type inference and autocomplete to your library based on user-provided ABIs or EIP-712 Typed Data, like [wagmi](https://wagmi.sh) and [viem](https://viem.sh).
- You need to [convert ABI types](https://abitype.dev/api/utilities#abiparameterstoprimitivetypes) (e.g. `'string'`) to TypeScript types (e.g. `string`) or other type transformations.
- You need to validate ABIs at [runtime](https://abitype.dev/api/zod) (e.g. after fetching from external resource).
- You don’t want to set up a build process to generate types (e.g. TypeChain).

## Install

```fish
npm i abitype
yarn add abitype
pnpm add abitype
```

## Documentation

For full documentation, visit [abitype.dev](https://abitype.dev).

## Community

If you have questions or need help, reach out to the community at the [ABIType GitHub Discussions](https://github.com/wagmi-dev/abitype/discussions).

## Sponsor

If you find ABIType useful, please consider supporting development on [GitHub Sponsors](https://github.com/sponsors/wagmi-dev?metadata_campaign=abitype_readme) or sending crypto to `wagmi-dev.eth`. Thank you 🙏

## Contributing

If you're interested in contributing to ABIType, please read our [contributing docs](https://github.com/wagmi-dev/abitype/blob/main/.github/CONTRIBUTING.md) **before submitting a pull request**.
