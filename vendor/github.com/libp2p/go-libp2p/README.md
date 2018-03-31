
<h1 align="center">
  <a href="libp2p.io"><img width="250" src="https://github.com/libp2p/libp2p/blob/master/logo/alternates/libp2p-logo-alt-2.png?raw=true" alt="libp2p hex logo" /></a>
</h1>

<h3 align="center">The Go implementation of the libp2p Networking Stack.</h3>

<p align="center">
  <a href="http://ipn.io"><img src="https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square" /></a>
  <a href="http://libp2p.io/"><img src="https://img.shields.io/badge/project-libp2p-blue.svg?style=flat-square" /></a>
  <a href="http://webchat.freenode.net/?channels=%23ipfs"><img src="https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square" /></a>
  <a href="https://waffle.io/libp2p/libp2p"><img src="https://img.shields.io/badge/pm-waffle-blue.svg?style=flat-square" /></a>
</p>

<p align="center">
  <a href="https://travis-ci.org/libp2p/go-libp2p"><img src="https://travis-ci.org/libp2p/go-libp2p.svg?branch=master" /></a>
  <!--<a href="https://circleci.com/gh/libp2p/go-libp2p"><img src="https://circleci.com/gh/libp2p/go-libp2p.svg?style=svg" /></a>-->
  <!--<a href="https://coveralls.io/github/libp2p/go-libp2p?branch=master"><img src="https://coveralls.io/repos/github/libp2p/go-libp2p/badge.svg?branch=master"></a>-->
  <br>
  <a href="https://github.com/RichardLitt/standard-readme"><img src="https://img.shields.io/badge/standard--readme-OK-green.svg?style=flat-square" /></a>
  <a href="https://godoc.org/github.com/libp2p/go-libp2p"><img src="https://godoc.org/github.com/ipfs/go-libp2p?status.svg" /></a>
  <a href=""><img src="https://img.shields.io/badge/golang-%3E%3D1.8.0-orange.svg?style=flat-square" /></a>
  <br>
</p>

# Project status

