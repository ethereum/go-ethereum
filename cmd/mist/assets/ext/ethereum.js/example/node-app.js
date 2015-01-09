#!/usr/bin/env node

require('es6-promise').polyfill();

var web3 = require("../index.js");

web3.setProvider(new web3.providers.HttpRpcProvider('http://localhost:8080'));

web3.eth.coinbase.then(function(result){
  console.log(result);
  return web3.eth.balanceAt(result);
}).then(function(balance){
  console.log(web3.toDecimal(balance));
}).catch(function(err){
  console.log(err);
});