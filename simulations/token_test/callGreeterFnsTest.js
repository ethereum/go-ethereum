var Web3 = require('web3')
var web3 = new Web3(new Web3.providers.HttpProvider('http://127.0.0.1:8545'))
var sendTxes = require('./call_greeter_fns').sendTxes
var greeter_addr = process.env.GREETER
var proxy_greeter_addr = process.env.PROXYGREETER

sendTxes(web3, greeter_addr, proxy_greeter_addr)