[![Throughput Graph](https://graphs.waffle.io/libp2p/go-libp2p/throughput.svg)](https://waffle.io/libp2p/go-libp2p/metrics/throughput)

# Table of Contents

- [Background](#background)
- [Bundles](#bundles)
- [Usage](#usage)
  - [Install](#install)
  - [API](#api)
  - [Examples](#examples)
- [Development](#development)
  - [Tests](#tests)
  - [Packages](#packages)
- [Contribute](#contribute)
- [License](#license)

## Background

[libp2p](https://github.com/libp2p/specs) is a networking stack and library modularized out of [The IPFS Project](https://github.com/ipfs/ipfs), and bundled separately for other tools to use.
>
libp2p is the product of a long, and arduous quest of understanding -- a deep dive into the internet's network stack, and plentiful peer-to-peer protocols from the past. Building large scale peer-to-peer systems has been complex and difficult in the last 15 years, and libp2p is a way to fix that. It is a "network stack" -- a protocol suite -- that cleanly separates concerns, and enables sophisticated applications to only use the protocols they absolutely need, without giving up interoperability and upgradeability. libp2p grew out of IPFS, but it is built so that lots of people can use it, for lots of different projects.
>
> We will be writing a set of docs, posts, tutorials, and talks to explain what p2p is, why it is tremendously useful, and how it can help your existing and new projects. But in the meantime, check out
>
> - [**The libp2p Specification**](https://github.com/libp2p/specs)
> - [**go-libp2p implementation**](https://github.com/libp2p/go-libp2p)
> - [**js-libp2p implementation**](https://github.com/libp2p/js-libp2p)


## Bundles

There is currently only one bundle of `go-libp2p`, this package. This bundle is used by [`go-ipfs`](https://github.com/ipfs/go-ipfs).

## Usage

`go-libp2p` repo is a place holder for the list of Go modules that compose Go libp2p, as well as its entry point.

### Install

```bash
> go get -d github.com/libp2p/go-libp2p/...
> cd $GOPATH/src/github.com/libp2p/go-libp2p
> make
> make deps
```

### API

[![GoDoc](https://godoc.org/github.com/ipfs/go-libp2p?status.svg)](https://godoc.org/github.com/libp2p/go-libp2p)

### Examples

Examples can be found on the [examples folder](examples).

## Development

### Dependencies

While developing, you need to use [gx to install and link your dependencies](https://github.com/whyrusleeping/gx#dependencies), to do that, run:

```sh
> make deps
```

Before commiting and pushing to Github, make sure to rewind the gx'ify of dependencies. You can do that with:

```sh
> make publish
```

### Tests

Running of individual tests is done through `gx test <path to test>`

```bash
$ cd $GOPATH/src/github.com/libp2p/go-libp2p
$ make deps
$ gx test ./p2p/<path of module you want to run tests for>
```

### Packages

> **WIP**

List of packages currently in existence for libp2p:

| Package            | Version | CI                  |
|--------------------|---------|---------------------|
| **Transports**                                     |
| **Connection Upgrades**                            |
| **Stream Muxers**                                  |
| **Discovery**                                      |
| **Crypto Channels**                                |
| **Peer Routing**                                   |
| **Content Routing**                                |
| **Miscellaneous**                                  |
| **Data Types**                                     |

# Contribute

go-libp2p is part of [The IPFS Project](https://github.com/ipfs/ipfs), and is MIT licensed open source software. We welcome contributions big and small! Take a look at the [community contributing notes](https://github.com/ipfs/community/blob/master/contributing.md). Please make sure to check the [issues](https://github.com/ipfs/go-libp2p/issues). Search the closed ones before reporting things, and help us with the open ones.

Guidelines:

- read the [libp2p spec](https://github.com/libp2p/specs)
- please make branches + pull-request, even if working on the main repository
- ask questions or talk about things in [Issues](https://github.com/libp2p/go-libp2p/issues) or #ipfs on freenode.
- ensure you are able to contribute (no legal issues please-- we use the DCO)
- run `go fmt` before pushing any code
- run `golint` and `go vet` too -- some things (like protobuf files) are expected to fail.
- get in touch with @jbenet and @diasdavid about how best to contribute
- have fun!

There's a few things you can do right now to help out:
 - Go through the modules below and **check out existing issues**. This would be especially useful for modules in active development. Some knowledge of IPFS/libp2p may be required, as well as the infrasture behind it - for instance, you may need to read up on p2p and more complex operations like muxing to be able to help technically.
 - **Perform code reviews**.
 - **Add tests**. There can never be enough tests.

## Modularizing go-libp2p

We have currently a work in progress of modularizing go-libp2p from a repo monolith to several packages in different repos that can be reused for other projects of swapped for custom libp2p builds.

We want to maintain history, so we'll use git-subtree for extracting packages. Find instructions below:

```sh
# 1) create the extracted tree (has the directory specified as -P as its root)
> cd go-libp2p/
> git subtree split -P p2p/crypto/secio/ -b libp2p-secio
62b0a5c21574bcbe06c422785cd5feff378ae5bd
# important to delete the tree now, so that outdated imports fail in step 5
> git rm -r p2p/crypto/secio/
> git commit
> cd ../

# 2) make the new repo
> mkdir go-libp2p-secio
> cd go-libp2p-secio/
> git init && git commit --allow-empty

# 3) fetch the extracted tree from the previous repo
> git remote add libp2p ../go-libp2p
> git fetch libp2p
> git reset --hard libp2p/libp2p-secio

# 4) update self import paths
> sed -someflagsidontknow 'go-libp2p/p2p/crypto/secio' 'golibp2p-secio'
> git commit

# 5) create package.json and check all imports are correct
> vim package.json
> gx --verbose install --global
> gx-go rewrite
> go test ./...
> gx-go rewrite --undo
> git commit

# 4) make the package ready
> vim README.md LICENSE
> git commit

# 5) bump the version separately
> vim package.json
> gx publish
> git add package.json .gx/
> git commit -m 'Publish 1.2.3'

# 6) clean up and push
> git remote rm libp2p
> git push origin master
```
