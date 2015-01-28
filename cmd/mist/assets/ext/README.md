# Ethereum JavaScript API

This is the Ethereum compatible [JavaScript API](https://github.com/ethereum/wiki/wiki/JavaScript-API)
which implements the [Generic JSON RPC](https://github.com/ethereum/wiki/wiki/Generic-JSON-RPC) spec. It's available on npm as a node module and also for bower and component as an embeddable js

[![NPM version][npm-image]][npm-url] [![Build Status][travis-image]][travis-url] [![dependency status][dep-image]][dep-url] [![dev dependency status][dep-dev-image]][dep-dev-url]

<!-- [![browser support](https://ci.testling.com/ethereum/ethereum.js.png)](https://ci.testling.com/ethereum/ethereum.js) -->

## Installation

### Node.js

    npm install ethereum.js

### For browser
Bower

	bower install ethereum.js

Component

	component install ethereum/ethereum.js

* Include `ethereum.min.js` in your html file.
* Include [bignumber.js](https://github.com/MikeMcl/bignumber.js/)

## Usage
Require the library:

	var web3 = require('web3');

Set a provider (QtProvider, WebSocketProvider, HttpRpcProvider)

	var web3.setProvider(new web3.providers.WebSocketProvider('ws://localhost:40404/eth'));

There you go, now you can use it:

```
var coinbase = web3.eth.coinbase;
var balance = web3.eth.balanceAt(coinbase);
```


For another example see `example/index.html`.

## Contribute!

### Requirements

* Node.js
* npm
* gulp (build)
* mocha (tests)

```bash
sudo apt-get update
sudo apt-get install nodejs
sudo apt-get install npm
sudo apt-get install nodejs-legacy
```

### Building (gulp)

```bash
npm run-script build
```


### Testing (mocha)

```bash
npm test
```

**Please note this repo is in it's early stage.**

If you'd like to run a WebSocket ethereum node check out
[go-ethereum](https://github.com/ethereum/go-ethereum).

To install ethereum and spawn a node:

```
go get github.com/ethereum/go-ethereum/ethereum
ethereum -ws -loglevel=4
```

[npm-image]: https://badge.fury.io/js/ethereum.js.png
[npm-url]: https://npmjs.org/package/ethereum.js
[travis-image]: https://travis-ci.org/ethereum/ethereum.js.svg
[travis-url]: https://travis-ci.org/ethereum/ethereum.js
[dep-image]: https://david-dm.org/ethereum/ethereum.js.svg
[dep-url]: https://david-dm.org/ethereum/ethereum.js
[dep-dev-image]: https://david-dm.org/ethereum/ethereum.js/dev-status.svg
[dep-dev-url]: https://david-dm.org/ethereum/ethereum.js#info=devDependencies

