var web3 = require('./main');
web3.providers.WebSocketProvider = require('./websocket');
web3.providers.HttpRpcProvider = require('./httprpc');
web3.providers.QtProvider = require('./qt');

module.exports = web3;