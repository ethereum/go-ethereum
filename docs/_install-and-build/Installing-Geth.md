---
title: Installing Geth
sort_key: A
---

There are several ways to install Geth, including via a package manager, downloading a pre-built bundle, running as a docker container or building from downloaded source code. On this page the various installation options are explained for several major operating systems. Users prioritizing ease of installation should choose to use a package manager or prebuilt bundle. Users prioritizing customization should build from source. It is important to run the latest version of Geth because each release includes bugfixes and improvement over the previous versions. The stable releases are recommended for most users because they have been fully tested. A list of stable releases can be found [here][geth-releases]. Instructions for updating existing Geth installations are also provided in each section.


{:toc}

-   this will be removed by the toc

## Package managers

### MacOS via Homebrew

The easiest way to install go-ethereum is to use the Geth Homebrew tap. The first step is to check that Homebrew is installed. The following command should return a version number.

```shell
brew -v
```

If a version number is returned, then Homebrew is installed. If not, Homebrew can be installed by following the instructions [here][brew]. With Homebrew installed, the following commands add the Geth tap and install Geth:


```shell
brew tap ethereum/ethereum
brew install ethereum
```

The previous command installs the latest stable release. Developers that wish to install the most up-to-date version can install the Geth repository's master branch by adding the `--devel` parameter to the install command:

```shell
brew install ethereum --devel
```

These commands install the core Geth software and the following developer tools: `clef`, `devp2p`, `abigen`, `bootnode`, `evm`, `rlpdump` and `puppeth`. The binaries for each of these tools are saved in `/usr/local/bin/`. The full list of command line options can be viewed [here][geth-cl-options] or in the terminal by running `geth --help`.

Updating an existing Geth installation to the latest version can be achieved by stopping the node and running the following commands:

```shell
brew update 
brew upgrade 
brew reinstall ethereum
```

When the node is started again, Geth will automatically use all the data from the previous version and sync the blocks that were missed while the node was offline.


### Ubuntu via PPAs

The easiest way to install Geth on Ubuntu-based distributions is with the built-in launchpad PPAs (Personal Package Archives). A single PPA repository is provided, containing stable and development releases for Ubuntu versions `xenial`, `trusty`, `impish`, `focal`, `bionic`.

The following command enables the launchpad repository:

```shell
sudo add-apt-repository -y ppa:ethereum/ethereum
```

Then, to install the stable version of go-ethereum:

```shell
sudo apt-get update
sudo apt-get install ethereum
```

Or, alternatively the develop version:

```shell
sudo apt-get update
sudo apt-get install ethereum-unstable
```

These commands install the core Geth software and the following developer tools: `clef`, `devp2p`, `abigen`, `bootnode`, `evm`, `rlpdump` and `puppeth`. The binaries for each of these tools are saved in `/usr/local/bin/`. The full list of command line options can be viewed [here][geth-cl-options] or in the terminal by running `geth --help`.


Updating an existing Geth installation to the latest version can be achieved by stopping the node and running the following commands:

```shell
sudo apt-get update
sudo apt-get install ethereum
sudo apt-get upgrade geth
```

When the node is started again, Geth will automatically use all the data from the previous version and sync the blocks that were missed while the node was offline.


### Windows

The easiest way to install Geth is to download a pre-compiled binary from the [downloads][geth-dl] page. The page provides an installer as well as a zip file containing the Geth source code. The install wizard offers the user the option to install Geth, or Geth and the developer tools. The installer adds `geth` to the system's `PATH` automatically. The zip file contains the command `.exe` files that can be run from the command prompt. The full list of command line options can be viewed [here][geth-cl-options] or in the terminal by running `geth --help`.

Updating an existing Geth installation can be achieved by stopping the node, downloading and installing the latest version following the instructions above. When the node is started again, Geth will automatically use all the data from the previous version and sync the blocks that were missed while the node was offline.

### FreeBSD via pkg

Geth can be installed on FreeBSD using the package manager `pkg`. The following command downloads and installs Geth:

```shell
pkg install go-ethereum
```

These commands install the core Geth software and the following developer tools: `clef`, `devp2p`, `abigen`, `bootnode`, `evm`, `rlpdump` and `puppeth`. 

The full list of command line options can be viewed [here][geth-cl-options] or in the terminal by running `geth --help`.


Updating an existing Geth installation to the latest version can be achieved by stopping the node and running the following commands:

```shell
pkg upgrade
```

When the node is started again, Geth will automatically use all the data from the previous version and sync the blocks that were missed while the node was offline.


