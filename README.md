## Expanse Go

Expanse Go Client, by Christopher Franko (forked from Jeffrey Wilcke (and some other people)'s Expanse Go client).

          | Linux   | OSX | ARM | Windows | Tests
----------|---------|-----|-----|---------|------
develop   | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Linux%20Go%20develop%20branch)](https://build.ethdev.com/builders/Linux%20Go%20develop%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Linux%20Go%20develop%20branch)](https://build.ethdev.com/builders/OSX%20Go%20develop%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=ARM%20Go%20develop%20branch)](https://build.ethdev.com/builders/ARM%20Go%20develop%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Windows%20Go%20develop%20branch)](https://build.ethdev.com/builders/Windows%20Go%20develop%20branch/builds/-1) | [![Buildr+Status](https://travis-ci.org/expanse/go-expanse.svg?branch=develop)](https://travis-ci.org/expanse/go-expanse) [![Coverage Status](https://coveralls.io/repos/expanse/go-expanse/badge.svg?branch=develop)](https://coveralls.io/r/expanse/go-expanse?branch=develop)
master    | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Linux%20Go%20master%20branch)](https://build.ethdev.com/builders/Linux%20Go%20master%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=OSX%20Go%20master%20branch)](https://build.ethdev.com/builders/OSX%20Go%20master%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=ARM%20Go%20master%20branch)](https://build.ethdev.com/builders/ARM%20Go%20master%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Windows%20Go%20master%20branch)](https://build.ethdev.com/builders/Windows%20Go%20master%20branch/builds/-1) | [![Buildr+Status](https://travis-ci.org/expanse/go-expanse.svg?branch=master)](https://travis-ci.org/expanse/go-expanse) [![Coverage Status](https://coveralls.io/repos/expanse/go-expanse/badge.svg?branch=master)](https://coveralls.io/r/expanse/go-expanse?branch=master)

[![Bugs](https://badge.waffle.io/expanse/go-expanse.png?label=bug&title=Bugs)](https://waffle.io/expanse/go-expanse)
[![Stories in Ready](https://badge.waffle.io/expanse/go-expanse.png?label=ready&title=Ready)](https://waffle.io/expanse/go-expanse)
[![Stories in Progress](https://badge.waffle.io/expanse/go-expanse.svg?label=in%20progress&title=In Progress)](http://waffle.io/expanse/go-expanse)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/expanse/go-expanse?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

## Automated development builds

The following builds are build automatically by our build servers after each push to the [develop](https://github.com/expanse-project/go-expanse/tree/develop) branch.

* [Docker](https://registry.hub.docker.com/u/expanse/client-go/)
* [OS X](http://build.ethdev.com/builds/OSX%20Go%20develop%20branch/Mist-OSX-latest.dmg)
* Ubuntu
  [trusty](https://build.ethdev.com/builds/Linux%20Go%20develop%20deb%20i386-trusty/latest/) |
  [utopic](https://build.ethdev.com/builds/Linux%20Go%20develop%20deb%20i386-utopic/latest/)
* [Windows 64-bit](https://build.ethdev.com/builds/Windows%20Go%20develop%20branch/Geth-Win64-latest.zip)
* [ARM](https://build.ethdev.com/builds/ARM%20Go%20develop%20branch/gexp-ARM-latest.tar.bz2)

## Building the source

For prerequisites and detailed build instructions please read the
[Installation Instructions](https://github.com/expanse-project/go-expanse/wiki/Building-Expanse)
on the wiki.

Building gexp requires two external dependencies, Go and GMP.
You can install them using your favourite package manager.
Once the dependencies are installed, run

    make gexp

## Executables

Go Expanse comes with several wrappers/executables found in
[the `cmd` directory](https://github.com/expanse-project/go-expanse/tree/develop/cmd):

* `gexp` Expanse CLI (expanse command line interface client)
* `bootnode` runs a bootstrap node for the Discovery Protocol
* `ethtest` test tool which runs with the [tests](https://github.com/expanse-project/tests) suite:
  `/path/to/test.json > ethtest --test BlockTests --stdin`.
* `evm` is a generic Expanse Virtual Machine: `evm -code 60ff60ff -gas
  10000 -price 0 -dump`. See `-h` for a detailed description.
* `disasm` disassembles EVM code: `echo "6001" | disasm`
* `rlpdump` prints RLP structures

## Command line options

`gexp` can be configured via command line options, environment variables and config files.

To get the options available:

<<<<<<< HEAD
    geth help

For further details on options, see the [wiki](https://github.com/expanse-project/go-expanse/wiki/Command-Line-Options)

## Contribution

If you'd like to contribute to go-expanse please fork, fix, commit and
send a pull request. Commits who do not comply with the coding standards
are ignored (use gofmt!). If you send pull requests make absolute sure that you
commit on the `develop` branch and that you do not merge to master.
Commits that are directly based on master are simply ignored.

See [Developers' Guide](https://github.com/expanse-project/go-expanse/wiki/Developers'-Guide)
for more details on configuring your environment, testing, and
dependency management.
