#!/usr/bin/env node

var web3 = require("../index.js");

web3.setProvider(new web3.providers.HttpSyncProvider('http://localhost:8080'));

var coinbase = web3.eth.coinbase;
console.log(coinbase);

var balance = web3.eth.balanceAt(coinbase);
console.log(balance);