### FreeBSD via ports

Installing Geth using ports, simply requires navigating to the `net-p2p/go-ethereum` ports directory and running `make install` as root:

```shell
cd /usr/ports/net-p2p/go-ethereum
make install
```

These commands install the core Geth software and the following developer tools: `clef`, `devp2p`, `abigen`, `bootnode`, `evm`, `rlpdump` and `puppeth`. The binaries for each of these tools are saved in `/usr/local/bin/`. 

The full list of command line options can be viewed [here][geth-cl-options] or in the terminal by running `geth --help`.


Updating an existing Geth installation can be achieved by stopping the node and running the following command:

```shell
portsnap fetch
```

When the node is started again, Geth will automatically use all the data from the previous version and sync the blocks that were missed while the node was offline.

### Arch Linux via pacman

The Geth package is available from the [community repo][geth-archlinux]. It can be installed by running:

```shell
pacman -S geth
```

These commands install the core Geth software and the following developer tools: `clef`, `devp2p`, `abigen`, `bootnode`, `evm`, `rlpdump` and `puppeth`. The binaries for each of these tools are saved in `/usr/bin/`. 

The full list of command line options can be viewed [here][geth-cl-options] or in the terminal by running `geth --help`.

Updating an existing Geth installation can be achieved by stopping the node and running the following command:

```shell
sudo pacman -Sy
```

When the node is started again, Geth will automatically use all the data from the previous version and sync the blocks that were missed while the node was offline.

## Standalone bundle

Stable releases and development builds are provided as standalone bundles. These are useful for users who: a) wish to install a specific version of Geth (e.g., for reproducible environments); b) wish to install on machines without internet access (e.g. air-gapped computers); or c) wish to avoid automatic updates and instead prefer to manually install software.

The following standalone bundles are available:

-   32bit, 64bit, ARMv5, ARMv6, ARMv7 and ARM64 archives (`.tar.gz`) on Linux
-   64bit archives (`.tar.gz`) on macOS
-   32bit and 64bit archives (`.zip`) and installers (`.exe`) on Windows


Some archives contain only Geth, while other archives containing Geth and the various developer tools (`clef`, `devp2p`, `abigen`, `bootnode`, `evm`, `rlpdump` and `puppeth`). More information about these executables is available at the [`README`][geth-readme-exe].

The standalone bundles can be downloaded from the [Geth Downloads][geth-dl] page. To update an existing installation, download and manually install the latest version.

## Docker container

A Docker image with recent snapshot builds from our `develop` branch is maintained on DockerHub to support users who prefer to run containerized processes. There four different Docker images available for running the latest stable or development versions of Geth.

-   `ethereum/client-go:latest` is the latest development version of Geth (default)
-   `ethereum/client-go:stable` is the latest stable version of Geth
-   `ethereum/client-go:{version}` is the stable version of Geth at a specific version number
-   `ethereum/client-go:release-{version}` is the latest stable version of Geth at a specific version family

Pulling an image and starting a node is achieved by running these commands:

```shell
docker pull ethereum/client-go
docker run -it -p 30303:30303 ethereum/client-go
```

There are also four different Docker images for running the latest stable or development versions of miscellaneous Ethereum tools.

-   `ethereum/client-go:alltools-latest` is the latest development version of the Ethereum tools
-   `ethereum/client-go:alltools-stable` is the latest stable version of the Ethereum tools
-   `ethereum/client-go:alltools-{version}` is the stable version of the Ethereum tools at a specific version number
-   `ethereum/client-go:alltools-release-{version}` is the latest stable version of the Ethereum tools at a specific version family

The image has the following ports automatically exposed:

-   `8545` TCP, used by the HTTP based JSON RPC API
-   `8546` TCP, used by the WebSocket based JSON RPC API
-   `8547` TCP, used by the GraphQL API
-   `30303` TCP and UDP, used by the P2P protocol running the network


**Note:** if you are running an Ethereum client inside a Docker container, you should mount a data volume as the client's data directory (located at `/root/.ethereum` inside the container) to ensure that downloaded data is preserved between restarts and/or container life-cycles.


Updating Geth to the latest version simply requires stopping the container, pulling the latest version from Docker and running it:

```shell
docker stop ethereum/client-go
docker pull ethereum/client-go:latest
docker run -it -p 30303:30303 ethereum/client-go
```

## Build from source code

### Most Linux systems and macOS

