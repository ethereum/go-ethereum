---
title: Geth
---
`geth` is the the command line interface for running a full ethereum node implemented in Go. 
It is the main deliverable of the [Frontier Release](https://ethereum.gitbooks.io/frontier-guide/content/frontier.html)

## Capabilities

By installing and running `geth`, you can take part in the ethereum frontier live network and
* mine real ether 
* transfer funds between addresses
* create contracts and send transactions
* explore block history
* and much much more

## Install 

Supported Platforms are Linux, Mac Os and Windows.

We support two types of installation: binary or scripted install for users. 
See [Install instructions](Building-Ethereum) for binary and scripted installs.

Developers and community enthusiast are advised to read the [Developers' Guide](Developers-Guide), which contains detailed instructions for manual build from source (on any platform) as well as detailed tips on testing, monitoring, contributing, debugging and submitting pull requests on github.

## Interfaces

* Javascript Console: `geth` can be launched with an interactive console, that provides a javascript runtime environment exposing a javascript API to interact with your node. [Javascript Console API](JavaScript-Console) includes the `web3` javascript Ðapp API as well as an additional admin API. 
* JSON-RPC server: `geth` can be launched with a json-rpc server that exposes the [JSON-RPC API](https://github.com/ethereum/wiki/JSON-RPC)
* [Command line options](Command-Line-Options) documents command line parameters as well as subcommands.

## Basic Use Case Documentation

* [Managing accounts](Managing-your-accounts)
* [Mining](Mining)

**Note** buying and selling ether through exchanges is not discussed here. 

## License

The Ethereum Core Protocol licensed under the [GNU Lesser General Public License](https://www.gnu.org/licenses/lgpl.html). All frontend client software (under [cmd](https://github.com/ethereum/go-ethereum/tree/develop/cmd)) is licensed under the [GNU General Public License](https://www.gnu.org/copyleft/gpl.html).

## Reporting 

Security issues are best sent to security@ethereum.org or shared in PM with devs on one of the channels (see Community and Suppport).

Non-sensitive bug reports are welcome on github. Please always state the version (on master) or commit of your build (if on develop), give as much detail as possible about the situation and the anomaly that occurred. Provide logs or stacktrace if you can.

## Contributors

Ethereum is joint work of ETHDEV and the community.

Name or blame = list of contributors:
* [go-ethereum](https://github.com/ethereum/go-ethereum/graphs/contributors)
* [cpp-ethereum](https://github.com/ethereum/cpp-ethereum/graphs/contributors)
* [web3.js](https://github.com/ethereum/web3.js/graphs/contributors)
* [ethash](https://github.com/ethereum/ethash/graphs/contributors)
* [netstats](https://github.com/cubedro/eth-netstats/graphs/contributors), 
[netintelligence-api](https://github.com/cubedro/eth-net-intelligence-api/graphs/contributors)

## Community and support

### Ethereum on social media

- Main site: https://www.ethereum.org
- Forum: https://forum.ethereum.org
- Github: https://github.com/ethereum
- Blog: https://blog.ethereum.org
- Wiki: http://wiki.ethereum.org
- Twitter: http://twitter.com/ethereumproject
- Reddit: http://reddit.com/r/ethereum
- Meetups: http://ethereum.meetup.com
- Facebook: https://www.facebook.com/ethereumproject
- Youtube: http://www.youtube.com/ethereumproject
- Google+: http://google.com/+EthereumOrgOfficial

### IRC 

IRC Freenode channels:
* `#ethereum`: for general discussion
* `#ethereum-dev`: for development specific questions and discussions
* `##ethereum`: for offtopic and banter
* `#ethereumjs`: for questions related to web3.js and node-ethereum
* `#ethereum-markets`: Trading 
* `#ethereum-mining` Mining
* `#dappdevs`: Dapp developers channel
* `#ethdev`: buildserver etc

[IRC Logs by ZeroGox](https://zerogox.com/bot/log)

### Gitter 

* [go-ethereum Gitter](https://gitter.im/ethereum/go-ethereum)
* [cpp-ethereum Gitter](https://gitter.im/ethereum/cpp-ethereum)
* [web3.js Gitter](https://gitter.im/ethereum/web3.js)
* [ethereum documentation project Gitter](https://gitter.im/ethereum/frontier-guide)

### Forum

- [Forum](https://forum.ethereum.org/categories/go-implementation)

### The ZeroGox Bot

[ZeroGox Bot](https://zerogox.com/bot)

### Dapp developers' mailing list

https://dapplist.net/

### Helpdesk 

On gitter, irc, skype or mail to helpdesk@ethereum.org
