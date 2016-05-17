## Expanse Go

Expanse Go Client, by Christopher Franko (forked from Jeffrey Wilcke (and some other people)'s Expanse Go client).

          | Linux   | OSX | ARM | Windows | Tests
----------|---------|-----|-----|---------|------
develop   | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Linux%20Go%20develop%20branch)](https://build.ethdev.com/builders/Linux%20Go%20develop%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Linux%20Go%20develop%20branch)](https://build.ethdev.com/builders/OSX%20Go%20develop%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=ARM%20Go%20develop%20branch)](https://build.ethdev.com/builders/ARM%20Go%20develop%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Windows%20Go%20develop%20branch)](https://build.ethdev.com/builders/Windows%20Go%20develop%20branch/builds/-1) | [![Buildr+Status](https://travis-ci.org/expanse-project/go-expanse.svg?branch=develop)](https://travis-ci.org/expanse/go-expanse) [![Coverage Status](https://coveralls.io/repos/expanse-project/go-expanse/badge.svg?branch=develop)](https://coveralls.io/r/expanse/go-expanse?branch=develop)
master    | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Linux%20Go%20master%20branch)](https://build.ethdev.com/builders/Linux%20Go%20master%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=OSX%20Go%20master%20branch)](https://build.ethdev.com/builders/OSX%20Go%20master%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=ARM%20Go%20master%20branch)](https://build.ethdev.com/builders/ARM%20Go%20master%20branch/builds/-1) | [![Build+Status](https://build.ethdev.com/buildstatusimage?builder=Windows%20Go%20master%20branch)](https://build.ethdev.com/builders/Windows%20Go%20master%20branch/builds/-1) | [![Buildr+Status](https://travis-ci.org/expanse-project/go-expanse.svg?branch=master)](https://travis-ci.org/expanse-project/go-expanse) [![Coverage Status](https://coveralls.io/repos/expanse-project/go-expanse/badge.svg?branch=master)](https://coveralls.io/r/expanse-project/go-expanse?branch=master)

[![Bugs](https://badge.waffle.io/expanse-project/go-expanse.png?label=bug&title=Bugs)](https://waffle.io/expanse/go-expanse)
[![Stories in Ready](https://badge.waffle.io/expanse-project/go-expanse.png?label=ready&title=Ready)](https://waffle.io/expanse/go-expanse)
[![Stories in Progress](https://badge.waffle.io/expanse-project/go-expanse.svg?label=in%20progress&title=In Progress)](http://waffle.io/expanse/go-expanse)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/expanse/go-expanse?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)
## Automated development builds

The following builds are build automatically by our build servers after each push to the [develop](https://github.com/expanse-project/go-expanse/tree/develop) branch.

* [Docker](https://registry.hub.docker.com/u/expanse/go-expanse/)
* [OS X](http://build.ethdev.com/builds/OSX%20Go%20develop%20branch/Mist-OSX-latest.dmg)
* Ubuntu
  [trusty](https://build.ethdev.com/builds/Linux%20Go%20develop%20deb%20i386-trusty/latest/) |
  [utopic](https://build.ethdev.com/builds/Linux%20Go%20develop%20deb%20i386-utopic/latest/)
* [Windows 64-bit](https://build.ethdev.com/builds/Windows%20Go%20develop%20branch/Gexp-Win64-latest.zip)
* [ARM](https://build.ethdev.com/builds/ARM%20Go%20develop%20branch/gexp-ARM-latest.tar.bz2)

## Building the source

For prerequisites and detailed build instructions please read the
[Installation Instructions](https://github.com/expanse-project/go-expanse/wiki/Building-Expanse)
on the wiki.

Building gexp requires both a Go and a C compiler.
You can install them using your favourite package manager.
Once the dependencies are installed, run

    make gexp

## Executables

Go Expanse comes with several wrappers/executables found in
[the `cmd` directory](https://github.com/expanse-project/go-expanse/tree/develop/cmd):

* `gexp` Expanse CLI (expanse command line interface client)
* `bootnode` runs a bootstrap node for the Discovery Protocol
* `exptest` test tool which runs with the [tests](https://github.com/expanse-project/tests) suite:
  `/path/to/test.json > exptest --test BlockTests --stdin`.
* `evm` is a generic Expanse Virtual Machine: `evm -code 60ff60ff -gas
  10000 -price 0 -dump`. See `-h` for a detailed description.
* `disasm` disassembles EVM code: `echo "6001" | disasm`
* `rlpdump` prints RLP structures

## Contribution

`gexp` can be configured via command line options, environment variables and config files.

If you'd like to contribute to go-expanse, please fork, fix, commit and send a pull request
for the maintainers to review and merge into the main code base. If you wish to submit more
complex changes though, please check up with the core devs first on [our gitter channel](https://gitter.im/ethereum/go-ethereum)
to ensure those changes are in line with the general philosophy of the project and/or get some
early feedback which can make both your efforts much lighter as well as our review and merge
procedures quick and simple.

Please make sure your contributions adhere to our coding guidlines:

 * Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
 * Code must be documented adherign to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
 * Pull requests need to be based on and opened against the `develop` branch.
 * Commit messages should be prefixed with the package(s) they modify.
   * E.g. "exp, rpc: make trace configs optional"


Please see the [Developers' Guide](https://github.com/expanse-project/go-expanse/wiki/Developers'-Guide)
for more details on configuring your environment, managing project dependencies and testing procedures.

## License

The go-expanse library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](http://www.gnu.org/licenses/lgpl-3.0.en.html), also
included in our repository in the `COPYING.LESSER` file.

The go-expanse binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](http://www.gnu.org/licenses/gpl-3.0.en.html), also included
in our repository in the `COPYING` file.
