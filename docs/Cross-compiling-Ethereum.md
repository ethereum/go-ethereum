**Note: All of these and much more have been merged into the project Makefile.
You can cross build via `make geth-<os>-<platform>` without needing to know any
of these details from below.**

Developers usually have a preferred platform that they feel most comfortable
working in, with all the necessary tools, libraries and environments set up for
an optimal workflow. However, there's often need to build for either a different
CPU architecture, or an entirely different operating system; but maintaining a
development environment for each and switching between the them quickly becomes
unwieldy.

Here we present a very simple way to cross compile Ethereum to various operating
systems and architectures using a minimal set of prerequisites and a completely
containerized approach, guaranteeing that your development environment remains
clean even after the complex requirements and mechanisms of a cross compilation.

The currently supported target platforms are:

 - ARMv7 Android and iOS
 - 32 bit, 64 bit and ARMv5 Linux
 - 32 bit and 64 bit Mac OSX
 - 32 bit and 64 bit Windows

Please note, that cross compilation does not replace a release build. Although
resulting binaries can usually run perfectly on the desired platform, compiling
on a native system with the specialized tools provided by the official vendor
can often result in more a finely optimized code.

## Cross compilation environment

Although the `go-ethereum` project is written in Go, it does include a bit of C
code shared between all implementations to ensure that all perform equally well,
including a dependency to the GNU Multiple Precision Arithmetic Library. Because
of these, Go cannot by itself compile to a different platform than the host. To
overcome this limitation, we will use [`xgo`](https://github.com/karalabe/xgo),
a Go cross compiler package based on Docker containers that has been architected
specifically to allow both embedded C snippets as well as simpler external C
dependencies during compilation.

The `xgo` project has two simple dependencies: Docker (to ensure that the build
environment is completely contained) and Go. On most platforms these should be
available from the official package repositories. For manually installing them,
please consult their install guides at [Docker](https://docs.docker.com/installation/)
and [Go](https://golang.org/doc/install) respectively. This guide assumes that these
two dependencies are met.

To install and/or update xgo, simply type:

    $ go get -u github.com/karalabe/xgo

You can test whether `xgo` is functioning correctly by requesting it to cross
compile itself and verifying that all cross compilations succeeded or not.

    $ xgo github.com/karalabe/xgo
    ...

    $ ls -al
    -rwxr-xr-x  1 root     root      2792436 Sep 14 16:45 xgo-android-21-arm
    -rwxr-xr-x  1 root     root      2353212 Sep 14 16:45 xgo-darwin-386
    -rwxr-xr-x  1 root     root      2906128 Sep 14 16:45 xgo-darwin-amd64
    -rwxr-xr-x  1 root     root      2388288 Sep 14 16:45 xgo-linux-386
    -rwxr-xr-x  1 root     root      2960560 Sep 14 16:45 xgo-linux-amd64
    -rwxr-xr-x  1 root     root      2437864 Sep 14 16:45 xgo-linux-arm
    -rwxr-xr-x  1 root     root      2551808 Sep 14 16:45 xgo-windows-386.exe
    -rwxr-xr-x  1 root     root      3130368 Sep 14 16:45 xgo-windows-amd64.exe


## Building Ethereum

Cross compiling Ethereum is analogous to the above example, but an additional
flags is required to satisfy the dependencies:

 - `--deps` is used to inject arbitrary C dependency packages and pre-build them

Injecting the GNU Arithmetic Library dependency and selecting `geth` would be:

    $ xgo --deps=https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2 \
          github.com/ethereum/go-ethereum/cmd/geth
    ...

    $ ls -al
    -rwxr-xr-x  1 root     root     23213372 Sep 14 17:59 geth-android-21-arm
    -rwxr-xr-x  1 root     root     14373980 Sep 14 17:59 geth-darwin-386
    -rwxr-xr-x  1 root     root     17373676 Sep 14 17:59 geth-darwin-amd64
    -rwxr-xr-x  1 root     root     21098910 Sep 14 17:59 geth-linux-386
    -rwxr-xr-x  1 root     root     25049693 Sep 14 17:59 geth-linux-amd64
    -rwxr-xr-x  1 root     root     20578535 Sep 14 17:59 geth-linux-arm
    -rwxr-xr-x  1 root     root     16351260 Sep 14 17:59 geth-windows-386.exe
    -rwxr-xr-x  1 root     root     19418071 Sep 14 17:59 geth-windows-amd64.exe


As the cross compiler needs to build all the dependencies as well as the main
project itself for each platform, it may take a while for the build to complete
(approximately 3-4 minutes on a Core i7 3770K machine).

### Fine tuning the build

By default Go, and inherently `xgo`, checks out and tries to build the master
branch of a source repository. However, more often than not, you'll probably
want to build a different branch from possibly an entirely different remote
repository. These can be controlled via the `--remote` and `--branch` flags.

To build the `develop` branch of the official `go-ethereum` repository instead
of the default `master` branch, you just need to specify it as an additional
command line flag (`--branch`):

    $ xgo --deps=https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2 \
          --branch=develop                                          \
          github.com/ethereum/go-ethereum/cmd/geth

Additionally, during development you will most probably want to not only build
a custom branch, but also one originating from your own fork of the repository
instead of the upstream one. This can be done via the `--remote` flag:

    $ xgo --deps=https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2 \
          --remote=https://github.com/karalabe/go-ethereum          \
          --branch=rpi-staging                                      \
          github.com/ethereum/go-ethereum/cmd/geth

By default `xgo` builds binaries for all supported platforms and architectures,
with Android binaries defaulting to the highest released Android NDK platform.
To limit the build targets or compile to a different Android platform, use the
`--targets` CLI parameter.

    $ xgo --deps=https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2 \
          --targets=android-16/arm,windows/*                        \
          github.com/ethereum/go-ethereum/cmd/geth

### Building locally

If you would like to cross compile your local development version, simply specify
a local path (starting with `.` or `/`), and `xgo` will use all local code from
`GOPATH`, only downloading missing dependencies. In such a case of course, the
`--branch`, `--remote` and `--pkg` arguments are no-op:

    $ xgo --deps=https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2 \
          ./cmd/geth

## Using the Makefile

Having understood the gist of `xgo` based cross compilation, you do not need to
actually memorize and maintain these commands, as they have been incorporated into
the official [Makefile](https://github.com/ethereum/go-ethereum/blob/master/Makefile)
and can be invoked with a trivial `make` request:

 * `make geth-cross`: Cross compiles to every supported OS and architecture
 * `make geth-<os>`: Cross compiles supported architectures of a particular OS (e.g. `linux`)
 * `make geth-<os>-<arch>`: Cross compiles to a specific OS/architecture (e.g. `linux`, `arm`)

We advise using the `make` based commands opposed to manually invoking `xgo` as we do
maintain the Makefile actively whereas we cannot guarantee that this document will be
always readily updated to latest advancements.

### Tuning the cross builds

A few of the `xgo` build options have also been surfaced directly into the Makefile to
allow fine tuning builds to work around either upstream Go issues, or to enable some
fancier mechanics.

 - `make ... GO=<go>`: Use a specific Go runtime (e.g. `1.5.1`, `1.5-develop`, `develop`)
 - `make ... MODE=<mode>`: Build a specific target type (e.g. `exe`, `c-archive`).

Please note that these are not yet fully finalized, so they may or may not change in
the future as our code and the Go runtime features change.