Geth is written in [Go][go], so building from source code requires the most recent version of Go to be installed. Instructions for installing Go are available at the [Go installation page][go-install] and necessary bundles can be downloaded from the [Go download page][go-dl].

With Go installed, Geth can be downloaded into a `GOPATH` workspace via:


```shell
go get -d github.com/ethereum/go-ethereum
```

You can also install specific versions via:

```shell
go get -d github.com/ethereum/go-ethereum@v1.9.21
```


The above commands do not build any executables. To do that you can either build one specifically:

```shell
go install github.com/ethereum/go-ethereum/cmd/geth
```


Alternatively, the following command, run in the project root directory (`ethereum/go-ethereum`) in the GO workspace, builds the entire project and installs Geth and all the developer tools:


```shell
go install ./...
```

For macOS users, errors related to macOS header files are usually fixed by installing XCode Command Line Tools with `xcode-select --install`.
Another common error is: `go: cannot use path@version syntax in GOPATH mode`. This and other similar errors can often be fixed by enabling gomodules using `export GO111MODULE=on`.

Updating an existing Geth installation can be achieved using `go get`:

```shell
go get -u github.com/ethereum/go-ethereum
```


### Windows

The Chocolatey package manager provides an easy way to install the required build tools. Chocolatey can be installed by following these [instructions][chocolatey]. Then, to install the build tool the following commands can be run in an Administrator command prompt:


```
C:\Windows\system32> choco install git
C:\Windows\system32> choco install golang
C:\Windows\system32> choco install mingw
```

Installing these packages sets up the path environment variables. To get the new path a new command prompt must be opened. To install Geth, a Go workspace directory must first be created, then the Geth source code can be created and built.

```
C:\Users\xxx> mkdir src\github.com\ethereum
C:\Users\xxx> git clone https://github.com/ethereum/go-ethereum src\github.com\ethereum\go-ethereum
C:\Users\xxx> cd src\github.com\ethereum\go-ethereum
C:\Users\xxx\src\github.com\ethereum\go-ethereum> go get -u -v golang.org/x/net/context
C:\Users\xxx\src\github.com\ethereum\go-ethereum> go install -v ./cmd/...
```

### FreeBSD

To build Geth from source code on FreeBSD, the Geth Github repository can be cloned into a local directory.


```shell
git clone https://github.com/ethereum/go-ethereum
```

Then, the Go compiler can be used to build Geth:

```shell
pkg install go
```


If the Go version currently installed is >= 1.5, Geth can be built using the following command:

```shell
cd go-ethereum
make geth
```

If the installed Go version is &lt; 1.5 (quarterly packages, for example), the following command can be used instead:

```shell
cd go-ethereum
CC=clang make geth
```

To start the node, the following command can be run:

```shell
build/bin/geth
```

### Building without a Go workflow

Geth can also be built without using Go workspaces. In this case, the repository should be cloned to a local repository. Then, the command
`make geth` configures everything for a temporary build and cleans up afterwards. This method of building only works on UNIX-like operating systems, and a Go installation is still required.

```shell
git clone https://github.com/ethereum/go-ethereum.git
cd go-ethereum
make geth
```

These commands create a Geth executable file in the `go-ethereum/build/bin` folder that can be moved and run from another directory if required. The binary is standalone and doesn't require any additional files.

To update an existing Geth installation simply stop the node, navigate to the project root directory and pull the latest version from the Geth Github repository. then rebuild and restart the node.

```shell
cd go-ethereum
git pull
make geth
```

Additionally all the developer tools provided with Geth (`clef`, `devp2p`, `abigen`, `bootnode`, `evm`, `rlpdump` and `puppeth`) can be compiled by running `make all`. More information about these tools can be found [here][geth-readme-exe].

Instructions for cross-compiling to another architecture are available in the [cross-compilation guide](./cross-compile).

To build a stable release, e.g. v1.9.21, the command `git checkout v1.9.21` retrieves that specific version. Executing that command before running `make geth` switches Geth to a stable branch.



[brew]: https://brew.sh/
[go]: https://golang.org/
[go-dl]: https://golang.org/dl/
[go-install]: https://golang.org/doc/install
[chocolatey]: https://chocolatey.org
[geth-releases]: https://github.com/ethereum/go-ethereum/releases
[geth-readme-exe]: https://github.com/ethereum/go-ethereum#executables
[geth-cl-options]: https://geth.ethereum.org/docs/interface/command-line-options
[geth-archlinux]: https://www.archlinux.org/packages/community/x86_64/geth/
[geth-dl]: ../../downloads/


