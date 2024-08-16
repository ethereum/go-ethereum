---
title: Developer docs
description: Documentation for Geth developers and dapp developers
---

Welcome to the Geth Developer docs!

This section includes information for builders. If you are building decentralized apps on top of Geth, head to the [Dapp developers](#dapp-developers) docs. If you are developing Geth itself, explore the [Geth developers](#geth-developers) docs.

## Dapp developers {#dapp-developers}

Geth has many features that support dapp developers. There are many built-in tracers implemented in Go or Javascript that allow developers to monitor what is happening in Geth from inside an app, and users can build their own custom tracers too. Geth also includes a suite of tools for interacting with Ethereum smart contracts using Geth functions using Go functions inside Go native applications. There is also information for Geth mobile developers.

- [Dev mode](/docs/developers/dapp-developer/dev-mode)
- [Go API](/docs/developers/dapp-developer/native)
- [Go Account management](/docs/developers/dapp-developer/native-accounts)
- [Go contract bindings](/docs/developers/dapp-developer/native-bindings)
- [Geth for mobile](/docs/developers/dapp-developer/mobile)

## EVM tracing

Tracing allows developers to analyze precisely what the EVM has done or will do given a certain set of commands. This section outlines the various ways tracing can be implemented in Geth.

- [Introduction](/docs/developers/evm-tracing/)
- [Basic traces](/docs/developers/evm-tracing/basic-traces)
- [Built-in tracers](/docs/developers/evm-tracing/built-in-tracers)
- [Custom EVM tracers](/docs/developers/evm-tracing/custom-tracer)
- [Live tracing](/docs/developers/evm-tracing/live-tracing)
- [Tutorial for Javascript tracing](/docs/developers/evm-tracing/javascript-tutorial)

## Geth developers {#geth-developers}

Geth developers add/remove features and fix bugs in Geth. The `geth-developer` section includes contribution guidelines and documentation relating to testing and disclosing vulnerabilities that will help you get started with working on Geth.

- [Developer guide](/docs/developers/geth-developer/dev-guide)
- [Disclosures](/docs/developers/geth-developer/disclosures)
- [DNS discovery setup guide](/docs/developers/geth-developer/dns-discovery-setup)
- [Code review guidelines](/docs/developers/geth-developer/code-review-guidelines)
- [Contributing](/docs/developers/geth-developer/contributing)
