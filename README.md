# Ethereum JavaScript API

This is the Ethereum compatible JavaScript API using `Promise`s
which implements the [Generic JSON RPC](https://github.com/ethereum/wiki/wiki/Generic-JSON-RPC) spec. It's available on npm as a node module and also for bower and component as an embeddable js

[![Build Status][1]][2] [![dependency status][3]][4] [![dev dependency status][5]][6]

[![browser support](https://ci.testling.com/cubedro/ethereum.js.png)](https://ci.testling.com/cubedro/ethereum.js)

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

## Building

* `gulp build`


### Testing

**Please note this repo is in it's early stage.**

If you'd like to run a WebSocket ethereum node check out
[go-ethereum](https://github.com/ethereum/go-ethereum).

To install ethereum and spawn a node:

```
go get github.com/ethereum/go-ethereum/ethereum
ethereum -ws -loglevel=4
```

[1]: https://travis-ci.org/cubedro/ethereum.js.svg
[2]: https://travis-ci.org/cubedro/ethereum.js
[3]: https://david-dm.org/cubedro/ethereum.js.svg
[4]: https://david-dm.org/cubedro/ethereum.js
[5]: https://david-dm.org/cubedro/ethereum.js/dev-status.svg
[6]: https://david-dm.org/cubedro/ethereum.js#info=devDependencies