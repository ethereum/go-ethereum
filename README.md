## Ethereum Go

Ethereum Go Client © 2014 Jeffrey Wilcke.

          | Linux   | OSX | Windows | Tests
----------|---------|-----|---------|------
develop   | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Linux%20Go%20develop%20branch)](https://build.ethdev.com/builders/Linux%20Go%20develop%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Linux%20Go%20develop%20branch)](https://build.ethdev.com/builders/OSX%20Go%20develop%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Windows%20Go%20develop%20branch)](https://build.ethdev.com/builders/Windows%20Go%20develop%20branch/builds/-1) | [![Buildr+Status](https://travis-ci.org/ethereum/go-ethereum.svg?branch=develop)](https://travis-ci.org/ethereum/go-ethereum) [![Coverage Status](https://coveralls.io/repos/ethereum/go-ethereum/badge.svg?branch=develop)](https://coveralls.io/r/ethereum/go-ethereum?branch=develop)
master    | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Linux%20Go%20master%20branch)](https://build.ethdev.com/builders/Linux%20Go%20master%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=OSX%20Go%20master%20branch)](https://build.ethdev.com/builders/OSX%20Go%20master%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Windows%20Go%20master%20branch)](https://build.ethdev.com/builders/Windows%20Go%20master%20branch/builds/-1) | [![Buildr+Status](https://travis-ci.org/ethereum/go-ethereum.svg?branch=master)](https://travis-ci.org/ethereum/go-ethereum) [![Coverage Status](https://coveralls.io/repos/ethereum/go-ethereum/badge.svg?branch=master)](https://coveralls.io/r/ethereum/go-ethereum?branch=master)

[![Bugs](https://badge.waffle.io/ethereum/go-ethereum.png?label=bug&title=Bugs)](https://waffle.io/ethereum/go-ethereum)
[![Stories in Ready](https://badge.waffle.io/ethereum/go-ethereum.png?label=ready&title=Ready)](https://waffle.io/ethereum/go-ethereum)
[![Stories in Progress](https://badge.waffle.io/ethereum/go-ethereum.svg?label=in%20progress&title=In Progress)](http://waffle.io/ethereum/go-ethereum)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/ethereum/go-ethereum?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)


Build
=====

Mist (GUI):

`go get github.com/ethereum/go-ethereum/cmd/mist`

Geth (CLI):

`go get github.com/ethereum/go-ethereum/cmd/geth`

As of POC-8, go-ethereum uses [Godep](https://github.com/tools/godep) to manage dependencies. Assuming you have [your environment all set up](https://github.com/ethereum/go-ethereum/wiki/Building-Ethereum), switch to the go-ethereum repository root folder, and build/install the executable you need:

Mist (GUI):

```
godep go build -v ./cmd/mist
```

Geth (CLI):

```
godep go build -v ./cmd/geth
```

Instead of `build`, you can use `install` which will also install the resulting binary.

For prerequisites and detailed build instructions please see the [Wiki](https://github.com/ethereum/go-ethereum/wiki/Building-Ethereum)

If you intend to develop on go-ethereum, check the [Developers' Guide](https://github.com/ethereum/go-ethereum/wiki/Developers'-Guide)

Automated (dev) builds
======================

* [Docker](https://registry.hub.docker.com/u/ethereum/client-go/)
* [OS X](http://build.ethdev.com/builds/OSX%20Go%20develop%20branch/Mist-OSX-latest.dmg)
* Ubuntu
  [trusty](https://build.ethdev.com/builds/Linux%20Go%20develop%20deb%20i386-trusty/latest/) |
  [utopic](https://build.ethdev.com/builds/Linux%20Go%20develop%20deb%20i386-utopic/latest/)
* [Windows] Coming soon&trade;

Executables
===========

Go Ethereum comes with several wrappers/executables found in 
[the `cmd` directory](https://github.com/ethereum/go-ethereum/tree/develop/cmd):

* `mist` Official Ethereum Browser (ethereum GUI client)
* `geth` Ethereum CLI (ethereum command line interface client)
* `bootnode` runs a bootstrap node for the Discovery Protocol
* `ethtest` test tool which runs with the [tests](https://github.com/ethereum/testes) suite: 
  `cat file | ethtest`.
* `evm` is a generic Ethereum Virtual Machine: `evm -code 60ff60ff -gas
  10000 -price 0 -dump`. See `-h` for a detailed description.
* `disasm` disassembles EVM code: `echo "6001" | disasm`
* `rlpdump` converts a rlp stream to `interface{}`.

Command line options
============================

Both `mist` and `geth` can be configured via command line options, environment variables and config files.

To get the options available:

```
geth -help
```

For further details on options, see the [wiki](https://github.com/ethereum/go-ethereum/wiki/Command-Line-Options)

Contribution
============

If you'd like to contribute to go-ethereum please fork, fix, commit and
send a pull request. Commits who do not comply with the coding standards
are ignored (use gofmt!). If you send pull requests make absolute sure that you
commit on the `develop` branch and that you do not merge to master.
Commits that are directly based on master are simply ignored.

For dependency management, we use [godep](https://github.com/tools/godep). After installing with `go get github.com/tools/godep`, run `godep restore` to ensure that changes to other repositories do not break the build. To update a dependency version (for example, to include a new upstream fix), run `go get -u <foo/bar>` then `godep update <foo/...>`. To track a new dependency, add it to the project as normal than run `godep save ./...`. Changes to the [Godeps folder](https://github.com/ethereum/go-ethereum/tree/develop/Godeps): should be manually verified then commited.

To make life easier try [git flow](http://nvie.com/posts/a-successful-git-branching-model/) it sets this all up and streamlines your work flow.

See [Developers' Guide](https://github.com/ethereum/go-ethereum/wiki/Developers'-Guide)

