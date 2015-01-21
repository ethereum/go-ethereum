var web3 = require('./lib/web3');
var ProviderManager = require('./lib/providermanager');
web3.provider = new ProviderManager();
web3.filter = require('./lib/filter');
web3.providers.WebSocketProvider = require('./lib/websocket');
web3.providers.HttpRpcProvider = require('./lib/httprpc');
web3.providers.QtProvider = require('./lib/qt');
web3.providers.HttpSyncProvider = require('./lib/httpsync');
web3.providers.AutoProvider = require('./lib/autoprovider');
web3.eth.contract = require('./lib/contract');
web3.abi = require('./lib/abi');


module.exports = web3;
