# Tomochain

[![Build Status](https://travis-ci.org/tomochain/tomochain.svg?branch=master)](https://travis-ci.org/tomochain/tomochain) [![Join the chat at https://gitter.im/tomochain/tomochain](https://badges.gitter.im/tomochain/tomochain.svg)](https://gitter.im/tomochain/tomochain?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

## About Tomochain

TomoChain is an innovative solution to the scalability problem with the Ethereum blockchain. Our mission is to be a leading force in building the Internet of Value, and its infrastructure. We are working to create an alternative, scalable financial system which is more secure, transparent, efficient, inclusive and equitable for everyone.

TomoChain relies on a system of 150 Masternodes with Proof of Stake Voting consensus that can support near-zero fee, and 2-second transaction confirmation time. Security, stability and chain finality are guaranteed via novel techniques such as double validation, staking via smart-contracts and "true" randomization processes.

Tomochain supports all EVM-compatible smart-contracts, protocols, and atomic cross-chain token transfers. New scaling techniques such as sharding, private-chain generation, hardware integration will be continuously researched and incorporated into Tomochain's masternode architecture which will be an ideal scalable smart-contract public blockchain for decentralized apps, token issuances and token integrations for small and big businesses.

More details can be found at our [technical white paper](https://tomochain.com/docs/technical-whitepaper---1.0.pdf)

Reading more about us on:

- our website: http://tomochain.com
- our blogs and announcements: https://medium.com/tomochain
- our documentation site: https://docs.tomochain.com

## Tomochain vs Giants

Tomochain is built by the mindset of standing on the giants shoulder. We have learned from all advanced technical design concept of many well-known public blockchains on the market and shaped up the platform with our own ingredients. See below the overall technical comparison table that we try to make clear the position of Tomochain comparing to some popular blockchains at the top-tier.

![Tomochain](https://cdn-images-1.medium.com/max/1600/1*LkiIWFHPXh-0Whv3Hm1yMQ.png)

**we just updated the number of masternodes accepted in the network upto 150*

## Building the source

Tomochain provides client binary called `tomo` for both running a masternode and running a full-node. Building `tomo` requires both a Go (1.7+) and a C compiler. Install them by your own way. Once the dependencies are installed, just run below commands:

```bash
$ git clone https://github.com/tomochain/tomochain tomochain
$ cd tomochain
$ make tomo
```

Alternatively, you could quickly download pre-complied binary on our [github release page](https://github.com/tomochain/tomochain/releases)

## Running tomo

## Road map

These following implementation items are eventually dropped into the source code:

- Layer 2 scalability with state sharding
- Asynchronize EVM execution
- Multi-chains interoperable
- Spam filtering
- DEX integration

## Contribution and technical discuss

Thank you for considering to try out our network and/or help out with the source code. We would love to get your help, feel free to lend a hand. Even the smallest bit of code or bug reporting or just discussing ideas are highly appreciated.

If you would like to contribute to tomochain source code, please refer to our Developer Guide for details on configuring development environment, managing dependencies, compiling, testing and submitting your code changes to our repo.

Please also make sure your contributions adhere to the base coding guidelines:

- Code must adhere the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e uses [gofmt](https://golang.org/cmd/gofmt/)).
- Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
- Pull requests need to be based on and opened against the `master` branch.
- Problem you are trying to contribute must be well-explained as an issue on our [github issue page](https://github.com/tomochain/tomochain/issues)
- Commit messages should be short but clear enough and should refer to the corresponding pre-logged issue mentioned above.

For technical discussion, feel free to join our chat at [Gitter](https://gitter.im/tomochain/tomochain).
