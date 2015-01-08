var web3 = require('./lib/web3');
web3.providers.WebSocketProvider = require('./lib/websocket');
web3.providers.HttpRpcProvider = require('./lib/httprpc');
web3.providers.QtProvider = require('./lib/qt');
web3.providers.AutoProvider = require('./lib/autoprovider');
web3.contract = require('./lib/contract');

module.exports = web3;
