# Ethereum JavaScript API

This is the Ethereum compatible JavaScript API using `Promise`s
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
* Include [es6-promise](https://github.com/jakearchibald/es6-promise) or another ES6-Shim if your browser doesn't support ECMAScript 6.

## Usage
Require the library:

	var web3 = require('web3');

Set a provider (QtProvider, WebSocketProvider, HttpRpcProvider)

	var web3.setProvider(new web3.providers.WebSocketProvider('ws://localhost:40404/eth'));

There you go, now you can use it:

```
web3.eth.coinbase.then(function(result){
  console.log(result);
  return web3.eth.balanceAt(result);
}).then(function(balance){
  console.log(web3.toDecimal(balance));
}).catch(function(err){
  console.log(err);
});
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

## Building

```bash (gulp)
npm run-script build
```


### Testing

```bash (mocha)
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
