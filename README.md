# Bor Overview
Bor is the Official Golang implementation of the Polygon PoS blockchain. It is a fork of [geth](https://github.com/ethereum/go-ethereum) and is EVM compatible (upto London fork).

[![API Reference](
https://camo.githubusercontent.com/915b7be44ada53c290eb157634330494ebe3e30a/68747470733a2f2f676f646f632e6f72672f6769746875622e636f6d2f676f6c616e672f6764646f3f7374617475732e737667
)](https://pkg.go.dev/github.com/maticnetwork/bor)
[![Go Report Card](https://goreportcard.com/badge/github.com/maticnetwork/bor)](https://goreportcard.com/report/github.com/maticnetwork/bor)
![MIT License](https://img.shields.io/github/license/maticnetwork/bor)
[![Discord](https://img.shields.io/discord/714888181740339261?color=1C1CE1&label=Polygon%20%7C%20Discord%20%F0%9F%91%8B%20&style=flat-square)](https://discord.gg/0xpolygon)
[![Twitter Follow](https://img.shields.io/twitter/follow/0xPolygon.svg?style=social)](https://twitter.com/0xPolygon)

### Installing bor using packaging

The easiest way to get started with bor is to install the packages using the command below. Refer to the [releases](https://github.com/maticnetwork/bor/releases) section to find the latest stable version of bor.
    
    curl -L https://raw.githubusercontent.com/maticnetwork/install/main/bor.sh | bash -s -- v0.4.0 <network> <node_type>

The network accepts `mainnet`,`amoy` or `mumbai` and the node type accepts `validator` or `sentry` or `archive`. The installation script does the following things:
- Create a new user named `bor`.
- Install the bor binary at `/usr/bin/bor`.
- Dump the suitable config file (based on the network and node type provided) at `/var/lib/bor` and uses it as the home dir.
- Create a systemd service named `bor` at `/lib/systemd/system/bor.service` which starts bor using the config file as `bor` user.

The releases supports both the networks i.e. Polygon Mainnet, Amoy and Mumbai (Testnet) unless explicitly specified. Before the stable release for mainnet, pre-releases will be available marked with `beta` tag for deploying on Mumbai/Amoy (testnet). On sufficient testing, stable release for mainnet will be announced with a forum post.

### Building from source

- Install Go (version 1.19 or later) and a C compiler.
- Clone the repository and build the binary using the following commands:
    ```shell
    make bor
    ```
- Start bor using the ideal config files for validator and sentry provided in the `packaging` folder.
    ```shell
    ./build/bin/bor server --config ./packaging/templates/mainnet-v1/sentry/sentry/bor/config.toml
    ```
- To build full set of utilities, run:
    ```shell
    make all
    ```
- Run unit and integration tests
    ```shell
    make test && make test-integration
    ```

#### Using the new cli

Post `v0.3.0` release, bor uses a new command line interface (cli). The new-cli (located at `internal/cli`) has been built with keeping the flag usage similar to old-cli (located at `cmd/geth`) with a few notable changes. Please refer to [docs](./docs) section for flag usage guide and example.

### Documentation

- The official documentation for the Polygon PoS chain can be found [here](https://wiki.polygon.technology/docs/pos/getting-started/). It contains all the conceptual and architectural details of the chain along with operational guide for users running the nodes.
- New release announcements and discussions can be found on our [forum page](https://forum.polygon.technology/).
- Polygon improvement proposals can be found [here](https://github.com/maticnetwork/Polygon-Improvement-Proposals/)

### Contribution guidelines

Thank you for considering helping out with the source code! We welcome contributions from anyone on the internet, and are grateful for even the smallest of fixes! If you'd like to contribute to bor, please fork, fix, commit and send a pull request for the maintainers to review and merge into the main code base. 

From the outset we defined some guidelines to ensure new contributions only ever enhance the project:

* Quality: Code in the Polygon project should meet the style guidelines, with sufficient test-cases, descriptive commit messages, evidence that the contribution does not break any compatibility commitments or cause adverse feature interactions, and evidence of high-quality peer-review. Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
* Testing: Please ensure that the updated code passes all the tests locally before submitting a pull request. In order to run unit tests, run `make test` and to run integration tests, run `make test-integration`.
* Size: The Polygon project’s culture is one of small pull-requests, regularly submitted. The larger a pull-request, the more likely it is that you will be asked to resubmit as a series of self-contained and individually reviewable smaller PRs.
* Maintainability: If the feature will require ongoing maintenance (e.g. support for a particular brand of database), we may ask you to accept responsibility for maintaining this feature
* Pull requests need to be based on and opened against the `develop` branch.
* PR title should be prefixed with package(s) they modify.
  * E.g. "eth, rpc: make trace configs optional"

## License

The go-ethereum library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html),
also included in our repository in the `COPYING.LESSER` file.

The go-ethereum binaries (i.e. all code inside of the `cmd` directory) are licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also
included in our repository in the `COPYING` file.

## Join our Discord server

Join Polygon community  – share your ideas or just say hi over [on Discord](https://discord.com/invite/0xPolygonDevs).
