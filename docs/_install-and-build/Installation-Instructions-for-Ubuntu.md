---
title: Installation instructions for Ubuntu
---

## Install on Ubuntu via PPAs

You can install go-ethereum on Ubuntu-based distributions using the built-in launchpad PPAs (Personal Package Archives). We provide a single PPA repository with both our stable and our development releases for Ubuntu versions `trusty`, `xenial`, `zesty` and `artful`.

Install dependencies first:

```shell
sudo apt-get install software-properties-common
```

To enable our launchpad repository run:

```shell
sudo add-apt-repository -y ppa:ethereum/ethereum
```

After that you can install the stable version of go-ethereum:

```shell
sudo apt-get update
sudo apt-get install ethereum
```

Or the development version with:

```shell
sudo apt-get update
sudo apt-get install ethereum-unstable
```
